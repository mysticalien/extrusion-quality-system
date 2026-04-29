package domain

import "time"

// TelemetryReadingID identifies a telemetry reading stored by the application.
type TelemetryReadingID int64

// SourceID identifies the data source, such as a sensor, PLC tag, OPC UA node, or simulator.
type SourceID string

// ParameterType identifies a technological parameter measured on the extrusion line.
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

// Unit describes a measurement unit used for telemetry values.
type Unit string

const (
	UnitBar     Unit = "bar"
	UnitPercent Unit = "percent"
	UnitCelsius Unit = "celsius"
	UnitRPM     Unit = "rpm"
)

// TelemetryReading represents a single measured value received from the extrusion process.
type TelemetryReading struct {
	ID            TelemetryReadingID `json:"id"`
	ParameterType ParameterType      `json:"parameterType"`
	Value         float64            `json:"value"`
	Unit          Unit               `json:"unit"`
	SourceID      SourceID           `json:"sourceId"`

	// MeasuredAt is the time when the value was measured by the source system.
	MeasuredAt time.Time `json:"measuredAt"`

	// CreatedAt is the time when the reading was stored by the application.
	CreatedAt time.Time `json:"createdAt"`
}
