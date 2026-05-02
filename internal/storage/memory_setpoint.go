package storage

import (
	"context"
	"extrusion-quality-system/internal/domain"
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
