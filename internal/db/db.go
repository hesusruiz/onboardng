package db

import (
	"database/sql"
	"errors"
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

	// The first step of registration, called a "solo" registration, without real verification
	//

	// Unique identifier of the registration
	RegistrationID string `json:"registration_id"`

	// Email of the person/user performing the registration. The email will be verified by sending a verification code
	// to the email supplied.
	Email string `json:"email"`

	// Name and surname of the user
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`

	// Identification of the company that the user is registering
	// Official name of the company as it appears in the registry of incorporation
	CompanyName string `json:"company_name"`

	// Country code (two-letter code) of incorporation of the company
	Country string `json:"country"`

	// VAT ID of the company. Together with the country code, it is the unique identifier of the company
	VatID string `json:"vat_id"`

	// Street address of the company
	StreetAddress string `json:"street_address"`

	// City of the company
	City string `json:"city"`

	// Postal code of the company
	PostalCode string `json:"postal_code"`

	// Role of the company in DOME (Seller or Buyer)
	Role string `json:"role"`

	// Timestamps of the registration record
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Flags for the sub-steps of the first step of registration
	// true if an email has been sent to the user to notify him about the registration
	Notified bool `json:"notified"`
	// true if the Verifiable credential has been issued to the user
	Issued bool `json:"issued"`
	// true if the user has been registered in the TMF
	TMFRegistered bool `json:"tmf_registered"`

	//
	// The second step of the registration, when the company formalizes the registration
	//

	// Identification data about the Legal Representative of the company
	LRFirstName string `json:"lr_first_name"`
	LRLastName  string `json:"lr_last_name"`
	LREmail     string `json:"lr_email"`
	LRCountry   string `json:"lr_country"`
	LRIdCard    string `json:"lr_id_card"`

	// Identification data about the Legal Entity Authorized Representative (LEAR) of the company
	// This is the person who will represent the company in DOME and act as the main contact for the company
	LEARFirstName    string `json:"lear_first_name"`
	LEARLastName     string `json:"lear_last_name"`
	LEAREmail        string `json:"lear_email"`
	LEARCountry      string `json:"lear_country"`
	LEARAddress      string `json:"lear_address"`
	LEARIdCard       string `json:"lear_id_card"`
	LEARMobileNumber string `json:"lear_mobile_number"`

	// State of the second step of registration
	// Data about the LR has been successfully registered
	LRCompleted bool `json:"lr_completed"`
	// Data about the LEAR has been successfully registered
	LEARCompleted bool `json:"lear_completed"`
	// Files about the company's official registration and the LEAR have been provided, but may be pending its verification
	FilesUploaded bool `json:"files_uploaded"`
	// Status of the second step: 0 = pending, 1 = approved, 2 = rejected
	Approved int `json:"approved"`
}

// RegistrationLog represents a log entry (info, warning, error) during the registration process
type RegistrationLog struct {
	RegistrationID string    `json:"registration_id"`
	Email          string    `json:"email"`
	VatID          string    `json:"vat_id"`
	Type           string    `json:"type"` // info, warning, error
	Message        string    `json:"message"`
	CreatedAt      time.Time `json:"created_at"`
}

// RegistrationFile represents a file uploaded by a user during registration
type RegistrationFile struct {
	FileID         string    `json:"file_id"`
	RegistrationID string    `json:"registration_id"`
	VatID          string    `json:"vat_id"`
	Name           string    `json:"name"`
	MimeType       string    `json:"mime_type"`
	Size           int64     `json:"size"`
	Status         string    `json:"status"`
	Content        []byte    `json:"content"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// Service provides database operations for registrations
type Service struct {
	dbPath  string
	conn    *sql.DB
	runtime configuration.RuntimeEnv
}

func NewDBService(runtime configuration.RuntimeEnv, path string) (*Service, error) {

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
	// Note: Older DB versions may have 'approved' defined as BOOLEAN.
	// SQLite affinity treats both as NUMERIC/INTEGER, so it's compatible with values > 1.
	query := `
	CREATE TABLE IF NOT EXISTS registrations (
		registration_id TEXT UNIQUE,
		email TEXT UNIQUE,
		first_name TEXT,
		last_name TEXT,
		company_name TEXT,
		country TEXT,
		vat_id TEXT UNIQUE,
		street_address TEXT,
		city TEXT,
		postal_code TEXT,
		role TEXT,
		notified BOOLEAN,
		issued BOOLEAN,
		tmf_registered BOOLEAN,
		created_at DATETIME,
		updated_at DATETIME,
		lr_first_name TEXT,
		lr_last_name TEXT,
		lr_email TEXT,
		lr_country TEXT,
		lr_id_card TEXT,
		lear_first_name TEXT,
		lear_last_name TEXT,
		lear_email TEXT,
		lear_country TEXT,
		lear_address TEXT,
		lear_id_card TEXT,
		lear_mobile_number TEXT,
		lear_completed BOOLEAN,
		files_uploaded BOOLEAN,
		approved INTEGER
	);
	CREATE TABLE IF NOT EXISTS registration_log (
		registration_id TEXT,
		email TEXT,
		vat_id TEXT,
		type TEXT,
		message TEXT,
		created_at DATETIME
	);
	CREATE TABLE IF NOT EXISTS registration_files (
		file_id TEXT UNIQUE,
		registration_id TEXT,
		vat_id TEXT,
		name TEXT,
		mime_type TEXT,
		size INTEGER,
		status TEXT,
		content BLOB,
		created_at DATETIME,
		updated_at DATETIME
	);`
	if _, err := dbConn.Exec(query); err != nil {
		dbConn.Close()
		return nil, errl.Errorf("failed to create tables: %w", err)
	}

	return &Service{dbPath: path, conn: dbConn, runtime: runtime}, nil
}

func (s *Service) Close() error {
	return s.conn.Close()
}

var ErrorAlreadyExists = errors.New("already exists")

func (s *Service) SaveRegistration(reg *RegistrationRecord) error {
	insertQuery := `
	INSERT INTO registrations (
		registration_id, email, first_name, last_name, company_name, country, vat_id,
		street_address, city, postal_code, role,
		created_at, updated_at, notified, issued, tmf_registered,
		lr_first_name, lr_last_name, lr_email, lr_country, lr_id_card,
		lear_first_name, lear_last_name, lear_email, lear_country, lear_address, lear_id_card, lear_mobile_number,
		lear_completed, files_uploaded, approved
	) VALUES (
		:registration_id, :email, :first_name, :last_name, :company_name, :country, :vat_id,
		:street_address, :city, :postal_code, :role,
		:created_at, :updated_at, :notified, :issued, :tmf_registered,
		:lr_first_name, :lr_last_name, :lr_email, :lr_country, :lr_id_card,
		:lear_first_name, :lear_last_name, :lear_email, :lear_country, :lear_address, :lear_id_card, :lear_mobile_number,
		:lear_completed, :files_uploaded, :approved
	)`

	now := time.Now()
	reg.CreatedAt = now
	reg.UpdatedAt = now
	reg.Notified = false
	reg.Issued = false
	reg.TMFRegistered = false

	// Check if a registration with the same VAT ID already exists
	oldRegVat, err := s.GetRegistrationByVatID(reg.VatID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return errl.Errorf("failed to search for VAT ID: %w", err)
	}

	// Check if a registration with the same email already exists
	oldRegEmail, err := s.GetRegistrationByEmail(reg.Email)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return errl.Errorf("failed to search for email: %w", err)
	}

	if oldRegVat != nil && oldRegEmail != nil && oldRegVat.RegistrationID == oldRegEmail.RegistrationID {
		slog.Info("Registration exists with same email and VAT ID, amending", "environment", s.runtime, "vat_id", reg.VatID, "email", reg.Email)
		// Reuse the existing registration ID
		reg.RegistrationID = oldRegVat.RegistrationID
		return s.AmendRegistration(reg)
	}

	// Otherwise, if either exists, it's a conflict since they don't belong to the same record
	if oldRegVat != nil {
		return errl.Errorf("%w: company with VAT ID %s already registered with a different email", ErrorAlreadyExists, reg.VatID)
	}
	if oldRegEmail != nil {
		return errl.Errorf("%w: user with email %s already registered with a different VAT ID", ErrorAlreadyExists, reg.Email)
	}

	// No conflicts, insert as new
	_, err = s.conn.Exec(insertQuery,
		sql.Named("registration_id", reg.RegistrationID),
		sql.Named("email", reg.Email),
		sql.Named("first_name", reg.FirstName),
		sql.Named("last_name", reg.LastName),
		sql.Named("company_name", reg.CompanyName),
		sql.Named("country", reg.Country),
		sql.Named("vat_id", reg.VatID),
		sql.Named("street_address", reg.StreetAddress),
		sql.Named("city", reg.City),
		sql.Named("postal_code", reg.PostalCode),
		sql.Named("role", reg.Role),
		sql.Named("created_at", reg.CreatedAt),
		sql.Named("updated_at", reg.UpdatedAt),
		sql.Named("notified", reg.Notified),
		sql.Named("issued", reg.Issued),
		sql.Named("tmf_registered", reg.TMFRegistered),
		sql.Named("lr_first_name", reg.LRFirstName),
		sql.Named("lr_last_name", reg.LRLastName),
		sql.Named("lr_email", reg.LREmail),
		sql.Named("lr_country", reg.LRCountry),
		sql.Named("lr_id_card", reg.LRIdCard),
		sql.Named("lear_first_name", reg.LEARFirstName),
		sql.Named("lear_last_name", reg.LEARLastName),
		sql.Named("lear_email", reg.LEAREmail),
		sql.Named("lear_country", reg.LEARCountry),
		sql.Named("lear_address", reg.LEARAddress),
		sql.Named("lear_id_card", reg.LEARIdCard),
		sql.Named("lear_mobile_number", reg.LEARMobileNumber),
		sql.Named("lear_completed", reg.LEARCompleted),
		sql.Named("files_uploaded", reg.FilesUploaded),
		sql.Named("approved", reg.Approved),
	)
	if err != nil {
		return errl.Errorf("failed to insert registration: %w", err)
	}
	return nil

}

