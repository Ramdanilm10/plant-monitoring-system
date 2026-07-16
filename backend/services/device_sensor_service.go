package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"plant-monitoring-backend/config"
	"plant-monitoring-backend/models"
)

// ErrSequencePayloadConflict terjadi ketika satu
// sequence_no yang sudah tersimpan digunakan kembali,
// tetapi nilai sensor yang dikirim berbeda.
var ErrSequencePayloadConflict = errors.New(
	"sequence_no sudah digunakan dengan payload berbeda",
)

type devicePlantMapping struct {
	PlantID int

	SoilReading float64
}

// SaveDeviceSensorReadings menyimpan satu paket ESP32
// menjadi dua baris sensor_logs:
//
// soil_1 → SOIL_1_PLANT_ID
// soil_2 → SOIL_2_PLANT_ID
//
// SequenceNo bersifat opsional.
//
// Jika SequenceNo dikirim, unique index database
// membuat proses retry menjadi idempotent.
func SaveDeviceSensorReadings(
	ctx context.Context,
	deviceCode string,
	clientIP string,
	request models.DeviceSensorRequest,
) (models.DeviceSensorSaveResult, error) {
	soil1PlantID, err := requiredPositiveIntEnv(
		"SOIL_1_PLANT_ID",
	)

	if err != nil {
		return models.DeviceSensorSaveResult{}, err
	}

	soil2PlantID, err := requiredPositiveIntEnv(
		"SOIL_2_PLANT_ID",
	)

	if err != nil {
		return models.DeviceSensorSaveResult{}, err
	}

	if soil1PlantID == soil2PlantID {
		return models.DeviceSensorSaveResult{}, fmt.Errorf(
			"SOIL_1_PLANT_ID dan SOIL_2_PLANT_ID tidak boleh sama",
		)
	}

	recordedAt := time.Now()

	if request.RecordedAt != nil {
		recordedAt = *request.RecordedAt
	}

	receivedAt := time.Now()

	mappings := []devicePlantMapping{
		{
			PlantID: soil1PlantID,

			SoilReading: *request.Soil1,
		},
		{
			PlantID: soil2PlantID,

			SoilReading: *request.Soil2,
		},
	}

	// Aturan tanaman diambil sebelum transaksi insert.
	// Kalau salah satu tanaman belum memiliki rule,
	// tidak ada data yang disimpan.
	rules := make(
		map[int]models.PlantRule,
		len(mappings),
	)

	for _, mapping := range mappings {
		rule, ruleErr := GetPlantRule(
			mapping.PlantID,
		)

		if ruleErr != nil {
			return models.DeviceSensorSaveResult{}, fmt.Errorf(
				"aturan tanaman %d tidak ditemukan: %w",
				mapping.PlantID,
				ruleErr,
			)
		}

		rules[mapping.PlantID] = rule
	}

	transaction, err := config.DB.BeginTx(
		ctx,
		nil,
	)

	if err != nil {
		return models.DeviceSensorSaveResult{}, fmt.Errorf(
			"gagal memulai transaksi penyimpanan sensor: %w",
			err,
		)
	}

	committed := false

	defer func() {
		if !committed {
			_ = transaction.Rollback()
		}
	}()

	results := make(
		[]models.DevicePlantReadingResult,
		0,
		len(mappings),
	)

	duplicateCount := 0

	for _, mapping := range mappings {
		sensor, duplicate, saveErr :=
			saveOneDevicePlantReading(
				ctx,
				transaction,
				deviceCode,
				request.SequenceNo,
				*request.Temperature,
				*request.Humidity,
				mapping,
				recordedAt,
				receivedAt,
			)

		if saveErr != nil {
			return models.DeviceSensorSaveResult{}, saveErr
		}

		if duplicate {
			duplicateCount++
		}

		results = append(
			results,
			models.DevicePlantReadingResult{
				PlantID: mapping.PlantID,

				Duplicate: duplicate,

				Sensor: sensor,

				Condition: CheckPlantCondition(
					sensor,
					rules[mapping.PlantID],
				),
			},
		)
	}

	// Status perangkat selalu diperbarui, termasuk
	// ketika request adalah retry/duplicate yang valid.
	if err := upsertDeviceStatus(
		ctx,
		transaction,
		deviceCode,
		strings.TrimSpace(
			request.FirmwareVersion,
		),
		strings.TrimSpace(clientIP),
		request.SequenceNo,
		recordedAt,
		receivedAt,
	); err != nil {
		return models.DeviceSensorSaveResult{}, err
	}

	if err := transaction.Commit(); err != nil {
		return models.DeviceSensorSaveResult{}, fmt.Errorf(
			"gagal menyelesaikan transaksi penyimpanan sensor: %w",
			err,
		)
	}

	committed = true

	return models.DeviceSensorSaveResult{
		DuplicateRequest: duplicateCount == len(mappings),

		Readings: results,
	}, nil
}

