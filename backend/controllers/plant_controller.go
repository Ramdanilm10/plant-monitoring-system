package controllers

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"

	"plant-monitoring-backend/config"
	"plant-monitoring-backend/models"
)

func validatePlantRequest(
	request models.PlantRequest,
) string {
	if strings.TrimSpace(request.Name) == "" {
		return "nama tanaman wajib diisi"
	}

	if strings.TrimSpace(request.Type) == "" {
		return "jenis tanaman wajib diisi"
	}

	if request.MinTemperature >= request.MaxTemperature {
		return "minimum suhu harus lebih kecil dari maksimum suhu"
	}

	if request.MinTemperature < -20 ||
		request.MaxTemperature > 80 {
		return "batas suhu harus berada antara -20°C sampai 80°C"
	}

	if request.MinHumidity < 0 ||
		request.MaxHumidity > 100 ||
		request.MinHumidity >= request.MaxHumidity {
		return "batas kelembapan udara harus valid antara 0 sampai 100 persen"
	}

	if request.MinSoilMoisture < 0 ||
		request.MaxSoilMoisture > 100 ||
		request.MinSoilMoisture >= request.MaxSoilMoisture {
		return "batas kelembapan tanah harus valid antara 0 sampai 100 persen"
	}

	return ""
}

func parsePlantManagementID(
	c *gin.Context,
) (int64, bool) {
	plantID, err := strconv.ParseInt(
		c.Param("id"),
		10,
		64,
	)

	if err != nil || plantID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "ID tanaman tidak valid",
		})

		return 0, false
	}

	return plantID, true
}

func isUniqueViolation(err error) bool {
	var databaseError *pq.Error

	if !errors.As(err, &databaseError) {
		return false
	}

	return databaseError.Code == "23505"
}

func GetPlants(c *gin.Context) {
	query := `
		SELECT
			p.id,
			p.name,
			p.type,
			pr.min_temperature,
			pr.max_temperature,
			pr.min_humidity,
			pr.max_humidity,
			pr.min_soil_moisture,
			pr.max_soil_moisture
		FROM plants p
		INNER JOIN plant_rules pr
			ON pr.plant_id = p.id
		ORDER BY p.id ASC
	`

	rows, err := config.DB.QueryContext(
		c.Request.Context(),
		query,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "gagal mengambil daftar tanaman",
		})

		return
	}

	defer rows.Close()

	plants := make([]models.ManagedPlant, 0)

	for rows.Next() {
		var plant models.ManagedPlant

		err = rows.Scan(
			&plant.ID,
			&plant.Name,
			&plant.Type,
			&plant.MinTemperature,
			&plant.MaxTemperature,
			&plant.MinHumidity,
			&plant.MaxHumidity,
			&plant.MinSoilMoisture,
			&plant.MaxSoilMoisture,
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "gagal membaca data tanaman",
			})

			return
		}

		plants = append(plants, plant)
	}

	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "gagal menyelesaikan pembacaan tanaman",
		})

		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": plants,
	})
}

func GetPlant(c *gin.Context) {
	plantID, valid := parsePlantManagementID(c)

	if !valid {
		return
	}

	var plant models.ManagedPlant

	query := `
		SELECT
			p.id,
			p.name,
			p.type,
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
	`

	err := config.DB.QueryRowContext(
		c.Request.Context(),
		query,
		plantID,
	).Scan(
		&plant.ID,
		&plant.Name,
		&plant.Type,
		&plant.MinTemperature,
		&plant.MaxTemperature,
		&plant.MinHumidity,
		&plant.MaxHumidity,
		&plant.MinSoilMoisture,
		&plant.MaxSoilMoisture,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{
				"message": "tanaman tidak ditemukan",
			})

			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "gagal mengambil data tanaman",
		})

		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": plant,
	})
}

func CreatePlant(c *gin.Context) {
	var request models.PlantRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "format data tanaman tidak valid",
		})

		return
	}

	request.Name = strings.TrimSpace(request.Name)
	request.Type = strings.TrimSpace(request.Type)

	if message := validatePlantRequest(request); message != "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": message,
		})

		return
	}

	transaction, err := config.DB.BeginTx(
		c.Request.Context(),
		nil,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "gagal memulai transaksi database",
		})

		return
	}

	defer transaction.Rollback()

	var plantID int64

	err = transaction.QueryRowContext(
		c.Request.Context(),
		`
			INSERT INTO plants (
				name,
				type
			)
			VALUES ($1, $2)
			RETURNING id
		`,
		request.Name,
		request.Type,
	).Scan(&plantID)

	if err != nil {
		if isUniqueViolation(err) {
			c.JSON(http.StatusConflict, gin.H{
				"message": "nama tanaman sudah digunakan",
			})

			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "gagal menambahkan tanaman",
		})

		return
	}

	_, err = transaction.ExecContext(
		c.Request.Context(),
		`
			INSERT INTO plant_rules (
				plant_id,
				min_temperature,
				max_temperature,
				min_humidity,
				max_humidity,
				min_soil_moisture,
				max_soil_moisture
			)
			VALUES (
				$1,
				$2,
				$3,
				$4,
				$5,
				$6,
				$7
			)
		`,
		plantID,
		request.MinTemperature,
		request.MaxTemperature,
		request.MinHumidity,
		request.MaxHumidity,
		request.MinSoilMoisture,
		request.MaxSoilMoisture,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "gagal menyimpan aturan tanaman",
		})

		return
	}

	if err := transaction.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "gagal menyelesaikan penambahan tanaman",
		})

		return
	}

	plant := models.ManagedPlant{
		ID:              plantID,
		Name:            request.Name,
		Type:            request.Type,
		MinTemperature:  request.MinTemperature,
		MaxTemperature:  request.MaxTemperature,
		MinHumidity:     request.MinHumidity,
		MaxHumidity:     request.MaxHumidity,
		MinSoilMoisture: request.MinSoilMoisture,
		MaxSoilMoisture: request.MaxSoilMoisture,
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "tanaman berhasil ditambahkan",
		"data":    plant,
	})
}

