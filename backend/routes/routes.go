package routes

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"plant-monitoring-backend/controllers"
	"plant-monitoring-backend/middleware"
)

const (
	// Maksimal 10 percobaan login dari satu IP
	// dalam waktu 5 menit.
	loginMaximumRequests = 10

	loginRateLimitWindow = 5 * time.Minute

	// Endpoint direct device maksimal 30 request
	// per menit untuk satu IP.
	//
	// Nilai ini masih jauh di atas interval pengiriman
	// normal ESP32 sehingga retry tetap dapat berjalan.
	deviceMaximumRequests = 30

	deviceRateLimitWindow = time.Minute
)

func SetupRoutes(
	router *gin.Engine,
) {
	api := router.Group("/api")

	// Health check publik.
	api.GET(
		"/health",
		func(c *gin.Context) {
			c.JSON(
				http.StatusOK,
				gin.H{
					"status": "ok",

					"service": "plant-monitoring-backend",

					"server_time": time.Now().Format(
						time.RFC3339,
					),
				},
			)
		},
	)

	// Endpoint autentikasi dashboard.
	//
	// Rate limiter dipasang sebelum controller login
	// untuk membatasi brute-force.
	auth := api.Group("/auth")

	auth.Use(
		middleware.RateLimitByIP(
			loginMaximumRequests,
			loginRateLimitWindow,
		),
	)

	{
		auth.POST(
			"/login",
			controllers.Login,
		)
	}

	// Endpoint khusus perangkat ESP32.
	//
	// Urutan middleware:
	//
	// 1. Rate limiter
	// 2. Device authentication
	// 3. Controller
	device := api.Group("/device")

	device.Use(
		middleware.RateLimitByIP(
			deviceMaximumRequests,
			deviceRateLimitWindow,
		),
	)

	device.Use(
		middleware.DeviceAuthRequired(),
	)

	{
		device.POST(
			"/readings",
			controllers.CreateDeviceSensorReadings,
		)
	}

	// Endpoint Admin dan Viewer.
	protected := api.Group("")

	protected.Use(
		middleware.AuthRequired(),
	)

	{
		protected.GET(
			"/plants",
			controllers.GetPlants,
		)

		protected.GET(
			"/dashboard/:plant_id",
			controllers.GetDashboard,
		)

		protected.GET(
			"/plants/:plant_id/history",
			controllers.GetSensorHistory,
		)

		protected.GET(
			"/plants/:plant_id/dss",
			controllers.GetDSSAnalysis,
		)

		protected.GET(
			"/sensor/live",
			controllers.GetLiveSensor,
		)
	}

	// Endpoint khusus Admin.
	admin := api.Group("/admin")

	admin.Use(
		middleware.AuthRequired(),
		middleware.AdminOnly(),
	)

	{
		admin.GET(
			"/check",
			func(c *gin.Context) {
				c.JSON(
					http.StatusOK,
					gin.H{
						"message": "akses Admin berhasil",
					},
				)
			},
		)

		admin.GET(
			"/test-blynk",
			controllers.TestBlynk,
		)

		admin.GET(
			"/devices/status",
			controllers.GetDeviceStatuses,
		)

		admin.POST(
			"/sensor",
			controllers.CreateSensorData,
		)

		admin.GET(
			"/plants",
			controllers.GetPlants,
		)

		admin.GET(
			"/plants/:id",
			controllers.GetPlant,
		)

		admin.POST(
			"/plants",
			controllers.CreatePlant,
		)

		admin.PUT(
			"/plants/:id",
			controllers.UpdatePlant,
		)

		admin.DELETE(
			"/plants/:id",
			controllers.DeletePlant,
		)
	}
}
