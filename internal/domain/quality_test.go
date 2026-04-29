package domain

import "testing"

func TestQualityStateFromValue(t *testing.T) {
	tests := []struct {
		name     string
		value    float64
		expected QualityState
	}{
		{
			name:     "value below unstable threshold is critical",
			value:    39.9,
			expected: QualityStateCritical,
		},
		{
			name:     "value equal to unstable threshold is unstable",
			value:    40,
			expected: QualityStateUnstable,
		},
		{
			name:     "value inside unstable range is unstable",
			value:    44,
			expected: QualityStateUnstable,
		},
		{
			name:     "value below warning threshold is unstable",
			value:    59.9,
			expected: QualityStateUnstable,
		},
		{
			name:     "value equal to warning threshold is warning",
			value:    60,
			expected: QualityStateWarning,
		},
		{
			name:     "value inside warning range is warning",
			value:    66,
			expected: QualityStateWarning,
		},
		{
			name:     "value below stable threshold is warning",
			value:    79.9,
			expected: QualityStateWarning,
		},
		{
			name:     "value equal to stable threshold is stable",
			value:    80,
			expected: QualityStateStable,
		},
		{
			name:     "value inside stable range is stable",
			value:    88,
			expected: QualityStateStable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := QualityStateFromValue(tt.value)

			if actual != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, actual)
			}
		})
	}
}