// UpdateRegistrationStatus updates the status flags of a registration.
// Other fields are not updated, except the timestamp of the update.
func (s *Service) UpdateRegistrationStatus(reg *RegistrationRecord) error {
	reg.UpdatedAt = time.Now()
	query := `
	UPDATE registrations SET
		updated_at = :updated_at,
		notified = :notified,
		issued = :issued,
		tmf_registered = :tmf_registered
	WHERE registration_id = :registration_id AND email = :email`
	_, err := s.conn.Exec(query,
		sql.Named("updated_at", reg.UpdatedAt),
		sql.Named("notified", reg.Notified),
		sql.Named("issued", reg.Issued),
		sql.Named("tmf_registered", reg.TMFRegistered),
		sql.Named("registration_id", reg.RegistrationID),
		sql.Named("email", reg.Email),
	)
	if err != nil {
		return errl.Errorf("failed to update registration: %w", err)
	}
	return nil
}

func (s *Service) UpdateApproval(vatID string, approved int) error {
	now := time.Now()
	query := `
	UPDATE registrations SET
		approved = :approved,
		updated_at = :updated_at
	WHERE vat_id = :vat_id`
	_, err := s.conn.Exec(query,
		sql.Named("approved", approved),
		sql.Named("updated_at", now),
		sql.Named("vat_id", vatID),
	)
	if err != nil {
		return errl.Errorf("failed to update approval: %w", err)
	}
	return nil
}

