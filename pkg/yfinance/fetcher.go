package yfinance

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"sync"
	"time"

	"github.com/JavierZam/quant-indonesia-scraping/domain"
)

// yahooResponse represents the Yahoo Finance v8 chart JSON response.
type yahooResponse struct {
	Chart struct {
		Result []struct {
			Meta struct {
				Currency string  `json:"currency"`
				Symbol   string  `json:"symbol"`
				Price    float64 `json:"regularMarketPrice"`
			} `json:"meta"`
			Timestamp []int64 `json:"timestamp"`
			Indicators struct {
				Quote []struct {
					Open   []*float64 `json:"open"`
					High   []*float64 `json:"high"`
					Low    []*float64 `json:"low"`
					Close  []*float64 `json:"close"`
					Volume []*int64   `json:"volume"`
				} `json:"quote"`
			} `json:"indicators"`
		} `json:"result"`
		Error *struct {
			Code        string `json:"code"`
			Description string `json:"description"`
		} `json:"error"`
	} `json:"chart"`
}

// Fetcher fetches daily historical stock prices from Yahoo Finance.
type Fetcher struct {
	httpClient *http.Client
	crumb      string
	crumbMu    sync.Mutex
	crumbTime  time.Time
}

// NewFetcher creates a new Yahoo Finance fetcher instance.
func NewFetcher() *Fetcher {
	jar, _ := cookiejar.New(nil)
	return &Fetcher{
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
			Jar:     jar,
		},
	}
}

const userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"

// ensureCrumb obtains a Yahoo Finance crumb+cookie for authenticated API calls.
// Crumbs are cached for 30 minutes.
func (f *Fetcher) ensureCrumb(ctx context.Context) (string, error) {
	f.crumbMu.Lock()
	defer f.crumbMu.Unlock()

	// Return cached crumb if still fresh
	if f.crumb != "" && time.Since(f.crumbTime) < 30*time.Minute {
		return f.crumb, nil
	}

	// Step 1: Hit consent page to get cookies
	consentReq, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://fc.yahoo.com", nil)
	if err != nil {
		return "", fmt.Errorf("creating consent request: %w", err)
	}
	consentReq.Header.Set("User-Agent", userAgent)
	consentResp, err := f.httpClient.Do(consentReq)
	if err != nil {
		return "", fmt.Errorf("fetching consent cookies: %w", err)
	}
	io.Copy(io.Discard, consentResp.Body)
	consentResp.Body.Close()

	// Step 2: Get crumb using the cookies
	crumbReq, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://query2.finance.yahoo.com/v1/test/getcrumb", nil)
	if err != nil {
		return "", fmt.Errorf("creating crumb request: %w", err)
	}
	crumbReq.Header.Set("User-Agent", userAgent)
	crumbResp, err := f.httpClient.Do(crumbReq)
	if err != nil {
		return "", fmt.Errorf("fetching crumb: %w", err)
	}
	defer crumbResp.Body.Close()

	if crumbResp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("crumb endpoint returned status %d", crumbResp.StatusCode)
	}

	body, err := io.ReadAll(crumbResp.Body)
	if err != nil {
		return "", fmt.Errorf("reading crumb body: %w", err)
	}

	f.crumb = strings.TrimSpace(string(body))
	f.crumbTime = time.Now()
	return f.crumb, nil
}

