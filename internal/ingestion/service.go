package ingestion

import (
	"context"
	"errors"
	"extrusion-quality-system/internal/analytics"
	"extrusion-quality-system/internal/anomalies"
	"extrusion-quality-system/internal/domain"
	"extrusion-quality-system/internal/storage"
	"fmt"
	"log/slog"
	"time"
)

type Option func(*Service)

func WithQualityWeights(weights analytics.QualityWeights) Option {
	return func(service *Service) {
		service.qualityWeights = weights
	}
}

func WithQualityWeightRepository(repository storage.QualityWeightRepository) Option {
	return func(service *Service) {
		service.qualityWeightRepository = repository
	}
}

// TelemetryInput describes telemetry data received from HTTP or MQTT.
type TelemetryInput struct {
	ParameterType domain.ParameterType `json:"parameterType"`
	Value         float64              `json:"value"`
	Unit          domain.Unit          `json:"unit"`
	SourceID      domain.SourceID      `json:"sourceId"`
	MeasuredAt    time.Time            `json:"measuredAt"`
}

// TelemetryResult describes telemetry processing result.
type TelemetryResult struct {
	Accepted       bool                  `json:"accepted"`
	ParameterType  domain.ParameterType  `json:"parameterType"`
	Value          float64               `json:"value"`
	Unit           domain.Unit           `json:"unit"`
	SourceID       domain.SourceID       `json:"sourceId"`
	MeasuredAt     time.Time             `json:"measuredAt"`
	State          domain.ParameterState `json:"state"`
	AlertCreated   bool                  `json:"alertCreated"`
	AlertUpdated   bool                  `json:"alertUpdated,omitempty"`
	ResolvedAlerts int64                 `json:"resolvedAlerts,omitempty"`
	AlertID        *domain.AlertID       `json:"alertId,omitempty"`
	AlertLevel     *domain.AlertLevel    `json:"alertLevel,omitempty"`
	QualityIndex   float64               `json:"qualityIndex"`
	QualityState   domain.QualityState   `json:"qualityState"`
}

// ValidationError is returned when incoming telemetry data is invalid.
type ValidationError struct {
	Message string
}

func (e ValidationError) Error() string {
	return e.Message
}

// IsValidationError checks whether an error is caused by invalid telemetry input.
func IsValidationError(err error) bool {
	var validationError ValidationError
	return errors.As(err, &validationError)
}

// Service processes telemetry from different transports.
type Service struct {
	logger                  *slog.Logger
	telemetryRepository     storage.TelemetryRepository
	alertRepository         storage.AlertRepository
	qualityRepository       storage.QualityRepository
	setpointRepository      storage.SetpointRepository
	anomalyRepository       storage.AnomalyRepository
	anomalyDetector         *anomalies.Detector
	qualityWeightRepository storage.QualityWeightRepository
	qualityWeights          analytics.QualityWeights
}

// NewService creates telemetry ingestion service.
func NewService(
	logger *slog.Logger,
	telemetryRepository storage.TelemetryRepository,
	alertRepository storage.AlertRepository,
	qualityRepository storage.QualityRepository,
	setpointSource any,
	optionalArgs ...any,
) *Service {
	setpointRepository := resolveSetpointRepository(setpointSource)

	anomalyRepository := storage.AnomalyRepository(storage.NewMemoryAnomalyRepository())

	service := &Service{
		logger:              logger,
		telemetryRepository: telemetryRepository,
		alertRepository:     alertRepository,
		qualityRepository:   qualityRepository,
		setpointRepository:  setpointRepository,
		anomalyRepository:   anomalyRepository,
		anomalyDetector:     anomalies.NewDetector(telemetryRepository),
		qualityWeights:      analytics.DefaultQualityWeights(),
	}

	for _, arg := range optionalArgs {
		switch value := arg.(type) {
		case storage.AnomalyRepository:
			if value != nil {
				service.anomalyRepository = value
			}
		case storage.QualityWeightRepository:
			if value != nil {
				service.qualityWeightRepository = value
			}

		case Option:
			value(service)
		}
	}

	return service
}

func resolveSetpointRepository(setpointSource any) storage.SetpointRepository {
	switch value := setpointSource.(type) {
	case storage.SetpointRepository:
		return value

	case map[domain.ParameterType]domain.Setpoint:
		setpoints := make([]domain.Setpoint, 0, len(value))

		for _, setpoint := range value {
			setpoints = append(setpoints, setpoint)
		}

		return storage.NewMemorySetpointRepository(setpoints)

	default:
		return storage.NewMemorySetpointRepository(nil)
	}
}

