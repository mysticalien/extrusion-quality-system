package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"extrusion-quality-system/internal/analytics"
	"extrusion-quality-system/internal/domain"
)

func LoadQualityWeightsFromEnv() (analytics.QualityWeights, error) {
	weights := analytics.DefaultQualityWeights()

	rawWeights := strings.TrimSpace(os.Getenv("QUALITY_WEIGHTS"))
	if rawWeights == "" {
		return weights, nil
	}

	items := strings.Split(rawWeights, ",")

	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}

		parts := strings.SplitN(item, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid quality weight item %q", item)
		}

		parameterType := domain.ParameterType(strings.TrimSpace(parts[0]))

		weight, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if err != nil {
			return nil, fmt.Errorf("invalid quality weight value for %s: %w", parameterType, err)
		}

		if weight <= 0 {
			return nil, fmt.Errorf("quality weight for %s must be positive", parameterType)
		}

		weights[parameterType] = weight
	}

	return weights, nil
}