// AmendRegistration updates all fields of an existing registration, except the vat_id, email and registration_id.
func (s *Service) AmendRegistration(reg *RegistrationRecord) error {
	reg.UpdatedAt = time.Now()
	query := `
	UPDATE registrations SET
		first_name = :first_name,
		last_name = :last_name,
		company_name = :company_name,
		country = :country,
		street_address = :street_address,
		city = :city,
		postal_code = :postal_code,
		role = :role,
		updated_at = :updated_at,
		notified = :notified,
		issued = :issued,
		tmf_registered = :tmf_registered
	WHERE vat_id = :vat_id`
	_, err := s.conn.Exec(query,
		sql.Named("first_name", reg.FirstName),
		sql.Named("last_name", reg.LastName),
		sql.Named("company_name", reg.CompanyName),
		sql.Named("country", reg.Country),
		sql.Named("street_address", reg.StreetAddress),
		sql.Named("city", reg.City),
		sql.Named("postal_code", reg.PostalCode),
		sql.Named("role", reg.Role),
		sql.Named("updated_at", reg.UpdatedAt),
		sql.Named("notified", reg.Notified),
		sql.Named("issued", reg.Issued),
		sql.Named("tmf_registered", reg.TMFRegistered),
		sql.Named("vat_id", reg.VatID),
	)
	if err != nil {
		return errl.Errorf("failed to amend registration: %w", err)
	}
	return nil
}

