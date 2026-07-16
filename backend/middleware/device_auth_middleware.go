package middleware

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	DeviceCodeContextKey = "device_code"

	expectedSHA256HexLength = 64
	maximumDeviceKeyBytes   = 256
)

// DeviceAuthRequired memverifikasi perangkat melalui:
//
// X-Device-Code
// X-Device-Key
//
// Backend hanya menyimpan SHA-256 API key melalui
// DEVICE_API_KEY_SHA256, bukan API key plaintext.
func DeviceAuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		expectedCode := strings.TrimSpace(
			os.Getenv("DEVICE_CODE"),
		)

		expectedKeyHash := strings.ToLower(
			strings.TrimSpace(
				os.Getenv("DEVICE_API_KEY_SHA256"),
			),
		)

		if expectedCode == "" ||
			!isValidSHA256Hex(expectedKeyHash) {
			c.AbortWithStatusJSON(
				http.StatusInternalServerError,
				gin.H{
					"message":    "konfigurasi autentikasi perangkat belum lengkap",
					"request_id": getRequestID(c),
				},
			)

			return
		}

		receivedCode := strings.TrimSpace(
			c.GetHeader("X-Device-Code"),
		)

		receivedKey := strings.TrimSpace(
			c.GetHeader("X-Device-Key"),
		)

		if receivedCode == "" ||
			receivedKey == "" {
			c.AbortWithStatusJSON(
				http.StatusUnauthorized,
				gin.H{
					"message":    "X-Device-Code dan X-Device-Key wajib dikirim",
					"request_id": getRequestID(c),
				},
			)

			return
		}

		if len([]byte(receivedKey)) > maximumDeviceKeyBytes {
			c.AbortWithStatusJSON(
				http.StatusUnauthorized,
				gin.H{
					"message":    "identitas atau API key perangkat tidak valid",
					"request_id": getRequestID(c),
				},
			)

			return
		}

		receivedKeyDigest := sha256.Sum256(
			[]byte(receivedKey),
		)

		receivedKeyHash := hex.EncodeToString(
			receivedKeyDigest[:],
		)

		codeValid := subtle.ConstantTimeCompare(
			[]byte(receivedCode),
			[]byte(expectedCode),
		) == 1

		keyValid := subtle.ConstantTimeCompare(
			[]byte(receivedKeyHash),
			[]byte(expectedKeyHash),
		) == 1

		if !codeValid || !keyValid {
			c.AbortWithStatusJSON(
				http.StatusUnauthorized,
				gin.H{
					"message":    "identitas atau API key perangkat tidak valid",
					"request_id": getRequestID(c),
				},
			)

			return
		}

		c.Set(
			DeviceCodeContextKey,
			receivedCode,
		)

		c.Next()
	}
}

func isValidSHA256Hex(value string) bool {
	if len(value) != expectedSHA256HexLength {
		return false
	}

	decoded, err := hex.DecodeString(value)

	return err == nil &&
		len(decoded) == sha256.Size
}