// FetchPrices retrieves daily OHLCV prices for an IDX ticker symbol.
func (f *Fetcher) FetchPrices(ctx context.Context, symbol string, rangeStr string) ([]domain.StockPrice, error) {
	if rangeStr == "" {
		rangeStr = "1y"
	}
	switch rangeStr {
	case "1mo", "6mo", "1y", "5y", "max":
		// valid
	default:
		return nil, fmt.Errorf("invalid range parameter: %s", rangeStr)
	}

	ticker := strings.TrimSpace(strings.ToUpper(symbol))
	if !strings.HasPrefix(ticker, "^") && !strings.HasSuffix(ticker, ".JK") {
		ticker += ".JK"
	}

	url := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s?interval=1d&range=%s", ticker, rangeStr)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request for %s: %w", ticker, err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching Yahoo Finance data for %s: %w", ticker, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("yahoo finance returned status %d for %s", resp.StatusCode, ticker)
	}

	var yr yahooResponse
	if err := json.NewDecoder(resp.Body).Decode(&yr); err != nil {
		return nil, fmt.Errorf("decoding yahoo response for %s: %w", ticker, err)
	}

	if yr.Chart.Error != nil {
		return nil, fmt.Errorf("yahoo finance API error: %s - %s", yr.Chart.Error.Code, yr.Chart.Error.Description)
	}

	if len(yr.Chart.Result) == 0 {
		return nil, fmt.Errorf("empty chart result for %s", ticker)
	}

	res := yr.Chart.Result[0]
	if len(res.Timestamp) == 0 || len(res.Indicators.Quote) == 0 {
		return nil, nil // No data available
	}

	quote := res.Indicators.Quote[0]
	var prices []domain.StockPrice

	cleanSymbol := ticker
	if strings.HasSuffix(ticker, ".JK") {
		cleanSymbol = strings.TrimSuffix(ticker, ".JK")
	} else if strings.HasPrefix(ticker, "^") {
		// Map index symbols to readable names
		switch ticker {
		case "^JKSE":
			cleanSymbol = "IHSG"
		default:
			cleanSymbol = strings.TrimPrefix(ticker, "^")
		}
	}

	for i, ts := range res.Timestamp {
		if i >= len(quote.Close) || quote.Close[i] == nil {
			continue
		}

		closePrice := *quote.Close[i]
		if closePrice <= 0 {
			continue
		}

		t := time.Unix(ts, 0).UTC()
		dateStr := t.Format("2006-01-02")

		p := domain.StockPrice{
			Symbol:     cleanSymbol,
			Date:       dateStr,
			ClosePrice: closePrice,
		}

		if i < len(quote.Open) && quote.Open[i] != nil {
			p.OpenPrice = quote.Open[i]
		}
		if i < len(quote.High) && quote.High[i] != nil {
			p.HighPrice = quote.High[i]
		}
		if i < len(quote.Low) && quote.Low[i] != nil {
			p.LowPrice = quote.Low[i]
		}
		if i < len(quote.Volume) && quote.Volume[i] != nil {
			p.Volume = quote.Volume[i]
		}

		prices = append(prices, p)
	}

	return prices, nil
}

type YahooProfile struct {
	LongName            string
	Sector              string
	Industry            string
	LongBusinessSummary string
	City                string
	Country             string
	Website             string
	FullTimeEmployees   int
	MarketCap           int64
	SharesOutstanding   int64
	FloatShares         int64
	TrailingPE          float64
	PriceToBook         float64
	TrailingEps         float64
	DividendYield       float64
	FiftyTwoWeekHigh    float64
	FiftyTwoWeekLow     float64
	TotalRevenue        int64
	NetIncome           int64
	TotalDebt           int64
	TotalAssets         int64
	ReturnOnEquity      float64
	DebtToEquity        float64
}

