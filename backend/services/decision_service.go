package services

import (
	"strings"

	"plant-monitoring-backend/models"
)

func CheckPlantCondition(
	sensor models.SensorLog,
	rule models.PlantRule,
) models.Alert {
	messages := make([]string, 0)
	recommendations := make([]string, 0)

	if sensor.SoilMoisture < rule.MinSoilMoisture {
		messages = append(
			messages,
			"Kelembapan tanah terlalu rendah",
		)

		recommendations = append(
			recommendations,
			"Lakukan penyiraman tanaman",
		)
	}

	if sensor.SoilMoisture > rule.MaxSoilMoisture {
		messages = append(
			messages,
			"Kelembapan tanah terlalu tinggi",
		)

		recommendations = append(
			recommendations,
			"Kurangi penyiraman dan periksa drainase",
		)
	}

	if sensor.Temperature < rule.MinTemperature {
		messages = append(
			messages,
			"Suhu udara terlalu rendah",
		)

		recommendations = append(
			recommendations,
			"Pindahkan tanaman ke lokasi yang lebih hangat",
		)
	}

	if sensor.Temperature > rule.MaxTemperature {
		messages = append(
			messages,
			"Suhu udara terlalu tinggi",
		)

		recommendations = append(
			recommendations,
			"Kurangi paparan panas dan perbaiki sirkulasi udara",
		)
	}

	if sensor.Humidity < rule.MinHumidity {
		messages = append(
			messages,
			"Kelembapan udara terlalu rendah",
		)

		recommendations = append(
			recommendations,
			"Tingkatkan kelembapan udara di sekitar tanaman",
		)
	}

	if sensor.Humidity > rule.MaxHumidity {
		messages = append(
			messages,
			"Kelembapan udara terlalu tinggi",
		)

		recommendations = append(
			recommendations,
			"Tingkatkan ventilasi di sekitar tanaman",
		)
	}

	if len(messages) == 0 {
		return models.Alert{
			PlantID: sensor.PlantID,
			Status:  "NORMAL",

			Message: "Kondisi tanaman normal",

			Recommendation: "Pertahankan kondisi lingkungan dan pola perawatan saat ini",
		}
	}

	return models.Alert{
		PlantID: sensor.PlantID,
		Status:  "WARNING",

		Message: strings.Join(
			messages,
			"; ",
		),

		Recommendation: strings.Join(
			recommendations,
			"; ",
		),
	}
}
