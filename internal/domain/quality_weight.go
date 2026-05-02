package domain

import (
	"errors"
	"time"
)

type QualityWeightID int64

type QualityWeight struct {
	ID            QualityWeightID `json:"id"`
	ParameterType ParameterType   `json:"parameterType"`
	Weight        float64         `json:"weight"`
	CreatedAt     time.Time       `json:"createdAt"`
	UpdatedAt     time.Time       `json:"updatedAt"`
	UpdatedBy     string          `json:"updatedBy,omitempty"`
}

type QualityWeightUpdate struct {
	Weight float64 `json:"weight"`
}

func ValidateQualityWeightUpdate(update QualityWeightUpdate) error {
	if update.Weight <= 0 {
		return errors.New("weight must be positive")
	}

	if update.Weight > 10 {
		return errors.New("weight must not be greater than 10")
	}

	return nil
}