func (s *Service) GetRegistrations(limit, offset int) ([]RegistrationRecord, error) {
	query := `
	SELECT 
		registration_id, email, first_name, last_name, company_name, country, vat_id,
		street_address, city, postal_code, role,
		created_at, updated_at, notified, issued, tmf_registered,
		lr_first_name, lr_last_name, lr_email, lr_country, lr_id_card,
		lear_first_name, lear_last_name, lear_email, lear_country, lear_address, lear_id_card, lear_mobile_number,
		lear_completed, files_uploaded, approved
	FROM registrations
	ORDER BY created_at DESC
	LIMIT :limit OFFSET :offset`

	rows, err := s.conn.Query(query,
		sql.Named("limit", limit),
		sql.Named("offset", offset),
	)
	if err != nil {
		return nil, errl.Errorf("failed to get registrations: %w", err)
	}
	defer rows.Close()

	var regs []RegistrationRecord
	for rows.Next() {
		var reg RegistrationRecord
		err := rows.Scan(
			&reg.RegistrationID, &reg.Email, &reg.FirstName, &reg.LastName, &reg.CompanyName, &reg.Country, &reg.VatID,
			&reg.StreetAddress, &reg.City, &reg.PostalCode, &reg.Role,
			&reg.CreatedAt, &reg.UpdatedAt, &reg.Notified, &reg.Issued, &reg.TMFRegistered,
			&reg.LRFirstName, &reg.LRLastName, &reg.LREmail, &reg.LRCountry, &reg.LRIdCard,
			&reg.LEARFirstName, &reg.LEARLastName, &reg.LEAREmail, &reg.LEARCountry, &reg.LEARAddress, &reg.LEARIdCard, &reg.LEARMobileNumber,
			&reg.LEARCompleted, &reg.FilesUploaded, &reg.Approved,
		)
		if err != nil {
			return nil, errl.Errorf("failed to scan registration row: %w", err)
		}
		regs = append(regs, reg)
	}

	if err = rows.Err(); err != nil {
		return nil, errl.Errorf("failed to iterate over registration rows: %w", err)
	}

	return regs, nil
}

func (s *Service) GetRegistration(vatID string, email string) (*RegistrationRecord, error) {
	query := `
	SELECT 
		registration_id, email, first_name, last_name, company_name, country, vat_id,
		street_address, city, postal_code, role,
		created_at, updated_at, notified, issued, tmf_registered,
		lr_first_name, lr_last_name, lr_email, lr_country, lr_id_card,
		lear_first_name, lear_last_name, lear_email, lear_country, lear_address, lear_id_card, lear_mobile_number,
		lear_completed, files_uploaded, approved
	FROM registrations
	WHERE vat_id = :vat_id AND email = :email`

	var reg RegistrationRecord
	err := s.conn.QueryRow(query,
		sql.Named("vat_id", vatID),
		sql.Named("email", email),
	).Scan(
		&reg.RegistrationID, &reg.Email, &reg.FirstName, &reg.LastName, &reg.CompanyName, &reg.Country, &reg.VatID,
		&reg.StreetAddress, &reg.City, &reg.PostalCode, &reg.Role,
		&reg.CreatedAt, &reg.UpdatedAt, &reg.Notified, &reg.Issued, &reg.TMFRegistered,
		&reg.LRFirstName, &reg.LRLastName, &reg.LREmail, &reg.LRCountry, &reg.LRIdCard,
		&reg.LEARFirstName, &reg.LEARLastName, &reg.LEAREmail, &reg.LEARCountry, &reg.LEARAddress, &reg.LEARIdCard, &reg.LEARMobileNumber,
		&reg.LEARCompleted, &reg.FilesUploaded, &reg.Approved,
	)
	if err != nil {
		return nil, errl.Errorf("failed to get registration by VAT ID and email: %w", err)
	}
	return &reg, nil
}