// Process validates, stores and analytically processes one telemetry reading.
// It also prevents duplicate active alerts for the same parameter.
func (s *Service) Process(ctx context.Context, input TelemetryInput) (TelemetryResult, error) {
	setpoint, err := s.validate(ctx, input)
	if err != nil {
		return TelemetryResult{}, err
	}

	reading := domain.TelemetryReading{
		ParameterType: input.ParameterType,
		Value:         input.Value,
		Unit:          input.Unit,
		SourceID:      input.SourceID,
		MeasuredAt:    input.MeasuredAt,
		CreatedAt:     time.Now().UTC(),
	}

	reading, err = s.telemetryRepository.Save(ctx, reading)
	if err != nil {
		return TelemetryResult{}, fmt.Errorf("save telemetry reading: %w", err)
	}

	state := setpoint.Evaluate(reading.Value)

	var alertID *domain.AlertID
	var alertLevel *domain.AlertLevel
	var alertCreated bool
	var alertUpdated bool
	var resolvedAlerts int64

	level, shouldCreateOrUpdateAlert := alertLevelFromParameterState(state)
	if shouldCreateOrUpdateAlert {
		alert, created, updated, err := s.createOrUpdateOpenAlert(ctx, reading, level)
		if err != nil {
			return TelemetryResult{}, err
		}

		alertCreated = created
		alertUpdated = updated

		id := alert.ID
		alertID = &id

		levelCopy := alert.Level
		alertLevel = &levelCopy
	} else {
		resolvedAlerts, err = s.alertRepository.ResolveOpenByParameter(ctx, reading.ParameterType)
		if err != nil {
			return TelemetryResult{}, fmt.Errorf("resolve open alerts by parameter: %w", err)
		}
	}

	activeAlerts, err := s.alertRepository.Active(ctx)
	if err != nil {
		return TelemetryResult{}, fmt.Errorf("load active alerts: %w", err)
	}

	detectedAnomalies, err := s.anomalyDetector.Detect(ctx, reading)
	if err != nil {
		return TelemetryResult{}, fmt.Errorf("detect anomalies: %w", err)
	}

	if err := s.syncAnomalies(ctx, reading.ParameterType, detectedAnomalies); err != nil {
		return TelemetryResult{}, err
	}

	activeAnomalies, err := s.anomalyRepository.Active(ctx)
	if err != nil {
		return TelemetryResult{}, fmt.Errorf("load active anomalies: %w", err)
	}

	qualityWeights, err := s.loadQualityWeights(ctx)
	if err != nil {
		return TelemetryResult{}, err
	}

	qualityIndex := analytics.CalculateQualityIndex(
		activeAlerts,
		qualityWeights,
		activeAnomalies,
	)
	qualityIndex, err = s.qualityRepository.Save(ctx, qualityIndex)
	if err != nil {
		return TelemetryResult{}, fmt.Errorf("save quality index: %w", err)
	}

	s.logger.Info(
		"telemetry processed",
		"parameterType", reading.ParameterType,
		"value", reading.Value,
		"unit", reading.Unit,
		"sourceId", reading.SourceID,
		"state", state,
		"alertCreated", alertCreated,
		"alertUpdated", alertUpdated,
		"resolvedAlerts", resolvedAlerts,
		"qualityIndex", qualityIndex.Value,
		"qualityState", qualityIndex.State,
	)

	return TelemetryResult{
		Accepted:       true,
		ParameterType:  reading.ParameterType,
		Value:          reading.Value,
		Unit:           reading.Unit,
		SourceID:       reading.SourceID,
		MeasuredAt:     reading.MeasuredAt,
		State:          state,
		AlertCreated:   alertCreated,
		AlertUpdated:   alertUpdated,
		ResolvedAlerts: resolvedAlerts,
		AlertID:        alertID,
		AlertLevel:     alertLevel,
		QualityIndex:   qualityIndex.Value,
		QualityState:   qualityIndex.State,
	}, nil
}

func (s *Service) createOrUpdateOpenAlert(
	ctx context.Context,
	reading domain.TelemetryReading,
	level domain.AlertLevel,
) (domain.AlertEvent, bool, bool, error) {
	alert := domain.AlertEvent{
		ParameterType: reading.ParameterType,
		Level:         level,
		Status:        domain.AlertStatusActive,
		Value:         reading.Value,
		Unit:          reading.Unit,
		SourceID:      reading.SourceID,
		Message:       buildAlertMessage(reading, level),
		CreatedAt:     time.Now().UTC(),
	}

	existingAlert, found, err := s.alertRepository.FindOpenByParameter(ctx, reading.ParameterType)
	if err != nil {
		return domain.AlertEvent{}, false, false, fmt.Errorf("find open alert by parameter: %w", err)
	}

	if found {
		alert.ID = existingAlert.ID

		updatedAlert, updated, err := s.alertRepository.UpdateOpen(ctx, alert)
		if err != nil {
			return domain.AlertEvent{}, false, false, fmt.Errorf("update open alert: %w", err)
		}

		if updated {
			return updatedAlert, false, true, nil
		}
	}

	createdAlert, err := s.alertRepository.Create(ctx, alert)
	if err != nil {
		return domain.AlertEvent{}, false, false, fmt.Errorf("save alert event: %w", err)
	}

	return createdAlert, true, false, nil
}

