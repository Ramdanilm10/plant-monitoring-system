package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"plant-monitoring-backend/services"
)

// AuditTrail mencatat aktivitas keamanan dan perubahan
// penting setelah request selesai diproses.
//
// Middleware tidak membaca ataupun menyimpan body request.
func AuditTrail() gin.HandlerFunc {
	return func(c *gin.Context) {
		startedAt := time.Now()

		c.Next()

		statusCode := c.Writer.Status()
		requestPath := c.Request.URL.Path

		if !shouldCreateAuditLog(
			c.Request.Method,
			requestPath,
			statusCode,
		) {
			return
		}

		action := determineAuditAction(
			c.Request.Method,
			requestPath,
			statusCode,
		)

		result := determineAuditResult(
			statusCode,
		)

		actorType,
			actorID,
			actorUsername,
			actorRole,
			deviceCode :=
			resolveAuditActor(c)

		routePattern := strings.TrimSpace(
			c.FullPath(),
		)

		if routePattern == "" {
			routePattern = requestPath
		}

		entry := services.AuditLogEntry{
			OccurredAt: time.Now().UTC(),

			RequestID: getRequestID(c),

			ActorType: actorType,

			ActorID: actorID,

			ActorUsername: actorUsername,

			ActorRole: actorRole,

			DeviceCode: deviceCode,

			Action: action,

			HTTPMethod: c.Request.Method,

			RequestPath: truncateAuditText(
				requestPath,
				255,
			),

			StatusCode: statusCode,

			Result: result,

			ClientIP: truncateAuditText(
				c.ClientIP(),
				64,
			),

			UserAgent: truncateAuditText(
				c.Request.UserAgent(),
				500,
			),

			LatencyMilliseconds: time.Since(
				startedAt,
			).Milliseconds(),

			Details: map[string]any{
				"route": routePattern,

				"status_text": http.StatusText(
					statusCode,
				),

				"content_length": c.Request.ContentLength,
			},
		}

		services.EnqueueAuditLog(entry)
	}
}

func shouldCreateAuditLog(
	method string,
	path string,
	statusCode int,
) bool {
	if path == "/api/auth/login" {
		return true
	}

	if path == "/api/device/readings" {
		return true
	}

	if isAuditWriteMethod(method) &&
		strings.HasPrefix(
			path,
			"/api/admin/",
		) {
		return true
	}

	if strings.HasPrefix(
		path,
		"/api/",
	) {
		switch statusCode {
		case http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusTooManyRequests:
			return true
		}
	}

	return false
}

func determineAuditAction(
	method string,
	path string,
	statusCode int,
) string {
	if path == "/api/auth/login" {
		if statusCode >= 200 &&
			statusCode < 300 {
			return "auth.login.success"
		}

		if statusCode ==
			http.StatusTooManyRequests {
			return "auth.login.rate_limited"
		}

		return "auth.login.failed"
	}

	if path == "/api/device/readings" {
		switch statusCode {
		case http.StatusOK,
			http.StatusCreated:
			return "device.readings.accepted"

		case http.StatusConflict:
			return "device.readings.conflict"

		case http.StatusTooManyRequests:
			return "device.readings.rate_limited"

		default:
			return "device.readings.rejected"
		}
	}

	if path == "/api/admin/sensor" &&
		method == http.MethodPost {
		return "admin.sensor.create"
	}

	if path == "/api/admin/plants" &&
		method == http.MethodPost {
		return "admin.plant.create"
	}

	if strings.HasPrefix(
		path,
		"/api/admin/plants/",
	) {
		switch method {
		case http.MethodPut,
			http.MethodPatch:
			return "admin.plant.update"

		case http.MethodDelete:
			return "admin.plant.delete"
		}
	}

	if statusCode == http.StatusUnauthorized ||
		statusCode == http.StatusForbidden {
		return "security.access.denied"
	}

	if statusCode ==
		http.StatusTooManyRequests {
		return "security.rate_limited"
	}

	return "admin.write"
}

func determineAuditResult(
	statusCode int,
) string {
	switch {
	case statusCode >= 200 &&
		statusCode < 400:
		return "success"

	case statusCode ==
		http.StatusUnauthorized:
		return "denied"

	case statusCode ==
		http.StatusForbidden:
		return "forbidden"

	case statusCode ==
		http.StatusTooManyRequests:
		return "rate_limited"

	case statusCode ==
		http.StatusConflict:
		return "conflict"

	case statusCode >= 500:
		return "error"

	default:
		return "failed"
	}
}

func resolveAuditActor(
	c *gin.Context,
) (
	string,
	int64,
	string,
	string,
	string,
) {
	userID := getAuditContextInt64(
		c,
		"user_id",
	)

	username := getAuditContextString(
		c,
		"username",
	)

	role := getAuditContextString(
		c,
		"role",
	)

	if username != "" {
		return "user",
			userID,
			truncateAuditText(
				username,
				64,
			),
			truncateAuditText(
				role,
				32,
			),
			""
	}

	attemptedUsername :=
		getAuditContextString(
			c,
			"audit_username",
		)

	attemptedRole :=
		getAuditContextString(
			c,
			"audit_role",
		)

	if attemptedUsername != "" {
		return "user_attempt",
			0,
			truncateAuditText(
				attemptedUsername,
				64,
			),
			truncateAuditText(
				attemptedRole,
				32,
			),
			""
	}

	deviceCode := getAuditContextString(
		c,
		DeviceCodeContextKey,
	)

	if deviceCode != "" {
		return "device",
			0,
			"",
			"",
			truncateAuditText(
				deviceCode,
				64,
			)
	}

	attemptedDeviceCode := strings.TrimSpace(
		c.GetHeader(
			"X-Device-Code",
		),
	)

	if attemptedDeviceCode != "" {
		return "device_attempt",
			0,
			"",
			"",
			truncateAuditText(
				attemptedDeviceCode,
				64,
			)
	}

	return "anonymous",
		0,
		"",
		"",
		""
}

func isAuditWriteMethod(
	method string,
) bool {
	switch method {
	case http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete:
		return true

	default:
		return false
	}
}

func getAuditContextString(
	c *gin.Context,
	key string,
) string {
	value, exists := c.Get(key)

	if !exists {
		return ""
	}

	stringValue, valid := value.(string)

	if !valid {
		return ""
	}

	return strings.TrimSpace(
		stringValue,
	)
}

func getAuditContextInt64(
	c *gin.Context,
	key string,
) int64 {
	value, exists := c.Get(key)

	if !exists {
		return 0
	}

	switch typedValue := value.(type) {
	case int64:
		return typedValue

	case int:
		return int64(typedValue)

	case int32:
		return int64(typedValue)

	case uint:
		return int64(typedValue)

	case uint64:
		if typedValue >
			uint64(^uint64(0)>>1) {
			return 0
		}

		return int64(typedValue)

	default:
		return 0
	}
}

func truncateAuditText(
	value string,
	maximumRunes int,
) string {
	value = strings.TrimSpace(value)

	if maximumRunes <= 0 {
		return ""
	}

	runes := []rune(value)

	if len(runes) <= maximumRunes {
		return value
	}

	return string(
		runes[:maximumRunes],
	)
}
