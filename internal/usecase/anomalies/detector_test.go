package anomalies

import (
	"context"
	"testing"
	"time"

	"extrusion-quality-system/internal/domain"
)

type fakeTelemetryRepository struct {
	historyByParameter map[domain.ParameterType][]domain.TelemetryReading
}

func (r *fakeTelemetryRepository) Save(
	ctx context.Context,
	reading domain.TelemetryReading,
) (domain.TelemetryReading, error) {
	_ = ctx

	return reading, nil
}

func (r *fakeTelemetryRepository) All(ctx context.Context) ([]domain.TelemetryReading, error) {
	_ = ctx

	return nil, nil
}

func (r *fakeTelemetryRepository) Latest(ctx context.Context) ([]domain.TelemetryReading, error) {
	_ = ctx

	return nil, nil
}

func (r *fakeTelemetryRepository) HistoryByParameter(
	ctx context.Context,
	parameterType domain.ParameterType,
	from time.Time,
	to time.Time,
	limit int,
) ([]domain.TelemetryReading, error) {
	_ = ctx
	_ = from
	_ = to

	history := append([]domain.TelemetryReading(nil), r.historyByParameter[parameterType]...)

	if limit > 0 && len(history) > limit {
		return history[len(history)-limit:], nil
	}

	return history, nil
}

func TestDetectorDetectsPressureJump(t *testing.T) {
	now := time.Now().UTC()

	current := testReading(domain.ParameterPressure, 75, domain.UnitBar, now)

	repository := &fakeTelemetryRepository{
		historyByParameter: map[domain.ParameterType][]domain.TelemetryReading{
			domain.ParameterPressure: {
				testReading(domain.ParameterPressure, 60, domain.UnitBar, now.Add(-time.Second)),
				current,
			},
		},
	}

	detector := NewDetector(repository)

	got, err := detector.Detect(context.Background(), current)
	if err != nil {
		t.Fatalf("Detect returned error: %v", err)
	}

	if !containsAnomalyType(got, domain.AnomalyTypeJump) {
		t.Fatalf("expected jump anomaly, got %+v", got)
	}
}

func TestDetectorDoesNotDetectSmallPressureChangeAsJump(t *testing.T) {
	now := time.Now().UTC()

	current := testReading(domain.ParameterPressure, 66, domain.UnitBar, now)

	repository := &fakeTelemetryRepository{
		historyByParameter: map[domain.ParameterType][]domain.TelemetryReading{
			domain.ParameterPressure: {
				testReading(domain.ParameterPressure, 60, domain.UnitBar, now.Add(-time.Second)),
				current,
			},
		},
	}

	detector := NewDetector(repository)

	got, err := detector.Detect(context.Background(), current)
	if err != nil {
		t.Fatalf("Detect returned error: %v", err)
	}

	if containsAnomalyType(got, domain.AnomalyTypeJump) {
		t.Fatalf("did not expect jump anomaly, got %+v", got)
	}
}

func TestDetectorDetectsMoistureDropJump(t *testing.T) {
	now := time.Now().UTC()

	current := testReading(domain.ParameterMoisture, 23, domain.UnitPercent, now)

	repository := &fakeTelemetryRepository{
		historyByParameter: map[domain.ParameterType][]domain.TelemetryReading{
			domain.ParameterMoisture: {
				testReading(domain.ParameterMoisture, 27, domain.UnitPercent, now.Add(-time.Second)),
				current,
			},
		},
	}

	detector := NewDetector(repository)

	got, err := detector.Detect(context.Background(), current)
	if err != nil {
		t.Fatalf("Detect returned error: %v", err)
	}

	if !containsAnomalyType(got, domain.AnomalyTypeJump) {
		t.Fatalf("expected moisture jump anomaly, got %+v", got)
	}
}

func TestDetectorDoesNotDetectMoistureIncreaseAsExpectedJump(t *testing.T) {
	now := time.Now().UTC()

	current := testReading(domain.ParameterMoisture, 31, domain.UnitPercent, now)

	repository := &fakeTelemetryRepository{
		historyByParameter: map[domain.ParameterType][]domain.TelemetryReading{
			domain.ParameterMoisture: {
				testReading(domain.ParameterMoisture, 27, domain.UnitPercent, now.Add(-time.Second)),
				current,
			},
		},
	}

	detector := NewDetector(repository)

	got, err := detector.Detect(context.Background(), current)
	if err != nil {
		t.Fatalf("Detect returned error: %v", err)
	}

	if containsAnomalyType(got, domain.AnomalyTypeJump) {
		t.Fatalf("did not expect moisture increase jump anomaly, got %+v", got)
	}
}