// FetchProfile retrieves stock profile and key statistics from Yahoo Finance.
func (f *Fetcher) FetchProfile(ctx context.Context, symbol string) (*YahooProfile, error) {
	ticker := strings.TrimSpace(strings.ToUpper(symbol))
	if !strings.HasPrefix(ticker, "^") && !strings.HasSuffix(ticker, ".JK") {
		ticker += ".JK"
	}

	// Get crumb for authenticated API access
	crumb, err := f.ensureCrumb(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting crumb for %s: %w", ticker, err)
	}

	url := fmt.Sprintf("https://query2.finance.yahoo.com/v10/finance/quoteSummary/%s?modules=summaryProfile,defaultKeyStatistics,financialData,price&crumb=%s", ticker, crumb)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request for %s: %w", ticker, err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching profile data for %s: %w", ticker, err)
	}
	defer resp.Body.Close()

	// If 401, invalidate crumb cache and retry once
	if resp.StatusCode == http.StatusUnauthorized {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		f.crumbMu.Lock()
		f.crumb = ""
		f.crumbTime = time.Time{}
		f.crumbMu.Unlock()

		crumb, err = f.ensureCrumb(ctx)
		if err != nil {
			return nil, fmt.Errorf("refreshing crumb for %s: %w", ticker, err)
		}

		url = fmt.Sprintf("https://query2.finance.yahoo.com/v10/finance/quoteSummary/%s?modules=summaryProfile,defaultKeyStatistics,financialData,price&crumb=%s", ticker, crumb)
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("creating retry request for %s: %w", ticker, err)
		}
		req.Header.Set("User-Agent", userAgent)

		resp, err = f.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("retry fetching profile for %s: %w", ticker, err)
		}
		defer resp.Body.Close()
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("yahoo finance returned status %d for %s", resp.StatusCode, ticker)
	}


	var data struct {
		QuoteSummary struct {
			Result []struct {
				SummaryProfile struct {
					Sector              string `json:"sector"`
					Industry            string `json:"industry"`
					LongBusinessSummary string `json:"longBusinessSummary"`
					City                string `json:"city"`
					Country             string `json:"country"`
					Website             string `json:"website"`
					FullTimeEmployees   int    `json:"fullTimeEmployees"`
				} `json:"summaryProfile"`
				Price struct {
					LongName     string `json:"longName"`
					MarketCap    struct{ Raw int64 `json:"raw"` } `json:"marketCap"`
				} `json:"price"`
				DefaultKeyStatistics struct {
					SharesOutstanding struct{ Raw int64 `json:"raw"` } `json:"sharesOutstanding"`
					FloatShares       struct{ Raw int64 `json:"raw"` } `json:"floatShares"`
					TrailingEps       struct{ Raw float64 `json:"raw"` } `json:"trailingEps"`
					FiftyTwoWeekHigh  struct{ Raw float64 `json:"raw"` } `json:"52WeekHigh"`
					FiftyTwoWeekLow   struct{ Raw float64 `json:"raw"` } `json:"52WeekLow"`
				} `json:"defaultKeyStatistics"`
				FinancialData struct {
					TrailingPE          struct{ Raw float64 `json:"raw"` } `json:"trailingPE"`
					PriceToBook         struct{ Raw float64 `json:"raw"` } `json:"priceToBook"`
					DividendYield       struct{ Raw float64 `json:"raw"` } `json:"dividendYield"`
					TotalRevenue        struct{ Raw int64 `json:"raw"` } `json:"totalRevenue"`
					NetIncomeToCommon   struct{ Raw int64 `json:"raw"` } `json:"netIncomeToCommon"`
					TotalDebt           struct{ Raw int64 `json:"raw"` } `json:"totalDebt"`
					TotalAssets         struct{ Raw int64 `json:"raw"` } `json:"totalAssets"`
					ReturnOnEquity      struct{ Raw float64 `json:"raw"` } `json:"returnOnEquity"`
					DebtToEquity        struct{ Raw float64 `json:"raw"` } `json:"debtToEquity"`
				} `json:"financialData"`
			} `json:"result"`
			Error *struct {
				Code        string `json:"code"`
				Description string `json:"description"`
			} `json:"error"`
		} `json:"quoteSummary"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("decoding profile response for %s: %w", ticker, err)
	}

	if data.QuoteSummary.Error != nil {
		return nil, fmt.Errorf("yahoo finance API error: %s - %s", data.QuoteSummary.Error.Code, data.QuoteSummary.Error.Description)
	}

	if len(data.QuoteSummary.Result) == 0 {
		return nil, fmt.Errorf("empty profile result for %s", ticker)
	}

	res := data.QuoteSummary.Result[0]
	profile := &YahooProfile{
		LongName:            res.Price.LongName,
		Sector:              res.SummaryProfile.Sector,
		Industry:            res.SummaryProfile.Industry,
		LongBusinessSummary: res.SummaryProfile.LongBusinessSummary,
		City:                res.SummaryProfile.City,
		Country:             res.SummaryProfile.Country,
		Website:             res.SummaryProfile.Website,
		FullTimeEmployees:   res.SummaryProfile.FullTimeEmployees,
		MarketCap:           res.Price.MarketCap.Raw,
		SharesOutstanding:   res.DefaultKeyStatistics.SharesOutstanding.Raw,
		FloatShares:         res.DefaultKeyStatistics.FloatShares.Raw,
		TrailingPE:          res.FinancialData.TrailingPE.Raw,
		PriceToBook:         res.FinancialData.PriceToBook.Raw,
		TrailingEps:         res.DefaultKeyStatistics.TrailingEps.Raw,
		DividendYield:       res.FinancialData.DividendYield.Raw,
		FiftyTwoWeekHigh:    res.DefaultKeyStatistics.FiftyTwoWeekHigh.Raw,
		FiftyTwoWeekLow:     res.DefaultKeyStatistics.FiftyTwoWeekLow.Raw,
		TotalRevenue:        res.FinancialData.TotalRevenue.Raw,
		NetIncome:           res.FinancialData.NetIncomeToCommon.Raw,
		TotalDebt:           res.FinancialData.TotalDebt.Raw,
		TotalAssets:         res.FinancialData.TotalAssets.Raw,
		ReturnOnEquity:      res.FinancialData.ReturnOnEquity.Raw,
		DebtToEquity:        res.FinancialData.DebtToEquity.Raw,
	}

	return profile, nil
}
