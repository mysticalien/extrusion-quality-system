package domain

import "testing"

func TestSetpointEvaluate(t *testing.T) {
	setpoint := Setpoint{
		ParameterType: ParameterPressure,
		Unit:          UnitBar,
		CriticalMin:   30,
		WarningMin:    35,
		NormalMin:     40,
		NormalMax:     75,
		WarningMax:    90,
		CriticalMax:   95,
	}

	tests := []struct {
		name  string
		value float64
		want  ParameterState
	}{
		{
			name:  "normal inside range",
			value: 60,
			want:  ParameterStateNormal,
		},
		{
			name:  "normal at lower boundary",
			value: 40,
			want:  ParameterStateNormal,
		},
		{
			name:  "normal at upper boundary",
			value: 75,
			want:  ParameterStateNormal,
		},
		{
			name:  "warning below normal",
			value: 37,
			want:  ParameterStateWarning,
		},
		{
			name:  "warning at warning min",
			value: 35,
			want:  ParameterStateWarning,
		},
		{
			name:  "warning above normal",
			value: 80,
			want:  ParameterStateWarning,
		},
		{
			name:  "warning at warning max",
			value: 90,
			want:  ParameterStateWarning,
		},
		{
			name:  "critical below warning",
			value: 29,
			want:  ParameterStateCritical,
		},
		{
			name:  "critical at critical min",
			value: 30,
			want:  ParameterStateCritical,
		},
		{
			name:  "critical above warning",
			value: 96,
			want:  ParameterStateCritical,
		},
		{
			name:  "critical at critical max",
			value: 95,
			want:  ParameterStateCritical,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := setpoint.Evaluate(tt.value)

			if got != tt.want {
				t.Fatalf("Evaluate(%v) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestValidateSetpointUpdate(t *testing.T) {
	tests := []struct {
		name    string
		update  SetpointUpdate
		wantErr bool
	}{
		{
			name: "valid ranges",
			update: SetpointUpdate{
				CriticalMin: 30,
				WarningMin:  35,
				NormalMin:   40,
				NormalMax:   75,
				WarningMax:  90,
				CriticalMax: 95,
			},
			wantErr: false,
		},
		{
			name: "critical min greater than warning min",
			update: SetpointUpdate{
				CriticalMin: 40,
				WarningMin:  35,
				NormalMin:   45,
				NormalMax:   75,
				WarningMax:  90,
				CriticalMax: 95,
			},
			wantErr: true,
		},
		{
			name: "warning min greater than normal min",
			update: SetpointUpdate{
				CriticalMin: 30,
				WarningMin:  50,
				NormalMin:   40,
				NormalMax:   75,
				WarningMax:  90,
				CriticalMax: 95,
			},
			wantErr: true,
		},
		{
			name: "normal min greater than normal max",
			update: SetpointUpdate{
				CriticalMin: 30,
				WarningMin:  35,
				NormalMin:   80,
				NormalMax:   75,
				WarningMax:  90,
				CriticalMax: 95,
			},
			wantErr: true,
		},
		{
			name: "normal max greater than warning max",
			update: SetpointUpdate{
				CriticalMin: 30,
				WarningMin:  35,
				NormalMin:   40,
				NormalMax:   95,
				WarningMax:  90,
				CriticalMax: 100,
			},
			wantErr: true,
		},
		{
			name: "warning max greater than critical max",
			update: SetpointUpdate{
				CriticalMin: 30,
				WarningMin:  35,
				NormalMin:   40,
				NormalMax:   75,
				WarningMax:  100,
				CriticalMax: 95,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSetpointUpdate(tt.update)

			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}

			if !tt.wantErr && err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
		})
	}
}