// GetRegistrationByEmailOrVatID returns a registration by email or VAT ID
// It returns an error if the registration is not found under any of the provided identifiers
func (s *Service) GetRegistrationByEmailOrVatID(email string, vatID string) (*RegistrationRecord, error) {
	query := `
	SELECT 
		registration_id, email, first_name, last_name, company_name, country, vat_id,
		street_address, city, postal_code, role,
		created_at, updated_at, notified, issued, tmf_registered,
		lr_first_name, lr_last_name, lr_email, lr_country, lr_id_card,
		lear_first_name, lear_last_name, lear_email, lear_country, lear_address, lear_id_card, lear_mobile_number,
		lear_completed, files_uploaded, approved
	FROM registrations
	WHERE email = :email OR vat_id = :vat_id`

	var reg RegistrationRecord
	err := s.conn.QueryRow(query,
		sql.Named("email", email),
		sql.Named("vat_id", vatID),
	).Scan(
		&reg.RegistrationID, &reg.Email, &reg.FirstName, &reg.LastName, &reg.CompanyName, &reg.Country, &reg.VatID,
		&reg.StreetAddress, &reg.City, &reg.PostalCode, &reg.Role,
		&reg.CreatedAt, &reg.UpdatedAt, &reg.Notified, &reg.Issued, &reg.TMFRegistered,
		&reg.LRFirstName, &reg.LRLastName, &reg.LREmail, &reg.LRCountry, &reg.LRIdCard,
		&reg.LEARFirstName, &reg.LEARLastName, &reg.LEAREmail, &reg.LEARCountry, &reg.LEARAddress, &reg.LEARIdCard, &reg.LEARMobileNumber,
		&reg.LEARCompleted, &reg.FilesUploaded, &reg.Approved,
	)
	if err != nil {
		return nil, errl.Errorf("failed to get registration by email or VAT ID: %w", err)
	}
	return &reg, nil
}

func (s *Service) GetRegistrationByVatID(vatID string) (*RegistrationRecord, error) {
	query := `
	SELECT 
		registration_id, email, first_name, last_name, company_name, country, vat_id,
		street_address, city, postal_code, role,
		created_at, updated_at, notified, issued, tmf_registered,
		lr_first_name, lr_last_name, lr_email, lr_country, lr_id_card,
		lear_first_name, lear_last_name, lear_email, lear_country, lear_address, lear_id_card, lear_mobile_number,
		lear_completed, files_uploaded, approved
	FROM registrations
	WHERE vat_id = :vat_id`

	var reg RegistrationRecord
	err := s.conn.QueryRow(query, sql.Named("vat_id", vatID)).Scan(
		&reg.RegistrationID, &reg.Email, &reg.FirstName, &reg.LastName, &reg.CompanyName, &reg.Country, &reg.VatID,
		&reg.StreetAddress, &reg.City, &reg.PostalCode, &reg.Role,
		&reg.CreatedAt, &reg.UpdatedAt, &reg.Notified, &reg.Issued, &reg.TMFRegistered,
		&reg.LRFirstName, &reg.LRLastName, &reg.LREmail, &reg.LRCountry, &reg.LRIdCard,
		&reg.LEARFirstName, &reg.LEARLastName, &reg.LEAREmail, &reg.LEARCountry, &reg.LEARAddress, &reg.LEARIdCard, &reg.LEARMobileNumber,
		&reg.LEARCompleted, &reg.FilesUploaded, &reg.Approved,
	)
	if err != nil {
		return nil, errl.Errorf("failed to get registration by VAT ID: %w", err)
	}
	return &reg, nil
}

