package controllers

import (
	"database/sql"
	"errors"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"plant-monitoring-backend/config"
	"plant-monitoring-backend/models"
)

type dssRangeConfig struct {
	Code    string
	Label   string
	Seconds int64
}

type dssMetric struct {
	Label         string  `json:"label"`
	Unit          string  `json:"unit"`
	Minimum       float64 `json:"minimum"`
	Average       float64 `json:"average"`
	Maximum       float64 `json:"maximum"`
	IdealMinimum  float64 `json:"ideal_minimum"`
	IdealMaximum  float64 `json:"ideal_maximum"`
	BelowPercent  float64 `json:"below_percent"`
	NormalPercent float64 `json:"normal_percent"`
	AbovePercent  float64 `json:"above_percent"`
	Trend         string  `json:"trend"`
	TrendChange   float64 `json:"trend_change"`
}

type dssRecommendation struct {
	Level  string `json:"level"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

type dssResponse struct {
	HasData         bool                 `json:"has_data"`
	Plant           models.Plant         `json:"plant"`
	Range           string               `json:"range"`
	RangeLabel      string               `json:"range_label"`
	TotalReadings   int64                `json:"total_readings"`
	PeriodStart     *time.Time           `json:"period_start"`
	PeriodEnd       *time.Time           `json:"period_end"`
	HealthScore     float64              `json:"health_score"`
	Status          string               `json:"status"`
	Summary         string               `json:"summary"`
	Metrics         map[string]dssMetric `json:"metrics"`
	Recommendations []dssRecommendation  `json:"recommendations"`
}

type dssAggregate struct {
	Total int64

	PeriodStart sql.NullTime
	PeriodEnd   sql.NullTime

	TemperatureAverage sql.NullFloat64
	TemperatureMinimum sql.NullFloat64
	TemperatureMaximum sql.NullFloat64
	TemperatureFirst   sql.NullFloat64
	TemperatureLast    sql.NullFloat64

	HumidityAverage sql.NullFloat64
	HumidityMinimum sql.NullFloat64
	HumidityMaximum sql.NullFloat64
	HumidityFirst   sql.NullFloat64
	HumidityLast    sql.NullFloat64

	SoilAverage sql.NullFloat64
	SoilMinimum sql.NullFloat64
	SoilMaximum sql.NullFloat64
	SoilFirst   sql.NullFloat64
	SoilLast    sql.NullFloat64

	TemperatureBelow  int64
	TemperatureNormal int64
	TemperatureAbove  int64

	HumidityBelow  int64
	HumidityNormal int64
	HumidityAbove  int64

	SoilBelow  int64
	SoilNormal int64
	SoilAbove  int64
}

var allowedDSSRanges = map[string]dssRangeConfig{
	"1h": {
		Code:    "1h",
		Label:   "1 Jam",
		Seconds: 60 * 60,
	},
	"6h": {
		Code:    "6h",
		Label:   "6 Jam",
		Seconds: 6 * 60 * 60,
	},
	"12h": {
		Code:    "12h",
		Label:   "12 Jam",
		Seconds: 12 * 60 * 60,
	},
	"24h": {
		Code:    "24h",
		Label:   "24 Jam",
		Seconds: 24 * 60 * 60,
	},
	"7d": {
		Code:    "7d",
		Label:   "7 Hari",
		Seconds: 7 * 24 * 60 * 60,
	},
	"30d": {
		Code:    "30d",
		Label:   "30 Hari",
		Seconds: 30 * 24 * 60 * 60,
	},
}

func GetDSSAnalysis(c *gin.Context) {
	plantID, err := strconv.Atoi(
		c.Param("plant_id"),
	)

	if err != nil || plantID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "plant_id tidak valid",
		})

		return
	}

	rangeCode := c.DefaultQuery(
		"range",
		"24h",
	)

	selectedRange, valid :=
		allowedDSSRanges[rangeCode]

	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Rentang DSS tidak valid. Gunakan 1h, 6h, 12h, 24h, 7d, atau 30d.",
		})

		return
	}

	var plant models.Plant
	var rule models.PlantRule

	err = config.DB.QueryRowContext(
		c.Request.Context(),
		`
			SELECT
				p.id,
				p.name,
				p.type,
				pr.plant_id,
				pr.min_temperature,
				pr.max_temperature,
				pr.min_humidity,
				pr.max_humidity,
				pr.min_soil_moisture,
				pr.max_soil_moisture
			FROM plants p
			INNER JOIN plant_rules pr
				ON pr.plant_id = p.id
			WHERE p.id = $1
		`,
		plantID,
	).Scan(
		&plant.ID,
		&plant.Name,
		&plant.Type,
		&rule.PlantID,
		&rule.MinTemperature,
		&rule.MaxTemperature,
		&rule.MinHumidity,
		&rule.MaxHumidity,
		&rule.MinSoilMoisture,
		&rule.MaxSoilMoisture,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{
				"message": "tanaman atau aturan tanaman tidak ditemukan",
			})

			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "gagal mengambil aturan tanaman",
		})

		return
	}

	var aggregate dssAggregate

	err = config.DB.QueryRowContext(
		c.Request.Context(),
		`
			SELECT
				COUNT(*)::bigint,
				MIN(recorded_at),
				MAX(recorded_at),

				AVG(temperature),
				MIN(temperature),
				MAX(temperature),
				(ARRAY_AGG(
					temperature
					ORDER BY recorded_at ASC
				))[1],
				(ARRAY_AGG(
					temperature
					ORDER BY recorded_at DESC
				))[1],

				AVG(humidity),
				MIN(humidity),
				MAX(humidity),
				(ARRAY_AGG(
					humidity
					ORDER BY recorded_at ASC
				))[1],
				(ARRAY_AGG(
					humidity
					ORDER BY recorded_at DESC
				))[1],

				AVG(soil_moisture),
				MIN(soil_moisture),
				MAX(soil_moisture),
				(ARRAY_AGG(
					soil_moisture
					ORDER BY recorded_at ASC
				))[1],
				(ARRAY_AGG(
					soil_moisture
					ORDER BY recorded_at DESC
				))[1],

				COUNT(*) FILTER (
					WHERE temperature < $2
				)::bigint,

				COUNT(*) FILTER (
					WHERE temperature >= $2
					AND temperature <= $3
				)::bigint,

				COUNT(*) FILTER (
					WHERE temperature > $3
				)::bigint,

				COUNT(*) FILTER (
					WHERE humidity < $4
				)::bigint,

				COUNT(*) FILTER (
					WHERE humidity >= $4
					AND humidity <= $5
				)::bigint,

				COUNT(*) FILTER (
					WHERE humidity > $5
				)::bigint,

				COUNT(*) FILTER (
					WHERE soil_moisture < $6
				)::bigint,

				COUNT(*) FILTER (
					WHERE soil_moisture >= $6
					AND soil_moisture <= $7
				)::bigint,

				COUNT(*) FILTER (
					WHERE soil_moisture > $7
				)::bigint

			FROM sensor_logs

			WHERE
				plant_id = $1
				AND recorded_at >=
					NOW() - (
						$8 * INTERVAL '1 second'
					)
		`,
		plantID,
		rule.MinTemperature,
		rule.MaxTemperature,
		rule.MinHumidity,
		rule.MaxHumidity,
		rule.MinSoilMoisture,
		rule.MaxSoilMoisture,
		selectedRange.Seconds,
	).Scan(
		&aggregate.Total,
		&aggregate.PeriodStart,
		&aggregate.PeriodEnd,

		&aggregate.TemperatureAverage,
		&aggregate.TemperatureMinimum,
		&aggregate.TemperatureMaximum,
		&aggregate.TemperatureFirst,
		&aggregate.TemperatureLast,

		&aggregate.HumidityAverage,
		&aggregate.HumidityMinimum,
		&aggregate.HumidityMaximum,
		&aggregate.HumidityFirst,
		&aggregate.HumidityLast,

		&aggregate.SoilAverage,
		&aggregate.SoilMinimum,
		&aggregate.SoilMaximum,
		&aggregate.SoilFirst,
		&aggregate.SoilLast,

		&aggregate.TemperatureBelow,
		&aggregate.TemperatureNormal,
		&aggregate.TemperatureAbove,

		&aggregate.HumidityBelow,
		&aggregate.HumidityNormal,
		&aggregate.HumidityAbove,

		&aggregate.SoilBelow,
		&aggregate.SoilNormal,
		&aggregate.SoilAbove,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "gagal menghitung analisis DSS",
			"error":   err.Error(),
		})

		return
	}

	if aggregate.Total == 0 {
		response := dssResponse{
			HasData:         false,
			Plant:           plant,
			Range:           selectedRange.Code,
			RangeLabel:      selectedRange.Label,
			TotalReadings:   0,
			HealthScore:     0,
			Status:          "BELUM ADA DATA",
			Summary:         "Belum tersedia data sensor pada rentang waktu yang dipilih.",
			Metrics:         map[string]dssMetric{},
			Recommendations: []dssRecommendation{},
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Belum ada data untuk analisis DSS.",
			"data":    response,
		})

		return
	}

	temperatureMetric := buildDSSMetric(
		"Suhu udara",
		"°C",
		aggregate.TemperatureMinimum,
		aggregate.TemperatureAverage,
		aggregate.TemperatureMaximum,
		aggregate.TemperatureFirst,
		aggregate.TemperatureLast,
		rule.MinTemperature,
		rule.MaxTemperature,
		aggregate.TemperatureBelow,
		aggregate.TemperatureNormal,
		aggregate.TemperatureAbove,
		aggregate.Total,
		0.5,
	)

	humidityMetric := buildDSSMetric(
		"Kelembapan udara",
		"%",
		aggregate.HumidityMinimum,
		aggregate.HumidityAverage,
		aggregate.HumidityMaximum,
		aggregate.HumidityFirst,
		aggregate.HumidityLast,
		rule.MinHumidity,
		rule.MaxHumidity,
		aggregate.HumidityBelow,
		aggregate.HumidityNormal,
		aggregate.HumidityAbove,
		aggregate.Total,
		1,
	)

	soilMetric := buildDSSMetric(
		"Kelembapan tanah",
		"%",
		aggregate.SoilMinimum,
		aggregate.SoilAverage,
		aggregate.SoilMaximum,
		aggregate.SoilFirst,
		aggregate.SoilLast,
		rule.MinSoilMoisture,
		rule.MaxSoilMoisture,
		aggregate.SoilBelow,
		aggregate.SoilNormal,
		aggregate.SoilAbove,
		aggregate.Total,
		1,
	)

	healthScore := roundNumber(
		temperatureMetric.NormalPercent*0.30+
			humidityMetric.NormalPercent*0.25+
			soilMetric.NormalPercent*0.45,
		2,
	)

	status, summary :=
		createHealthStatus(healthScore)

	recommendations :=
		createDSSRecommendations(
			aggregate.Total,
			temperatureMetric,
			humidityMetric,
			soilMetric,
		)

	var periodStart *time.Time
	var periodEnd *time.Time

	if aggregate.PeriodStart.Valid {
		value := aggregate.PeriodStart.Time
		periodStart = &value
	}

	if aggregate.PeriodEnd.Valid {
		value := aggregate.PeriodEnd.Time
		periodEnd = &value
	}

	response := dssResponse{
		HasData:       true,
		Plant:         plant,
		Range:         selectedRange.Code,
		RangeLabel:    selectedRange.Label,
		TotalReadings: aggregate.Total,
		PeriodStart:   periodStart,
		PeriodEnd:     periodEnd,
		HealthScore:   healthScore,
		Status:        status,
		Summary:       summary,

		Metrics: map[string]dssMetric{
			"temperature":   temperatureMetric,
			"humidity":      humidityMetric,
			"soil_moisture": soilMetric,
		},

		Recommendations: recommendations,
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Analisis DSS berhasil dihitung.",
		"data":    response,
	})
}

func buildDSSMetric(
	label string,
	unit string,
	minimum sql.NullFloat64,
	average sql.NullFloat64,
	maximum sql.NullFloat64,
	first sql.NullFloat64,
	last sql.NullFloat64,
	idealMinimum float64,
	idealMaximum float64,
	belowCount int64,
	normalCount int64,
	aboveCount int64,
	total int64,
	trendThreshold float64,
) dssMetric {
	firstValue := nullFloatValue(first)
	lastValue := nullFloatValue(last)

	trendChange := roundNumber(
		lastValue-firstValue,
		2,
	)

	trend := "stabil"

	if trendChange > trendThreshold {
		trend = "naik"
	}

	if trendChange < -trendThreshold {
		trend = "turun"
	}

	return dssMetric{
		Label:        label,
		Unit:         unit,
		Minimum:      roundNumber(nullFloatValue(minimum), 2),
		Average:      roundNumber(nullFloatValue(average), 2),
		Maximum:      roundNumber(nullFloatValue(maximum), 2),
		IdealMinimum: idealMinimum,
		IdealMaximum: idealMaximum,
		BelowPercent: percentage(belowCount, total),
		NormalPercent: percentage(
			normalCount,
			total,
		),
		AbovePercent: percentage(aboveCount, total),
		Trend:        trend,
		TrendChange:  trendChange,
	}
}

func createHealthStatus(
	score float64,
) (string, string) {
	switch {
	case score >= 85:
		return "SANGAT BAIK",
			"Sebagian besar pembacaan sensor berada di dalam batas ideal tanaman."

	case score >= 70:
		return "BAIK",
			"Kondisi tanaman secara umum baik, tetapi masih terdapat beberapa pembacaan di luar batas ideal."

	case score >= 50:
		return "PERLU PERHATIAN",
			"Sejumlah kondisi lingkungan berada di luar batas ideal dan memerlukan tindakan perawatan."

	default:
		return "KRITIS",
			"Mayoritas kondisi lingkungan tidak sesuai dengan kebutuhan tanaman dan memerlukan tindakan segera."
	}
}

func createDSSRecommendations(
	totalReadings int64,
	temperature dssMetric,
	humidity dssMetric,
	soil dssMetric,
) []dssRecommendation {
	recommendations :=
		make([]dssRecommendation, 0)

	if totalReadings < 3 {
		recommendations = append(
			recommendations,
			dssRecommendation{
				Level:  "info",
				Title:  "Data belum cukup",
				Detail: "Jumlah pembacaan masih sedikit. Kumpulkan minimal tiga pembacaan sebelum menyimpulkan kondisi perawatan.",
			},
		)
	}

	if soil.BelowPercent >= 20 {
		recommendations = append(
			recommendations,
			dssRecommendation{
				Level: recommendationLevel(
					soil.BelowPercent,
				),
				Title:  "Kelembapan tanah sering rendah",
				Detail: "Periksa media tanam dan tingkatkan frekuensi penyiraman secara bertahap. Hindari penyiraman berlebihan sekaligus.",
			},
		)
	}

	if soil.AbovePercent >= 20 {
		recommendations = append(
			recommendations,
			dssRecommendation{
				Level: recommendationLevel(
					soil.AbovePercent,
				),
				Title:  "Kelembapan tanah sering tinggi",
				Detail: "Kurangi penyiraman dan periksa drainase agar akar tidak terlalu lama berada pada media yang basah.",
			},
		)
	}

	if temperature.BelowPercent >= 20 {
		recommendations = append(
			recommendations,
			dssRecommendation{
				Level: recommendationLevel(
					temperature.BelowPercent,
				),
				Title:  "Suhu sering terlalu rendah",
				Detail: "Pindahkan tanaman ke area yang lebih hangat atau kurangi paparan udara dingin.",
			},
		)
	}

	if temperature.AbovePercent >= 20 {
		recommendations = append(
			recommendations,
			dssRecommendation{
				Level: recommendationLevel(
					temperature.AbovePercent,
				),
				Title:  "Suhu sering terlalu tinggi",
				Detail: "Kurangi paparan panas langsung dan perbaiki sirkulasi udara di sekitar tanaman.",
			},
		)
	}

	if humidity.BelowPercent >= 20 {
		recommendations = append(
			recommendations,
			dssRecommendation{
				Level: recommendationLevel(
					humidity.BelowPercent,
				),
				Title:  "Kelembapan udara sering rendah",
				Detail: "Pertimbangkan peningkatan kelembapan lingkungan tanpa membuat media tanam terlalu basah.",
			},
		)
	}

	if humidity.AbovePercent >= 20 {
		recommendations = append(
			recommendations,
			dssRecommendation{
				Level: recommendationLevel(
					humidity.AbovePercent,
				),
				Title:  "Kelembapan udara sering tinggi",
				Detail: "Tingkatkan ventilasi untuk mengurangi risiko jamur dan penyakit tanaman.",
			},
		)
	}

	if len(recommendations) == 0 {
		recommendations = append(
			recommendations,
			dssRecommendation{
				Level:  "success",
				Title:  "Pertahankan perawatan",
				Detail: "Kondisi historis berada dalam rentang ideal. Pertahankan pola penyiraman dan lingkungan saat ini.",
			},
		)
	}

	return recommendations
}

func recommendationLevel(
	abnormalPercent float64,
) string {
	if abnormalPercent >= 50 {
		return "danger"
	}

	if abnormalPercent >= 30 {
		return "warning"
	}

	return "info"
}

func percentage(
	value int64,
	total int64,
) float64 {
	if total <= 0 {
		return 0
	}

	return roundNumber(
		float64(value)/
			float64(total)*
			100,
		2,
	)
}

func nullFloatValue(
	value sql.NullFloat64,
) float64 {
	if !value.Valid {
		return 0
	}

	return value.Float64
}

func roundNumber(
	value float64,
	precision int,
) float64 {
	factor := math.Pow(
		10,
		float64(precision),
	)

	return math.Round(
		value*factor,
	) / factor
}
