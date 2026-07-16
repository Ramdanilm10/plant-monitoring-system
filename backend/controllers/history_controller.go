package controllers

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"plant-monitoring-backend/config"
	"plant-monitoring-backend/models"
)

type historyRange struct {
	Code          string
	Label         string
	RangeSeconds  int64
	BucketSeconds int64
}

type sensorHistoryPoint struct {
	RecordedAt   time.Time `json:"recorded_at"`
	Temperature  float64   `json:"temperature"`
	Humidity     float64   `json:"humidity"`
	SoilMoisture float64   `json:"soil_moisture"`
}

var allowedHistoryRanges = map[string]historyRange{
	"1h": {
		Code:          "1h",
		Label:         "1 Jam",
		RangeSeconds:  60 * 60,
		BucketSeconds: 60,
	},
	"6h": {
		Code:          "6h",
		Label:         "6 Jam",
		RangeSeconds:  6 * 60 * 60,
		BucketSeconds: 5 * 60,
	},
	"24h": {
		Code:          "24h",
		Label:         "24 Jam",
		RangeSeconds:  24 * 60 * 60,
		BucketSeconds: 15 * 60,
	},
	"7d": {
		Code:          "7d",
		Label:         "7 Hari",
		RangeSeconds:  7 * 24 * 60 * 60,
		BucketSeconds: 60 * 60,
	},
	"30d": {
		Code:          "30d",
		Label:         "30 Hari",
		RangeSeconds:  30 * 24 * 60 * 60,
		BucketSeconds: 6 * 60 * 60,
	},
}

func GetSensorHistory(c *gin.Context) {
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

	rangeCode := c.DefaultQuery(
		"range",
		"24h",
	)

	selectedRange, validRange :=
		allowedHistoryRanges[rangeCode]

	if !validRange {
		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"message": "Rentang histori tidak valid. Gunakan 1h, 6h, 24h, 7d, atau 30d.",
			},
		)

		return
	}

	var plant models.Plant

	err = config.DB.QueryRowContext(
		c.Request.Context(),
		`
			SELECT
				id,
				name,
				type
			FROM plants
			WHERE id = $1
		`,
		plantID,
	).Scan(
		&plant.ID,
		&plant.Name,
		&plant.Type,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(
				http.StatusNotFound,
				gin.H{
					"message": "tanaman tidak ditemukan",
				},
			)

			return
		}

		c.JSON(
			http.StatusInternalServerError,
			gin.H{
				"message": "gagal mengambil data tanaman",
			},
		)

		return
	}

	rows, err := config.DB.QueryContext(
		c.Request.Context(),
		`
			SELECT
				to_timestamp(
					FLOOR(
						EXTRACT(
							EPOCH FROM recorded_at
						) / $2
					) * $2
				) AS bucket_time,

				ROUND(
					AVG(temperature)::numeric,
					2
				)::double precision,

				ROUND(
					AVG(humidity)::numeric,
					2
				)::double precision,

				ROUND(
					AVG(soil_moisture)::numeric,
					2
				)::double precision

			FROM sensor_logs

			WHERE
				plant_id = $1
				AND recorded_at >=
					NOW() - (
						$3 * INTERVAL '1 second'
					)

			GROUP BY bucket_time
			ORDER BY bucket_time ASC
		`,
		plantID,
		selectedRange.BucketSeconds,
		selectedRange.RangeSeconds,
	)

	if err != nil {
		c.JSON(
			http.StatusInternalServerError,
			gin.H{
				"message": "gagal mengambil histori sensor",
			},
		)

		return
	}

	defer rows.Close()

	readings := make(
		[]sensorHistoryPoint,
		0,
	)

	for rows.Next() {
		var reading sensorHistoryPoint

		err = rows.Scan(
			&reading.RecordedAt,
			&reading.Temperature,
			&reading.Humidity,
			&reading.SoilMoisture,
		)

		if err != nil {
			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"message": "gagal membaca histori sensor",
				},
			)

			return
		}

		readings = append(
			readings,
			reading,
		)
	}

	if err = rows.Err(); err != nil {
		c.JSON(
			http.StatusInternalServerError,
			gin.H{
				"message": "gagal memproses histori sensor",
			},
		)

		return
	}

	endAt := time.Now().UTC()

	startAt := endAt.Add(
		-time.Duration(
			selectedRange.RangeSeconds,
		) * time.Second,
	)

	c.JSON(
		http.StatusOK,
		gin.H{
			"message": "Histori sensor berhasil dimuat.",
			"data": gin.H{
				"plant": plant,

				"range": selectedRange.Code,

				"range_label": selectedRange.Label,

				"bucket_seconds": selectedRange.BucketSeconds,

				"start_at": startAt,

				"end_at": endAt,

				"total_points": len(readings),

				"readings": readings,
			},
		},
	)
}
