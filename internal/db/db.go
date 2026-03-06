package db

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/hesusruiz/onboardng/internal/configuration"
	"github.com/hesusruiz/utils/errl"
	_ "modernc.org/sqlite"
)

// RegistrationRecord represents a user registration record in the database
type RegistrationRecord struct {
	RegistrationID  string    `json:"registration_id"`
	Email           string    `json:"email"`
	FirstName       string    `json:"first_name"`
	LastName        string    `json:"last_name"`
	CompanyName     string    `json:"company_name"`
	Country         string    `json:"country"`
	VatID           string    `json:"vat_id"`
	StreetAddress   string    `json:"street_address"`
	PostalCode      string    `json:"postal_code"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	IssuanceAt      time.Time `json:"issuance_at,omitempty"`
	IssuanceError   string    `json:"issuance_error,omitempty"`
	NotifEmailAt    time.Time `json:"notif_email_at,omitempty"`
	NotifEmailError string    `json:"notif_email_error,omitempty"`
}

const dbDir = "data"
const dbPath = dbDir + "/onboarding.db"

// Service provides database operations for registrations
type Service struct {
	dbPath  string
	conn    *sql.DB
	runtime configuration.RuntimeEnv
}

func NewService(runtime configuration.RuntimeEnv, path string) (*Service, error) {

	// Create the directory if it does not exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, errl.Errorf("failed to create data directory: %w", err)
	}

	// Open the database
	dbConn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, errl.Errorf("failed to open database: %w", err)
	}

	// Create table if not exists
	query := `
	CREATE TABLE IF NOT EXISTS registrations (
		registration_id TEXT UNIQUE,
		email TEXT,
		first_name TEXT,
		last_name TEXT,
		company_name TEXT,
		country TEXT,
		vat_id TEXT UNIQUE,
		street_address TEXT,
		postal_code TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		issuance_at DATETIME,
		issuance_error TEXT,
		notif_email_at DATETIME,
		notif_email_error TEXT
	);`
	if _, err := dbConn.Exec(query); err != nil {
		dbConn.Close()
		return nil, errl.Errorf("failed to create table: %w", err)
	}

	return &Service{dbPath: path, conn: dbConn, runtime: runtime}, nil
}

func (s *Service) Close() error {
	return s.conn.Close()
}

func (s *Service) SaveRegistration(reg *RegistrationRecord) error {
	insertQuery := `
	INSERT INTO registrations (
		registration_id, email, first_name, last_name, company_name, country, vat_id,
		street_address, postal_code,
		created_at, updated_at, issuance_at, issuance_error, notif_email_at, notif_email_error
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	now := time.Now()
	reg.CreatedAt = now
	reg.UpdatedAt = now
	reg.IssuanceAt = now
	reg.NotifEmailAt = now
	reg.IssuanceError = ""
	reg.NotifEmailError = ""

	switch s.runtime {
	case configuration.Development, configuration.Preproduction:
		slog.Info("Saving registration in development or preproduction", "vat_id", reg.VatID, "email", reg.Email)
		oldReg, err := s.GetRegistrationByVatID(reg.VatID)
		if err != nil && err != sql.ErrNoRows {
			// A database error, we can not continue
			return err
		}

		// If the registration already exists, we amend it reusing the old registration id
		if oldReg != nil {
			slog.Info("Registration already exists, amending", "vat_id", reg.VatID, "email", reg.Email)
			return s.AmendRegistration(reg)
		} else {
			slog.Info("Registration does not exist, inserting", "vat_id", reg.VatID, "email", reg.Email)
			// If the registration does not exist, we insert it
			_, err := s.conn.Exec(insertQuery,
				reg.RegistrationID, reg.Email, reg.FirstName, reg.LastName, reg.CompanyName, reg.Country, reg.VatID,
				reg.StreetAddress, reg.PostalCode,
				reg.CreatedAt, reg.UpdatedAt, reg.IssuanceAt, reg.IssuanceError, reg.NotifEmailAt, reg.NotifEmailError,
			)
			return err
		}
	case configuration.Production:
		slog.Info("Saving registration in production", "vat_id", reg.VatID, "email", reg.Email)

		// In production, we check if the company (vat_id) already exists.
		// One individual (email) can register multiple companies.
		oldReg, err := s.GetRegistrationByVatID(reg.VatID)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		if oldReg != nil {
			return fmt.Errorf("company with VAT ID %s already registered", reg.VatID)
		}

		_, err = s.conn.Exec(insertQuery,
			reg.RegistrationID, reg.Email, reg.FirstName, reg.LastName, reg.CompanyName, reg.Country, reg.VatID,
			reg.StreetAddress, reg.PostalCode,
			reg.CreatedAt, reg.UpdatedAt, reg.IssuanceAt, reg.IssuanceError, reg.NotifEmailAt, reg.NotifEmailError,
		)
		return err
	}

	// Should never happen, return an error
	return fmt.Errorf("unknown runtime environment: %s", s.runtime)
}

func (s *Service) UpdateRegistrationStatus(reg *RegistrationRecord) error {
	reg.UpdatedAt = time.Now()
	query := `
	UPDATE registrations SET
		updated_at = ?,
		issuance_at = ?,
		issuance_error = ?,
		notif_email_at = ?,
		notif_email_error = ?
	WHERE registration_id = ? AND email = ?`
	_, err := s.conn.Exec(query,
		reg.UpdatedAt, reg.IssuanceAt, reg.IssuanceError, reg.NotifEmailAt, reg.NotifEmailError,
		reg.RegistrationID, reg.Email,
	)
	return err
}

func (s *Service) AmendRegistration(reg *RegistrationRecord) error {
	reg.UpdatedAt = time.Now()
	query := `
	UPDATE registrations SET
		registration_id = ?,
		first_name = ?,
		last_name = ?,
		company_name = ?,
		country = ?,
		street_address = ?,
		postal_code = ?,
		updated_at = ?,
		issuance_at = ?,
		issuance_error = ?,
		notif_email_at = ?,
		notif_email_error = ?,
		email = ?
	WHERE vat_id = ?`
	_, err := s.conn.Exec(query,
		reg.RegistrationID,
		reg.FirstName, reg.LastName, reg.CompanyName, reg.Country,
		reg.StreetAddress, reg.PostalCode,
		reg.UpdatedAt,
		reg.IssuanceAt, reg.IssuanceError, reg.NotifEmailAt, reg.NotifEmailError,
		reg.Email,
		reg.VatID,
	)
	return err
}

func (s *Service) GetRegistrations(limit, offset int) ([]RegistrationRecord, error) {
	query := `
	SELECT 
		registration_id, email, first_name, last_name, company_name, country, vat_id,
		street_address, postal_code,
		created_at, updated_at, issuance_at, issuance_error, notif_email_at, notif_email_error
	FROM registrations
	ORDER BY created_at DESC
	LIMIT ? OFFSET ?`

	rows, err := s.conn.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var regs []RegistrationRecord
	for rows.Next() {
		var reg RegistrationRecord
		err := rows.Scan(
			&reg.RegistrationID, &reg.Email, &reg.FirstName, &reg.LastName, &reg.CompanyName, &reg.Country, &reg.VatID,
			&reg.StreetAddress, &reg.PostalCode,
			&reg.CreatedAt, &reg.UpdatedAt, &reg.IssuanceAt, &reg.IssuanceError, &reg.NotifEmailAt, &reg.NotifEmailError,
		)
		if err != nil {
			return nil, err
		}
		regs = append(regs, reg)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return regs, nil
}

func (s *Service) GetRegistration(vatID string, email string) (*RegistrationRecord, error) {
	query := `
	SELECT 
		registration_id, email, first_name, last_name, company_name, country, vat_id,
		street_address, postal_code,
		created_at, updated_at, issuance_at, issuance_error, notif_email_at, notif_email_error
	FROM registrations
	WHERE vat_id = ? AND email = ?`

	var reg RegistrationRecord
	err := s.conn.QueryRow(query, vatID, email).Scan(
		&reg.RegistrationID, &reg.Email, &reg.FirstName, &reg.LastName, &reg.CompanyName, &reg.Country, &reg.VatID,
		&reg.StreetAddress, &reg.PostalCode,
		&reg.CreatedAt, &reg.UpdatedAt, &reg.IssuanceAt, &reg.IssuanceError, &reg.NotifEmailAt, &reg.NotifEmailError,
	)
	if err != nil {
		return nil, err
	}
	return &reg, nil
}

func (s *Service) GetRegistrationByVatID(vatID string) (*RegistrationRecord, error) {
	query := `
	SELECT 
		registration_id, email, first_name, last_name, company_name, country, vat_id,
		street_address, postal_code,
		created_at, updated_at, issuance_at, issuance_error, notif_email_at, notif_email_error
	FROM registrations
	WHERE vat_id = ?`

	var reg RegistrationRecord
	err := s.conn.QueryRow(query, vatID).Scan(
		&reg.RegistrationID, &reg.Email, &reg.FirstName, &reg.LastName, &reg.CompanyName, &reg.Country, &reg.VatID,
		&reg.StreetAddress, &reg.PostalCode,
		&reg.CreatedAt, &reg.UpdatedAt, &reg.IssuanceAt, &reg.IssuanceError, &reg.NotifEmailAt, &reg.NotifEmailError,
	)
	if err != nil {
		return nil, err
	}
	return &reg, nil
}
