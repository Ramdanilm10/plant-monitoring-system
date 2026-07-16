package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	defaultRequestIDBytes = 16
)

// rateLimitState menyimpan jumlah request dari satu IP
// dalam satu rentang waktu.
type rateLimitState struct {
	WindowStartedAt time.Time

	LastSeenAt time.Time

	RequestCount int
}

// RequestID membuat ID unik untuk setiap HTTP request.
//
// ID dikirim kembali melalui:
//
// X-Request-ID
//
// ID ini dapat digunakan untuk mencocokkan laporan error
// pengguna dengan log backend.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := generateRequestID()

		c.Set(
			"request_id",
			requestID,
		)

		c.Header(
			"X-Request-ID",
			requestID,
		)

		c.Next()
	}
}

// SecurityHeaders menambahkan header keamanan HTTP.
//
// Pada production, HSTS hanya dikirim apabila request
// benar-benar menggunakan HTTPS atau diteruskan oleh
// Cloudflare melalui HTTPS.
func SecurityHeaders(
	isProduction bool,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header(
			"X-Content-Type-Options",
			"nosniff",
		)

		c.Header(
			"X-Frame-Options",
			"DENY",
		)

		c.Header(
			"Referrer-Policy",
			"strict-origin-when-cross-origin",
		)

		c.Header(
			"Permissions-Policy",
			"camera=(), microphone=(), geolocation=(), payment=(), usb=()",
		)

		c.Header(
			"Cross-Origin-Opener-Policy",
			"same-origin",
		)

		// same-site tetap kompatibel dengan frontend Vite
		// pada localhost:5173 dan backend localhost:8080.
		c.Header(
			"Cross-Origin-Resource-Policy",
			"same-site",
		)

		c.Header(
			"Content-Security-Policy",
			strings.Join(
				[]string{
					"default-src 'self'",
					"script-src 'self'",
					"style-src 'self' 'unsafe-inline'",
					"img-src 'self' data: blob:",
					"font-src 'self' data:",
					"connect-src 'self'",
					"object-src 'none'",
					"base-uri 'self'",
					"frame-ancestors 'none'",
					"form-action 'self'",
				},
				"; ",
			),
		)

		// Respons API tidak boleh disimpan oleh browser
		// atau intermediary cache.
		if c.Request.URL.Path == "/api" ||
			strings.HasPrefix(
				c.Request.URL.Path,
				"/api/",
			) {
			c.Header(
				"Cache-Control",
				"no-store, no-cache, must-revalidate, private",
			)

			c.Header(
				"Pragma",
				"no-cache",
			)
		}

		if isProduction &&
			requestUsesHTTPS(c) {
			c.Header(
				"Strict-Transport-Security",
				"max-age=31536000; includeSubDomains",
			)
		}

		c.Next()
	}
}

// LimitRequestBody membatasi ukuran body request.
//
// Berlaku untuk:
//
// POST
// PUT
// PATCH
//
// Request dengan Content-Length melebihi batas langsung
// ditolak menggunakan HTTP 413.
func LimitRequestBody(
	maximumBytes int64,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		if maximumBytes <= 0 {
			c.Next()

			return
		}

		switch c.Request.Method {
		case http.MethodPost,
			http.MethodPut,
			http.MethodPatch:

			if c.Request.ContentLength > maximumBytes {
				c.AbortWithStatusJSON(
					http.StatusRequestEntityTooLarge,
					gin.H{
						"message": "ukuran request melebihi batas yang diizinkan",

						"maximum_bytes": maximumBytes,

						"request_id": getRequestID(c),
					},
				)

				return
			}

			if c.Request.Body != nil {
				c.Request.Body = http.MaxBytesReader(
					c.Writer,
					c.Request.Body,
					maximumBytes,
				)
			}
		}

		c.Next()
	}
}

// RateLimitByIP membatasi request berdasarkan IP.
//
// Contoh:
//
// RateLimitByIP(10, 5*time.Minute)
//
// berarti satu IP maksimal melakukan 10 request
// dalam rentang 5 menit.
func RateLimitByIP(
	maximumRequests int,
	window time.Duration,
) gin.HandlerFunc {
	if maximumRequests <= 0 {
		panic(
			"maximumRequests harus lebih besar dari 0",
		)
	}

	if window <= 0 {
		panic(
			"window rate limiter harus lebih besar dari 0",
		)
	}

	var mutex sync.Mutex

	clients := make(
		map[string]*rateLimitState,
	)

	lastCleanupAt := time.Now()

	return func(c *gin.Context) {
		now := time.Now()

		clientIP := strings.TrimSpace(
			c.ClientIP(),
		)

		if clientIP == "" {
			clientIP = "unknown"
		}

		mutex.Lock()

		// Membersihkan entry lama agar map tidak tumbuh
		// tanpa batas.
		if now.Sub(lastCleanupAt) >= window {
			cleanupBefore := now.Add(
				-2 * window,
			)

			for ipAddress, state := range clients {
				if state.LastSeenAt.Before(
					cleanupBefore,
				) {
					delete(
						clients,
						ipAddress,
					)
				}
			}

			lastCleanupAt = now
		}

		state, exists := clients[clientIP]

		if !exists ||
			now.Sub(
				state.WindowStartedAt,
			) >= window {
			state = &rateLimitState{
				WindowStartedAt: now,

				LastSeenAt: now,

				RequestCount: 0,
			}

			clients[clientIP] = state
		}

		state.RequestCount++

		state.LastSeenAt = now

		requestCount := state.RequestCount

		windowEndsAt := state.WindowStartedAt.Add(
			window,
		)

		remainingRequests :=
			maximumRequests - requestCount

		if remainingRequests < 0 {
			remainingRequests = 0
		}

		allowed :=
			requestCount <= maximumRequests

		mutex.Unlock()

		c.Header(
			"X-RateLimit-Limit",
			strconv.Itoa(
				maximumRequests,
			),
		)

		c.Header(
			"X-RateLimit-Remaining",
			strconv.Itoa(
				remainingRequests,
			),
		)

		c.Header(
			"X-RateLimit-Reset",
			strconv.FormatInt(
				windowEndsAt.Unix(),
				10,
			),
		)

		if !allowed {
			retryAfterDuration :=
				time.Until(windowEndsAt)

			retryAfterSeconds := int(
				retryAfterDuration.Seconds(),
			)

			if retryAfterSeconds < 1 {
				retryAfterSeconds = 1
			}

			c.Header(
				"Retry-After",
				strconv.Itoa(
					retryAfterSeconds,
				),
			)

			c.AbortWithStatusJSON(
				http.StatusTooManyRequests,
				gin.H{
					"message": "terlalu banyak request; coba kembali setelah beberapa saat",

					"retry_after_seconds": retryAfterSeconds,

					"request_id": getRequestID(c),
				},
			)

			return
		}

		c.Next()
	}
}

func generateRequestID() string {
	randomBytes := make(
		[]byte,
		defaultRequestIDBytes,
	)

	if _, err := rand.Read(
		randomBytes,
	); err == nil {
		return hex.EncodeToString(
			randomBytes,
		)
	}

	// Fallback ketika sumber random sistem gagal.
	return fmt.Sprintf(
		"fallback-%d",
		time.Now().UnixNano(),
	)
}

func getRequestID(
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

func requestUsesHTTPS(
	c *gin.Context,
) bool {
	if c.Request.TLS != nil {
		return true
	}

	forwardedProtocol := strings.ToLower(
		strings.TrimSpace(
			c.GetHeader(
				"X-Forwarded-Proto",
			),
		),
	)

	return forwardedProtocol == "https"
}
