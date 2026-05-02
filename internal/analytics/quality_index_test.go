package analytics

import (
	"testing"

	"extrusion-quality-system/internal/domain"
)

func TestCalculateQualityIndex(t *testing.T) {
	tests := []struct {
		name                     string
		alerts                   []domain.AlertEvent
		expectedValue            float64
		expectedState            domain.QualityState
		expectedParameterPenalty float64
	}{
		{
			name:                     "no alerts returns stable quality index",
			alerts:                   nil,
			expectedValue:            100,
			expectedState:            domain.QualityStateStable,
			expectedParameterPenalty: 0,
		},
		{
			name: "active warning alert decreases quality index",
			alerts: []domain.AlertEvent{
				{
					Level:  domain.AlertLevelWarning,
					Status: domain.AlertStatusActive,
				},
			},
			expectedValue:            85,
			expectedState:            domain.QualityStateStable,
			expectedParameterPenalty: 15,
		},
		{
			name: "acknowledged warning alert still affects quality index",
			alerts: []domain.AlertEvent{
				{
					Level:  domain.AlertLevelWarning,
					Status: domain.AlertStatusAcknowledged,
				},
			},
			expectedValue:            85,
			expectedState:            domain.QualityStateStable,
			expectedParameterPenalty: 15,
		},
		{
			name: "active critical alert decreases quality index stronger than warning",
			alerts: []domain.AlertEvent{
				{
					Level:  domain.AlertLevelCritical,
					Status: domain.AlertStatusActive,
				},
			},
			expectedValue:            70,
			expectedState:            domain.QualityStateWarning,
			expectedParameterPenalty: 30,
		},
		{
			name: "warning and critical alerts are accumulated",
			alerts: []domain.AlertEvent{
				{
					Level:  domain.AlertLevelWarning,
					Status: domain.AlertStatusActive,
				},
				{
					Level:  domain.AlertLevelCritical,
					Status: domain.AlertStatusActive,
				},
			},
			expectedValue:            55,
			expectedState:            domain.QualityStateUnstable,
			expectedParameterPenalty: 45,
		},
		{
			name: "resolved alerts do not affect quality index",
			alerts: []domain.AlertEvent{
				{
					Level:  domain.AlertLevelWarning,
					Status: domain.AlertStatusResolved,
				},
				{
					Level:  domain.AlertLevelCritical,
					Status: domain.AlertStatusResolved,
				},
			},
			expectedValue:            100,
			expectedState:            domain.QualityStateStable,
			expectedParameterPenalty: 0,
		},
		{
			name: "quality index is clamped to zero",
			alerts: []domain.AlertEvent{
				{Level: domain.AlertLevelCritical, Status: domain.AlertStatusActive},
				{Level: domain.AlertLevelCritical, Status: domain.AlertStatusActive},
				{Level: domain.AlertLevelCritical, Status: domain.AlertStatusActive},
				{Level: domain.AlertLevelCritical, Status: domain.AlertStatusActive},
			},
			expectedValue:            0,
			expectedState:            domain.QualityStateCritical,
			expectedParameterPenalty: 120,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := CalculateQualityIndex(tt.alerts)

			if actual.Value != tt.expectedValue {
				t.Fatalf("expected value %.2f, got %.2f", tt.expectedValue, actual.Value)
			}

			if actual.State != tt.expectedState {
				t.Fatalf("expected state %q, got %q", tt.expectedState, actual.State)
			}

			if actual.ParameterPenalty != tt.expectedParameterPenalty {
				t.Fatalf(
					"expected parameterPenalty %.2f, got %.2f",
					tt.expectedParameterPenalty,
					actual.ParameterPenalty,
				)
			}

			if actual.AnomalyPenalty != 0 {
				t.Fatalf("expected anomalyPenalty 0, got %.2f", actual.AnomalyPenalty)
			}

			if actual.CalculatedAt.IsZero() {
				t.Fatalf("expected calculatedAt to be set")
			}
		})
	}
}

func TestCalculateQualityIndexWithAnomalyPenalty(t *testing.T) {
	activeAlerts := []domain.AlertEvent{
		{
			Level:  domain.AlertLevelWarning,
			Status: domain.AlertStatusActive,
		},
	}

	activeAnomalies := []domain.AnomalyEvent{
		{
			Type:   domain.AnomalyTypeJump,
			Status: domain.AlertStatusActive,
		},
		{
			Type:   domain.AnomalyTypeCombinedRisk,
			Status: domain.AlertStatusActive,
		},
	}

	index := CalculateQualityIndex(activeAlerts, activeAnomalies)

	expectedValue := float64(50)

	if index.Value != expectedValue {
		t.Fatalf("expected quality index %.2f, got %.2f", expectedValue, index.Value)
	}

	if index.ParameterPenalty != 15 {
		t.Fatalf("expected parameter penalty 15, got %.2f", index.ParameterPenalty)
	}

	if index.AnomalyPenalty != 35 {
		t.Fatalf("expected anomaly penalty 35, got %.2f", index.AnomalyPenalty)
	}

	if index.State != domain.QualityStateUnstable {
		t.Fatalf("expected state %q, got %q", domain.QualityStateUnstable, index.State)
	}
}

func TestCalculateQualityIndexIgnoresResolvedAnomalies(t *testing.T) {
	activeAlerts := []domain.AlertEvent{}

	anomalies := []domain.AnomalyEvent{
		{
			Type:   domain.AnomalyTypeJump,
			Status: domain.AlertStatusResolved,
		},
		{
			Type:   domain.AnomalyTypeDrift,
			Status: domain.AlertStatusResolved,
		},
	}

	index := CalculateQualityIndex(activeAlerts, anomalies)

	if index.Value != 100 {
		t.Fatalf("expected quality index 100, got %.2f", index.Value)
	}

	if index.AnomalyPenalty != 0 {
		t.Fatalf("expected anomaly penalty 0, got %.2f", index.AnomalyPenalty)
	}
}
