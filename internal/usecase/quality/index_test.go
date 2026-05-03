package quality

import (
	"testing"

	"extrusion-quality-system/internal/domain"
)

func TestCalculateIndexWithoutPenalties(t *testing.T) {
	got := CalculateIndex(nil, DefaultWeights())

	if got.Value != 100 {
		t.Fatalf("Value = %v, want 100", got.Value)
	}

	if got.ParameterPenalty != 0 {
		t.Fatalf("ParameterPenalty = %v, want 0", got.ParameterPenalty)
	}

	if got.AnomalyPenalty != 0 {
		t.Fatalf("AnomalyPenalty = %v, want 0", got.AnomalyPenalty)
	}

	if got.State != domain.QualityStateStable {
		t.Fatalf("State = %q, want %q", got.State, domain.QualityStateStable)
	}
}

func TestCalculateIndexWarningAlertDecreasesIndex(t *testing.T) {
	alerts := []domain.AlertEvent{
		{
			ParameterType: domain.ParameterPressure,
			Level:         domain.AlertLevelWarning,
			Status:        domain.AlertStatusActive,
		},
	}

	got := CalculateIndex(alerts, DefaultWeights())

	if got.Value >= 100 {
		t.Fatalf("Value = %v, want less than 100", got.Value)
	}

	if got.ParameterPenalty <= 0 {
		t.Fatalf("ParameterPenalty = %v, want positive", got.ParameterPenalty)
	}
}

func TestCalculateIndexCriticalAlertDecreasesMoreThanWarning(t *testing.T) {
	warningAlerts := []domain.AlertEvent{
		{
			ParameterType: domain.ParameterPressure,
			Level:         domain.AlertLevelWarning,
			Status:        domain.AlertStatusActive,
		},
	}

	criticalAlerts := []domain.AlertEvent{
		{
			ParameterType: domain.ParameterPressure,
			Level:         domain.AlertLevelCritical,
			Status:        domain.AlertStatusActive,
		},
	}

	warningIndex := CalculateIndex(warningAlerts, DefaultWeights())
	criticalIndex := CalculateIndex(criticalAlerts, DefaultWeights())

	if criticalIndex.Value >= warningIndex.Value {
		t.Fatalf(
			"critical index = %v, warning index = %v, want critical lower",
			criticalIndex.Value,
			warningIndex.Value,
		)
	}
}

func TestCalculateIndexParameterWeightAffectsPenalty(t *testing.T) {
	alerts := []domain.AlertEvent{
		{
			ParameterType: domain.ParameterPressure,
			Level:         domain.AlertLevelWarning,
			Status:        domain.AlertStatusActive,
		},
	}

	defaultIndex := CalculateIndex(alerts, DefaultWeights())

	weights := DefaultWeights()
	weights[domain.ParameterPressure] = 2

	weightedIndex := CalculateIndex(alerts, weights)

	if weightedIndex.Value >= defaultIndex.Value {
		t.Fatalf(
			"weighted index = %v, default index = %v, want weighted lower",
			weightedIndex.Value,
			defaultIndex.Value,
		)
	}

	if weightedIndex.ParameterPenalty <= defaultIndex.ParameterPenalty {
		t.Fatalf(
			"weighted penalty = %v, default penalty = %v, want weighted greater",
			weightedIndex.ParameterPenalty,
			defaultIndex.ParameterPenalty,
		)
	}
}

func TestCalculateIndexAcknowledgedAlertStillAffectsIndex(t *testing.T) {
	alerts := []domain.AlertEvent{
		{
			ParameterType: domain.ParameterPressure,
			Level:         domain.AlertLevelWarning,
			Status:        domain.AlertStatusAcknowledged,
		},
	}

	got := CalculateIndex(alerts, DefaultWeights())

	if got.Value >= 100 {
		t.Fatalf("Value = %v, want less than 100", got.Value)
	}
}

func TestCalculateIndexResolvedAlertDoesNotAffectIndex(t *testing.T) {
	alerts := []domain.AlertEvent{
		{
			ParameterType: domain.ParameterPressure,
			Level:         domain.AlertLevelCritical,
			Status:        domain.AlertStatusResolved,
		},
	}

	got := CalculateIndex(alerts, DefaultWeights())

	if got.Value != 100 {
		t.Fatalf("Value = %v, want 100", got.Value)
	}

	if got.ParameterPenalty != 0 {
		t.Fatalf("ParameterPenalty = %v, want 0", got.ParameterPenalty)
	}
}

