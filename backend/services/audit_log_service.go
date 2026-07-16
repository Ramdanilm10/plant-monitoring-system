package services

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"sync"
	"time"

	"plant-monitoring-backend/config"
)

const (
	auditLogQueueSize = 256

	auditLogInsertTimeout = 5 * time.Second
)

// AuditLogEntry adalah data audit internal.
//
// Struct ini sengaja tidak memiliki password,
// Authorization header, JWT, API key, ataupun body request.
type AuditLogEntry struct {
	OccurredAt time.Time

	RequestID string

	ActorType string

	ActorID int64

	ActorUsername string

	ActorRole string

	DeviceCode string

	Action string

	HTTPMethod string

	RequestPath string

	StatusCode int

	Result string

	ClientIP string

	UserAgent string

	LatencyMilliseconds int64

	Details map[string]any
}

var (
	auditLogWorkerOnce sync.Once

	auditLogQueue chan AuditLogEntry
)

// StartAuditLogWorker menjalankan worker tunggal untuk
// menyimpan audit log secara asynchronous.
//
// Dengan cara ini, proses penyimpanan audit tidak
// memperlambat respons utama pengguna atau perangkat.
func StartAuditLogWorker() {
	auditLogWorkerOnce.Do(
		func() {
			auditLogQueue = make(
				chan AuditLogEntry,
				auditLogQueueSize,
			)

			go runAuditLogWorker()

			log.Printf(
				"Audit log worker aktif dengan kapasitas antrean %d",
				auditLogQueueSize,
			)
		},
	)
}

// EnqueueAuditLog menambahkan audit log ke antrean.
//
// Fungsi mengembalikan false apabila antrean penuh.
// Request utama tetap berjalan walaupun audit gagal.
func EnqueueAuditLog(
	entry AuditLogEntry,
) bool {
	StartAuditLogWorker()

	select {
	case auditLogQueue <- entry:
		return true

	default:
		log.Printf(
			"Audit log dilewati karena antrean penuh. request_id=%s action=%s",
			entry.RequestID,
			entry.Action,
		)

		return false
	}
}

func runAuditLogWorker() {
	for entry := range auditLogQueue {
		if err := persistAuditLog(entry); err != nil {
			log.Printf(
				"Gagal menyimpan audit log. request_id=%s action=%s error=%v",
				entry.RequestID,
				entry.Action,
				err,
			)
		}
	}
}

func persistAuditLog(
	entry AuditLogEntry,
) error {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		auditLogInsertTimeout,
	)

	defer cancel()

	if entry.OccurredAt.IsZero() {
		entry.OccurredAt = time.Now().UTC()
	}

	detailsJSON := []byte("{}")

	if entry.Details != nil {
		encodedDetails, err := json.Marshal(
			entry.Details,
		)

		if err == nil {
			detailsJSON = encodedDetails
		}
	}

	var actorID any

	if entry.ActorID > 0 {
		actorID = entry.ActorID
	}

	const query = `
		INSERT INTO audit_logs
		(
			occurred_at,
			request_id,
			actor_type,
			actor_id,
			actor_username,
			actor_role,
			device_code,
			action,
			http_method,
			request_path,
			status_code,
			result,
			client_ip,
			user_agent,
			latency_ms,
			details
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
			$9,
			$10,
			$11,
			$12,
			$13,
			$14,
			$15,
			$16
		)
	`

	_, err := config.DB.ExecContext(
		ctx,
		query,
		entry.OccurredAt,
		nullableAuditString(entry.RequestID),
		normalizeAuditValue(
			entry.ActorType,
			"anonymous",
		),
		actorID,
		nullableAuditString(entry.ActorUsername),
		nullableAuditString(entry.ActorRole),
		nullableAuditString(entry.DeviceCode),
		normalizeAuditValue(
			entry.Action,
			"http.request",
		),
		normalizeAuditValue(
			entry.HTTPMethod,
			"UNKNOWN",
		),
		normalizeAuditValue(
			entry.RequestPath,
			"/",
		),
		entry.StatusCode,
		normalizeAuditValue(
			entry.Result,
			"unknown",
		),
		nullableAuditString(entry.ClientIP),
		nullableAuditString(entry.UserAgent),
		entry.LatencyMilliseconds,
		detailsJSON,
	)

	return err
}

func nullableAuditString(
	value string,
) any {
	value = strings.TrimSpace(value)

	if value == "" {
		return nil
	}

	return value
}

func normalizeAuditValue(
	value string,
	fallback string,
) string {
	value = strings.TrimSpace(value)

	if value == "" {
		return fallback
	}

	return value
}