// saveOneDevicePlantReading menyimpan data untuk
// satu tanaman.
func saveOneDevicePlantReading(
	ctx context.Context,
	transaction *sql.Tx,
	deviceCode string,
	sequenceNo *int64,
	temperature float64,
	humidity float64,
	mapping devicePlantMapping,
	recordedAt time.Time,
	receivedAt time.Time,
) (models.SensorLog, bool, error) {
	sensor := models.SensorLog{
		PlantID: mapping.PlantID,

		DeviceCode: deviceCode,

		SequenceNo: sequenceNo,

		Temperature: temperature,

		Humidity: humidity,

		SoilMoisture: mapping.SoilReading,

		RecordedAt: recordedAt,

		ReceivedAt: receivedAt,

		Source: "esp32-direct",
	}

	var storedSequence sql.NullInt64

	const insertQuery = `
		INSERT INTO sensor_logs
		(
			plant_id,
			device_code,
			sequence_no,
			temperature,
			humidity,
			soil_moisture,
			recorded_at,
			received_at,
			source
		)
		VALUES
		(
			$1,
			$2,
			$3,
			$4,
			$5,
			$6,
			$7,
			$8,
			$9
		)
		ON CONFLICT
		(
			device_code,
			sequence_no,
			plant_id
		)
		WHERE device_code IS NOT NULL
		  AND sequence_no IS NOT NULL
		DO NOTHING
		RETURNING
			id,
			sequence_no,
			recorded_at,
			received_at
	`

	err := transaction.QueryRowContext(
		ctx,
		insertQuery,
		sensor.PlantID,
		sensor.DeviceCode,
		sensor.SequenceNo,
		sensor.Temperature,
		sensor.Humidity,
		sensor.SoilMoisture,
		sensor.RecordedAt,
		sensor.ReceivedAt,
		sensor.Source,
	).Scan(
		&sensor.ID,
		&storedSequence,
		&sensor.RecordedAt,
		&sensor.ReceivedAt,
	)

	// Data baru berhasil disimpan.
	if err == nil {
		sensor.SequenceNo = nullableInt64Pointer(
			storedSequence,
		)

		return sensor, false, nil
	}

	// Error selain sql.ErrNoRows berarti insert memang gagal.
	if !errors.Is(err, sql.ErrNoRows) {
		return models.SensorLog{}, false, fmt.Errorf(
			"gagal menyimpan data untuk tanaman %d dari perangkat %s: %w",
			sensor.PlantID,
			deviceCode,
			err,
		)
	}

	// Jika sequence nil, seharusnya tidak mungkin terjadi
	// conflict karena NULL tidak berbenturan pada unique index.
	if sequenceNo == nil {
		return models.SensorLog{}, false, fmt.Errorf(
			"insert tanaman %d tidak mengembalikan data tanpa sequence_no",
			sensor.PlantID,
		)
	}

	// ON CONFLICT DO NOTHING menghasilkan sql.ErrNoRows.
	// Ambil kembali data yang sebelumnya sudah tersimpan.
	const existingQuery = `
		SELECT
			id,
			plant_id,
			device_code,
			sequence_no,
			temperature,
			humidity,
			soil_moisture,
			recorded_at,
			received_at,
			source
		FROM sensor_logs
		WHERE device_code = $1
		  AND sequence_no = $2
		  AND plant_id = $3
		ORDER BY id DESC
		LIMIT 1
	`

	var existing models.SensorLog

	var existingSequence sql.NullInt64

	err = transaction.QueryRowContext(
		ctx,
		existingQuery,
		deviceCode,
		*sequenceNo,
		sensor.PlantID,
	).Scan(
		&existing.ID,
		&existing.PlantID,
		&existing.DeviceCode,
		&existingSequence,
		&existing.Temperature,
		&existing.Humidity,
		&existing.SoilMoisture,
		&existing.RecordedAt,
		&existing.ReceivedAt,
		&existing.Source,
	)

	if err != nil {
		return models.SensorLog{}, false, fmt.Errorf(
			"data duplikat tanaman %d tidak dapat dibaca kembali: %w",
			sensor.PlantID,
			err,
		)
	}

	existing.SequenceNo = nullableInt64Pointer(
		existingSequence,
	)

	// Sequence sama tetapi payload berbeda tidak boleh
	// dianggap sebagai retry normal.
	if !sameSensorPayload(
		existing,
		temperature,
		humidity,
		mapping.SoilReading,
	) {
		return models.SensorLog{}, false, fmt.Errorf(
			"%w: perangkat %s, sequence_no %d, plant_id %d",
			ErrSequencePayloadConflict,
			deviceCode,
			*sequenceNo,
			sensor.PlantID,
		)
	}

	// Sequence dan payload sama berarti request retry.
	return existing, true, nil
}