func (s *Service) GetRegistrationByEmail(email string) (*RegistrationRecord, error) {
	query := `
	SELECT 
		registration_id, email, first_name, last_name, company_name, country, vat_id,
		street_address, city, postal_code, role,
		created_at, updated_at, notified, issued, tmf_registered,
		lr_first_name, lr_last_name, lr_email, lr_country, lr_id_card,
		lear_first_name, lear_last_name, lear_email, lear_country, lear_address, lear_id_card, lear_mobile_number,
		lear_completed, files_uploaded, approved
	FROM registrations
	WHERE email = :email`

	var reg RegistrationRecord
	err := s.conn.QueryRow(query, sql.Named("email", email)).Scan(
		&reg.RegistrationID, &reg.Email, &reg.FirstName, &reg.LastName, &reg.CompanyName, &reg.Country, &reg.VatID,
		&reg.StreetAddress, &reg.City, &reg.PostalCode, &reg.Role,
		&reg.CreatedAt, &reg.UpdatedAt, &reg.Notified, &reg.Issued, &reg.TMFRegistered,
		&reg.LRFirstName, &reg.LRLastName, &reg.LREmail, &reg.LRCountry, &reg.LRIdCard,
		&reg.LEARFirstName, &reg.LEARLastName, &reg.LEAREmail, &reg.LEARCountry, &reg.LEARAddress, &reg.LEARIdCard, &reg.LEARMobileNumber,
		&reg.LEARCompleted, &reg.FilesUploaded, &reg.Approved,
	)
	if err != nil {
		return nil, errl.Errorf("failed to get registration by email: %w", err)
	}
	return &reg, nil
}

func (s *Service) SaveRegistrationLog(logEntry *RegistrationLog) error {
	query := `
	INSERT INTO registration_log (
		registration_id, email, vat_id, type, message, created_at
	) VALUES (:registration_id, :email, :vat_id, :type, :message, :created_at)`

	logEntry.CreatedAt = time.Now()

	_, err := s.conn.Exec(query,
		sql.Named("registration_id", logEntry.RegistrationID),
		sql.Named("email", logEntry.Email),
		sql.Named("vat_id", logEntry.VatID),
		sql.Named("type", logEntry.Type),
		sql.Named("message", logEntry.Message),
		sql.Named("created_at", logEntry.CreatedAt),
	)
	if err != nil {
		return errl.Errorf("failed to insert registration log: %w", err)
	}
	return nil
}

func (s *Service) SaveRegistrationFile(f *RegistrationFile) error {
	query := `
	INSERT INTO registration_files (
		file_id, registration_id, vat_id, name, mime_type, size, status, content, created_at, updated_at
	) VALUES (
		:file_id, :registration_id, :vat_id, :name, :mime_type, :size, :status, :content, :created_at, :updated_at
	)`

	now := time.Now()
	f.CreatedAt = now
	f.UpdatedAt = now
	if f.Status == "" {
		f.Status = "uploaded" // Default status
	}

	_, err := s.conn.Exec(query,
		sql.Named("file_id", f.FileID),
		sql.Named("registration_id", f.RegistrationID),
		sql.Named("vat_id", f.VatID),
		sql.Named("name", f.Name),
		sql.Named("mime_type", f.MimeType),
		sql.Named("size", f.Size),
		sql.Named("status", f.Status),
		sql.Named("content", f.Content),
		sql.Named("created_at", f.CreatedAt),
		sql.Named("updated_at", f.UpdatedAt),
	)
	if err != nil {
		return errl.Errorf("failed to insert registration file: %w", err)
	}
	return nil
}

func (s *Service) GetRegistrationFilesByVatID(vatID string) ([]RegistrationFile, error) {
	query := `
	SELECT 
		file_id, registration_id, vat_id, name, mime_type, size, status, content, created_at, updated_at
	FROM registration_files
	WHERE vat_id = :vat_id
	ORDER BY created_at ASC`

	rows, err := s.conn.Query(query, sql.Named("vat_id", vatID))
	if err != nil {
		return nil, errl.Errorf("failed to get registration files: %w", err)
	}
	defer rows.Close()

	var files []RegistrationFile
	for rows.Next() {
		var f RegistrationFile
		err := rows.Scan(
			&f.FileID, &f.RegistrationID, &f.VatID, &f.Name, &f.MimeType, &f.Size, &f.Status, &f.Content, &f.CreatedAt, &f.UpdatedAt,
		)
		if err != nil {
			return nil, errl.Errorf("failed to scan registration file row: %w", err)
		}
		files = append(files, f)
	}

	if err = rows.Err(); err != nil {
		return nil, errl.Errorf("failed to iterate over registration file rows: %w", err)
	}

	return files, nil
}

func (s *Service) UpdateRegistrationFileStatus(fileID string, status string) error {
	query := `
	UPDATE registration_files SET
		status = :status,
		updated_at = :updated_at
	WHERE file_id = :file_id`

	_, err := s.conn.Exec(query,
		sql.Named("status", status),
		sql.Named("updated_at", time.Now()),
		sql.Named("file_id", fileID),
	)
	if err != nil {
		return errl.Errorf("failed to update registration file status: %w", err)
	}
	return nil
}

