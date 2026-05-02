package domain

import (
	"fmt"
	"time"
)

// SetpointID identifies a setpoint configuration.
type SetpointID int64

// ParameterState describes the current parameter state relative to configured setpoints.
type ParameterState string

const (
	ParameterStateNormal   ParameterState = "normal"
	ParameterStateWarning  ParameterState = "warning"
	ParameterStateCritical ParameterState = "critical"
)

// Setpoint defines normal, warning, and critical ranges for a technological parameter.
type Setpoint struct {
	ID            SetpointID    `json:"id"`
	ParameterType ParameterType `json:"parameterType"`
	Unit          Unit          `json:"unit"`

	CriticalMin float64 `json:"criticalMin"`
	WarningMin  float64 `json:"warningMin"`
	NormalMin   float64 `json:"normalMin"`
	NormalMax   float64 `json:"normalMax"`
	WarningMax  float64 `json:"warningMax"`
	CriticalMax float64 `json:"criticalMax"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	// UpdatedBy stores the user who last changed the setpoint configuration.
	UpdatedBy *UserID `json:"updatedBy,omitempty"`
}

type SetpointUpdate struct {
	CriticalMin float64 `json:"criticalMin"`
	WarningMin  float64 `json:"warningMin"`
	NormalMin   float64 `json:"normalMin"`
	NormalMax   float64 `json:"normalMax"`
	WarningMax  float64 `json:"warningMax"`
	CriticalMax float64 `json:"criticalMax"`
}

func (s Setpoint) Evaluate(value float64) ParameterState {
	if value < s.WarningMin || value > s.WarningMax {
		return ParameterStateCritical
	}

	if value < s.NormalMin || value > s.NormalMax {
		return ParameterStateWarning
	}

	return ParameterStateNormal
}

func ValidateSetpointUpdate(update SetpointUpdate) error {
	if update.CriticalMin > update.WarningMin {
		return fmt.Errorf("criticalMin must be less than or equal to warningMin")
	}

	if update.WarningMin > update.NormalMin {
		return fmt.Errorf("warningMin must be less than or equal to normalMin")
	}

	if update.NormalMin > update.NormalMax {
		return fmt.Errorf("normalMin must be less than or equal to normalMax")
	}

	if update.NormalMax > update.WarningMax {
		return fmt.Errorf("normalMax must be less than or equal to warningMax")
	}

	if update.WarningMax > update.CriticalMax {
		return fmt.Errorf("warningMax must be less than or equal to criticalMax")
	}

	return nil
}
