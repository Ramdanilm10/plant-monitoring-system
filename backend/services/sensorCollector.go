package services

import (
	"context"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	"plant-monitoring-backend/config"
)

const (
	lidahMertuaPlantID = 1
	anthuriumPlantID   = 2
)

func CollectSensorData() {
	if err := collectAndSaveSensorData(); err != nil {
		log.Printf(
			"Collector Blynk gagal: %v",
			err,
		)

		return
	}

	log.Println(
		"Collector Blynk berhasil menyimpan data untuk tanaman 1 dan 2",
	)
}

func collectAndSaveSensorData() error {
	soil1Raw, err :=
		GetBlynkValue("V0")

	if err != nil {
		return fmt.Errorf(
			"gagal mengambil V0: %w",
			err,
		)
	}

	soil2Raw, err :=
		GetBlynkValue("V1")

	if err != nil {
		return fmt.Errorf(
			"gagal mengambil V1: %w",
			err,
		)
	}

	temperatureRaw, err :=
		GetBlynkValue("V2")

	if err != nil {
		return fmt.Errorf(
			"gagal mengambil V2: %w",
			err,
		)
	}

	humidityRaw, err :=
		GetBlynkValue("V3")

	if err != nil {
		return fmt.Errorf(
			"gagal mengambil V3: %w",
			err,
		)
	}

	soil1, err :=
		parseBlynkNumber(
			soil1Raw,
			"V0",
		)

	if err != nil {
		return err
	}

	soil2, err :=
		parseBlynkNumber(
			soil2Raw,
			"V1",
		)

	if err != nil {
		return err
	}

	temperature, err :=
		parseBlynkNumber(
			temperatureRaw,
			"V2",
		)

	if err != nil {
		return err
	}

	humidity, err :=
		parseBlynkNumber(
			humidityRaw,
			"V3",
		)

	if err != nil {
		return err
	}

	ctx, cancel :=
		context.WithTimeout(
			context.Background(),
			15*time.Second,
		)

	defer cancel()

	transaction, err :=
		config.DB.BeginTx(
			ctx,
			nil,
		)

	if err != nil {
		return fmt.Errorf(
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

	recordedAt := time.Now()

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

	_, err = transaction.ExecContext(
		ctx,
		insertQuery,
		lidahMertuaPlantID,
		temperature,
		humidity,
		soil1,
		recordedAt,
		"blynk",
	)

	if err != nil {
		return fmt.Errorf(
			"gagal menyimpan data Lidah Mertua: %w",
			err,
		)
	}

	_, err = transaction.ExecContext(
		ctx,
		insertQuery,
		anthuriumPlantID,
		temperature,
		humidity,
		soil2,
		recordedAt,
		"blynk",
	)

	if err != nil {
		return fmt.Errorf(
			"gagal menyimpan data Anthurium Jenmanii: %w",
			err,
		)
	}

	if err := transaction.Commit(); err != nil {
		return fmt.Errorf(
			"gagal menyelesaikan transaksi database: %w",
			err,
		)
	}

	committed = true

	return nil
}

func parseBlynkNumber(
	rawValue string,
	pin string,
) (float64, error) {
	cleanValue := strings.TrimSpace(
		rawValue,
	)

	cleanValue = strings.Trim(
		cleanValue,
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
