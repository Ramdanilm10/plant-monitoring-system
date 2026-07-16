package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"

	"plant-monitoring-backend/config"
)

const (
	defaultBcryptCost = 12
	minimumBcryptCost = 10
	maximumBcryptCost = 14

	minimumPasswordBytes = 12
	maximumPasswordBytes = 72
)

type seedUser struct {
	Username string
	Password string
	Role     string
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println(
			"File .env tidak ditemukan, memakai environment sistem",
		)
	}

	users, err := loadSeedUsers()

	if err != nil {
		log.Fatal(err)
	}

	bcryptCost, err := loadBcryptCost()

	if err != nil {
		log.Fatal(err)
	}

	config.ConnectDatabase()
	defer config.DB.Close()

	ctx, cancel := context.WithTimeout(
		context.Background(),
		30*time.Second,
	)
	defer cancel()

	transaction, err := config.DB.BeginTx(
		ctx,
		nil,
	)

	if err != nil {
		log.Fatal(
			"gagal memulai transaksi reset akun: ",
			err,
		)
	}

	committed := false

	defer func() {
		if !committed {
			_ = transaction.Rollback()
		}
	}()

	const query = `
		INSERT INTO users
		(
			username,
			password_hash,
			role
		)
		VALUES
		(
			$1,
			$2,
			$3
		)
		ON CONFLICT (username)
		DO UPDATE SET
			password_hash = EXCLUDED.password_hash,
			role = EXCLUDED.role
	`

	for _, user := range users {
		passwordHash, err := bcrypt.GenerateFromPassword(
			[]byte(user.Password),
			bcryptCost,
		)

		if err != nil {
			log.Fatalf(
				"gagal membuat hash untuk %s: %v",
				user.Username,
				err,
			)
		}

		if _, err := transaction.ExecContext(
			ctx,
			query,
			user.Username,
			string(passwordHash),
			user.Role,
		); err != nil {
			log.Fatalf(
				"gagal menyimpan akun %s: %v",
				user.Username,
				err,
			)
		}

		log.Printf(
			"akun %s dengan role %s berhasil diperbarui",
			user.Username,
			user.Role,
		)
	}

	if err := transaction.Commit(); err != nil {
		log.Fatal(
			"gagal menyelesaikan transaksi reset akun: ",
			err,
		)
	}

	committed = true

	log.Printf(
		"semua akun berhasil diperbarui menggunakan bcrypt cost %d",
		bcryptCost,
	)

	log.Println(
		"hapus ADMIN_PASSWORD dan VIEWER_PASSWORD dari environment setelah proses selesai",
	)
}

func loadSeedUsers() ([]seedUser, error) {
	adminUsername := readEnvironmentWithDefault(
		"ADMIN_USERNAME",
		"admin",
	)

	viewerUsername := readEnvironmentWithDefault(
		"VIEWER_USERNAME",
		"viewer",
	)

	adminPassword := os.Getenv(
		"ADMIN_PASSWORD",
	)

	viewerPassword := os.Getenv(
		"VIEWER_PASSWORD",
	)

	if strings.EqualFold(
		adminUsername,
		viewerUsername,
	) {
		return nil, fmt.Errorf(
			"ADMIN_USERNAME dan VIEWER_USERNAME tidak boleh sama",
		)
	}

	if err := validateUsername(
		"ADMIN_USERNAME",
		adminUsername,
	); err != nil {
		return nil, err
	}

	if err := validateUsername(
		"VIEWER_USERNAME",
		viewerUsername,
	); err != nil {
		return nil, err
	}

	if err := validatePassword(
		"ADMIN_PASSWORD",
		adminPassword,
	); err != nil {
		return nil, err
	}

	if err := validatePassword(
		"VIEWER_PASSWORD",
		viewerPassword,
	); err != nil {
		return nil, err
	}

	if adminPassword == viewerPassword {
		return nil, fmt.Errorf(
			"ADMIN_PASSWORD dan VIEWER_PASSWORD tidak boleh sama",
		)
	}

	return []seedUser{
		{
			Username: adminUsername,
			Password: adminPassword,
			Role:     "admin",
		},
		{
			Username: viewerUsername,
			Password: viewerPassword,
			Role:     "viewer",
		},
	}, nil
}

func validateUsername(
	name string,
	value string,
) error {
	value = strings.TrimSpace(value)

	if len(value) < 3 ||
		len([]byte(value)) > 64 {
		return fmt.Errorf(
			"%s harus memiliki panjang 3 sampai 64 byte",
			name,
		)
	}

	for _, character := range value {
		if unicode.IsLetter(character) ||
			unicode.IsDigit(character) ||
			character == '_' ||
			character == '-' ||
			character == '.' {
			continue
		}

		return fmt.Errorf(
			"%s hanya boleh berisi huruf, angka, titik, underscore, atau tanda hubung",
			name,
		)
	}

	return nil
}

func validatePassword(
	name string,
	password string,
) error {
	passwordLength := len(
		[]byte(password),
	)

	if passwordLength < minimumPasswordBytes ||
		passwordLength > maximumPasswordBytes {
		return fmt.Errorf(
			"%s harus memiliki panjang %d sampai %d byte",
			name,
			minimumPasswordBytes,
			maximumPasswordBytes,
		)
	}

	var hasUpper bool
	var hasLower bool
	var hasDigit bool
	var hasSymbol bool

	for _, character := range password {
		switch {
		case unicode.IsUpper(character):
			hasUpper = true

		case unicode.IsLower(character):
			hasLower = true

		case unicode.IsDigit(character):
			hasDigit = true

		case unicode.IsSpace(character):
			return fmt.Errorf(
				"%s tidak boleh mengandung spasi",
				name,
			)

		default:
			hasSymbol = true
		}
	}

	if !hasUpper ||
		!hasLower ||
		!hasDigit ||
		!hasSymbol {
		return fmt.Errorf(
			"%s harus mengandung huruf besar, huruf kecil, angka, dan simbol",
			name,
		)
	}

	return nil
}

func loadBcryptCost() (int, error) {
	rawValue := strings.TrimSpace(
		os.Getenv("BCRYPT_COST"),
	)

	if rawValue == "" {
		return defaultBcryptCost, nil
	}

	cost, err := strconv.Atoi(rawValue)

	if err != nil ||
		cost < minimumBcryptCost ||
		cost > maximumBcryptCost {
		return 0, fmt.Errorf(
			"BCRYPT_COST harus berada pada rentang %d sampai %d",
			minimumBcryptCost,
			maximumBcryptCost,
		)
	}

	return cost, nil
}

func readEnvironmentWithDefault(
	name string,
	defaultValue string,
) string {
	value := strings.TrimSpace(
		os.Getenv(name),
	)

	if value == "" {
		return defaultValue
	}

	return value
}
