package services

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"plant-monitoring-backend/config"
	"plant-monitoring-backend/models"
)

const (
	defaultJWTIssuer      = "plant-monitoring-backend"
	defaultJWTAudience    = "plant-monitoring-web"
	defaultJWTTTLMinutes  = 120
	minimumJWTTTLMinutes  = 15
	maximumJWTTTLMinutes  = 1440
	minimumJWTSecretBytes = 32

	defaultBcryptCost = 12
	minimumBcryptCost = 10
	maximumBcryptCost = 14

	maximumUsernameLengthBytes = 64
	maximumPasswordLengthBytes = 72
)

var (
	ErrInvalidCredentials = errors.New(
		"username, password, atau role tidak sesuai",
	)

	ErrJWTConfiguration = errors.New(
		"konfigurasi JWT tidak valid",
	)

	ErrInvalidToken = errors.New(
		"token tidak valid atau kedaluwarsa",
	)

	dummyPasswordHashOnce sync.Once
	dummyPasswordHash     []byte
)

type AuthClaims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`

	jwt.RegisteredClaims
}

type authConfiguration struct {
	Secret   []byte
	Issuer   string
	Audience string
	TTL      time.Duration
}

// AuthenticateUser memverifikasi username, password,
// dan role dengan respons error yang tetap generik.
//
// Password hash akan ditingkatkan otomatis ketika cost
// pada database lebih rendah dari BCRYPT_COST.
func AuthenticateUser(
	ctx context.Context,
	username string,
	password string,
	requestedRole string,
) (models.User, error) {
	username = strings.TrimSpace(username)
	requestedRole = normalizeRole(requestedRole)

	if username == "" ||
		len([]byte(username)) > maximumUsernameLengthBytes ||
		password == "" ||
		len([]byte(password)) > maximumPasswordLengthBytes ||
		!isAllowedRole(requestedRole) {
		return models.User{}, ErrInvalidCredentials
	}

	var user models.User

	const query = `
		SELECT
			id,
			username,
			password_hash,
			role
		FROM users
		WHERE LOWER(username) = LOWER($1)
		LIMIT 1
	`

	err := config.DB.QueryRowContext(
		ctx,
		query,
		username,
	).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.Role,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Tetap jalankan bcrypt agar perbedaan waktu antara
			// username ada dan tidak ada tidak terlalu mencolok.
			_ = bcrypt.CompareHashAndPassword(
				getDummyPasswordHash(),
				[]byte(password),
			)

			return models.User{}, ErrInvalidCredentials
		}

		return models.User{}, fmt.Errorf(
			"gagal mengambil pengguna: %w",
			err,
		)
	}

	user.Role = normalizeRole(user.Role)

	passwordErr := bcrypt.CompareHashAndPassword(
		[]byte(user.PasswordHash),
		[]byte(password),
	)

	if passwordErr != nil ||
		user.Role != requestedRole ||
		!isAllowedRole(user.Role) {
		return models.User{}, ErrInvalidCredentials
	}

	if err := upgradePasswordHashIfNeeded(
		ctx,
		user.ID,
		password,
		user.PasswordHash,
	); err != nil {
		return models.User{}, err
	}

	return user, nil
}

// GenerateAuthToken membuat JWT HS256 dengan issuer,
// audience, expiration, issued-at, not-before, subject,
// serta token ID unik.
func GenerateAuthToken(
	user models.User,
) (string, time.Time, error) {
	configuration, err := loadAuthConfiguration()

	if err != nil {
		return "", time.Time{}, err
	}

	role := normalizeRole(user.Role)

	if user.ID <= 0 ||
		strings.TrimSpace(user.Username) == "" ||
		!isAllowedRole(role) {
		return "", time.Time{}, ErrInvalidCredentials
	}

	now := time.Now().UTC()
	expiresAt := now.Add(configuration.TTL)

	tokenID, err := generateSecureTokenID()

	if err != nil {
		return "", time.Time{}, fmt.Errorf(
			"gagal membuat token ID: %w",
			err,
		)
	}

	claims := AuthClaims{
		UserID:   user.ID,
		Username: strings.TrimSpace(user.Username),
		Role:     role,

		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: configuration.Issuer,

			Subject: strconv.FormatInt(
				user.ID,
				10,
			),

			Audience: jwt.ClaimStrings{
				configuration.Audience,
			},

			ExpiresAt: jwt.NewNumericDate(expiresAt),

			NotBefore: jwt.NewNumericDate(
				now.Add(-30 * time.Second),
			),

			IssuedAt: jwt.NewNumericDate(now),

			ID: tokenID,
		},
	}

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		claims,
	)

	signedToken, err := token.SignedString(
		configuration.Secret,
	)

	if err != nil {
		return "", time.Time{}, fmt.Errorf(
			"gagal menandatangani token: %w",
			err,
		)
	}

	return signedToken, expiresAt, nil
}

// ParseAuthToken memverifikasi signature dan seluruh
// registered claims yang diwajibkan aplikasi.
func ParseAuthToken(
	tokenString string,
) (*AuthClaims, error) {
	configuration, err := loadAuthConfiguration()

	if err != nil {
		return nil, err
	}

	tokenString = strings.TrimSpace(tokenString)

	if tokenString == "" ||
		len(tokenString) > 8192 {
		return nil, ErrInvalidToken
	}

	claims := &AuthClaims{}

	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(token *jwt.Token) (any, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, ErrInvalidToken
			}

			return configuration.Secret, nil
		},
		jwt.WithValidMethods(
			[]string{
				jwt.SigningMethodHS256.Alg(),
			},
		),
		jwt.WithIssuer(configuration.Issuer),
		jwt.WithAudience(configuration.Audience),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithNotBeforeRequired(),
		jwt.WithLeeway(30*time.Second),
	)

	if err != nil ||
		token == nil ||
		!token.Valid {
		return nil, ErrInvalidToken
	}

	claims.Username = strings.TrimSpace(claims.Username)
	claims.Role = normalizeRole(claims.Role)

	expectedSubject := strconv.FormatInt(
		claims.UserID,
		10,
	)

	if claims.UserID <= 0 ||
		claims.Username == "" ||
		!isAllowedRole(claims.Role) ||
		claims.Subject != expectedSubject ||
		strings.TrimSpace(claims.ID) == "" {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

func loadAuthConfiguration() (
	authConfiguration,
	error,
) {
	secret := strings.TrimSpace(
		os.Getenv("JWT_SECRET"),
	)

	if len([]byte(secret)) < minimumJWTSecretBytes {
		return authConfiguration{}, fmt.Errorf(
			"%w: JWT_SECRET minimal %d byte",
			ErrJWTConfiguration,
			minimumJWTSecretBytes,
		)
	}

	issuer := strings.TrimSpace(
		os.Getenv("JWT_ISSUER"),
	)

	if issuer == "" {
		issuer = defaultJWTIssuer
	}

	audience := strings.TrimSpace(
		os.Getenv("JWT_AUDIENCE"),
	)

	if audience == "" {
		audience = defaultJWTAudience
	}

	ttlMinutes, err := readBoundedIntEnvironment(
		"JWT_TTL_MINUTES",
		defaultJWTTTLMinutes,
		minimumJWTTTLMinutes,
		maximumJWTTTLMinutes,
	)

	if err != nil {
		return authConfiguration{}, fmt.Errorf(
			"%w: %v",
			ErrJWTConfiguration,
			err,
		)
	}

	return authConfiguration{
		Secret:   []byte(secret),
		Issuer:   issuer,
		Audience: audience,
		TTL: time.Duration(ttlMinutes) *
			time.Minute,
	}, nil
}

func upgradePasswordHashIfNeeded(
	ctx context.Context,
	userID int64,
	plainPassword string,
	currentHash string,
) error {
	configuredCost, err := getConfiguredBcryptCost()

	if err != nil {
		return err
	}

	currentCost, err := bcrypt.Cost(
		[]byte(currentHash),
	)

	if err != nil {
		return fmt.Errorf(
			"hash password pengguna tidak valid: %w",
			err,
		)
	}

	if currentCost >= configuredCost {
		return nil
	}

	newHash, err := bcrypt.GenerateFromPassword(
		[]byte(plainPassword),
		configuredCost,
	)

	if err != nil {
		return fmt.Errorf(
			"gagal meningkatkan hash password: %w",
			err,
		)
	}

	const updateQuery = `
		UPDATE users
		SET password_hash = $1
		WHERE id = $2
	`

	if _, err := config.DB.ExecContext(
		ctx,
		updateQuery,
		string(newHash),
		userID,
	); err != nil {
		return fmt.Errorf(
			"gagal memperbarui hash password: %w",
			err,
		)
	}

	return nil
}

func getConfiguredBcryptCost() (int, error) {
	cost, err := readBoundedIntEnvironment(
		"BCRYPT_COST",
		defaultBcryptCost,
		minimumBcryptCost,
		maximumBcryptCost,
	)

	if err != nil {
		return 0, fmt.Errorf(
			"konfigurasi BCRYPT_COST tidak valid: %w",
			err,
		)
	}

	return cost, nil
}

func getDummyPasswordHash() []byte {
	dummyPasswordHashOnce.Do(
		func() {
			cost, err := getConfiguredBcryptCost()

			if err != nil {
				cost = defaultBcryptCost
			}

			hash, err := bcrypt.GenerateFromPassword(
				[]byte("invalid-password-placeholder"),
				cost,
			)

			if err != nil {
				dummyPasswordHash = []byte(
					"$2a$10$abcdefghijklmnopqrstuuuuuuuuuuuuuuuuuuuuuuuuuuuu",
				)

				return
			}

			dummyPasswordHash = hash
		},
	)

	return dummyPasswordHash
}

func generateSecureTokenID() (string, error) {
	randomBytes := make([]byte, 16)

	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}

	return hex.EncodeToString(randomBytes), nil
}

func readBoundedIntEnvironment(
	name string,
	defaultValue int,
	minimumValue int,
	maximumValue int,
) (int, error) {
	rawValue := strings.TrimSpace(
		os.Getenv(name),
	)

	if rawValue == "" {
		return defaultValue, nil
	}

	value, err := strconv.Atoi(rawValue)

	if err != nil {
		return 0, fmt.Errorf(
			"%s harus berupa angka",
			name,
		)
	}

	if value < minimumValue ||
		value > maximumValue {
		return 0, fmt.Errorf(
			"%s harus berada pada rentang %d sampai %d",
			name,
			minimumValue,
			maximumValue,
		)
	}

	return value, nil
}

func normalizeRole(role string) string {
	return strings.ToLower(
		strings.TrimSpace(role),
	)
}

func isAllowedRole(role string) bool {
	return role == "admin" ||
		role == "viewer"
}
