package telemetry

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"extrusion-quality-system/internal/domain"
)

func TestProcessNormalTelemetrySavesReadingWithoutAlertAndQualityIsStable(t *testing.T) {
	ctx := context.Background()

	telemetryRepository := &fakeTelemetryRepository{}
	alertRepository := &fakeAlertRepository{}
	qualityRepository := &fakeQualityRepository{}
	setpointRepository := newFakeSetpointRepository()
	anomalyRepository := &fakeAnomalyRepository{}

	service := NewService(
		slog.Default(),
		telemetryRepository,
		alertRepository,
		qualityRepository,
		setpointRepository,
		anomalyRepository,
	)

	result, err := service.Process(ctx, Input{
		ParameterType: domain.ParameterPressure,
		Value:         60,
		Unit:          domain.UnitBar,
		SourceID:      "test-source",
		MeasuredAt:    time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("Process returned error: %v", err)
	}

	if !result.Accepted {
		t.Fatal("expected accepted result")
	}

	if result.State != domain.ParameterStateNormal {
		t.Fatalf("State = %q, want %q", result.State, domain.ParameterStateNormal)
	}

	if len(telemetryRepository.readings) != 1 {
		t.Fatalf("saved readings = %d, want 1", len(telemetryRepository.readings))
	}

	if len(alertRepository.alerts) != 0 {
		t.Fatalf("alerts = %d, want 0", len(alertRepository.alerts))
	}

	if len(qualityRepository.indexes) != 1 {
		t.Fatalf("quality indexes = %d, want 1", len(qualityRepository.indexes))
	}

	if result.QualityIndex != 100 {
		t.Fatalf("QualityIndex = %v, want 100", result.QualityIndex)
	}
}

func TestProcessWarningTelemetryCreatesAlertAndDecreasesQualityIndex(t *testing.T) {
	ctx := context.Background()

	telemetryRepository := &fakeTelemetryRepository{}
	alertRepository := &fakeAlertRepository{}
	qualityRepository := &fakeQualityRepository{}
	setpointRepository := newFakeSetpointRepository()
	anomalyRepository := &fakeAnomalyRepository{}

	service := NewService(
		slog.Default(),
		telemetryRepository,
		alertRepository,
		qualityRepository,
		setpointRepository,
		anomalyRepository,
	)

	result, err := service.Process(ctx, Input{
		ParameterType: domain.ParameterPressure,
		Value:         80,
		Unit:          domain.UnitBar,
		SourceID:      "test-source",
		MeasuredAt:    time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("Process returned error: %v", err)
	}

	if result.State != domain.ParameterStateWarning {
		t.Fatalf("State = %q, want %q", result.State, domain.ParameterStateWarning)
	}

	if !result.AlertCreated {
		t.Fatal("expected alert to be created")
	}

	if result.AlertID == nil {
		t.Fatal("expected alert id")
	}

	if len(alertRepository.alerts) != 1 {
		t.Fatalf("alerts = %d, want 1", len(alertRepository.alerts))
	}

	if result.QualityIndex >= 100 {
		t.Fatalf("QualityIndex = %v, want less than 100", result.QualityIndex)
	}
}

func TestProcessRepeatedWarningTelemetryUpdatesOpenAlertInsteadOfCreatingDuplicate(t *testing.T) {
	ctx := context.Background()

	telemetryRepository := &fakeTelemetryRepository{}
	alertRepository := &fakeAlertRepository{}
	qualityRepository := &fakeQualityRepository{}
	setpointRepository := newFakeSetpointRepository()
	anomalyRepository := &fakeAnomalyRepository{}

	service := NewService(
		slog.Default(),
		telemetryRepository,
		alertRepository,
		qualityRepository,
		setpointRepository,
		anomalyRepository,
	)

	firstResult, err := service.Process(ctx, Input{
		ParameterType: domain.ParameterPressure,
		Value:         80,
		Unit:          domain.UnitBar,
		SourceID:      "test-source",
		MeasuredAt:    time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("first Process returned error: %v", err)
	}

	secondResult, err := service.Process(ctx, Input{
		ParameterType: domain.ParameterPressure,
		Value:         85,
		Unit:          domain.UnitBar,
		SourceID:      "test-source",
		MeasuredAt:    time.Now().UTC().Add(time.Second),
	})
	if err != nil {
		t.Fatalf("second Process returned error: %v", err)
	}

	if !firstResult.AlertCreated {
		t.Fatal("expected first warning to create alert")
	}

	if !secondResult.AlertUpdated {
		t.Fatal("expected second warning to update alert")
	}

	if secondResult.AlertCreated {
		t.Fatal("second warning should not create duplicate alert")
	}

	if len(alertRepository.alerts) != 1 {
		t.Fatalf("alerts = %d, want 1", len(alertRepository.alerts))
	}

	if alertRepository.alerts[0].Value != 85 {
		t.Fatalf("updated alert value = %v, want 85", alertRepository.alerts[0].Value)
	}
}

func TestProcessNormalTelemetryResolvesOpenAlert(t *testing.T) {
	ctx := context.Background()

	telemetryRepository := &fakeTelemetryRepository{}
	alertRepository := &fakeAlertRepository{}
	qualityRepository := &fakeQualityRepository{}
	setpointRepository := newFakeSetpointRepository()
	anomalyRepository := &fakeAnomalyRepository{}

	service := NewService(
		slog.Default(),
		telemetryRepository,
		alertRepository,
		qualityRepository,
		setpointRepository,
		anomalyRepository,
	)

	_, err := service.Process(ctx, Input{
		ParameterType: domain.ParameterPressure,
		Value:         80,
		Unit:          domain.UnitBar,
		SourceID:      "test-source",
		MeasuredAt:    time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("warning Process returned error: %v", err)
	}

	result, err := service.Process(ctx, Input{
		ParameterType: domain.ParameterPressure,
		Value:         60,
		Unit:          domain.UnitBar,
		SourceID:      "test-source",
		MeasuredAt:    time.Now().UTC().Add(time.Second),
	})
	if err != nil {
		t.Fatalf("normal Process returned error: %v", err)
	}

	if result.State != domain.ParameterStateNormal {
		t.Fatalf("State = %q, want %q", result.State, domain.ParameterStateNormal)
	}

	if result.ResolvedAlerts != 1 {
		t.Fatalf("ResolvedAlerts = %d, want 1", result.ResolvedAlerts)
	}

	if alertRepository.alerts[0].Status != domain.AlertStatusResolved {
		t.Fatalf("alert status = %q, want %q", alertRepository.alerts[0].Status, domain.AlertStatusResolved)
	}
}

func TestProcessUnknownParameterReturnsValidationError(t *testing.T) {
	service := newTestTelemetryService()

	_, err := service.Process(context.Background(), Input{
		ParameterType: "unknown_parameter",
		Value:         80,
		Unit:          domain.UnitBar,
		SourceID:      "test-source",
		MeasuredAt:    time.Now().UTC(),
	})

	assertValidationError(t, err)
}

func TestProcessWrongUnitReturnsValidationError(t *testing.T) {
	service := newTestTelemetryService()

	_, err := service.Process(context.Background(), Input{
		ParameterType: domain.ParameterPressure,
		Value:         80,
		Unit:          domain.UnitPercent,
		SourceID:      "test-source",
		MeasuredAt:    time.Now().UTC(),
	})

	assertValidationError(t, err)
}

func TestProcessEmptySourceIDReturnsValidationError(t *testing.T) {
	service := newTestTelemetryService()

	_, err := service.Process(context.Background(), Input{
		ParameterType: domain.ParameterPressure,
		Value:         80,
		Unit:          domain.UnitBar,
		SourceID:      "",
		MeasuredAt:    time.Now().UTC(),
	})

	assertValidationError(t, err)
}

func TestProcessEmptyMeasuredAtReturnsValidationError(t *testing.T) {
	service := newTestTelemetryService()

	_, err := service.Process(context.Background(), Input{
		ParameterType: domain.ParameterPressure,
		Value:         80,
		Unit:          domain.UnitBar,
		SourceID:      "test-source",
	})

	assertValidationError(t, err)
}

func TestProcessActiveAnomalyAffectsQualityIndex(t *testing.T) {
	ctx := context.Background()

	telemetryRepository := &fakeTelemetryRepository{}
	alertRepository := &fakeAlertRepository{}
	qualityRepository := &fakeQualityRepository{}
	setpointRepository := newFakeSetpointRepository()
	anomalyRepository := &fakeAnomalyRepository{
		anomalies: []domain.AnomalyEvent{
			{
				ID:            1,
				Type:          domain.AnomalyTypeDrift,
				ParameterType: domain.ParameterMoisture,
				Level:         domain.AlertLevelWarning,
				Status:        domain.AlertStatusActive,
				Message:       "test moisture drift anomaly",
				SourceID:      "test-source",
				ObservedAt:    time.Now().UTC(),
				CreatedAt:     time.Now().UTC(),
				UpdatedAt:     time.Now().UTC(),
			},
		},
	}

	service := NewService(
		slog.Default(),
		telemetryRepository,
		alertRepository,
		qualityRepository,
		setpointRepository,
		anomalyRepository,
	)

	result, err := service.Process(ctx, Input{
		ParameterType: domain.ParameterPressure,
		Value:         60,
		Unit:          domain.UnitBar,
		SourceID:      "test-source",
		MeasuredAt:    time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("Process returned error: %v", err)
	}

	if result.QualityIndex >= 100 {
		t.Fatalf("QualityIndex = %v, want less than 100 because of active anomaly", result.QualityIndex)
	}

	if len(qualityRepository.indexes) != 1 {
		t.Fatalf("quality indexes = %d, want 1", len(qualityRepository.indexes))
	}

	if qualityRepository.indexes[0].AnomalyPenalty <= 0 {
		t.Fatalf("AnomalyPenalty = %v, want positive", qualityRepository.indexes[0].AnomalyPenalty)
	}
}

func newTestTelemetryService() *Service {
	return NewService(
		slog.Default(),
		&fakeTelemetryRepository{},
		&fakeAlertRepository{},
		&fakeQualityRepository{},
		newFakeSetpointRepository(),
		&fakeAnomalyRepository{},
	)
}

func newFakeSetpointRepository() *fakeSetpointRepository {
	return &fakeSetpointRepository{
		setpoints: map[domain.ParameterType]domain.Setpoint{
			domain.ParameterPressure: {
				ID:            1,
				ParameterType: domain.ParameterPressure,
				Unit:          domain.UnitBar,
				CriticalMin:   30,
				WarningMin:    35,
				NormalMin:     40,
				NormalMax:     75,
				WarningMax:    90,
				CriticalMax:   95,
			},
			domain.ParameterMoisture: {
				ID:            2,
				ParameterType: domain.ParameterMoisture,
				Unit:          domain.UnitPercent,
				CriticalMin:   10,
				WarningMin:    18,
				NormalMin:     22,
				NormalMax:     28,
				WarningMax:    32,
				CriticalMax:   40,
			},
		},
	}
}

func assertValidationError(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !IsValidationError(err) {
		t.Fatalf("expected validation error, got %T: %v", err, err)
	}
}

type fakeTelemetryRepository struct {
	nextID   domain.TelemetryReadingID
	readings []domain.TelemetryReading

	saveErr error
}

func (r *fakeTelemetryRepository) Save(
	ctx context.Context,
	reading domain.TelemetryReading,
) (domain.TelemetryReading, error) {
	_ = ctx

	if r.saveErr != nil {
		return domain.TelemetryReading{}, r.saveErr
	}

	if r.nextID == 0 {
		r.nextID = 1
	}

	reading.ID = r.nextID
	r.nextID++

	r.readings = append(r.readings, reading)

	return reading, nil
}

func (r *fakeTelemetryRepository) All(ctx context.Context) ([]domain.TelemetryReading, error) {
	_ = ctx

	return append([]domain.TelemetryReading(nil), r.readings...), nil
}

func (r *fakeTelemetryRepository) Latest(ctx context.Context) ([]domain.TelemetryReading, error) {
	_ = ctx

	return append([]domain.TelemetryReading(nil), r.readings...), nil
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

	result := make([]domain.TelemetryReading, 0)

	for _, reading := range r.readings {
		if reading.ParameterType == parameterType {
			result = append(result, reading)
		}
	}

	if limit > 0 && len(result) > limit {
		result = result[len(result)-limit:]
	}

	return result, nil
}

type fakeAlertRepository struct {
	nextID domain.AlertID
	alerts []domain.AlertEvent

	createErr                 error
	activeErr                 error
	findOpenByParameterErr    error
	updateOpenErr             error
	resolveOpenByParameterErr error
}

func (r *fakeAlertRepository) Create(
	ctx context.Context,
	alert domain.AlertEvent,
) (domain.AlertEvent, error) {
	_ = ctx

	if r.createErr != nil {
		return domain.AlertEvent{}, r.createErr
	}

	if r.nextID == 0 {
		r.nextID = 1
	}

	alert.ID = r.nextID
	r.nextID++

	if alert.Status == "" {
		alert.Status = domain.AlertStatusActive
	}

	r.alerts = append(r.alerts, alert)

	return alert, nil
}

func (r *fakeAlertRepository) All(ctx context.Context) ([]domain.AlertEvent, error) {
	_ = ctx

	return append([]domain.AlertEvent(nil), r.alerts...), nil
}

func (r *fakeAlertRepository) Active(ctx context.Context) ([]domain.AlertEvent, error) {
	_ = ctx

	if r.activeErr != nil {
		return nil, r.activeErr
	}

	result := make([]domain.AlertEvent, 0)

	for _, alert := range r.alerts {
		if isOpenAlert(alert.Status) {
			result = append(result, alert)
		}
	}

	return result, nil
}

func (r *fakeAlertRepository) FindOpenByParameter(
	ctx context.Context,
	parameterType domain.ParameterType,
) (domain.AlertEvent, bool, error) {
	_ = ctx

	if r.findOpenByParameterErr != nil {
		return domain.AlertEvent{}, false, r.findOpenByParameterErr
	}

	for index := len(r.alerts) - 1; index >= 0; index-- {
		alert := r.alerts[index]

		if alert.ParameterType == parameterType && isOpenAlert(alert.Status) {
			return alert, true, nil
		}
	}

	return domain.AlertEvent{}, false, nil
}

func (r *fakeAlertRepository) UpdateOpen(
	ctx context.Context,
	alert domain.AlertEvent,
) (domain.AlertEvent, bool, error) {
	_ = ctx

	if r.updateOpenErr != nil {
		return domain.AlertEvent{}, false, r.updateOpenErr
	}

	for index := range r.alerts {
		if r.alerts[index].ID != alert.ID || !isOpenAlert(r.alerts[index].Status) {
			continue
		}

		r.alerts[index].Level = alert.Level
		r.alerts[index].Value = alert.Value
		r.alerts[index].Unit = alert.Unit
		r.alerts[index].SourceID = alert.SourceID
		r.alerts[index].Message = alert.Message

		return r.alerts[index], true, nil
	}

	return domain.AlertEvent{}, false, nil
}

func (r *fakeAlertRepository) ResolveOpenByParameter(
	ctx context.Context,
	parameterType domain.ParameterType,
) (int64, error) {
	_ = ctx

	if r.resolveOpenByParameterErr != nil {
		return 0, r.resolveOpenByParameterErr
	}

	var resolvedCount int64
	now := time.Now().UTC()

	for index := range r.alerts {
		if r.alerts[index].ParameterType != parameterType || !isOpenAlert(r.alerts[index].Status) {
			continue
		}

		r.alerts[index].Status = domain.AlertStatusResolved
		r.alerts[index].ResolvedAt = &now
		resolvedCount++
	}

	return resolvedCount, nil
}

func (r *fakeAlertRepository) Acknowledge(
	ctx context.Context,
	id domain.AlertID,
	userID *domain.UserID,
) (domain.AlertEvent, bool, error) {
	_ = ctx
	_ = userID

	for index := range r.alerts {
		if r.alerts[index].ID == id {
			r.alerts[index].Status = domain.AlertStatusAcknowledged
			return r.alerts[index], true, nil
		}
	}

	return domain.AlertEvent{}, false, nil
}

func (r *fakeAlertRepository) Resolve(
	ctx context.Context,
	id domain.AlertID,
) (domain.AlertEvent, bool, error) {
	_ = ctx

	for index := range r.alerts {
		if r.alerts[index].ID == id {
			r.alerts[index].Status = domain.AlertStatusResolved
			return r.alerts[index], true, nil
		}
	}

	return domain.AlertEvent{}, false, nil
}

type fakeQualityRepository struct {
	nextID  domain.QualityIndexID
	indexes []domain.QualityIndex

	saveErr error
}

func (r *fakeQualityRepository) Save(
	ctx context.Context,
	index domain.QualityIndex,
) (domain.QualityIndex, error) {
	_ = ctx

	if r.saveErr != nil {
		return domain.QualityIndex{}, r.saveErr
	}

	if r.nextID == 0 {
		r.nextID = 1
	}

	index.ID = r.nextID
	r.nextID++

	r.indexes = append(r.indexes, index)

	return index, nil
}

func (r *fakeQualityRepository) Latest(
	ctx context.Context,
) (domain.QualityIndex, bool, error) {
	_ = ctx

	if len(r.indexes) == 0 {
		return domain.QualityIndex{}, false, nil
	}

	return r.indexes[len(r.indexes)-1], true, nil
}

func (r *fakeQualityRepository) History(
	ctx context.Context,
	from time.Time,
	to time.Time,
	limit int,
) ([]domain.QualityIndex, error) {
	_ = ctx
	_ = from
	_ = to
	_ = limit

	return append([]domain.QualityIndex(nil), r.indexes...), nil
}

type fakeSetpointRepository struct {
	setpoints map[domain.ParameterType]domain.Setpoint
}

func (r *fakeSetpointRepository) All(ctx context.Context) ([]domain.Setpoint, error) {
	_ = ctx

	result := make([]domain.Setpoint, 0, len(r.setpoints))

	for _, setpoint := range r.setpoints {
		result = append(result, setpoint)
	}

	return result, nil
}

func (r *fakeSetpointRepository) GetByParameter(
	ctx context.Context,
	parameterType domain.ParameterType,
) (domain.Setpoint, bool, error) {
	_ = ctx

	setpoint, ok := r.setpoints[parameterType]

	return setpoint, ok, nil
}

func (r *fakeSetpointRepository) Update(
	ctx context.Context,
	id int64,
	update domain.SetpointUpdate,
) (domain.Setpoint, bool, error) {
	_ = ctx

	for parameterType, setpoint := range r.setpoints {
		if int64(setpoint.ID) != id {
			continue
		}

		setpoint.CriticalMin = update.CriticalMin
		setpoint.WarningMin = update.WarningMin
		setpoint.NormalMin = update.NormalMin
		setpoint.NormalMax = update.NormalMax
		setpoint.WarningMax = update.WarningMax
		setpoint.CriticalMax = update.CriticalMax
		setpoint.UpdatedAt = time.Now().UTC()

		r.setpoints[parameterType] = setpoint

		return setpoint, true, nil
	}

	return domain.Setpoint{}, false, nil
}

type fakeAnomalyRepository struct {
	nextID    domain.AnomalyID
	anomalies []domain.AnomalyEvent

	createErr                        error
	activeErr                        error
	findOpenByTypeAndParameterErr    error
	updateOpenErr                    error
	resolveOpenByTypeAndParameterErr error
}

func (r *fakeAnomalyRepository) Create(
	ctx context.Context,
	anomaly domain.AnomalyEvent,
) (domain.AnomalyEvent, error) {
	_ = ctx

	if r.createErr != nil {
		return domain.AnomalyEvent{}, r.createErr
	}

	if r.nextID == 0 {
		r.nextID = 1
	}

	anomaly.ID = r.nextID
	r.nextID++

	if anomaly.Status == "" {
		anomaly.Status = domain.AlertStatusActive
	}

	r.anomalies = append(r.anomalies, anomaly)

	return anomaly, nil
}

func (r *fakeAnomalyRepository) All(ctx context.Context) ([]domain.AnomalyEvent, error) {
	_ = ctx

	return append([]domain.AnomalyEvent(nil), r.anomalies...), nil
}

func (r *fakeAnomalyRepository) Active(ctx context.Context) ([]domain.AnomalyEvent, error) {
	_ = ctx

	if r.activeErr != nil {
		return nil, r.activeErr
	}

	result := make([]domain.AnomalyEvent, 0)

	for _, anomaly := range r.anomalies {
		if isOpenAlert(anomaly.Status) {
			result = append(result, anomaly)
		}
	}

	return result, nil
}

func (r *fakeAnomalyRepository) FindOpenByTypeAndParameter(
	ctx context.Context,
	anomalyType domain.AnomalyType,
	parameterType domain.ParameterType,
) (domain.AnomalyEvent, bool, error) {
	_ = ctx

	if r.findOpenByTypeAndParameterErr != nil {
		return domain.AnomalyEvent{}, false, r.findOpenByTypeAndParameterErr
	}

	for index := len(r.anomalies) - 1; index >= 0; index-- {
		anomaly := r.anomalies[index]

		if anomaly.Type == anomalyType &&
			anomaly.ParameterType == parameterType &&
			isOpenAlert(anomaly.Status) {
			return anomaly, true, nil
		}
	}

	return domain.AnomalyEvent{}, false, nil
}

func (r *fakeAnomalyRepository) UpdateOpen(
	ctx context.Context,
	anomaly domain.AnomalyEvent,
) (domain.AnomalyEvent, bool, error) {
	_ = ctx

	if r.updateOpenErr != nil {
		return domain.AnomalyEvent{}, false, r.updateOpenErr
	}

	for index := range r.anomalies {
		if r.anomalies[index].ID != anomaly.ID || !isOpenAlert(r.anomalies[index].Status) {
			continue
		}

		r.anomalies[index] = anomaly

		return r.anomalies[index], true, nil
	}

	return domain.AnomalyEvent{}, false, nil
}

func (r *fakeAnomalyRepository) ResolveOpenByTypeAndParameter(
	ctx context.Context,
	anomalyType domain.AnomalyType,
	parameterType domain.ParameterType,
) (int64, error) {
	_ = ctx

	if r.resolveOpenByTypeAndParameterErr != nil {
		return 0, r.resolveOpenByTypeAndParameterErr
	}

	var resolvedCount int64
	now := time.Now().UTC()

	for index := range r.anomalies {
		if r.anomalies[index].Type != anomalyType ||
			r.anomalies[index].ParameterType != parameterType ||
			!isOpenAlert(r.anomalies[index].Status) {
			continue
		}

		r.anomalies[index].Status = domain.AlertStatusResolved
		r.anomalies[index].ResolvedAt = &now
		resolvedCount++
	}

	return resolvedCount, nil
}

func isOpenAlert(status domain.AlertStatus) bool {
	return status == domain.AlertStatusActive ||
		status == domain.AlertStatusAcknowledged
}

func TestProcessCriticalTelemetryCreatesCriticalAlert(t *testing.T) {
	ctx := context.Background()

	telemetryRepository := &fakeTelemetryRepository{}
	alertRepository := &fakeAlertRepository{}
	qualityRepository := &fakeQualityRepository{}
	setpointRepository := newFakeSetpointRepository()
	anomalyRepository := &fakeAnomalyRepository{}

	service := NewService(
		slog.Default(),
		telemetryRepository,
		alertRepository,
		qualityRepository,
		setpointRepository,
		anomalyRepository,
	)

	result, err := service.Process(ctx, Input{
		ParameterType: domain.ParameterPressure,
		Value:         100,
		Unit:          domain.UnitBar,
		SourceID:      "test-source",
		MeasuredAt:    time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("Process returned error: %v", err)
	}

	if result.State != domain.ParameterStateCritical {
		t.Fatalf("State = %q, want %q", result.State, domain.ParameterStateCritical)
	}

	if !result.AlertCreated {
		t.Fatal("expected alert to be created")
	}

	if result.AlertLevel == nil {
		t.Fatal("expected alert level")
	}

	if *result.AlertLevel != domain.AlertLevelCritical {
		t.Fatalf("AlertLevel = %q, want %q", *result.AlertLevel, domain.AlertLevelCritical)
	}

	if len(alertRepository.alerts) != 1 {
		t.Fatalf("alerts = %d, want 1", len(alertRepository.alerts))
	}

	if alertRepository.alerts[0].Level != domain.AlertLevelCritical {
		t.Fatalf("alert level = %q, want %q", alertRepository.alerts[0].Level, domain.AlertLevelCritical)
	}

	if result.QualityIndex >= 100 {
		t.Fatalf("QualityIndex = %v, want less than 100", result.QualityIndex)
	}
}

func TestProcessUsesQualityWeightsFromRepository(t *testing.T) {
	ctx := context.Background()

	telemetryRepository := &fakeTelemetryRepository{}
	alertRepository := &fakeAlertRepository{}
	defaultQualityRepository := &fakeQualityRepository{}
	weightedQualityRepository := &fakeQualityRepository{}
	setpointRepository := newFakeSetpointRepository()

	defaultService := NewService(
		slog.Default(),
		telemetryRepository,
		alertRepository,
		defaultQualityRepository,
		setpointRepository,
		&fakeAnomalyRepository{},
	)

	defaultResult, err := defaultService.Process(ctx, Input{
		ParameterType: domain.ParameterPressure,
		Value:         80,
		Unit:          domain.UnitBar,
		SourceID:      "test-source",
		MeasuredAt:    time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("default Process returned error: %v", err)
	}

	telemetryRepository = &fakeTelemetryRepository{}
	alertRepository = &fakeAlertRepository{}
	setpointRepository = newFakeSetpointRepository()

	weightedService := NewService(
		slog.Default(),
		telemetryRepository,
		alertRepository,
		weightedQualityRepository,
		setpointRepository,
		&fakeAnomalyRepository{},
		WithQualityWeightRepository(&fakeQualityWeightRepository{
			weights: []domain.QualityWeight{
				{
					ParameterType: domain.ParameterPressure,
					Weight:        2,
				},
			},
		}),
	)

	weightedResult, err := weightedService.Process(ctx, Input{
		ParameterType: domain.ParameterPressure,
		Value:         80,
		Unit:          domain.UnitBar,
		SourceID:      "test-source",
		MeasuredAt:    time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("weighted Process returned error: %v", err)
	}

	if weightedResult.QualityIndex >= defaultResult.QualityIndex {
		t.Fatalf(
			"weighted quality index = %v, default quality index = %v, want weighted lower",
			weightedResult.QualityIndex,
			defaultResult.QualityIndex,
		)
	}

	if weightedQualityRepository.indexes[0].ParameterPenalty <= defaultQualityRepository.indexes[0].ParameterPenalty {
		t.Fatalf(
			"weighted penalty = %v, default penalty = %v, want weighted greater",
			weightedQualityRepository.indexes[0].ParameterPenalty,
			defaultQualityRepository.indexes[0].ParameterPenalty,
		)
	}
}

type fakeQualityWeightRepository struct {
	weights []domain.QualityWeight
	err     error
}

func (r *fakeQualityWeightRepository) List(ctx context.Context) ([]domain.QualityWeight, error) {
	_ = ctx

	if r.err != nil {
		return nil, r.err
	}

	return append([]domain.QualityWeight(nil), r.weights...), nil
}

func (r *fakeQualityWeightRepository) Update(
	ctx context.Context,
	id domain.QualityWeightID,
	update domain.QualityWeightUpdate,
	updatedBy string,
) (domain.QualityWeight, bool, error) {
	_ = ctx
	_ = id
	_ = update
	_ = updatedBy

	return domain.QualityWeight{}, false, nil
}

func TestProcessReturnsErrorWhenSaveTelemetryFails(t *testing.T) {
	ctx := context.Background()

	service := NewService(
		slog.Default(),
		&fakeTelemetryRepository{
			saveErr: errRepositoryFailure,
		},
		&fakeAlertRepository{},
		&fakeQualityRepository{},
		newFakeSetpointRepository(),
		&fakeAnomalyRepository{},
	)

	_, err := service.Process(ctx, Input{
		ParameterType: domain.ParameterPressure,
		Value:         60,
		Unit:          domain.UnitBar,
		SourceID:      "test-source",
		MeasuredAt:    time.Now().UTC(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, errRepositoryFailure) {
		t.Fatalf("error = %v, want wrapped errRepositoryFailure", err)
	}
}

func TestProcessReturnsErrorWhenSaveQualityIndexFails(t *testing.T) {
	ctx := context.Background()

	service := NewService(
		slog.Default(),
		&fakeTelemetryRepository{},
		&fakeAlertRepository{},
		&fakeQualityRepository{
			saveErr: errRepositoryFailure,
		},
		newFakeSetpointRepository(),
		&fakeAnomalyRepository{},
	)

	_, err := service.Process(ctx, Input{
		ParameterType: domain.ParameterPressure,
		Value:         60,
		Unit:          domain.UnitBar,
		SourceID:      "test-source",
		MeasuredAt:    time.Now().UTC(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, errRepositoryFailure) {
		t.Fatalf("error = %v, want wrapped errRepositoryFailure", err)
	}
}

func TestProcessReturnsErrorWhenLoadQualityWeightsFails(t *testing.T) {
	ctx := context.Background()

	service := NewService(
		slog.Default(),
		&fakeTelemetryRepository{},
		&fakeAlertRepository{},
		&fakeQualityRepository{},
		newFakeSetpointRepository(),
		&fakeAnomalyRepository{},
		WithQualityWeightRepository(&fakeQualityWeightRepository{
			err: errRepositoryFailure,
		}),
	)

	_, err := service.Process(ctx, Input{
		ParameterType: domain.ParameterPressure,
		Value:         60,
		Unit:          domain.UnitBar,
		SourceID:      "test-source",
		MeasuredAt:    time.Now().UTC(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, errRepositoryFailure) {
		t.Fatalf("error = %v, want wrapped errRepositoryFailure", err)
	}
}

func TestProcessReturnsErrorWhenCreateAlertFails(t *testing.T) {
	ctx := context.Background()

	service := NewService(
		slog.Default(),
		&fakeTelemetryRepository{},
		&fakeAlertRepository{
			createErr: errRepositoryFailure,
		},
		&fakeQualityRepository{},
		newFakeSetpointRepository(),
		&fakeAnomalyRepository{},
	)

	_, err := service.Process(ctx, Input{
		ParameterType: domain.ParameterPressure,
		Value:         80,
		Unit:          domain.UnitBar,
		SourceID:      "test-source",
		MeasuredAt:    time.Now().UTC(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, errRepositoryFailure) {
		t.Fatalf("error = %v, want wrapped errRepositoryFailure", err)
	}
}

func TestProcessReturnsErrorWhenFindOpenAlertFails(t *testing.T) {
	ctx := context.Background()

	service := NewService(
		slog.Default(),
		&fakeTelemetryRepository{},
		&fakeAlertRepository{
			findOpenByParameterErr: errRepositoryFailure,
		},
		&fakeQualityRepository{},
		newFakeSetpointRepository(),
		&fakeAnomalyRepository{},
	)

	_, err := service.Process(ctx, Input{
		ParameterType: domain.ParameterPressure,
		Value:         80,
		Unit:          domain.UnitBar,
		SourceID:      "test-source",
		MeasuredAt:    time.Now().UTC(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, errRepositoryFailure) {
		t.Fatalf("error = %v, want wrapped errRepositoryFailure", err)
	}
}

func TestProcessReturnsErrorWhenUpdateOpenAlertFails(t *testing.T) {
	ctx := context.Background()

	alertRepository := &fakeAlertRepository{
		alerts: []domain.AlertEvent{
			{
				ID:            1,
				ParameterType: domain.ParameterPressure,
				Level:         domain.AlertLevelWarning,
				Status:        domain.AlertStatusActive,
				Value:         80,
				Unit:          domain.UnitBar,
				SourceID:      "test-source",
				Message:       "existing alert",
				CreatedAt:     time.Now().UTC(),
			},
		},
		updateOpenErr: errRepositoryFailure,
	}

	service := NewService(
		slog.Default(),
		&fakeTelemetryRepository{},
		alertRepository,
		&fakeQualityRepository{},
		newFakeSetpointRepository(),
		&fakeAnomalyRepository{},
	)

	_, err := service.Process(ctx, Input{
		ParameterType: domain.ParameterPressure,
		Value:         85,
		Unit:          domain.UnitBar,
		SourceID:      "test-source",
		MeasuredAt:    time.Now().UTC(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, errRepositoryFailure) {
		t.Fatalf("error = %v, want wrapped errRepositoryFailure", err)
	}
}

func TestProcessReturnsErrorWhenResolveOpenAlertFails(t *testing.T) {
	ctx := context.Background()

	service := NewService(
		slog.Default(),
		&fakeTelemetryRepository{},
		&fakeAlertRepository{
			resolveOpenByParameterErr: errRepositoryFailure,
		},
		&fakeQualityRepository{},
		newFakeSetpointRepository(),
		&fakeAnomalyRepository{},
	)

	_, err := service.Process(ctx, Input{
		ParameterType: domain.ParameterPressure,
		Value:         60,
		Unit:          domain.UnitBar,
		SourceID:      "test-source",
		MeasuredAt:    time.Now().UTC(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, errRepositoryFailure) {
		t.Fatalf("error = %v, want wrapped errRepositoryFailure", err)
	}
}

func TestProcessReturnsErrorWhenLoadActiveAlertsFails(t *testing.T) {
	ctx := context.Background()

	service := NewService(
		slog.Default(),
		&fakeTelemetryRepository{},
		&fakeAlertRepository{
			activeErr: errRepositoryFailure,
		},
		&fakeQualityRepository{},
		newFakeSetpointRepository(),
		&fakeAnomalyRepository{},
	)

	_, err := service.Process(ctx, Input{
		ParameterType: domain.ParameterPressure,
		Value:         60,
		Unit:          domain.UnitBar,
		SourceID:      "test-source",
		MeasuredAt:    time.Now().UTC(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, errRepositoryFailure) {
		t.Fatalf("error = %v, want wrapped errRepositoryFailure", err)
	}
}

func TestProcessReturnsErrorWhenLoadActiveAnomaliesFails(t *testing.T) {
	ctx := context.Background()

	service := NewService(
		slog.Default(),
		&fakeTelemetryRepository{},
		&fakeAlertRepository{},
		&fakeQualityRepository{},
		newFakeSetpointRepository(),
		&fakeAnomalyRepository{
			activeErr: errRepositoryFailure,
		},
	)

	_, err := service.Process(ctx, Input{
		ParameterType: domain.ParameterPressure,
		Value:         60,
		Unit:          domain.UnitBar,
		SourceID:      "test-source",
		MeasuredAt:    time.Now().UTC(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, errRepositoryFailure) {
		t.Fatalf("error = %v, want wrapped errRepositoryFailure", err)
	}
}

var errRepositoryFailure = errors.New("repository failure")