func (s *Service) validate(ctx context.Context, input TelemetryInput) (domain.Setpoint, error) {
	setpoint, found, err := s.setpointRepository.GetByParameter(ctx, input.ParameterType)
	if err != nil {
		return domain.Setpoint{}, fmt.Errorf("load setpoint by parameter: %w", err)
	}

	if !found {
		return domain.Setpoint{}, ValidationError{Message: "unknown parameterType"}
	}

	if input.Unit == "" {
		return domain.Setpoint{}, ValidationError{Message: "unit is required"}
	}

	if input.Unit != setpoint.Unit {
		return domain.Setpoint{}, ValidationError{Message: "unit does not match parameterType"}
	}

	if input.SourceID == "" {
		return domain.Setpoint{}, ValidationError{Message: "sourceId is required"}
	}

	if input.MeasuredAt.IsZero() {
		return domain.Setpoint{}, ValidationError{Message: "measuredAt is required"}
	}

	return setpoint, nil
}

func alertLevelFromParameterState(state domain.ParameterState) (domain.AlertLevel, bool) {
	switch state {
	case domain.ParameterStateWarning:
		return domain.AlertLevelWarning, true
	case domain.ParameterStateCritical:
		return domain.AlertLevelCritical, true
	default:
		return "", false
	}
}

func buildAlertMessage(reading domain.TelemetryReading, level domain.AlertLevel) string {
	return fmt.Sprintf(
		"parameter %s has %s value %.2f %s",
		reading.ParameterType,
		level,
		reading.Value,
		reading.Unit,
	)
}

func (s *Service) syncAnomalies(
	ctx context.Context,
	currentParameter domain.ParameterType,
	detectedAnomalies []domain.AnomalyEvent,
) error {
	detected := make(map[string]struct{})

	for _, anomaly := range detectedAnomalies {
		key := anomalyKey(anomaly.Type, anomaly.ParameterType)
		detected[key] = struct{}{}

		if _, err := s.createOrUpdateOpenAnomaly(ctx, anomaly); err != nil {
			return err
		}
	}

	if _, ok := detected[anomalyKey(domain.AnomalyTypeJump, currentParameter)]; !ok {
		if _, err := s.anomalyRepository.ResolveOpenByTypeAndParameter(
			ctx,
			domain.AnomalyTypeJump,
			currentParameter,
		); err != nil {
			return fmt.Errorf("resolve jump anomaly: %w", err)
		}
	}

	if _, ok := detected[anomalyKey(domain.AnomalyTypeDrift, currentParameter)]; !ok {
		if _, err := s.anomalyRepository.ResolveOpenByTypeAndParameter(
			ctx,
			domain.AnomalyTypeDrift,
			currentParameter,
		); err != nil {
			return fmt.Errorf("resolve drift anomaly: %w", err)
		}
	}

	if _, ok := detected[anomalyKey(domain.AnomalyTypeCombinedRisk, domain.ParameterProcessRisk)]; !ok {
		if _, err := s.anomalyRepository.ResolveOpenByTypeAndParameter(
			ctx,
			domain.AnomalyTypeCombinedRisk,
			domain.ParameterProcessRisk,
		); err != nil {
			return fmt.Errorf("resolve combined risk anomaly: %w", err)
		}
	}

	return nil
}

func (s *Service) createOrUpdateOpenAnomaly(
	ctx context.Context,
	anomaly domain.AnomalyEvent,
) (domain.AnomalyEvent, error) {
	existing, found, err := s.anomalyRepository.FindOpenByTypeAndParameter(
		ctx,
		anomaly.Type,
		anomaly.ParameterType,
	)
	if err != nil {
		return domain.AnomalyEvent{}, fmt.Errorf("find open anomaly: %w", err)
	}

	if found {
		anomaly.ID = existing.ID

		updated, ok, err := s.anomalyRepository.UpdateOpen(ctx, anomaly)
		if err != nil {
			return domain.AnomalyEvent{}, fmt.Errorf("update open anomaly: %w", err)
		}

		if ok {
			return updated, nil
		}
	}

	created, err := s.anomalyRepository.Create(ctx, anomaly)
	if err != nil {
		return domain.AnomalyEvent{}, fmt.Errorf("create anomaly: %w", err)
	}

	return created, nil
}

func anomalyKey(anomalyType domain.AnomalyType, parameterType domain.ParameterType) string {
	return string(anomalyType) + ":" + string(parameterType)
}

func (s *Service) loadQualityWeights(ctx context.Context) (analytics.QualityWeights, error) {
	if s.qualityWeightRepository == nil {
		return s.qualityWeights, nil
	}

	items, err := s.qualityWeightRepository.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("load quality weights: %w", err)
	}

	if len(items) == 0 {
		return s.qualityWeights, nil
	}

	return analytics.QualityWeightsFromDomain(items), nil
}