func UpdatePlant(c *gin.Context) {
	plantID, valid := parsePlantManagementID(c)

	if !valid {
		return
	}

	var request models.PlantRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "format data tanaman tidak valid",
		})

		return
	}

	request.Name = strings.TrimSpace(request.Name)
	request.Type = strings.TrimSpace(request.Type)

	if message := validatePlantRequest(request); message != "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": message,
		})

		return
	}

	transaction, err := config.DB.BeginTx(
		c.Request.Context(),
		nil,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "gagal memulai transaksi database",
		})

		return
	}

	defer transaction.Rollback()

	result, err := transaction.ExecContext(
		c.Request.Context(),
		`
			UPDATE plants
			SET
				name = $1,
				type = $2
			WHERE id = $3
		`,
		request.Name,
		request.Type,
		plantID,
	)

	if err != nil {
		if isUniqueViolation(err) {
			c.JSON(http.StatusConflict, gin.H{
				"message": "nama tanaman sudah digunakan",
			})

			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "gagal memperbarui tanaman",
		})

		return
	}

	affectedRows, err := result.RowsAffected()

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "gagal memeriksa perubahan tanaman",
		})

		return
	}

	if affectedRows == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "tanaman tidak ditemukan",
		})

		return
	}

	_, err = transaction.ExecContext(
		c.Request.Context(),
		`
			INSERT INTO plant_rules (
				plant_id,
				min_temperature,
				max_temperature,
				min_humidity,
				max_humidity,
				min_soil_moisture,
				max_soil_moisture
			)
			VALUES (
				$1,
				$2,
				$3,
				$4,
				$5,
				$6,
				$7
			)
			ON CONFLICT (plant_id)
			DO UPDATE SET
				min_temperature =
					EXCLUDED.min_temperature,
				max_temperature =
					EXCLUDED.max_temperature,
				min_humidity =
					EXCLUDED.min_humidity,
				max_humidity =
					EXCLUDED.max_humidity,
				min_soil_moisture =
					EXCLUDED.min_soil_moisture,
				max_soil_moisture =
					EXCLUDED.max_soil_moisture
		`,
		plantID,
		request.MinTemperature,
		request.MaxTemperature,
		request.MinHumidity,
		request.MaxHumidity,
		request.MinSoilMoisture,
		request.MaxSoilMoisture,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "gagal memperbarui aturan tanaman",
		})

		return
	}

	if err := transaction.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "gagal menyelesaikan perubahan tanaman",
		})

		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "tanaman berhasil diperbarui",
		"data": models.ManagedPlant{
			ID:              plantID,
			Name:            request.Name,
			Type:            request.Type,
			MinTemperature:  request.MinTemperature,
			MaxTemperature:  request.MaxTemperature,
			MinHumidity:     request.MinHumidity,
			MaxHumidity:     request.MaxHumidity,
			MinSoilMoisture: request.MinSoilMoisture,
			MaxSoilMoisture: request.MaxSoilMoisture,
		},
	})
}

func DeletePlant(c *gin.Context) {
	plantID, valid := parsePlantManagementID(c)

	if !valid {
		return
	}

	transaction, err := config.DB.BeginTx(
		c.Request.Context(),
		nil,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "gagal memulai transaksi database",
		})

		return
	}

	defer transaction.Rollback()

	var plantExists bool

	err = transaction.QueryRowContext(
		c.Request.Context(),
		`
			SELECT EXISTS (
				SELECT 1
				FROM plants
				WHERE id = $1
			)
		`,
		plantID,
	).Scan(&plantExists)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "gagal memeriksa tanaman",
		})

		return
	}

	if !plantExists {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "tanaman tidak ditemukan",
		})

		return
	}

	var sensorLogCount int64

	err = transaction.QueryRowContext(
		c.Request.Context(),
		`
			SELECT COUNT(*)
			FROM sensor_logs
			WHERE plant_id = $1
		`,
		plantID,
	).Scan(&sensorLogCount)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "gagal memeriksa histori sensor",
		})

		return
	}

	if sensorLogCount > 0 {
		c.JSON(http.StatusConflict, gin.H{
			"message": "tanaman memiliki histori sensor dan tidak dapat dihapus",
		})

		return
	}

	_, err = transaction.ExecContext(
		c.Request.Context(),
		`
			DELETE FROM alerts
			WHERE plant_id = $1
		`,
		plantID,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "gagal menghapus riwayat peringatan",
		})

		return
	}

	_, err = transaction.ExecContext(
		c.Request.Context(),
		`
			DELETE FROM plant_rules
			WHERE plant_id = $1
		`,
		plantID,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "gagal menghapus aturan tanaman",
		})

		return
	}

	_, err = transaction.ExecContext(
		c.Request.Context(),
		`
			DELETE FROM plants
			WHERE id = $1
		`,
		plantID,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "gagal menghapus tanaman",
		})

		return
	}

	if err := transaction.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "gagal menyelesaikan penghapusan tanaman",
		})

		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "tanaman berhasil dihapus",
	})
}