func TestDetectorDetectsDriveLoadDrift(t *testing.T) {
	now := time.Now().UTC()

	history := []domain.TelemetryReading{
		testReading(domain.ParameterDriveLoad, 50, domain.UnitPercent, now.Add(-4*time.Second)),
		testReading(domain.ParameterDriveLoad, 52, domain.UnitPercent, now.Add(-3*time.Second)),
		testReading(domain.ParameterDriveLoad, 54, domain.UnitPercent, now.Add(-2*time.Second)),
		testReading(domain.ParameterDriveLoad, 56, domain.UnitPercent, now.Add(-time.Second)),
		testReading(domain.ParameterDriveLoad, 58, domain.UnitPercent, now),
	}

	current := history[len(history)-1]

	repository := &fakeTelemetryRepository{
		historyByParameter: map[domain.ParameterType][]domain.TelemetryReading{
			domain.ParameterDriveLoad: history,
		},
	}

	detector := NewDetector(repository)

	got, err := detector.Detect(context.Background(), current)
	if err != nil {
		t.Fatalf("Detect returned error: %v", err)
	}

	if !containsAnomalyType(got, domain.AnomalyTypeDrift) {
		t.Fatalf("expected drift anomaly, got %+v", got)
	}
}

func TestDetectorDoesNotDetectDriftWhenValuesFluctuate(t *testing.T) {
	now := time.Now().UTC()

	history := []domain.TelemetryReading{
		testReading(domain.ParameterDriveLoad, 50, domain.UnitPercent, now.Add(-4*time.Second)),
		testReading(domain.ParameterDriveLoad, 54, domain.UnitPercent, now.Add(-3*time.Second)),
		testReading(domain.ParameterDriveLoad, 52, domain.UnitPercent, now.Add(-2*time.Second)),
		testReading(domain.ParameterDriveLoad, 57, domain.UnitPercent, now.Add(-time.Second)),
		testReading(domain.ParameterDriveLoad, 58, domain.UnitPercent, now),
	}

	current := history[len(history)-1]

	repository := &fakeTelemetryRepository{
		historyByParameter: map[domain.ParameterType][]domain.TelemetryReading{
			domain.ParameterDriveLoad: history,
		},
	}

	detector := NewDetector(repository)

	got, err := detector.Detect(context.Background(), current)
	if err != nil {
		t.Fatalf("Detect returned error: %v", err)
	}

	if containsAnomalyType(got, domain.AnomalyTypeDrift) {
		t.Fatalf("did not expect drift anomaly, got %+v", got)
	}
}

func TestDetectorDoesNotDetectDriftWithShortHistory(t *testing.T) {
	now := time.Now().UTC()

	history := []domain.TelemetryReading{
		testReading(domain.ParameterDriveLoad, 50, domain.UnitPercent, now.Add(-2*time.Second)),
		testReading(domain.ParameterDriveLoad, 54, domain.UnitPercent, now.Add(-time.Second)),
		testReading(domain.ParameterDriveLoad, 58, domain.UnitPercent, now),
	}

	current := history[len(history)-1]

	repository := &fakeTelemetryRepository{
		historyByParameter: map[domain.ParameterType][]domain.TelemetryReading{
			domain.ParameterDriveLoad: history,
		},
	}

	detector := NewDetector(repository)

	got, err := detector.Detect(context.Background(), current)
	if err != nil {
		t.Fatalf("Detect returned error: %v", err)
	}

	if containsAnomalyType(got, domain.AnomalyTypeDrift) {
		t.Fatalf("did not expect drift anomaly with short history, got %+v", got)
	}
}

func TestDetectorDetectsMoistureFallingDrift(t *testing.T) {
	now := time.Now().UTC()

	history := []domain.TelemetryReading{
		testReading(domain.ParameterMoisture, 27, domain.UnitPercent, now.Add(-4*time.Second)),
		testReading(domain.ParameterMoisture, 26.5, domain.UnitPercent, now.Add(-3*time.Second)),
		testReading(domain.ParameterMoisture, 26, domain.UnitPercent, now.Add(-2*time.Second)),
		testReading(domain.ParameterMoisture, 25.5, domain.UnitPercent, now.Add(-time.Second)),
		testReading(domain.ParameterMoisture, 25, domain.UnitPercent, now),
	}

	current := history[len(history)-1]

	repository := &fakeTelemetryRepository{
		historyByParameter: map[domain.ParameterType][]domain.TelemetryReading{
			domain.ParameterMoisture: history,
		},
	}

	detector := NewDetector(repository)

	got, err := detector.Detect(context.Background(), current)
	if err != nil {
		t.Fatalf("Detect returned error: %v", err)
	}

	if !containsAnomalyType(got, domain.AnomalyTypeDrift) {
		t.Fatalf("expected moisture falling drift anomaly, got %+v", got)
	}
}

