package models

type PlantRequest struct {
	Name string `json:"name"`
	Type string `json:"type"`

	MinTemperature float64 `json:"min_temperature"`
	MaxTemperature float64 `json:"max_temperature"`

	MinHumidity float64 `json:"min_humidity"`
	MaxHumidity float64 `json:"max_humidity"`

	MinSoilMoisture float64 `json:"min_soil_moisture"`
	MaxSoilMoisture float64 `json:"max_soil_moisture"`
}

type ManagedPlant struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`

	MinTemperature float64 `json:"min_temperature"`
	MaxTemperature float64 `json:"max_temperature"`

	MinHumidity float64 `json:"min_humidity"`
	MaxHumidity float64 `json:"max_humidity"`

	MinSoilMoisture float64 `json:"min_soil_moisture"`
	MaxSoilMoisture float64 `json:"max_soil_moisture"`
}
