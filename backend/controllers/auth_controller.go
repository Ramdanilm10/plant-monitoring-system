package controllers

import (
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"plant-monitoring-backend/models"
	"plant-monitoring-backend/services"
)

const (
	maximumLoginUsernameBytes = 64
	maximumLoginPasswordBytes = 72
)

func Login(c *gin.Context) {
	var request models.LoginRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"message": "format login tidak valid",

				"request_id": getControllerRequestID(c),
			},
		)

		return
	}

	request.Username = strings.TrimSpace(
		request.Username,
	)

	request.Role = strings.ToLower(
		strings.TrimSpace(
			request.Role,
		),
	)

	// Informasi ini digunakan audit middleware setelah
	// request selesai. Password tidak pernah disimpan.
	c.Set(
		"audit_username",
		request.Username,
	)

	c.Set(
		"audit_role",
		request.Role,
	)

	if request.Username == "" ||
		len([]byte(request.Username)) >
			maximumLoginUsernameBytes ||
		request.Password == "" ||
		len([]byte(request.Password)) >
			maximumLoginPasswordBytes ||
		(request.Role != "admin" &&
			request.Role != "viewer") {
		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"message": "format login tidak valid",

				"request_id": getControllerRequestID(c),
			},
		)

		return
	}

	user, err := services.AuthenticateUser(
		c.Request.Context(),
		request.Username,
		request.Password,
		request.Role,
	)

	if err != nil {
		if errors.Is(
			err,
			services.ErrInvalidCredentials,
		) {
			c.JSON(
				http.StatusUnauthorized,
				gin.H{
					"message": "username, password, atau role tidak sesuai",

					"request_id": getControllerRequestID(c),
				},
			)

			return
		}

		log.Printf(
			"Login gagal diproses. request_id=%s error=%v",
			getControllerRequestID(c),
			err,
		)

		c.JSON(
			http.StatusInternalServerError,
			gin.H{
				"message": "gagal memproses login",

				"request_id": getControllerRequestID(c),
			},
		)

		return
	}

	// Identity disimpan ke context agar audit middleware
	// mengenali login yang berhasil.
	c.Set(
		"user_id",
		user.ID,
	)

	c.Set(
		"username",
		user.Username,
	)

	c.Set(
		"role",
		user.Role,
	)

	token, expiresAt, err :=
		services.GenerateAuthToken(user)

	if err != nil {
		log.Printf(
			"Pembuatan token gagal. request_id=%s error=%v",
			getControllerRequestID(c),
			err,
		)

		c.JSON(
			http.StatusInternalServerError,
			gin.H{
				"message": "gagal membuat token login",

				"request_id": getControllerRequestID(c),
			},
		)

		return
	}

	expiresInSeconds := int64(
		time.Until(expiresAt).Seconds(),
	)

	if expiresInSeconds < 0 {
		expiresInSeconds = 0
	}

	c.JSON(
		http.StatusOK,
		models.LoginResponse{
			Token: token,

			TokenType: "Bearer",

			ExpiresInSeconds: expiresInSeconds,

			User: user,
		},
	)
}

func getControllerRequestID(
	c *gin.Context,
) string {
	value, exists := c.Get(
		"request_id",
	)

	if !exists {
		return ""
	}

	requestID, valid :=
		value.(string)

	if !valid {
		return ""
	}

	return requestID
}
