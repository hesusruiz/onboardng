package vies

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/hesusruiz/utils/errl"
	_ "modernc.org/sqlite"
)

const (
	endpoint      = "https://ec.europa.eu/taxation_customs/vies/rest-api/check-vat-number"
	DefaultDBPath = "data/vies_cache.db"
)

type CheckVATRequest struct {
	CountryCode string `json:"countryCode"`
	VATNumber   string `json:"vatNumber"`

	// Optional fields for approximate matching
	RequesterMemberStateCode string `json:"requesterMemberStateCode,omitempty"`
	RequesterNumber          string `json:"requesterNumber,omitempty"`
	TraderName               string `json:"traderName,omitempty"`
	TraderStreet             string `json:"traderStreet,omitempty"`
	TraderPostalCode         string `json:"traderPostalCode,omitempty"`
	TraderCity               string `json:"traderCity,omitempty"`
	TraderCompanyType        string `json:"traderCompanyType,omitempty"`
}

type CheckVATResponse struct {
	CountryCode string    `json:"countryCode"`
	VATNumber   string    `json:"vatNumber"`
	RequestDate time.Time `json:"requestDate"`

	Valid             bool   `json:"valid"`
	RequestIdentifier string `json:"requestIdentifier"`

	Name    string `json:"name"`
	Address string `json:"address"`

	TraderName        string `json:"traderName"`
	TraderStreet      string `json:"traderStreet"`
	TraderPostalCode  string `json:"traderPostalCode"`
	TraderCity        string `json:"traderCity"`
	TraderCompanyType string `json:"traderCompanyType"`

	TraderNameMatch        string `json:"traderNameMatch"`
	TraderStreetMatch      string `json:"traderStreetMatch"`
	TraderPostalCodeMatch  string `json:"traderPostalCodeMatch"`
	TraderCityMatch        string `json:"traderCityMatch"`
	TraderCompanyTypeMatch string `json:"traderCompanyTypeMatch"`
}

// Service provides VIES VAT validation backed by a local SQLite cache.
type Service struct {
	db     *sql.DB
	client *http.Client
}

// NewService creates a new VIES validation service using the specified SQLite database path.
// If dbPath is empty, DefaultDBPath ("vies_cache.db" in the current directory) is used.
func NewService(dbPath string, client *http.Client) (*Service, error) {
	if dbPath == "" {
		dbPath = DefaultDBPath
	}
	if client == nil {
		client = &http.Client{
			Timeout: 15 * time.Second,
		}
	}

	dbConn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open VIES cache database: %w", err)
	}

	schema := `
	CREATE TABLE IF NOT EXISTS vat_validations (
		country_code TEXT NOT NULL,
		vat_number   TEXT NOT NULL,
		request_date DATETIME NOT NULL,
		valid        BOOLEAN NOT NULL,
		PRIMARY KEY (country_code, vat_number)
	);`
	if _, err := dbConn.Exec(schema); err != nil {
		dbConn.Close()
		return nil, fmt.Errorf("failed to initialize VIES cache table: %w", err)
	}

	return &Service{
		db:     dbConn,
		client: client,
	}, nil
}

// Close closes the underlying SQLite database connection.
func (s *Service) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// CheckVAT validates a VAT number for a given country code, returning a cached validation
// if one exists and is less than 1 month old, or querying the VIES API and caching the result.
func (s *Service) CheckVAT(ctx context.Context, countryCode, vatNumber string) (*CheckVATResponse, error) {
	countryCode = strings.ToUpper(strings.TrimSpace(countryCode))
	vatNumber = strings.ToUpper(strings.TrimSpace(vatNumber))

	// 1. Check cache
	var cached CheckVATResponse
	err := s.db.QueryRowContext(
		ctx,
		`SELECT country_code, vat_number, request_date, valid FROM vat_validations WHERE country_code = ? AND vat_number = ?`,
		countryCode, vatNumber,
	).Scan(&cached.CountryCode, &cached.VATNumber, &cached.RequestDate, &cached.Valid)

	if err == nil {
		// Entry is valid for 1 month from RequestDate
		if time.Now().Before(cached.RequestDate.AddDate(0, 1, 0)) {
			return &cached, nil
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, errl.Errorf("failed to query VIES cache for %s and %s: %w", countryCode, vatNumber, err)
	}

	// 2. Cache miss or expired: call external VIES API
	result, err := fetchVIES(ctx, s.client, countryCode, vatNumber)
	if err != nil {
		return nil, errl.Errorf("failed to fetch VIES validation for %s and %s: %w", countryCode, vatNumber, err)
	}

	if result.CountryCode == "" {
		result.CountryCode = countryCode
	}
	if result.VATNumber == "" {
		result.VATNumber = vatNumber
	}
	if result.RequestDate.IsZero() {
		result.RequestDate = time.Now().UTC()
	}

	// 3. Save to cache
	_, err = s.db.ExecContext(
		ctx,
		`INSERT INTO vat_validations (country_code, vat_number, request_date, valid)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(country_code, vat_number) DO UPDATE SET
		     request_date = excluded.request_date,
		     valid        = excluded.valid`,
		countryCode, vatNumber, result.RequestDate, result.Valid,
	)
	if err != nil {
		return nil, errl.Errorf("failed to store VIES validation for %s and %s in cache: %w", countryCode, vatNumber, err)
	}

	return result, nil
}

func fetchVIES(ctx context.Context, client *http.Client, countryCode, vatNumber string) (*CheckVATResponse, error) {
	if client == nil {
		client = &http.Client{
			Timeout: 60 * time.Second,
		}
	}

	reqBody := CheckVATRequest{
		CountryCode: countryCode,
		VATNumber:   vatNumber,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		endpoint,
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, errl.Errorf("failed to call VIES API for %s and %s: %w", countryCode, vatNumber, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errl.Errorf("unexpected HTTP status for %s and %s: %s", countryCode, vatNumber, resp.Status)
	}

	var result CheckVATResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, errl.Errorf("failed to decode VIES API response for %s and %s: %w", countryCode, vatNumber, err)
	}

	return &result, nil
}

var (
	defaultOnce    sync.Once
	defaultService *Service
	defaultInitErr error
)

// CheckVAT uses the default SQLite-backed VIES service ("vies_cache.db" in the current directory).
func CheckVAT(ctx context.Context, client *http.Client, countryCode, vatNumber string) (*CheckVATResponse, error) {
	defaultOnce.Do(func() {
		defaultService, defaultInitErr = NewService(DefaultDBPath, client)
	})
	if defaultInitErr != nil {
		return nil, errl.Errorf("failed to initialize VIES service: %w", defaultInitErr)
	}
	if client != nil {
		svc := &Service{
			db:     defaultService.db,
			client: client,
		}
		return svc.CheckVAT(ctx, countryCode, vatNumber)
	}
	return defaultService.CheckVAT(ctx, countryCode, vatNumber)
}