func TestDetectorDetectsCombinedRisk(t *testing.T) {
	now := time.Now().UTC()

	current := testReading(domain.ParameterPressure, 68, domain.UnitBar, now)

	repository := &fakeTelemetryRepository{
		historyByParameter: map[domain.ParameterType][]domain.TelemetryReading{
			domain.ParameterPressure: {
				testReading(domain.ParameterPressure, 60, domain.UnitBar, now.Add(-4*time.Second)),
				testReading(domain.ParameterPressure, 62, domain.UnitBar, now.Add(-3*time.Second)),
				testReading(domain.ParameterPressure, 64, domain.UnitBar, now.Add(-2*time.Second)),
				testReading(domain.ParameterPressure, 66, domain.UnitBar, now.Add(-time.Second)),
				current,
			},
			domain.ParameterMoisture: {
				testReading(domain.ParameterMoisture, 27, domain.UnitPercent, now.Add(-4*time.Second)),
				testReading(domain.ParameterMoisture, 26.5, domain.UnitPercent, now.Add(-3*time.Second)),
				testReading(domain.ParameterMoisture, 26, domain.UnitPercent, now.Add(-2*time.Second)),
				testReading(domain.ParameterMoisture, 25.5, domain.UnitPercent, now.Add(-time.Second)),
				testReading(domain.ParameterMoisture, 25, domain.UnitPercent, now),
			},
			domain.ParameterDriveLoad: {
				testReading(domain.ParameterDriveLoad, 50, domain.UnitPercent, now.Add(-4*time.Second)),
				testReading(domain.ParameterDriveLoad, 52, domain.UnitPercent, now.Add(-3*time.Second)),
				testReading(domain.ParameterDriveLoad, 54, domain.UnitPercent, now.Add(-2*time.Second)),
				testReading(domain.ParameterDriveLoad, 56, domain.UnitPercent, now.Add(-time.Second)),
				testReading(domain.ParameterDriveLoad, 58, domain.UnitPercent, now),
			},
		},
	}

	detector := NewDetector(repository)

	got, err := detector.Detect(context.Background(), current)
	if err != nil {
		t.Fatalf("Detect returned error: %v", err)
	}

	if !containsAnomalyType(got, domain.AnomalyTypeCombinedRisk) {
		t.Fatalf("expected combined risk anomaly, got %+v", got)
	}
}

func TestDetectorDoesNotDetectCombinedRiskWhenOneFactorIsMissing(t *testing.T) {
	now := time.Now().UTC()

	current := testReading(domain.ParameterPressure, 68, domain.UnitBar, now)

	repository := &fakeTelemetryRepository{
		historyByParameter: map[domain.ParameterType][]domain.TelemetryReading{
			domain.ParameterPressure: {
				testReading(domain.ParameterPressure, 60, domain.UnitBar, now.Add(-4*time.Second)),
				testReading(domain.ParameterPressure, 62, domain.UnitBar, now.Add(-3*time.Second)),
				testReading(domain.ParameterPressure, 64, domain.UnitBar, now.Add(-2*time.Second)),
				testReading(domain.ParameterPressure, 66, domain.UnitBar, now.Add(-time.Second)),
				current,
			},
			domain.ParameterMoisture: {
				testReading(domain.ParameterMoisture, 27, domain.UnitPercent, now.Add(-4*time.Second)),
				testReading(domain.ParameterMoisture, 26.5, domain.UnitPercent, now.Add(-3*time.Second)),
				testReading(domain.ParameterMoisture, 26, domain.UnitPercent, now.Add(-2*time.Second)),
				testReading(domain.ParameterMoisture, 25.5, domain.UnitPercent, now.Add(-time.Second)),
				testReading(domain.ParameterMoisture, 25, domain.UnitPercent, now),
			},
			domain.ParameterDriveLoad: {
				testReading(domain.ParameterDriveLoad, 50, domain.UnitPercent, now.Add(-4*time.Second)),
				testReading(domain.ParameterDriveLoad, 50.5, domain.UnitPercent, now.Add(-3*time.Second)),
				testReading(domain.ParameterDriveLoad, 51, domain.UnitPercent, now.Add(-2*time.Second)),
				testReading(domain.ParameterDriveLoad, 51.5, domain.UnitPercent, now.Add(-time.Second)),
				testReading(domain.ParameterDriveLoad, 52, domain.UnitPercent, now),
			},
		},
	}

	detector := NewDetector(repository)

	got, err := detector.Detect(context.Background(), current)
	if err != nil {
		t.Fatalf("Detect returned error: %v", err)
	}

	if containsAnomalyType(got, domain.AnomalyTypeCombinedRisk) {
		t.Fatalf("did not expect combined risk anomaly, got %+v", got)
	}
}

func testReading(
	parameterType domain.ParameterType,
	value float64,
	unit domain.Unit,
	measuredAt time.Time,
) domain.TelemetryReading {
	return domain.TelemetryReading{
		ParameterType: parameterType,
		Value:         value,
		Unit:          unit,
		SourceID:      "test-source",
		MeasuredAt:    measuredAt,
		CreatedAt:     measuredAt,
	}
}

func containsAnomalyType(
	anomalies []domain.AnomalyEvent,
	anomalyType domain.AnomalyType,
) bool {
	for _, anomaly := range anomalies {
		if anomaly.Type == anomalyType {
			return true
		}
	}

	return false
}
