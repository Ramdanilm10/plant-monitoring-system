package services

import (
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"plant-monitoring-backend/config"
)

const (
	defaultLidahMertuaPlantID int64 = 1

	defaultAnthuriumPlantID int64 = 2

	collectorInterval = 30 * time.Minute

	// Transaction-level advisory lock.
	//
	// Lock ini mencegah beberapa instance Vercel
	// menyimpan data pada slot waktu yang sama.
	collectorAdvisoryLockKey int64 = 2607193001
)

// SensorCollectionResult merupakan hasil satu proses
// pengambilan data dari Blynk.
type SensorCollectionResult struct {
	Inserted bool `json:"inserted"`

	RowsInserted int `json:"rows_inserted"`

	Source string `json:"source"`

	PlantIDs []int64 `json:"plant_ids"`

	Temperature float64 `json:"temperature"`

	Humidity float64 `json:"humidity"`

	SoilMoisture1 float64 `json:"soil_moisture_1"`

	SoilMoisture2 float64 `json:"soil_moisture_2"`

	RecordedAt time.Time `json:"recorded_at"`

	BucketStart time.Time `json:"bucket_start"`

	BucketEnd time.Time `json:"bucket_end"`
}

type blynkSensorSnapshot struct {
	temperature float64

	humidity float64

	soilMoisture1 float64

	soilMoisture2 float64
}

// CollectSensorData tetap dipertahankan agar kompatibel
// dengan scheduler lokal pada main.go.
//
// Pada Vercel, BLYNK_COLLECTOR_ENABLED harus tetap false.
func CollectSensorData() {
	result, err := CollectSensorDataNow()
	if err != nil {
		log.Printf(
			"Collector Blynk gagal: %v",
			err,
		)

		return
	}

	if !result.Inserted {
		log.Printf(
			"Collector Blynk dilewati karena slot %s sampai %s sudah memiliki data",
			result.BucketStart.Format(
				time.RFC3339,
			),
			result.BucketEnd.Format(
				time.RFC3339,
			),
		)

		return
	}

	log.Printf(
		"Collector Blynk berhasil menyimpan %d baris untuk tanaman %d dan %d pada %s",
		result.RowsInserted,
		result.PlantIDs[0],
		result.PlantIDs[1],
		result.RecordedAt.Format(
			time.RFC3339,
		),
	)
}

// CollectSensorDataNow melakukan satu proses pengambilan data:
//
// 1. Membaca virtual pin Blynk.
// 2. Menentukan slot pengukuran 30 menit.
// 3. Memeriksa duplikasi.
// 4. Menyimpan data untuk dua tanaman.
func CollectSensorDataNow() (
	*SensorCollectionResult,
	error,
) {
	plantID1, plantID2, err :=
		getCollectorPlantIDs()

	if err != nil {
		return nil, err
	}

	snapshot, err :=
		readBlynkSensorSnapshot()

	if err != nil {
		return nil, err
	}

	recordedAt := time.Now().UTC()

	bucketStart := recordedAt.Truncate(
		collectorInterval,
	)

	bucketEnd := bucketStart.Add(
		collectorInterval,
	)

	result := &SensorCollectionResult{
		Source: "blynk",

		PlantIDs: []int64{
			plantID1,
			plantID2,
		},

		Temperature: snapshot.temperature,

		Humidity: snapshot.humidity,

		SoilMoisture1: snapshot.soilMoisture1,

		SoilMoisture2: snapshot.soilMoisture2,

		RecordedAt: recordedAt,

		BucketStart: bucketStart,

		BucketEnd: bucketEnd,
	}

	rowsInserted, err := saveBlynkSnapshot(
		result,
		plantID1,
		plantID2,
	)

	if err != nil {
		return nil, err
	}

	result.RowsInserted = rowsInserted

	result.Inserted =
		rowsInserted > 0

	return result, nil
}

func readBlynkSensorSnapshot() (
	*blynkSensorSnapshot,
	error,
) {
	soil1, err := readBlynkNumber(
		"V0",
	)

	if err != nil {
		return nil, err
	}

	soil2, err := readBlynkNumber(
		"V1",
	)

	if err != nil {
		return nil, err
	}

	temperature, err := readBlynkNumber(
		"V2",
	)

	if err != nil {
		return nil, err
	}

	humidity, err := readBlynkNumber(
		"V3",
	)

	if err != nil {
		return nil, err
	}

	return &blynkSensorSnapshot{
		temperature: temperature,

		humidity: humidity,

		soilMoisture1: soil1,

		soilMoisture2: soil2,
	}, nil
}

func readBlynkNumber(
	pin string,
) (float64, error) {
	rawValue, err := GetBlynkValue(
		pin,
	)

	if err != nil {
		return 0, fmt.Errorf(
			"gagal mengambil %s: %w",
			pin,
			err,
		)
	}

	return parseBlynkNumber(
		rawValue,
		pin,
	)
}

