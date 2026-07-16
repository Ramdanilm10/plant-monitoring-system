package models

import "time"

type SensorLog struct {
	ID      int64 `json:"id"`
	PlantID int   `json:"plant_id"`

	DeviceCode string `json:"device_code,omitempty"`
	SequenceNo *int64 `json:"sequence_no,omitempty"`

	Temperature  float64 `json:"temperature"`
	Humidity     float64 `json:"humidity"`
	SoilMoisture float64 `json:"soil_moisture"`

	RecordedAt time.Time `json:"recorded_at"`
	ReceivedAt time.Time `json:"received_at"`

	Source string `json:"source"`
}
