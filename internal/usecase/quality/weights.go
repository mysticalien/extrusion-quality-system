package quality

import (
	"extrusion-quality-system/internal/domain"
	"sort"
)

type Weights map[domain.ParameterType]float64

func DefaultWeights() Weights {
	return Weights{
		domain.ParameterPressure:               1,
		domain.ParameterMoisture:               1,
		domain.ParameterBarrelTemperatureZone1: 1,
		domain.ParameterBarrelTemperatureZone2: 1,
		domain.ParameterBarrelTemperatureZone3: 1,
		domain.ParameterScrewSpeed:             1,
		domain.ParameterDriveLoad:              1,
		domain.ParameterOutletTemperature:      1,
	}
}

func WeightsFromDomain(items []domain.QualityWeight) Weights {
	weights := DefaultWeights()

	for _, item := range items {
		if item.Weight > 0 {
			weights[item.ParameterType] = item.Weight
		}
	}

	return weights
}

func (w Weights) WeightFor(parameterType domain.ParameterType) float64 {
	weight, ok := w[parameterType]
	if !ok || weight <= 0 {
		return 1
	}

	return weight
}

func SortedWeights(items []domain.QualityWeight) []domain.QualityWeight {
	result := append([]domain.QualityWeight(nil), items...)

	sort.Slice(result, func(i, j int) bool {
		return result[i].ParameterType < result[j].ParameterType
	})

	return result
}