func (s *Service) DeleteRegistrationFile(fileID string) error {
	query := `DELETE FROM registration_files WHERE file_id = :file_id`
	_, err := s.conn.Exec(query, sql.Named("file_id", fileID))
	if err != nil {
		return errl.Errorf("failed to delete registration file: %w", err)
	}
	return nil
}

func (s *Service) GetRegistrationFile(fileID string) (*RegistrationFile, error) {
	query := `
	SELECT 
		file_id, registration_id, vat_id, name, mime_type, size, status, content, created_at, updated_at
	FROM registration_files
	WHERE file_id = :file_id`

	var f RegistrationFile
	err := s.conn.QueryRow(query, sql.Named("file_id", fileID)).Scan(
		&f.FileID, &f.RegistrationID, &f.VatID, &f.Name, &f.MimeType, &f.Size, &f.Status, &f.Content, &f.CreatedAt, &f.UpdatedAt,
	)
	if err != nil {
		return nil, errl.Errorf("failed to get registration file: %w", err)
	}
	return &f, nil
}

func (s *Service) UpdateRegistrationFileContent(fileID string, content []byte) error {
	query := `
	UPDATE registration_files SET
		content = :content,
		updated_at = :updated_at
	WHERE file_id = :file_id`

	_, err := s.conn.Exec(query,
		sql.Named("content", content),
		sql.Named("updated_at", time.Now()),
		sql.Named("file_id", fileID),
	)
	if err != nil {
		return errl.Errorf("failed to update registration file content: %w", err)
	}
	return nil
}

func (s *Service) GetRegistrationLogs(vatID string, limit, offset int) ([]RegistrationLog, error) {

	var rows *sql.Rows
	var err error

	if vatID != "" {
		query := `
		SELECT 
			registration_id, email, vat_id, type, message, created_at
		FROM registration_log
		WHERE vat_id = :vat_id
		ORDER BY created_at DESC`

		rows, err = s.conn.Query(query,
			sql.Named("vat_id", vatID),
		)
	} else {
		query := `
		SELECT 
			registration_id, email, vat_id, type, message, created_at
		FROM registration_log
		ORDER BY created_at DESC
		LIMIT :limit OFFSET :offset`

		rows, err = s.conn.Query(query,
			sql.Named("limit", limit),
			sql.Named("offset", offset),
		)
	}
	if err != nil {
		return nil, errl.Errorf("failed to get registration logs: %w", err)
	}
	defer rows.Close()

	var logsList []RegistrationLog
	for rows.Next() {
		var l RegistrationLog
		err := rows.Scan(
			&l.RegistrationID, &l.Email, &l.VatID, &l.Type, &l.Message, &l.CreatedAt,
		)
		if err != nil {
			return nil, errl.Errorf("failed to scan registration log row: %w", err)
		}
		logsList = append(logsList, l)
	}

	if err = rows.Err(); err != nil {
		return nil, errl.Errorf("failed to iterate over registration log rows: %w", err)
	}

	return logsList, nil
}

func (s *Service) GetRegistrationFiles(vatID string, limit, offset int) ([]RegistrationFile, error) {

	var rows *sql.Rows
	var err error

	if vatID != "" {
		query := `
		SELECT 
			file_id, registration_id, vat_id, name, mime_type, size, status, content, created_at, updated_at
			FROM registration_files
			WHERE vat_id = :vat_id
			ORDER BY created_at DESC`

		rows, err = s.conn.Query(query,
			sql.Named("vat_id", vatID),
		)
	} else {
		query := `
		SELECT 
			file_id, registration_id, vat_id, name, mime_type, size, status, content, created_at, updated_at
			FROM registration_files
			ORDER BY created_at DESC
			LIMIT :limit OFFSET :offset`

		rows, err = s.conn.Query(query,
			sql.Named("limit", limit),
			sql.Named("offset", offset),
		)
	}
	if err != nil {
		return nil, errl.Errorf("failed to get registration files: %w", err)
	}
	defer rows.Close()

	var files []RegistrationFile
	for rows.Next() {
		var f RegistrationFile
		err := rows.Scan(
			&f.FileID, &f.RegistrationID, &f.VatID, &f.Name, &f.MimeType, &f.Size, &f.Status, &f.Content, &f.CreatedAt, &f.UpdatedAt,
		)
		if err != nil {
			return nil, errl.Errorf("failed to scan registration file row: %w", err)
		}
		files = append(files, f)
	}

	if err = rows.Err(); err != nil {
		return nil, errl.Errorf("failed to iterate over registration file rows: %w", err)
	}

	return files, nil
}

