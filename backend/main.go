package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/robfig/cron/v3"

	"plant-monitoring-backend/config"
	"plant-monitoring-backend/middleware"
	"plant-monitoring-backend/routes"
	"plant-monitoring-backend/services"
)

const (
	defaultPort = "8080"

	defaultSensorCron = "0 */30 * * * *"

	maximumRequestBodyBytes int64 = 1 * 1024 * 1024
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println(
			"File .env tidak ditemukan, memakai environment sistem",
		)
	}

	applicationEnvironment :=
		getApplicationEnvironment()

	if applicationEnvironment == "production" {
		gin.SetMode(
			gin.ReleaseMode,
		)
	}

	config.ConnectDatabase()

	// Worker dijalankan setelah koneksi database tersedia.
	services.StartAuditLogWorker()

	if isBlynkCollectorEnabled() {
		startSensorScheduler()
	} else {
		log.Println(
			"Collector Blynk dinonaktifkan melalui environment",
		)
	}

	router := gin.New()

	// AuditTrail diletakkan sebelum Recovery.
	//
	// Dengan urutan ini, panic yang ditangani Recovery
	// tetap menghasilkan status 500 yang dapat diaudit.
	router.Use(
		gin.Logger(),

		middleware.RequestID(),

		middleware.AuditTrail(),

		gin.Recovery(),

		middleware.SecurityHeaders(
			applicationEnvironment ==
				"production",
		),

		middleware.LimitRequestBody(
			maximumRequestBodyBytes,
		),
	)

	configureTrustedProxies(
		router,
	)

	configureCORS(
		router,
		applicationEnvironment,
	)

	routes.SetupRoutes(
		router,
	)

	configureFrontendServing(
		router,
	)

	port := strings.TrimSpace(
		os.Getenv("PORT"),
	)

	if port == "" {
		port = defaultPort
	}

	address := "0.0.0.0:" + port

	log.Printf(
		"Environment aplikasi: %s",
		applicationEnvironment,
	)

	log.Printf(
		"Batas body request: %d byte",
		maximumRequestBodyBytes,
	)

	log.Printf(
		"Backend dan frontend berjalan di http://%s",
		address,
	)

	if err := router.Run(
		address,
	); err != nil {
		log.Fatal(
			"Gagal menjalankan server: ",
			err,
		)
	}
}

func getApplicationEnvironment() string {
	applicationEnvironment := strings.ToLower(
		strings.TrimSpace(
			os.Getenv("APP_ENV"),
		),
	)

	switch applicationEnvironment {
	case "production":
		return "production"

	case "development":
		return "development"

	default:
		return "development"
	}
}

func configureTrustedProxies(
	router *gin.Engine,
) {
	trustedProxies := []string{
		"127.0.0.1",
		"::1",
	}

	if err := router.SetTrustedProxies(
		trustedProxies,
	); err != nil {
		log.Printf(
			"Gagal mengatur trusted proxies: %v",
			err,
		)
	}
}

func configureCORS(
	router *gin.Engine,
	applicationEnvironment string,
) {
	if applicationEnvironment ==
		"production" {
		log.Println(
			"CORS middleware tidak digunakan pada production karena frontend dan API memakai origin yang sama",
		)

		return
	}

	router.Use(
		cors.New(
			cors.Config{
				AllowOrigins: getAllowedOrigins(),

				AllowMethods: []string{
					http.MethodGet,
					http.MethodPost,
					http.MethodPut,
					http.MethodPatch,
					http.MethodDelete,
					http.MethodOptions,
				},

				AllowHeaders: []string{
					"Origin",
					"Content-Type",
					"Authorization",
					"X-Device-Code",
					"X-Device-Key",
				},

				ExposeHeaders: []string{
					"Content-Length",
					"X-Request-ID",
					"X-RateLimit-Limit",
					"X-RateLimit-Remaining",
					"X-RateLimit-Reset",
				},

				AllowCredentials: false,
			},
		),
	)

	log.Println(
		"CORS development aktif",
	)
}

