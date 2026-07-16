package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"plant-monitoring-backend/services"
)

func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		authorizationHeader := strings.TrimSpace(
			c.GetHeader("Authorization"),
		)

		headerParts := strings.Fields(
			authorizationHeader,
		)

		if len(headerParts) != 2 ||
			!strings.EqualFold(
				headerParts[0],
				"Bearer",
			) ||
			strings.TrimSpace(headerParts[1]) == "" {
			c.Header(
				"WWW-Authenticate",
				`Bearer realm="plant-monitoring"`,
			)

			c.AbortWithStatusJSON(
				http.StatusUnauthorized,
				gin.H{
					"message":    "token login tidak ditemukan",
					"request_id": getRequestID(c),
				},
			)

			return
		}

		tokenString := headerParts[1]

		claims, err := services.ParseAuthToken(
			tokenString,
		)

		if err != nil {
			if errors.Is(
				err,
				services.ErrJWTConfiguration,
			) {
				c.AbortWithStatusJSON(
					http.StatusInternalServerError,
					gin.H{
						"message":    "konfigurasi autentikasi bermasalah",
						"request_id": getRequestID(c),
					},
				)

				return
			}

			c.Header(
				"WWW-Authenticate",
				`Bearer error="invalid_token"`,
			)

			c.AbortWithStatusJSON(
				http.StatusUnauthorized,
				gin.H{
					"message":    "token tidak valid atau kedaluwarsa",
					"request_id": getRequestID(c),
				},
			)

			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Set("token_id", claims.ID)

		c.Next()
	}
}

func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		roleValue, exists := c.Get("role")

		if !exists {
			c.AbortWithStatusJSON(
				http.StatusUnauthorized,
				gin.H{
					"message":    "data sesi pengguna tidak ditemukan",
					"request_id": getRequestID(c),
				},
			)

			return
		}

		role, valid := roleValue.(string)

		if !valid {
			c.AbortWithStatusJSON(
				http.StatusUnauthorized,
				gin.H{
					"message":    "data role pengguna tidak valid",
					"request_id": getRequestID(c),
				},
			)

			return
		}

		role = strings.ToLower(
			strings.TrimSpace(role),
		)

		if role != "admin" {
			c.AbortWithStatusJSON(
				http.StatusForbidden,
				gin.H{
					"message":    "akses hanya tersedia untuk Admin",
					"request_id": getRequestID(c),
				},
			)

			return
		}

		c.Next()
	}
}
