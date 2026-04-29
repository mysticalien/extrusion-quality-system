package domain

import "testing"

func TestSetpointEvaluate(t *testing.T) {
	setpoint := Setpoint{
		ParameterType: ParameterPressure,
		Unit:          UnitBar,
		WarningMin:    30,
		NormalMin:     40,
		NormalMax:     75,
		WarningMax:    90,
	}

	tests := []struct {
		name     string
		value    float64
		expected ParameterState
	}{
		{
			name:     "value below warning minimum is critical",
			value:    29.9,
			expected: ParameterStateCritical,
		},
		{
			name:     "value equal to warning minimum is warning",
			value:    30,
			expected: ParameterStateWarning,
		},
		{
			name:     "value between warning minimum and normal minimum is warning",
			value:    35,
			expected: ParameterStateWarning,
		},
		{
			name:     "value equal to normal minimum is normal",
			value:    40,
			expected: ParameterStateNormal,
		},
		{
			name:     "value inside normal range is normal",
			value:    60,
			expected: ParameterStateNormal,
		},
		{
			name:     "value equal to normal maximum is normal",
			value:    75,
			expected: ParameterStateNormal,
		},
		{
			name:     "value between normal maximum and warning maximum is warning",
			value:    82,
			expected: ParameterStateWarning,
		},
		{
			name:     "value equal to warning maximum is warning",
			value:    90,
			expected: ParameterStateWarning,
		},
		{
			name:     "value above warning maximum is critical",
			value:    90.1,
			expected: ParameterStateCritical,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := setpoint.Evaluate(tt.value)

			if actual != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, actual)
			}
		})
	}
}
