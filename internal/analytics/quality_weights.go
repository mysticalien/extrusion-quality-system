package analytics

import (
	"sort"

	"extrusion-quality-system/internal/domain"
)

type QualityWeights map[domain.ParameterType]float64

func DefaultQualityWeights() QualityWeights {
	return QualityWeights{
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

func QualityWeightsFromDomain(items []domain.QualityWeight) QualityWeights {
	weights := DefaultQualityWeights()

	for _, item := range items {
		if item.Weight > 0 {
			weights[item.ParameterType] = item.Weight
		}
	}

	return weights
}

func (w QualityWeights) WeightFor(parameterType domain.ParameterType) float64 {
	weight, ok := w[parameterType]
	if !ok || weight <= 0 {
		return 1
	}

	return weight
}

func SortedQualityWeights(items []domain.QualityWeight) []domain.QualityWeight {
	result := append([]domain.QualityWeight(nil), items...)

	sort.Slice(result, func(i, j int) bool {
		return result[i].ParameterType < result[j].ParameterType
	})

	return result
}