func startSensorScheduler() {
	schedule := strings.TrimSpace(
		os.Getenv("SENSOR_CRON"),
	)

	if schedule == "" {
		schedule = defaultSensorCron
	}

	scheduler := cron.New(
		cron.WithSeconds(),
	)

	_, err := scheduler.AddFunc(
		schedule,
		services.CollectSensorData,
	)

	if err != nil {
		log.Fatal(
			"Jadwal collector sensor tidak valid: ",
			err,
		)
	}

	scheduler.Start()

	log.Printf(
		"Collector sensor aktif dengan jadwal: %s",
		schedule,
	)

	go services.CollectSensorData()
}

func getAllowedOrigins() []string {
	rawOrigins := strings.TrimSpace(
		os.Getenv("FRONTEND_ORIGINS"),
	)

	if rawOrigins == "" {
		return []string{
			"http://localhost:5173",
			"http://127.0.0.1:5173",
		}
	}

	origins := make(
		[]string,
		0,
	)

	for _, origin := range strings.Split(
		rawOrigins,
		",",
	) {
		cleanOrigin := strings.TrimSpace(
			origin,
		)

		if cleanOrigin == "" {
			continue
		}

		origins = append(
			origins,
			strings.TrimRight(
				cleanOrigin,
				"/",
			),
		)
	}

	if len(origins) == 0 {
		return []string{
			"http://localhost:5173",
			"http://127.0.0.1:5173",
		}
	}

	return origins
}

func isBlynkCollectorEnabled() bool {
	rawValue := strings.ToLower(
		strings.TrimSpace(
			os.Getenv(
				"BLYNK_COLLECTOR_ENABLED",
			),
		),
	)

	if rawValue == "" {
		return true
	}

	switch rawValue {
	case "true",
		"1",
		"yes",
		"on":
		return true

	default:
		return false
	}
}

func configureFrontendServing(
	router *gin.Engine,
) {
	distPath := strings.TrimSpace(
		os.Getenv(
			"FRONTEND_DIST_PATH",
		),
	)

	if distPath == "" {
		distPath = "../frontend/dist"
	}

	absoluteDistPath, err := filepath.Abs(
		distPath,
	)

	if err != nil {
		log.Printf(
			"Path frontend tidak valid: %v",
			err,
		)

		return
	}

	indexPath := filepath.Join(
		absoluteDistPath,
		"index.html",
	)

	if _, err := os.Stat(
		indexPath,
	); err != nil {
		log.Printf(
			"Frontend production build tidak ditemukan di %s",
			absoluteDistPath,
		)

		log.Println(
			"Jalankan npm run build pada folder frontend",
		)

		return
	}

	router.NoRoute(
		func(c *gin.Context) {
			requestPath :=
				c.Request.URL.Path

			if requestPath == "/api" ||
				strings.HasPrefix(
					requestPath,
					"/api/",
				) {
				c.JSON(
					http.StatusNotFound,
					gin.H{
						"message": "endpoint API tidak ditemukan",

						"request_id": getContextRequestID(
							c,
						),
					},
				)

				return
			}

			if requestPath != "/" {
				relativePath := strings.TrimPrefix(
					requestPath,
					"/",
				)

				candidatePath := filepath.Join(
					absoluteDistPath,
					filepath.FromSlash(
						relativePath,
					),
				)

				if isPathInsideDirectory(
					absoluteDistPath,
					candidatePath,
				) {
					fileInfo, statErr := os.Stat(
						candidatePath,
					)

					if statErr == nil &&
						!fileInfo.IsDir() {
						c.File(
							candidatePath,
						)

						return
					}
				}
			}

			c.File(
				indexPath,
			)
		},
	)

	log.Printf(
		"Frontend production build dilayani dari %s",
		absoluteDistPath,
	)
}

func isPathInsideDirectory(
	baseDirectory string,
	targetPath string,
) bool {
	absoluteBase, err := filepath.Abs(
		baseDirectory,
	)

	if err != nil {
		return false
	}

	absoluteTarget, err := filepath.Abs(
		targetPath,
	)

	if err != nil {
		return false
	}

	relativePath, err := filepath.Rel(
		absoluteBase,
		absoluteTarget,
	)

	if err != nil {
		return false
	}

	if relativePath == "." {
		return true
	}

	if relativePath == ".." {
		return false
	}

	if filepath.IsAbs(
		relativePath,
	) {
		return false
	}

	parentPrefix := ".." + string(
		os.PathSeparator,
	)

	return !strings.HasPrefix(
		relativePath,
		parentPrefix,
	)
}

func getContextRequestID(
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
