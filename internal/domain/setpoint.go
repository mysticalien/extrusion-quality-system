package domain

import (
	"time"
)

type SetpointID int64

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
	UpdatedBy *UserID   `json:"updatedBy,omitempty"`
}

//func (s Setpoint) Evaluate(value float64) ParameterState {
//	if value < s.CriticalMin || value > s.CriticalMax {
//		return ParameterStateCritical
//	}
//
//	if value < s.NormalMin || value > s.NormalMax {
//		return ParameterStateWarning
//	}
//
//	return ParameterStateNormal
//}
//
//func (s Setpoint) Validate() error {
//	if s.CriticalMin > s.WarningMin {
//		return errors.New("criticalMin must be less than or equal to warningMin")
//	}
//
//	if s.WarningMin > s.NormalMin {
//		return errors.New("warningMin must be less than or equal to normalMin")
//	}
//
//	if s.NormalMin > s.NormalMax {
//		return errors.New("normalMin must be less than or equal to normalMax")
//	}
//
//	if s.NormalMax > s.WarningMax {
//		return errors.New("normalMax must be less than or equal to warningMax")
//	}
//
//	if s.WarningMax > s.CriticalMax {
//		return errors.New("warningMax must be less than or equal to criticalMax")
//	}
//
//	return nil
//}
