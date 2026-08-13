package quant

import "math"

// Weights for composite scoring
const (
	WeightSentiment   = 0.40
	WeightTechnical   = 0.30
	WeightMomentum    = 0.15
	WeightForeignFlow = 0.15
)

// CompositeInput holds all factor inputs for computing the composite quant score.
type CompositeInput struct {
	// Sentiment factor: time-weighted average sentiment score (-1.0 to +1.0)
	SentimentScore float64

	// Technical factors
	RSI14      *float64 // 0-100, nil if not available
	PriceVsMA20 *float64 // ratio: current_price / MA20
	PriceVsMA50 *float64 // ratio: current_price / MA50
	MACDHist   *float64 // MACD histogram value
	TechSignal string   // "BUY", "SELL", "HOLD" from technical analysis

	// Momentum factors
	PriceChangePct float64  // Daily price change %
	VolumeRatio    *float64 // today_volume / avg_volume (> 1 = above avg)

	// Foreign flow factor
	NetForeignFlow *float64 // positive = foreign buying, negative = selling
}

// CompositeResult is the output of the composite scoring.
type CompositeResult struct {
	CompositeScore  float64 `json:"composite_score"`  // -1.0 to +1.0
	SentimentFactor float64 `json:"sentiment_factor"` // -1.0 to +1.0
	TechnicalFactor float64 `json:"technical_factor"` // -1.0 to +1.0
	MomentumFactor  float64 `json:"momentum_factor"`  // -1.0 to +1.0
	FlowFactor      float64 `json:"flow_factor"`      // -1.0 to +1.0
	Signal          string  `json:"signal"`           // "BUY", "SELL", "HOLD"
	Confidence      float64 `json:"confidence"`       // 0.0 to 1.0
}

// ComputeComposite calculates the multi-factor composite quant score.
func ComputeComposite(input CompositeInput) CompositeResult {
	result := CompositeResult{}

	// 1. Sentiment Factor (already -1 to +1)
	result.SentimentFactor = clamp(input.SentimentScore, -1.0, 1.0)

	// 2. Technical Factor
	result.TechnicalFactor = computeTechnicalFactor(input)

	// 3. Momentum Factor
	result.MomentumFactor = computeMomentumFactor(input)

	// 4. Foreign Flow Factor
	result.FlowFactor = computeFlowFactor(input)

	// Weighted composite
	result.CompositeScore = clamp(
		WeightSentiment*result.SentimentFactor+
			WeightTechnical*result.TechnicalFactor+
			WeightMomentum*result.MomentumFactor+
			WeightForeignFlow*result.FlowFactor,
		-1.0, 1.0,
	)

	// Determine signal
	result.Signal = determineSignal(result, input)

	// Confidence based on how many factors agree
	result.Confidence = computeConfidence(result)

	return result
}

func computeTechnicalFactor(input CompositeInput) float64 {
	if input.TechSignal == "" && input.RSI14 == nil {
		return 0 // No technical data available
	}

	score := 0.0
	count := 0.0

	// RSI component (normalized to -1..+1)
	if input.RSI14 != nil {
		// RSI 0-30: bullish (oversold), 30-50: slightly bullish, 50-70: slightly bearish, 70-100: bearish (overbought)
		rsiScore := (50.0 - *input.RSI14) / 50.0 // RSI 0 → +1.0, RSI 50 → 0, RSI 100 → -1.0
		score += rsiScore
		count++
	}

	// Price vs MA20 (trend)
	if input.PriceVsMA20 != nil {
		// >1 means above MA (bullish), <1 means below (bearish)
		maScore := clamp((*input.PriceVsMA20-1.0)*10, -1.0, 1.0) // 10% deviation → full signal
		score += maScore
		count++
	}

	// Price vs MA50 (longer trend)
	if input.PriceVsMA50 != nil {
		maScore := clamp((*input.PriceVsMA50-1.0)*8, -1.0, 1.0)
		score += maScore
		count++
	}

	// MACD histogram
	if input.MACDHist != nil {
		// Normalize: assume typical range is -5 to +5
		macdScore := clamp(*input.MACDHist/5.0, -1.0, 1.0)
		score += macdScore
		count++
	}

	if count == 0 {
		return 0
	}
	return clamp(score/count, -1.0, 1.0)
}

func computeMomentumFactor(input CompositeInput) float64 {
	score := 0.0
	count := 0.0

	// Price change component
	if input.PriceChangePct != 0 {
		// Normalize: 5% daily change → full signal
		priceScore := clamp(input.PriceChangePct/5.0, -1.0, 1.0)
		score += priceScore
		count++
	}

	// Volume ratio component
	if input.VolumeRatio != nil && *input.VolumeRatio > 0 {
		// Above average volume with positive price → bullish momentum
		volSignal := math.Log2(*input.VolumeRatio) // log2(2.0) = 1.0, log2(0.5) = -1.0
		if input.PriceChangePct < 0 {
			volSignal = -math.Abs(volSignal) // High volume + price drop = bearish
		}
		score += clamp(volSignal, -1.0, 1.0)
		count++
	}

	if count == 0 {
		return 0
	}
	return clamp(score/count, -1.0, 1.0)
}

func computeFlowFactor(input CompositeInput) float64 {
	if input.NetForeignFlow == nil {
		return 0
	}
	// Normalize: assume significant flow is ±100 billion IDR
	// Positive = foreign buying = bullish
	return clamp(*input.NetForeignFlow/100e9, -1.0, 1.0)
}

func determineSignal(result CompositeResult, input CompositeInput) string {
	// Strong BUY: composite >= 0.25 AND not overbought
	if result.CompositeScore >= 0.25 {
		if input.RSI14 != nil && *input.RSI14 > 75 {
			return "HOLD" // Overbought, don't buy even if sentiment is good
		}
		return "BUY"
	}

	// Strong SELL: composite <= -0.25 OR RSI > 80 with negative sentiment
	if result.CompositeScore <= -0.25 {
		return "SELL"
	}
	if input.RSI14 != nil && *input.RSI14 > 80 && result.SentimentFactor < 0 {
		return "SELL"
	}

	return "HOLD"
}

func computeConfidence(result CompositeResult) float64 {
	// Count how many factors agree with the final direction
	agreeing := 0
	total := 0
	signDirection := 0.0
	if result.CompositeScore > 0 {
		signDirection = 1.0
	} else if result.CompositeScore < 0 {
		signDirection = -1.0
	}

	factors := []float64{result.SentimentFactor, result.TechnicalFactor, result.MomentumFactor, result.FlowFactor}
	for _, f := range factors {
		if f != 0 {
			total++
			if (f > 0 && signDirection > 0) || (f < 0 && signDirection < 0) {
				agreeing++
			}
		}
	}

	if total == 0 {
		return 0
	}
	return float64(agreeing) / float64(total)
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