func TestCalculateIndexJumpAnomalyDecreasesIndex(t *testing.T) {
	anomalies := []domain.AnomalyEvent{
		{
			Type:   domain.AnomalyTypeJump,
			Status: domain.AlertStatusActive,
		},
	}

	got := CalculateIndex(nil, DefaultWeights(), anomalies)

	if got.Value >= 100 {
		t.Fatalf("Value = %v, want less than 100", got.Value)
	}

	if got.AnomalyPenalty <= 0 {
		t.Fatalf("AnomalyPenalty = %v, want positive", got.AnomalyPenalty)
	}
}

func TestCalculateIndexDriftAnomalyDecreasesMoreThanJump(t *testing.T) {
	jumpAnomalies := []domain.AnomalyEvent{
		{
			Type:   domain.AnomalyTypeJump,
			Status: domain.AlertStatusActive,
		},
	}

	driftAnomalies := []domain.AnomalyEvent{
		{
			Type:   domain.AnomalyTypeDrift,
			Status: domain.AlertStatusActive,
		},
	}

	jumpIndex := CalculateIndex(nil, DefaultWeights(), jumpAnomalies)
	driftIndex := CalculateIndex(nil, DefaultWeights(), driftAnomalies)

	if driftIndex.Value >= jumpIndex.Value {
		t.Fatalf(
			"drift index = %v, jump index = %v, want drift lower",
			driftIndex.Value,
			jumpIndex.Value,
		)
	}
}

func TestCalculateIndexCombinedRiskDecreasesMoreThanDrift(t *testing.T) {
	driftAnomalies := []domain.AnomalyEvent{
		{
			Type:   domain.AnomalyTypeDrift,
			Status: domain.AlertStatusActive,
		},
	}

	combinedRiskAnomalies := []domain.AnomalyEvent{
		{
			Type:   domain.AnomalyTypeCombinedRisk,
			Status: domain.AlertStatusActive,
		},
	}

	driftIndex := CalculateIndex(nil, DefaultWeights(), driftAnomalies)
	combinedRiskIndex := CalculateIndex(nil, DefaultWeights(), combinedRiskAnomalies)

	if combinedRiskIndex.Value >= driftIndex.Value {
		t.Fatalf(
			"combined risk index = %v, drift index = %v, want combined risk lower",
			combinedRiskIndex.Value,
			driftIndex.Value,
		)
	}
}

func TestCalculateIndexResolvedAnomalyDoesNotAffectIndex(t *testing.T) {
	anomalies := []domain.AnomalyEvent{
		{
			Type:   domain.AnomalyTypeCombinedRisk,
			Status: domain.AlertStatusResolved,
		},
	}

	got := CalculateIndex(nil, DefaultWeights(), anomalies)

	if got.Value != 100 {
		t.Fatalf("Value = %v, want 100", got.Value)
	}

	if got.AnomalyPenalty != 0 {
		t.Fatalf("AnomalyPenalty = %v, want 0", got.AnomalyPenalty)
	}
}

func TestCalculateIndexDoesNotGoBelowZero(t *testing.T) {
	alerts := make([]domain.AlertEvent, 0)

	for i := 0; i < 20; i++ {
		alerts = append(alerts, domain.AlertEvent{
			ParameterType: domain.ParameterPressure,
			Level:         domain.AlertLevelCritical,
			Status:        domain.AlertStatusActive,
		})
	}

	anomalies := []domain.AnomalyEvent{
		{
			Type:   domain.AnomalyTypeCombinedRisk,
			Status: domain.AlertStatusActive,
		},
	}

	got := CalculateIndex(alerts, DefaultWeights(), anomalies)

	if got.Value < 0 {
		t.Fatalf("Value = %v, want not less than 0", got.Value)
	}
}

func TestCalculateIndexDoesNotGoAboveOneHundred(t *testing.T) {
	got := CalculateIndex(nil, nil)

	if got.Value > 100 {
		t.Fatalf("Value = %v, want not greater than 100", got.Value)
	}
}

func TestWeightsFromDomainOverridesDefaultWeights(t *testing.T) {
	weights := WeightsFromDomain([]domain.QualityWeight{
		{
			ParameterType: domain.ParameterPressure,
			Weight:        2.5,
		},
	})

	got := weights.WeightFor(domain.ParameterPressure)

	if got != 2.5 {
		t.Fatalf("WeightFor(pressure) = %v, want 2.5", got)
	}
}

func TestWeightsFromDomainIgnoresInvalidWeight(t *testing.T) {
	weights := WeightsFromDomain([]domain.QualityWeight{
		{
			ParameterType: domain.ParameterPressure,
			Weight:        -1,
		},
	})

	got := weights.WeightFor(domain.ParameterPressure)

	if got != 1 {
		t.Fatalf("WeightFor(pressure) = %v, want default 1", got)
	}
}