func saveBlynkSnapshot(
	result *SensorCollectionResult,
	plantID1 int64,
	plantID2 int64,
) (int, error) {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		20*time.Second,
	)

	defer cancel()

	transaction, err := config.DB.BeginTx(
		ctx,
		nil,
	)

	if err != nil {
		return 0, fmt.Errorf(
			"gagal memulai transaksi database: %w",
			err,
		)
	}

	committed := false

	defer func() {
		if !committed {
			_ = transaction.Rollback()
		}
	}()

	// Transaction advisory lock mencegah dua invocation
	// menyimpan data secara bersamaan.
	_, err = transaction.ExecContext(
		ctx,
		"SELECT pg_advisory_xact_lock($1)",
		collectorAdvisoryLockKey,
	)

	if err != nil {
		return 0, fmt.Errorf(
			"gagal mengunci proses collector: %w",
			err,
		)
	}

	// Periksa apakah data masing-masing tanaman
	// sudah tersedia pada slot 30 menit saat ini.
	const existingQuery = `
		SELECT
			EXISTS
			(
				SELECT 1
				FROM sensor_logs
				WHERE source = $1
				  AND plant_id = $2
				  AND recorded_at >= $4
				  AND recorded_at < $5
			),
			EXISTS
			(
				SELECT 1
				FROM sensor_logs
				WHERE source = $1
				  AND plant_id = $3
				  AND recorded_at >= $4
				  AND recorded_at < $5
			)
	`

	var plant1Exists bool

	var plant2Exists bool

	err = transaction.QueryRowContext(
		ctx,
		existingQuery,
		"blynk",
		plantID1,
		plantID2,
		result.BucketStart,
		result.BucketEnd,
	).Scan(
		&plant1Exists,
		&plant2Exists,
	)

	if err != nil {
		return 0, fmt.Errorf(
			"gagal memeriksa duplikasi data collector: %w",
			err,
		)
	}

	if plant1Exists &&
		plant2Exists {
		if err := transaction.Commit(); err != nil {
			return 0, fmt.Errorf(
				"gagal menyelesaikan pemeriksaan duplikasi: %w",
				err,
			)
		}

		committed = true

		return 0, nil
	}

	const insertQuery = `
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
	`

	rowsInserted := 0

	if !plant1Exists {
		_, err = transaction.ExecContext(
			ctx,
			insertQuery,
			plantID1,
			result.Temperature,
			result.Humidity,
			result.SoilMoisture1,
			result.RecordedAt,
			result.Source,
		)

		if err != nil {
			return 0, fmt.Errorf(
				"gagal menyimpan data tanaman %d: %w",
				plantID1,
				err,
			)
		}

		rowsInserted++
	}

	if !plant2Exists {
		_, err = transaction.ExecContext(
			ctx,
			insertQuery,
			plantID2,
			result.Temperature,
			result.Humidity,
			result.SoilMoisture2,
			result.RecordedAt,
			result.Source,
		)

		if err != nil {
			return 0, fmt.Errorf(
				"gagal menyimpan data tanaman %d: %w",
				plantID2,
				err,
			)
		}

		rowsInserted++
	}

	if err := transaction.Commit(); err != nil {
		return 0, fmt.Errorf(
			"gagal menyelesaikan transaksi database: %w",
			err,
		)
	}

	committed = true

	return rowsInserted, nil
}

func getCollectorPlantIDs() (
	int64,
	int64,
	error,
) {
	plantID1, err :=
		getPositiveInt64Environment(
			"SOIL_1_PLANT_ID",
			defaultLidahMertuaPlantID,
		)

	if err != nil {
		return 0, 0, err
	}

	plantID2, err :=
		getPositiveInt64Environment(
			"SOIL_2_PLANT_ID",
			defaultAnthuriumPlantID,
		)

	if err != nil {
		return 0, 0, err
	}

	if plantID1 == plantID2 {
		return 0, 0, fmt.Errorf(
			"SOIL_1_PLANT_ID dan SOIL_2_PLANT_ID tidak boleh sama",
		)
	}

	return plantID1, plantID2, nil
}

func getPositiveInt64Environment(
	key string,
	defaultValue int64,
) (int64, error) {
	rawValue := strings.TrimSpace(
		os.Getenv(key),
	)

	if rawValue == "" {
		return defaultValue, nil
	}

	value, err := strconv.ParseInt(
		rawValue,
		10,
		64,
	)

	if err != nil ||
		value <= 0 {
		return 0, fmt.Errorf(
			"%s harus berupa bilangan bulat positif",
			key,
		)
	}

	return value, nil
}

func parseBlynkNumber(
	rawValue string,
	pin string,
) (float64, error) {
	cleanValue := strings.Trim(
		strings.TrimSpace(
			rawValue,
		),
		`"`,
	)

	value, err := strconv.ParseFloat(
		cleanValue,
		64,
	)

	if err != nil {
		return 0, fmt.Errorf(
			"nilai %s tidak valid (%q): %w",
			pin,
			rawValue,
			err,
		)
	}

	if math.IsNaN(value) ||
		math.IsInf(value, 0) {
		return 0, fmt.Errorf(
			"nilai %s bukan angka normal",
			pin,
		)
	}

	return value, nil
}
