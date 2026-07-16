package services

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

var blynkHTTPClient = &http.Client{
	Timeout: 10 * time.Second,
}

var blynkVirtualPinPattern = regexp.MustCompile(
	`^V[0-9]+$`,
)

// GetBlynkValue mengambil nilai terakhir
// dari satu virtual pin Blynk.
//
// Contoh:
// V0
// V1
// V2
// V3
func GetBlynkValue(
	pin string,
) (string, error) {
	server, token, err := getBlynkConfiguration()

	if err != nil {
		return "", err
	}

	pin = strings.TrimSpace(pin)

	if !blynkVirtualPinPattern.MatchString(pin) {
		return "", fmt.Errorf(
			"virtual pin Blynk tidak valid: %s",
			pin,
		)
	}

	endpoint := fmt.Sprintf(
		"%s/external/api/get?token=%s&%s",
		server,
		url.QueryEscape(token),
		pin,
	)

	responseBody, err := executeBlynkGET(endpoint)

	if err != nil {
		return "", err
	}

	return strings.TrimSpace(
		string(responseBody),
	), nil
}

// GetBlynkConnectionStatus mengambil status
// koneksi ESP32 langsung dari Blynk Cloud.
//
// true  = perangkat online
// false = perangkat offline
func GetBlynkConnectionStatus() (
	bool,
	error,
) {
	server, token, err := getBlynkConfiguration()

	if err != nil {
		return false, err
	}

	endpoint := fmt.Sprintf(
		"%s/external/api/isHardwareConnected?token=%s",
		server,
		url.QueryEscape(token),
	)

	responseBody, err := executeBlynkGET(endpoint)

	if err != nil {
		return false, err
	}

	statusValue := strings.ToLower(
		strings.TrimSpace(
			string(responseBody),
		),
	)

	switch statusValue {
	case "true":
		return true, nil

	case "false":
		return false, nil

	default:
		return false, fmt.Errorf(
			"respons status koneksi Blynk tidak valid: %q",
			statusValue,
		)
	}
}

func getBlynkConfiguration() (
	string,
	string,
	error,
) {
	token := strings.TrimSpace(
		os.Getenv("BLYNK_TOKEN"),
	)

	server := strings.TrimRight(
		strings.TrimSpace(
			os.Getenv("BLYNK_SERVER"),
		),
		"/",
	)

	if token == "" {
		return "", "", fmt.Errorf(
			"BLYNK_TOKEN belum diisi",
		)
	}

	if server == "" {
		return "", "", fmt.Errorf(
			"BLYNK_SERVER belum diisi",
		)
	}

	return server, token, nil
}

func executeBlynkGET(
	endpoint string,
) ([]byte, error) {
	request, err := http.NewRequest(
		http.MethodGet,
		endpoint,
		nil,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"gagal membuat request Blynk: %w",
			err,
		)
	}

	response, err := blynkHTTPClient.Do(request)

	if err != nil {
		return nil, fmt.Errorf(
			"gagal menghubungi Blynk: %w",
			err,
		)
	}

	defer response.Body.Close()

	body, err := io.ReadAll(
		io.LimitReader(
			response.Body,
			2048,
		),
	)

	if err != nil {
		return nil, fmt.Errorf(
			"gagal membaca respons Blynk: %w",
			err,
		)
	}

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"Blynk mengembalikan status %d: %s",
			response.StatusCode,
			strings.TrimSpace(
				string(body),
			),
		)
	}

	return body, nil
}
