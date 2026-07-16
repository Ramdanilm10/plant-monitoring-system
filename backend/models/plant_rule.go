package models

type PlantRule struct {

	PlantID int

	MinTemperature float64
	MaxTemperature float64

	MinHumidity float64
	MaxHumidity float64

	MinSoilMoisture float64
	MaxSoilMoisture float64

}