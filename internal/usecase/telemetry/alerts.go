package telemetry

import (
	"context"
	"fmt"
	"time"

	"extrusion-quality-system/internal/domain"
)

type alertProcessingResult struct {
	ID            *domain.AlertID
	Level         *domain.AlertLevel
	Created       bool
	Updated       bool
	ResolvedCount int64
}

func (s *Service) processAlerts(
	ctx context.Context,
	reading domain.TelemetryReading,
	state domain.ParameterState,
) (alertProcessingResult, error) {
	level, shouldCreateOrUpdateAlert := alertLevelFromParameterState(state)
	if shouldCreateOrUpdateAlert {
		alert, created, updated, err := s.createOrUpdateOpenAlert(ctx, reading, level)
		if err != nil {
			return alertProcessingResult{}, err
		}

		s.logAlertChange(alert, created, updated)

		id := alert.ID
		levelCopy := alert.Level

		return alertProcessingResult{
			ID:      &id,
			Level:   &levelCopy,
			Created: created,
			Updated: updated,
		}, nil
	}

	resolvedAlerts, err := s.alertRepository.ResolveOpenByParameter(ctx, reading.ParameterType)
	if err != nil {
		return alertProcessingResult{}, fmt.Errorf("resolve open alerts by parameter: %w", err)
	}

	if resolvedAlerts > 0 {
		s.logger.Info(
			"alerts resolved",
			"parameterType", reading.ParameterType,
			"resolvedAlerts", resolvedAlerts,
		)
	}

	return alertProcessingResult{
		ResolvedCount: resolvedAlerts,
	}, nil
}

func (s *Service) logAlertChange(alert domain.AlertEvent, created bool, updated bool) {
	if created {
		s.logger.Warn(
			"alert created",
			"alertId", alert.ID,
			"parameterType", alert.ParameterType,
			"level", alert.Level,
			"value", alert.Value,
			"unit", alert.Unit,
			"sourceId", alert.SourceID,
		)
	}

	if updated {
		s.logger.Warn(
			"alert updated",
			"alertId", alert.ID,
			"parameterType", alert.ParameterType,
			"level", alert.Level,
			"value", alert.Value,
			"unit", alert.Unit,
			"sourceId", alert.SourceID,
		)
	}
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
