package quant

import "testing"

func TestComputeComposite_BullishAllFactors(t *testing.T) {
	rsi := 35.0
	priceMA20 := 1.05
	priceMA50 := 1.08
	macd := 3.0
	volume := 1.5
	flow := 50e9

	result := ComputeComposite(CompositeInput{
		SentimentScore: 0.6,
		RSI14:          &rsi,
		PriceVsMA20:    &priceMA20,
		PriceVsMA50:    &priceMA50,
		MACDHist:       &macd,
		TechSignal:     "BUY",
		PriceChangePct: 2.0,
		VolumeRatio:    &volume,
		NetForeignFlow: &flow,
	})

	if result.Signal != "BUY" {
		t.Errorf("expected BUY, got %s (composite: %f)", result.Signal, result.CompositeScore)
	}
	if result.CompositeScore < 0.2 {
		t.Errorf("expected high composite score, got %f", result.CompositeScore)
	}
	if result.Confidence < 0.5 {
		t.Errorf("expected high confidence, got %f", result.Confidence)
	}
}

func TestComputeComposite_BearishAllFactors(t *testing.T) {
	rsi := 75.0
	priceMA20 := 0.92
	priceMA50 := 0.88
	macd := -4.0
	volume := 2.0
	flow := -80e9

	result := ComputeComposite(CompositeInput{
		SentimentScore: -0.5,
		RSI14:          &rsi,
		PriceVsMA20:    &priceMA20,
		PriceVsMA50:    &priceMA50,
		MACDHist:       &macd,
		TechSignal:     "SELL",
		PriceChangePct: -3.0,
		VolumeRatio:    &volume,
		NetForeignFlow: &flow,
	})

	if result.Signal != "SELL" {
		t.Errorf("expected SELL, got %s (composite: %f)", result.Signal, result.CompositeScore)
	}
	if result.CompositeScore > -0.2 {
		t.Errorf("expected negative composite score, got %f", result.CompositeScore)
	}
}

func TestComputeComposite_OverboughtOverride(t *testing.T) {
	rsi := 82.0

	result := ComputeComposite(CompositeInput{
		SentimentScore: 0.7, // Good sentiment but RSI overbought
		RSI14:          &rsi,
		TechSignal:     "SELL",
	})

	// Should NOT be BUY despite good sentiment, because RSI > 75
	if result.Signal == "BUY" {
		t.Error("expected non-BUY signal when RSI > 80 (overbought override)")
	}
}

func TestComputeComposite_NoTechnicalData(t *testing.T) {
	// Only sentiment available
	result := ComputeComposite(CompositeInput{
		SentimentScore: 0.8,
	})

	// Should still produce a valid result with sentiment only
	if result.CompositeScore <= 0 {
		t.Errorf("expected positive composite from positive sentiment, got %f", result.CompositeScore)
	}
}
