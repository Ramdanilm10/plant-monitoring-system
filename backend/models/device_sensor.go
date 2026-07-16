package models

import "time"

type DeviceSensorRequest struct {
	SequenceNo      *int64     `json:"sequence_no"`
	FirmwareVersion string     `json:"firmware_version"`
	Temperature     *float64   `json:"temperature" binding:"required"`
	Humidity        *float64   `json:"humidity" binding:"required"`
	Soil1           *float64   `json:"soil_1" binding:"required"`
	Soil2           *float64   `json:"soil_2" binding:"required"`
	RecordedAt      *time.Time `json:"recorded_at"`
}

// DevicePlantReadingResult adalah hasil penyimpanan
// data untuk satu tanaman.
type DevicePlantReadingResult struct {
	PlantID   int       `json:"plant_id"`
	Duplicate bool      `json:"duplicate"`
	Sensor    SensorLog `json:"sensor"`
	Condition Alert     `json:"condition"`
}

// DeviceSensorSaveResult adalah hasil penyimpanan
// satu paket ESP32 yang terdiri dari dua tanaman.
type DeviceSensorSaveResult struct {
	DuplicateRequest bool `json:"duplicate_request"`

	Readings []DevicePlantReadingResult `json:"readings"`
}

// DeviceStatus menyimpan status komunikasi
// terakhir dari perangkat ESP32.
type DeviceStatus struct {
	DeviceCode      string `json:"device_code"`
	FirmwareVersion string `json:"firmware_version"`

	LastSeenAt    time.Time  `json:"last_seen_at"`
	LastPayloadAt *time.Time `json:"last_payload_at"`

	LastIP         string `json:"last_ip"`
	LastSequenceNo *int64 `json:"last_sequence_no"`

	TotalRequests int64 `json:"total_requests"`
}