func sameSensorPayload(
	existing models.SensorLog,
	temperature float64,
	humidity float64,
	soilMoisture float64,
) bool {
	const tolerance = 0.000001

	abs := func(value float64) float64 {
		if value < 0 {
			return -value
		}

		return value
	}

	return abs(
		existing.Temperature-temperature,
	) <= tolerance &&
		abs(
			existing.Humidity-humidity,
		) <= tolerance &&
		abs(
			existing.SoilMoisture-soilMoisture,
		) <= tolerance
}

// upsertDeviceStatus membuat atau memperbarui
// status terakhir ESP32.
func upsertDeviceStatus(
	ctx context.Context,
	transaction *sql.Tx,
	deviceCode string,
	firmwareVersion string,
	clientIP string,
	sequenceNo *int64,
	payloadAt time.Time,
	receivedAt time.Time,
) error {
	const query = `
		INSERT INTO device_status
		(
			device_code,
			firmware_version,
			last_seen_at,
			last_payload_at,
			last_ip,
			last_sequence_no,
			total_requests,
			created_at,
			updated_at
		)
		VALUES
		(
			$1,
			NULLIF($2, ''),
			$3,
			$4,
			NULLIF($5, ''),
			$6,
			1,
			$3,
			$3
		)
		ON CONFLICT (device_code)
		DO UPDATE SET
			firmware_version = COALESCE(
				NULLIF(EXCLUDED.firmware_version, ''),
				device_status.firmware_version
			),

			last_seen_at = EXCLUDED.last_seen_at,

			last_payload_at = EXCLUDED.last_payload_at,

			last_ip = COALESCE(
				NULLIF(EXCLUDED.last_ip, ''),
				device_status.last_ip
			),

			last_sequence_no = COALESCE(
				EXCLUDED.last_sequence_no,
				device_status.last_sequence_no
			),

			total_requests =
				device_status.total_requests + 1,

			updated_at = EXCLUDED.updated_at
	`

	_, err := transaction.ExecContext(
		ctx,
		query,
		deviceCode,
		firmwareVersion,
		receivedAt,
		payloadAt,
		clientIP,
		sequenceNo,
	)

	if err != nil {
		return fmt.Errorf(
			"gagal memperbarui status perangkat %s: %w",
			deviceCode,
			err,
		)
	}

	return nil
}

func nullableInt64Pointer(
	value sql.NullInt64,
) *int64 {
	if !value.Valid {
		return nil
	}

	result := value.Int64

	return &result
}

func requiredPositiveIntEnv(
	name string,
) (int, error) {
	rawValue := strings.TrimSpace(
		os.Getenv(name),
	)

	if rawValue == "" {
		return 0, fmt.Errorf(
			"environment %s belum diisi",
			name,
		)
	}

	value, err := strconv.Atoi(
		rawValue,
	)

	if err != nil || value <= 0 {
		return 0, fmt.Errorf(
			"environment %s harus berupa angka positif",
			name,
		)
	}

	return value, nil
}
