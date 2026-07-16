package controllers

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"

	"plant-monitoring-backend/config"
	"plant-monitoring-backend/models"
)

// GetDeviceStatuses menampilkan status komunikasi
// seluruh perangkat yang pernah mengirim data langsung
// ke backend.
func GetDeviceStatuses(
	c *gin.Context,
) {
	const query = `
		SELECT
			device_code,
			COALESCE(
				firmware_version,
				''
			),
			last_seen_at,
			last_payload_at,
			COALESCE(
				last_ip,
				''
			),
			last_sequence_no,
			total_requests
		FROM device_status
		ORDER BY last_seen_at DESC
	`

	rows, err := config.DB.QueryContext(
		c.Request.Context(),
		query,
	)

	if err != nil {
		c.JSON(
			http.StatusInternalServerError,
			gin.H{
				"message": "gagal mengambil status perangkat",
				"error":   err.Error(),
			},
		)

		return
	}

	defer rows.Close()

	statuses := make(
		[]models.DeviceStatus,
		0,
	)

	for rows.Next() {
		var status models.DeviceStatus

		var lastPayloadAt sql.NullTime

		var lastSequenceNo sql.NullInt64

		if err := rows.Scan(
			&status.DeviceCode,
			&status.FirmwareVersion,
			&status.LastSeenAt,
			&lastPayloadAt,
			&status.LastIP,
			&lastSequenceNo,
			&status.TotalRequests,
		); err != nil {
			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"message": "gagal membaca status perangkat",
					"error":   err.Error(),
				},
			)

			return
		}

		if lastPayloadAt.Valid {
			value := lastPayloadAt.Time

			status.LastPayloadAt = &value
		}

		if lastSequenceNo.Valid {
			value := lastSequenceNo.Int64

			status.LastSequenceNo = &value
		}

		statuses = append(
			statuses,
			status,
		)
	}

	if err := rows.Err(); err != nil {
		c.JSON(
			http.StatusInternalServerError,
			gin.H{
				"message": "gagal menyelesaikan pembacaan status perangkat",
				"error":   err.Error(),
			},
		)

		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": statuses,
		},
	)
}
