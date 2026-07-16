package controllers

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"plant-monitoring-backend/middleware"
	"plant-monitoring-backend/models"
	"plant-monitoring-backend/services"
)

func CreateDeviceSensorReadings(
	c *gin.Context,
) {
	var request models.DeviceSensorRequest

	if err := c.ShouldBindJSON(
		&request,
	); err != nil {
		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"message": "format payload perangkat tidak valid",
				"error":   err.Error(),
			},
		)

		return
	}

	if err := validateDeviceSensorRequest(
		request,
	); err != nil {
		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"message": "nilai sensor tidak valid",
				"error":   err.Error(),
			},
		)

		return
	}

	deviceCodeValue, exists := c.Get(
		middleware.DeviceCodeContextKey,
	)

	if !exists {
		c.JSON(
			http.StatusInternalServerError,
			gin.H{
				"message": "identitas perangkat tidak tersedia pada request",
			},
		)

		return
	}

	deviceCode, validType :=
		deviceCodeValue.(string)

	if !validType ||
		strings.TrimSpace(deviceCode) == "" {
		c.JSON(
			http.StatusInternalServerError,
			gin.H{
				"message": "identitas perangkat tidak valid pada request",
			},
		)

		return
	}

	result, err := services.SaveDeviceSensorReadings(
		c.Request.Context(),
		deviceCode,
		c.ClientIP(),
		request,
	)

	if err != nil {
		statusCode := http.StatusInternalServerError

		message := "paket data perangkat gagal disimpan"

		if errors.Is(
			err,
			services.ErrSequencePayloadConflict,
		) {
			statusCode = http.StatusConflict

			message = "sequence_no sudah digunakan oleh payload berbeda"
		}

		c.JSON(
			statusCode,
			gin.H{
				"message": message,
				"error":   err.Error(),
			},
		)

		return
	}

	statusCode := http.StatusCreated

	message := "paket data perangkat berhasil disimpan"

	if result.DuplicateRequest {
		statusCode = http.StatusOK

		message = "paket data sudah pernah diterima; tidak dibuat duplikat"
	}

	c.JSON(
		statusCode,
		gin.H{
			"message": message,

			"device_code": deviceCode,

			"duplicate_request": result.DuplicateRequest,

			"results": result.Readings,
		},
	)
}

func validateDeviceSensorRequest(
	request models.DeviceSensorRequest,
) error {
	if request.Temperature == nil ||
		request.Humidity == nil ||
		request.Soil1 == nil ||
		request.Soil2 == nil {
		return fmt.Errorf(
			"temperature, humidity, soil_1, dan soil_2 wajib diisi",
		)
	}

	if request.SequenceNo != nil &&
		*request.SequenceNo <= 0 {
		return fmt.Errorf(
			"sequence_no harus lebih besar dari 0",
		)
	}

	if len(
		strings.TrimSpace(
			request.FirmwareVersion,
		),
	) > 100 {
		return fmt.Errorf(
			"firmware_version maksimal 100 karakter",
		)
	}

	if err := validateFiniteRange(
		"temperature",
		*request.Temperature,
		-40,
		80,
	); err != nil {
		return err
	}

	if err := validateFiniteRange(
		"humidity",
		*request.Humidity,
		0,
		100,
	); err != nil {
		return err
	}

	if err := validateFiniteRange(
		"soil_1",
		*request.Soil1,
		0,
		100,
	); err != nil {
		return err
	}

	if err := validateFiniteRange(
		"soil_2",
		*request.Soil2,
		0,
		100,
	); err != nil {
		return err
	}

	if request.RecordedAt != nil {
		maximumFutureTime := time.Now().Add(
			5 * time.Minute,
		)

		if request.RecordedAt.After(
			maximumFutureTime,
		) {
			return fmt.Errorf(
				"recorded_at tidak boleh lebih dari 5 menit di masa depan",
			)
		}
	}

	return nil
}

func validateFiniteRange(
	fieldName string,
	value float64,
	minimum float64,
	maximum float64,
) error {
	if math.IsNaN(value) ||
		math.IsInf(value, 0) {
		return fmt.Errorf(
			"%s harus berupa angka normal",
			fieldName,
		)
	}

	if value < minimum ||
		value > maximum {
		return fmt.Errorf(
			"%s harus berada pada rentang %.0f sampai %.0f",
			fieldName,
			minimum,
			maximum,
		)
	}

	return nil
}
