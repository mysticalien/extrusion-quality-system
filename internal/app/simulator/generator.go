package simulator

import (
	"time"

	"extrusion-quality-system/internal/domain"
)

func (a *app) generateReadings(measuredAt time.Time) []telemetryMessage {
	a.tickCount++

	switch a.mode {
	case SimulationModeWarning:
		return a.warningReadings(measuredAt)
	case SimulationModeCritical:
		return a.criticalReadings(measuredAt)
	case SimulationModeAnomaly:
		return a.anomalyReadings(measuredAt)
	default:
		return a.normalReadings(measuredAt)
	}
}

func (a *app) normalReadings(measuredAt time.Time) []telemetryMessage {
	return []telemetryMessage{
		a.reading(domain.ParameterPressure, round2(randomInRange(a.random, 62, 68)), domain.UnitBar, measuredAt),
		a.reading(domain.ParameterMoisture, round2(randomInRange(a.random, 24, 27)), domain.UnitPercent, measuredAt),
		a.reading(domain.ParameterBarrelTemperatureZone1, round2(randomInRange(a.random, 100, 115)), domain.UnitCelsius, measuredAt),
		a.reading(domain.ParameterBarrelTemperatureZone2, round2(randomInRange(a.random, 115, 130)), domain.UnitCelsius, measuredAt),
		a.reading(domain.ParameterBarrelTemperatureZone3, round2(randomInRange(a.random, 125, 145)), domain.UnitCelsius, measuredAt),
		a.reading(domain.ParameterScrewSpeed, round2(randomInRange(a.random, 280, 360)), domain.UnitRPM, measuredAt),
		a.reading(domain.ParameterDriveLoad, round2(randomInRange(a.random, 55, 70)), domain.UnitPercent, measuredAt),
		a.reading(domain.ParameterOutletTemperature, round2(randomInRange(a.random, 105, 120)), domain.UnitCelsius, measuredAt),
	}
}

func (a *app) warningReadings(measuredAt time.Time) []telemetryMessage {
	readings := a.normalReadings(measuredAt)

	replaceReading(readings, domain.ParameterPressure, round2(randomInRange(a.random, 81, 87)))
	replaceReading(readings, domain.ParameterDriveLoad, round2(randomInRange(a.random, 82, 88)))

	return readings
}

func (a *app) criticalReadings(measuredAt time.Time) []telemetryMessage {
	readings := a.normalReadings(measuredAt)

	replaceReading(readings, domain.ParameterPressure, round2(randomInRange(a.random, 94, 98)))
	replaceReading(readings, domain.ParameterDriveLoad, round2(randomInRange(a.random, 93, 98)))
	replaceReading(readings, domain.ParameterBarrelTemperatureZone3, round2(randomInRange(a.random, 165, 175)))
	replaceReading(readings, domain.ParameterOutletTemperature, round2(randomInRange(a.random, 145, 155)))

	return readings
}

func (a *app) anomalyReadings(measuredAt time.Time) []telemetryMessage {
	readings := a.normalReadings(measuredAt)

	tick := float64(a.tickCount)

	moisture := clamp(27-tick*0.3, 15, 27)
	pressure := clamp(55+tick*1.2, 55, 98)
	driveLoad := clamp(45+tick*1.0, 45, 96)

	replaceReading(readings, domain.ParameterMoisture, round2(moisture))
	replaceReading(readings, domain.ParameterPressure, round2(pressure))
	replaceReading(readings, domain.ParameterDriveLoad, round2(driveLoad))

	return readings
}

func (a *app) reading(
	parameterType domain.ParameterType,
	value float64,
	unit domain.Unit,
	measuredAt time.Time,
) telemetryMessage {
	return telemetryMessage{
		ParameterType: parameterType,
		Value:         value,
		Unit:          unit,
		SourceID:      domain.SourceID(a.cfg.SourceID),
		MeasuredAt:    measuredAt,
	}
}

func replaceReading(readings []telemetryMessage, parameterType domain.ParameterType, value float64) {
	for index := range readings {
		if readings[index].ParameterType == parameterType {
			readings[index].Value = value
			return
		}
	}
}
