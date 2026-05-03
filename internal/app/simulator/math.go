package simulator

import "math/rand"

func randomInRange(random *rand.Rand, minValue float64, maxValue float64) float64 {
	return minValue + random.Float64()*(maxValue-minValue)
}

func clamp(value float64, minValue float64, maxValue float64) float64 {
	if value < minValue {
		return minValue
	}

	if value > maxValue {
		return maxValue
	}

	return value
}

func round2(value float64) float64 {
	return float64(int(value*100)) / 100
}
