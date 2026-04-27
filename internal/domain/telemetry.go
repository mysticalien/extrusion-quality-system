package domain

import "time"

type ParameterType string

const (
	ParameterPressure               ParameterType = "pressure"
	ParameterMoisture               ParameterType = "moisture"
	ParameterBarrelTemperatureZone1 ParameterType = "barrel_temperature_zone_1"
	ParameterBarrelTemperatureZone2 ParameterType = "barrel_temperature_zone_2"
	ParameterBarrelTemperatureZone3 ParameterType = "barrel_temperature_zone_3"
	ParameterScrewSpeed             ParameterType = "screw_speed"
	ParameterDriveLoad              ParameterType = "drive_load"
	ParameterOutletTemperature      ParameterType = "outlet_temperature"
)

type Unit string

const (
	UnitBar     Unit = "bar"
	UnitPercent Unit = "%"
	UnitCelsius Unit = "°C"
	UnitRPM     Unit = "rpm"
)

// TelemetryReading represents a single measured value received from the extrusion process.
type TelemetryReading struct {
	ParameterType ParameterType
	Value         float64
	Unit          Unit
	SourceID      string
	MeasuredAt    time.Time
}
