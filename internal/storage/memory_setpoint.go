package storage

import (
	"context"
	"extrusion-quality-system/internal/domain"
	"time"
)

type MemorySetpointRepository struct {
	setpoints map[domain.ParameterType]domain.Setpoint
}

func NewMemorySetpointRepository(
	setpoints []domain.Setpoint,
) *MemorySetpointRepository {
	copiedSetpoints := make(map[domain.ParameterType]domain.Setpoint, len(setpoints))

	for _, setpoint := range setpoints {
		copiedSetpoints[setpoint.ParameterType] = setpoint
	}

	return &MemorySetpointRepository{
		setpoints: copiedSetpoints,
	}
}

func (r *MemorySetpointRepository) All(
	ctx context.Context,
) ([]domain.Setpoint, error) {
	_ = ctx

	result := make([]domain.Setpoint, 0, len(r.setpoints))

	for _, setpoint := range r.setpoints {
		result = append(result, setpoint)
	}

	return result, nil
}

func (r *MemorySetpointRepository) GetByParameter(
	ctx context.Context,
	parameterType domain.ParameterType,
) (domain.Setpoint, bool, error) {
	_ = ctx

	setpoint, found := r.setpoints[parameterType]

	return setpoint, found, nil
}

func (r *MemorySetpointRepository) Update(
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