// GetRegistrationByID returns a registration by its registration ID
func (s *Service) GetRegistrationByID(registrationID string) (*RegistrationRecord, error) {
	query := `
	SELECT 
		registration_id, email, first_name, last_name, company_name, country, vat_id,
		street_address, city, postal_code,
		created_at, updated_at, notified, issued, tmf_registered,
		lr_first_name, lr_last_name, lr_email, lr_country, lr_id_card,
		lear_first_name, lear_last_name, lear_email, lear_country, lear_address, lear_id_card, lear_mobile_number,
		lear_completed, files_uploaded, approved
	FROM registrations
	WHERE registration_id = :registration_id`

	var reg RegistrationRecord
	err := s.conn.QueryRow(query, sql.Named("registration_id", registrationID)).Scan(
		&reg.RegistrationID, &reg.Email, &reg.FirstName, &reg.LastName, &reg.CompanyName, &reg.Country, &reg.VatID,
		&reg.StreetAddress, &reg.City, &reg.PostalCode,
		&reg.CreatedAt, &reg.UpdatedAt, &reg.Notified, &reg.Issued, &reg.TMFRegistered,
		&reg.LRFirstName, &reg.LRLastName, &reg.LREmail, &reg.LRCountry, &reg.LRIdCard,
		&reg.LEARFirstName, &reg.LEARLastName, &reg.LEAREmail, &reg.LEARCountry, &reg.LEARAddress, &reg.LEARIdCard, &reg.LEARMobileNumber,
		&reg.LEARCompleted, &reg.FilesUploaded, &reg.Approved,
	)
	if err != nil {
		return nil, errl.Errorf("failed to get registration by ID: %w", err)
	}
	return &reg, nil
}

// UpdateRepresentativesByVatID updates the LR and LEAR fields of a registration by VAT ID.
// The update is ignored by the database if the registration is already approved (approved = 0).
func (s *Service) UpdateRepresentativesByVatID(vatID string, rep *RegistrationRecord) error {
	query := `
	UPDATE registrations SET
		role = :role,
		lr_first_name = :lr_first_name,
		lr_last_name = :lr_last_name,
		lr_email = :lr_email,
		lr_country = :lr_country,
		lr_id_card = :lr_id_card,
		lear_first_name = :lear_first_name,
		lear_last_name = :lear_last_name,
		lear_email = :lear_email,
		lear_country = :lear_country,
		lear_address = :lear_address,
		lear_id_card = :lear_id_card,
		lear_mobile_number = :lear_mobile_number,
		lear_completed = :lear_completed,
		updated_at = :updated_at
	WHERE vat_id = :vat_id AND approved = 0`

	rep.UpdatedAt = time.Now()
	_, err := s.conn.Exec(query,
		sql.Named("role", rep.Role),
		sql.Named("lr_first_name", rep.LRFirstName),
		sql.Named("lr_last_name", rep.LRLastName),
		sql.Named("lr_email", rep.LREmail),
		sql.Named("lr_country", rep.LRCountry),
		sql.Named("lr_id_card", rep.LRIdCard),
		sql.Named("lear_first_name", rep.LEARFirstName),
		sql.Named("lear_last_name", rep.LEARLastName),
		sql.Named("lear_email", rep.LEAREmail),
		sql.Named("lear_country", rep.LEARCountry),
		sql.Named("lear_address", rep.LEARAddress),
		sql.Named("lear_id_card", rep.LEARIdCard),
		sql.Named("lear_mobile_number", rep.LEARMobileNumber),
		sql.Named("lear_completed", rep.LEARCompleted),
		sql.Named("updated_at", rep.UpdatedAt),
		sql.Named("vat_id", vatID),
	)
	if err != nil {
		return errl.Errorf("failed to update representatives: %w", err)
	}
	return nil
}
