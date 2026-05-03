package simulator

import (
	"fmt"
	"strings"
)

type SimulationMode string

const (
	SimulationModeNormal   SimulationMode = "normal"
	SimulationModeWarning  SimulationMode = "warning"
	SimulationModeCritical SimulationMode = "critical"
	SimulationModeAnomaly  SimulationMode = "anomaly"
)

func parseMode(rawMode string) (SimulationMode, error) {
	mode := SimulationMode(strings.ToLower(strings.TrimSpace(rawMode)))

	switch mode {
	case SimulationModeNormal,
		SimulationModeWarning,
		SimulationModeCritical,
		SimulationModeAnomaly:
		return mode, nil
	default:
		return "", fmt.Errorf("unknown simulator mode %q", rawMode)
	}
}
