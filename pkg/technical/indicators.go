package technical

import "math"

// TechnicalResult holds computed technical indicators for a stock.
type TechnicalResult struct {
	MA20        *float64 `json:"ma20,omitempty"`
	MA50        *float64 `json:"ma50,omitempty"`
	RSI14       *float64 `json:"rsi14,omitempty"`
	MACDLine    *float64 `json:"macd_line,omitempty"`
	MACDSignal  *float64 `json:"macd_signal,omitempty"`
	MACDHist    *float64 `json:"macd_hist,omitempty"`
	LastPrice   float64  `json:"last_price"`
	PriceChange float64  `json:"price_change_pct"` // % change from previous close
	Signal      string   `json:"signal"`           // "BUY", "SELL", "HOLD"
}

// CalcSMA computes Simple Moving Average for the given period.
// prices should be ordered oldest-first.
func CalcSMA(prices []float64, period int) *float64 {
	if len(prices) < period || period <= 0 {
		return nil
	}
	sum := 0.0
	for _, p := range prices[len(prices)-period:] {
		sum += p
	}
	result := sum / float64(period)
	return &result
}

// CalcEMA computes Exponential Moving Average for the given period.
// prices should be ordered oldest-first.
func CalcEMA(prices []float64, period int) *float64 {
	if len(prices) < period || period <= 0 {
		return nil
	}
	multiplier := 2.0 / float64(period+1)
	// Start with SMA of first 'period' prices
	sma := 0.0
	for i := 0; i < period; i++ {
		sma += prices[i]
	}
	sma /= float64(period)

	ema := sma
	for i := period; i < len(prices); i++ {
		ema = (prices[i]-ema)*multiplier + ema
	}
	return &ema
}

// CalcRSI computes the Relative Strength Index using Wilder's smoothing method.
// prices should be ordered oldest-first. Standard period is 14.
func CalcRSI(prices []float64, period int) *float64 {
	if len(prices) < period+1 || period <= 0 {
		return nil
	}

	// Calculate initial average gain and loss
	var avgGain, avgLoss float64
	for i := 1; i <= period; i++ {
		change := prices[i] - prices[i-1]
		if change > 0 {
			avgGain += change
		} else {
			avgLoss += math.Abs(change)
		}
	}
	avgGain /= float64(period)
	avgLoss /= float64(period)

	// Apply Wilder's smoothing for remaining prices
	for i := period + 1; i < len(prices); i++ {
		change := prices[i] - prices[i-1]
		var gain, loss float64
		if change > 0 {
			gain = change
		} else {
			loss = math.Abs(change)
		}
		avgGain = (avgGain*float64(period-1) + gain) / float64(period)
		avgLoss = (avgLoss*float64(period-1) + loss) / float64(period)
	}

	if avgLoss == 0 {
		rsi := 100.0
		return &rsi
	}

	rs := avgGain / avgLoss
	rsi := 100.0 - (100.0 / (1.0 + rs))
	return &rsi
}

// CalcMACD computes the MACD indicator (12, 26, 9).
// Returns macdLine, signalLine, histogram.
func CalcMACD(prices []float64) (*float64, *float64, *float64) {
	if len(prices) < 26 {
		return nil, nil, nil
	}

	ema12 := CalcEMA(prices, 12)
	ema26 := CalcEMA(prices, 26)
	if ema12 == nil || ema26 == nil {
		return nil, nil, nil
	}

	macdLine := *ema12 - *ema26

	// For signal line, we need MACD history
	// Simplified: compute MACD values for last 9+ periods
	var macdValues []float64
	for i := 26; i <= len(prices); i++ {
		subset := prices[:i]
		e12 := CalcEMA(subset, 12)
		e26 := CalcEMA(subset, 26)
		if e12 != nil && e26 != nil {
			macdValues = append(macdValues, *e12-*e26)
		}
	}

	signalLine := CalcEMA(macdValues, 9)
	if signalLine == nil {
		return &macdLine, nil, nil
	}

	hist := macdLine - *signalLine
	return &macdLine, signalLine, &hist
}

// Analyze computes all technical indicators from a slice of closing prices (oldest-first)
// and generates a technical signal.
func Analyze(closePrices []float64) *TechnicalResult {
	if len(closePrices) < 2 {
		return nil
	}

	result := &TechnicalResult{
		LastPrice: closePrices[len(closePrices)-1],
	}

	// Price change
	prevPrice := closePrices[len(closePrices)-2]
	if prevPrice > 0 {
		result.PriceChange = ((result.LastPrice - prevPrice) / prevPrice) * 100
	}

	// Moving averages
	result.MA20 = CalcSMA(closePrices, 20)
	result.MA50 = CalcSMA(closePrices, 50)

	// RSI
	result.RSI14 = CalcRSI(closePrices, 14)

	// MACD
	result.MACDLine, result.MACDSignal, result.MACDHist = CalcMACD(closePrices)

	// Generate technical signal
	result.Signal = computeTechnicalSignal(result)

	return result
}

// computeTechnicalSignal determines BUY/SELL/HOLD based on technical indicators.
func computeTechnicalSignal(r *TechnicalResult) string {
	buyScore := 0
	sellScore := 0

	// RSI signals
	if r.RSI14 != nil {
		if *r.RSI14 < 30 {
			buyScore += 2 // Oversold — strong buy signal
		} else if *r.RSI14 < 40 {
			buyScore += 1
		} else if *r.RSI14 > 80 {
			sellScore += 2 // Heavily overbought — strong sell signal
		} else if *r.RSI14 > 70 {
			sellScore += 1 // Overbought
		}
	}

	// Price vs MA20 (short-term trend)
	if r.MA20 != nil && r.LastPrice > 0 {
		if r.LastPrice > *r.MA20*1.02 {
			buyScore++ // Price above MA20 by >2%
		} else if r.LastPrice < *r.MA20*0.98 {
			sellScore++ // Price below MA20 by >2%
		}
	}

	// MA20 vs MA50 (golden/death cross)
	if r.MA20 != nil && r.MA50 != nil {
		if *r.MA20 > *r.MA50 {
			buyScore++ // Golden cross territory
		} else {
			sellScore++ // Death cross territory
		}
	}

	// MACD histogram
	if r.MACDHist != nil {
		if *r.MACDHist > 0 {
			buyScore++
		} else {
			sellScore++
		}
	}

	if buyScore >= 3 && buyScore > sellScore {
		return "BUY"
	} else if sellScore >= 3 && sellScore > buyScore {
		return "SELL"
	}
	return "HOLD"
}
