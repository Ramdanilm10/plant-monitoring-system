package controllers

import (
	"crypto/subtle"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"

	"plant-monitoring-backend/services"
)

// CollectSensorDataCron menangani request dari scheduler eksternal.
//
// Endpoint ini tidak menggunakan autentikasi JWT pengguna.
// Autentikasi dilakukan menggunakan:
//
// Authorization: Bearer <CRON_SECRET>
func CollectSensorDataCron(c *gin.Context) {
	configuredSecret := strings.TrimSpace(
		os.Getenv("CRON_SECRET"),
	)

	if configuredSecret == "" {
		c.JSON(
			http.StatusServiceUnavailable,
			gin.H{
				"message": "endpoint cron belum dikonfigurasi",
			},
		)

		return
	}

	providedSecret := extractBearerToken(
		c.GetHeader("Authorization"),
	)

	if !constantTimeStringEqual(
		providedSecret,
		configuredSecret,
	) {
		c.Header(
			"WWW-Authenticate",
			"Bearer",
		)

		c.JSON(
			http.StatusUnauthorized,
			gin.H{
				"message": "akses cron tidak sah",
			},
		)

		return
	}

	result, err := services.CollectSensorDataNow()
	if err != nil {
		c.JSON(
			http.StatusInternalServerError,
			gin.H{
				"message": "pengambilan data sensor gagal",
				"error":   err.Error(),
			},
		)

		return
	}

	statusCode := http.StatusCreated

	message :=
		"data sensor berhasil ditarik dari Blynk dan disimpan"

	if !result.Inserted {
		statusCode = http.StatusOK

		message =
			"data pada slot 30 menit ini sudah tersimpan; duplikasi dilewati"
	}

	c.JSON(
		statusCode,
		gin.H{
			"message": message,
			"data":    result,
		},
	)
}

func extractBearerToken(
	authorizationHeader string,
) string {
	parts := strings.Fields(
		authorizationHeader,
	)

	if len(parts) != 2 {
		return ""
	}

	if !strings.EqualFold(
		parts[0],
		"Bearer",
	) {
		return ""
	}

	return strings.TrimSpace(
		parts[1],
	)
}

func constantTimeStringEqual(
	providedValue string,
	expectedValue string,
) bool {
	providedBytes := []byte(
		providedValue,
	)

	expectedBytes := []byte(
		expectedValue,
	)

	if len(providedBytes) !=
		len(expectedBytes) {
		return false
	}

	return subtle.ConstantTimeCompare(
		providedBytes,
		expectedBytes,
	) == 1
}
