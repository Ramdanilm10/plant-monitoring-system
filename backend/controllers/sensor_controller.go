package controllers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"plant-monitoring-backend/config"
	"plant-monitoring-backend/models"
	"plant-monitoring-backend/services"
)

func CreateSensorData(c *gin.Context) {
	var sensor models.SensorLog

	if err := c.ShouldBindJSON(&sensor); err != nil {
		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"message": "format data sensor tidak valid",
				"error":   err.Error(),
			},
		)

		return
	}

	if sensor.PlantID <= 0 {
		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"message": "plant_id wajib diisi",
			},
		)

		return
	}

	rule, err := services.GetPlantRule(
		sensor.PlantID,
	)

	if err != nil {
		c.JSON(
			http.StatusNotFound,
			gin.H{
				"message": "aturan tanaman tidak ditemukan",
				"error":   err.Error(),
			},
		)

		return
	}

	if strings.TrimSpace(sensor.Source) == "" {
		sensor.Source = "manual"
	}

	if sensor.RecordedAt.IsZero() {
		sensor.RecordedAt = time.Now()
	}

	const query = `
		INSERT INTO sensor_logs
		(
			plant_id,
			temperature,
			humidity,
			soil_moisture,
			recorded_at,
			source
		)
		VALUES
		(
			$1,
			$2,
			$3,
			$4,
			$5,
			$6
		)
		RETURNING
			id,
			recorded_at
	`

	err = config.DB.QueryRow(
		query,
		sensor.PlantID,
		sensor.Temperature,
		sensor.Humidity,
		sensor.SoilMoisture,
		sensor.RecordedAt,
		sensor.Source,
	).Scan(
		&sensor.ID,
		&sensor.RecordedAt,
	)

	if err != nil {
		c.JSON(
			http.StatusInternalServerError,
			gin.H{
				"message": "data sensor gagal disimpan",
				"error":   err.Error(),
			},
		)

		return
	}

	condition := services.CheckPlantCondition(
		sensor,
		rule,
	)

	c.JSON(
		http.StatusCreated,
		gin.H{
			"message":   "data sensor berhasil disimpan",
			"data":      sensor,
			"condition": condition,
		},
	)
}
