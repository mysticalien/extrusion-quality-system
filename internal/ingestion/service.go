package ingestion

import (
	"context"
	"errors"
	"extrusion-quality-system/internal/analytics"
	"extrusion-quality-system/internal/domain"
	"extrusion-quality-system/internal/storage"
	"fmt"
	"log/slog"
	"time"
)

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
	Accepted      bool                  `json:"accepted"`
	ParameterType domain.ParameterType  `json:"parameterType"`
	Value         float64               `json:"value"`
	Unit          domain.Unit           `json:"unit"`
	SourceID      domain.SourceID       `json:"sourceId"`
	MeasuredAt    time.Time             `json:"measuredAt"`
	State         domain.ParameterState `json:"state"`
	AlertCreated  bool                  `json:"alertCreated"`
	AlertID       *domain.AlertID       `json:"alertId,omitempty"`
	AlertLevel    *domain.AlertLevel    `json:"alertLevel,omitempty"`
	QualityIndex  float64               `json:"qualityIndex"`
	QualityState  domain.QualityState   `json:"qualityState"`
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
	logger              *slog.Logger
	telemetryRepository storage.TelemetryRepository
	alertRepository     storage.AlertRepository
	qualityRepository   storage.QualityRepository
	setpoints           map[domain.ParameterType]domain.Setpoint
}

// NewService creates telemetry ingestion service.
func NewService(
	logger *slog.Logger,
	telemetryRepository storage.TelemetryRepository,
	alertRepository storage.AlertRepository,
	qualityRepository storage.QualityRepository,
	setpoints map[domain.ParameterType]domain.Setpoint,
) *Service {
	return &Service{
		logger:              logger,
		telemetryRepository: telemetryRepository,
		alertRepository:     alertRepository,
		qualityRepository:   qualityRepository,
		setpoints:           setpoints,
	}
}

// Process validates, stores and analytically processes one telemetry reading.
func (s *Service) Process(ctx context.Context, input TelemetryInput) (TelemetryResult, error) {
	setpoint, err := s.validate(input)
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

	level, shouldCreateAlert := alertLevelFromParameterState(state)
	if shouldCreateAlert {
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

		alert, err = s.alertRepository.Create(ctx, alert)
		if err != nil {
			return TelemetryResult{}, fmt.Errorf("save alert event: %w", err)
		}

		id := alert.ID
		alertID = &id

		levelCopy := alert.Level
		alertLevel = &levelCopy
	}

	activeAlerts, err := s.alertRepository.Active(ctx)
	if err != nil {
		return TelemetryResult{}, fmt.Errorf("load active alerts: %w", err)
	}

	qualityIndex := analytics.CalculateQualityIndex(activeAlerts)

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
		"alertCreated", shouldCreateAlert,
		"qualityIndex", qualityIndex.Value,
		"qualityState", qualityIndex.State,
	)

	return TelemetryResult{
		Accepted:      true,
		ParameterType: reading.ParameterType,
		Value:         reading.Value,
		Unit:          reading.Unit,
		SourceID:      reading.SourceID,
		MeasuredAt:    reading.MeasuredAt,
		State:         state,
		AlertCreated:  shouldCreateAlert,
		AlertID:       alertID,
		AlertLevel:    alertLevel,
		QualityIndex:  qualityIndex.Value,
		QualityState:  qualityIndex.State,
	}, nil
}

func (s *Service) validate(input TelemetryInput) (domain.Setpoint, error) {
	setpoint, ok := s.setpoints[input.ParameterType]
	if !ok {
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
