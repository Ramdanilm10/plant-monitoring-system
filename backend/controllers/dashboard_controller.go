package controllers

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"plant-monitoring-backend/config"
	"plant-monitoring-backend/models"
	"plant-monitoring-backend/services"
)

const (
	syncCurrentMaximumAge = 35 * time.Minute
	syncDelayedMaximumAge = 65 * time.Minute
)

// GetDashboard mengambil data tanaman, sensor terbaru,
// kondisi tanaman, dan status monitoring perangkat.
func GetDashboard(c *gin.Context) {
	plantID, err := strconv.Atoi(
		c.Param("plant_id"),
	)

	if err != nil || plantID <= 0 {
		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"message": "plant_id tidak valid",
			},
		)

		return
	}

	plant, err := getDashboardPlant(
		c,
		plantID,
	)

	if err != nil {
		return
	}

	sensor, hasSensorData, err := getLatestDashboardSensor(
		c,
		plantID,
	)

	if err != nil {
		return
	}

	if !hasSensorData {
		c.JSON(
			http.StatusOK,
			gin.H{
				"message": "Tanaman sudah terdaftar, tetapi belum memiliki data sensor.",

				"plant": plant,

				"sensor": nil,

				"condition": nil,

				"monitoring": nil,

				"has_sensor_data": false,
			},
		)

		return
	}

	rule, err := services.GetPlantRule(
		plantID,
	)

	if err != nil {
		if errors.Is(
			err,
			sql.ErrNoRows,
		) {
			c.JSON(
				http.StatusNotFound,
				gin.H{
					"message": "aturan tanaman tidak ditemukan",
				},
			)

			return
		}

		c.JSON(
			http.StatusInternalServerError,
			gin.H{
				"message": "gagal mengambil aturan tanaman",
				"error":   err.Error(),
			},
		)

		return
	}

	condition := services.CheckPlantCondition(
		sensor,
		rule,
	)

	monitoring := buildMonitoringStatus(
		sensor,
	)

	c.JSON(
		http.StatusOK,
		gin.H{
			"message": "Data dashboard berhasil dimuat.",

			"plant": plant,

			"sensor": sensor,

			"condition": condition,

			"monitoring": monitoring,

			"has_sensor_data": true,
		},
	)
}

func getDashboardPlant(
	c *gin.Context,
	plantID int,
) (models.Plant, error) {
	var plant models.Plant

	const query = `
		SELECT
			id,
			name,
			type
		FROM plants
		WHERE id = $1
	`

	err := config.DB.QueryRowContext(
		c.Request.Context(),
		query,
		plantID,
	).Scan(
		&plant.ID,
		&plant.Name,
		&plant.Type,
	)

	if err == nil {
		return plant, nil
	}

	if errors.Is(
		err,
		sql.ErrNoRows,
	) {
		c.JSON(
			http.StatusNotFound,
			gin.H{
				"message": "tanaman tidak ditemukan",
			},
		)

		return models.Plant{}, err
	}

	c.JSON(
		http.StatusInternalServerError,
		gin.H{
			"message": "gagal mengambil data tanaman",
			"error":   err.Error(),
		},
	)

	return models.Plant{}, err
}

func getLatestDashboardSensor(
	c *gin.Context,
	plantID int,
) (
	models.SensorLog,
	bool,
	error,
) {
	var sensor models.SensorLog

	var sequenceNo sql.NullInt64

	const query = `
		SELECT
			id,
			plant_id,
			COALESCE(device_code, ''),
			sequence_no,
			temperature,
			humidity,
			soil_moisture,
			recorded_at,
			received_at,
			COALESCE(source, '')
		FROM sensor_logs
		WHERE plant_id = $1
		ORDER BY recorded_at DESC
		LIMIT 1
	`

	err := config.DB.QueryRowContext(
		c.Request.Context(),
		query,
		plantID,
	).Scan(
		&sensor.ID,
		&sensor.PlantID,
		&sensor.DeviceCode,
		&sequenceNo,
		&sensor.Temperature,
		&sensor.Humidity,
		&sensor.SoilMoisture,
		&sensor.RecordedAt,
		&sensor.ReceivedAt,
		&sensor.Source,
	)

	if err == nil {
		if sequenceNo.Valid {
			value := sequenceNo.Int64

			sensor.SequenceNo = &value
		}

		return sensor, true, nil
	}

	if errors.Is(
		err,
		sql.ErrNoRows,
	) {
		return models.SensorLog{}, false, nil
	}

	c.JSON(
		http.StatusInternalServerError,
		gin.H{
			"message": "gagal mengambil data sensor",
			"error":   err.Error(),
		},
	)

	return models.SensorLog{}, false, err
}

func buildMonitoringStatus(
	sensor models.SensorLog,
) gin.H {
	deviceStatus := "UNKNOWN"

	deviceMessage :=
		"Status koneksi perangkat belum dapat diverifikasi."

	var deviceOnline *bool

	isOnline, err := services.GetBlynkConnectionStatus()

	if err != nil {
		log.Printf(
			"Pengecekan status perangkat Blynk gagal: %v",
			err,
		)
	} else {
		deviceOnlineValue := isOnline

		deviceOnline = &deviceOnlineValue

		if isOnline {
			deviceStatus = "ONLINE"

			deviceMessage =
				"ESP32 sedang terhubung ke Blynk Cloud."
		} else {
			deviceStatus = "OFFLINE"

			deviceMessage =
				"ESP32 tidak sedang terhubung ke Blynk Cloud."
		}
	}

	dataAge := time.Since(
		sensor.ReceivedAt,
	)

	if dataAge < 0 {
		dataAge = 0
	}

	syncStatus := "CURRENT"

	syncMessage :=
		"Data database masih berada dalam jadwal penyimpanan normal."

	switch {
	case dataAge <= syncCurrentMaximumAge:
		syncStatus = "CURRENT"

		syncMessage =
			"Data database masih berada dalam jadwal penyimpanan normal."

	case dataAge <= syncDelayedMaximumAge:
		syncStatus = "DELAYED"

		syncMessage =
			"Data terbaru melewati satu jadwal penyimpanan."

	default:
		syncStatus = "STALE"

		syncMessage =
			"Data database sudah terlalu lama dan perlu diperiksa."
	}

	return gin.H{
		"device_status": deviceStatus,

		"device_online": deviceOnline,

		"device_message": deviceMessage,

		"backend_sync_status": syncStatus,

		"backend_sync_message": syncMessage,

		"data_age_seconds": int64(
			dataAge.Seconds(),
		),

		"last_recorded_at": sensor.RecordedAt,

		"last_received_at": sensor.ReceivedAt,

		"source": sensor.Source,

		"device_code": sensor.DeviceCode,

		"auto_refresh_seconds": 30,
	}
}