package technical

import (
	"math"
	"testing"
)

func approxEqual(a, b, tolerance float64) bool {
	return math.Abs(a-b) < tolerance
}

func TestCalcSMA(t *testing.T) {
	prices := []float64{10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}

	// SMA(5) of last 5 prices: 16+17+18+19+20 = 90/5 = 18
	result := CalcSMA(prices, 5)
	if result == nil {
		t.Fatal("expected non-nil SMA(5)")
	}
	if !approxEqual(*result, 18.0, 0.001) {
		t.Errorf("SMA(5) = %f, want 18.0", *result)
	}

	// Not enough data
	result = CalcSMA(prices[:2], 5)
	if result != nil {
		t.Error("expected nil SMA when not enough data")
	}
}

func TestCalcRSI(t *testing.T) {
	// Generate uptrending prices (should give RSI > 50)
	prices := make([]float64, 30)
	for i := range prices {
		prices[i] = 100 + float64(i)*0.5
	}

	result := CalcRSI(prices, 14)
	if result == nil {
		t.Fatal("expected non-nil RSI")
	}
	if *result < 50 {
		t.Errorf("RSI for uptrend = %f, expected > 50", *result)
	}

	// Generate downtrending prices (should give RSI < 50)
	for i := range prices {
		prices[i] = 200 - float64(i)*0.5
	}
	result = CalcRSI(prices, 14)
	if result == nil {
		t.Fatal("expected non-nil RSI")
	}
	if *result > 50 {
		t.Errorf("RSI for downtrend = %f, expected < 50", *result)
	}
}

func TestCalcMACD(t *testing.T) {
	prices := make([]float64, 40)
	for i := range prices {
		prices[i] = 100 + float64(i)*0.3
	}

	macdLine, signalLine, hist := CalcMACD(prices)
	if macdLine == nil {
		t.Fatal("expected non-nil MACD line")
	}
	if signalLine == nil {
		t.Fatal("expected non-nil signal line")
	}
	if hist == nil {
		t.Fatal("expected non-nil histogram")
	}

	// For steady uptrend, MACD should be positive
	if *macdLine < 0 {
		t.Errorf("MACD for uptrend = %f, expected > 0", *macdLine)
	}
}

func TestAnalyze(t *testing.T) {
	// Insufficient data
	result := Analyze([]float64{100})
	if result != nil {
		t.Error("expected nil for single price")
	}

	// Enough data for basic analysis
	prices := make([]float64, 60)
	for i := range prices {
		prices[i] = 1000 + float64(i)*5
	}
	result = Analyze(prices)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.MA20 == nil {
		t.Error("expected non-nil MA20")
	}
	if result.MA50 == nil {
		t.Error("expected non-nil MA50")
	}
	if result.RSI14 == nil {
		t.Error("expected non-nil RSI14")
	}
	if result.Signal == "" {
		t.Error("expected non-empty signal")
	}
}
