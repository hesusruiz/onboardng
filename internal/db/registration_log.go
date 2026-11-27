package db

import (
	"database/sql"
	"time"

	"github.com/hesusruiz/utils/errl"
)

// LogRegWarning logs a warning entry in the registration log.
// regId and email are optional (they may be empty strings)
func (s *Service) LogRegWarn(vatId string, message string, regId string, email string) error {
	return s.LogReg(WarnM, vatId, message, regId, email)
}

// LogRegError logs an error entry in the registration log.
// regId and email are optional (they may be empty strings)
func (s *Service) LogRegError(vatId string, message string, regId string, email string) error {
	return s.LogReg(ErrorM, vatId, message, regId, email)
}

// LogRegInfo logs an info entry in the registration log.
// regId and email are optional (they may be empty strings)
func (s *Service) LogRegInfo(vatId string, message string, regId string, email string) error {
	return s.LogReg(InfoM, vatId, message, regId, email)
}

// LogRegDebug logs a debug entry in the registration log.
// regId and email are optional (they may be empty strings)
func (s *Service) LogRegDebug(vatId string, message string, regId string, email string) error {
	return s.LogReg(DebugM, vatId, message, regId, email)
}

type SeverityLevel int

const (
	DebugM SeverityLevel = iota
	InfoM
	WarnM
	ErrorM
)

func (s SeverityLevel) String() string {
	switch s {
	case DebugM:
		return "debug"
	case InfoM:
		return "info"
	case WarnM:
		return "warn"
	case ErrorM:
		return "error"
	default:
		return "unknown"
	}
}

// RegistrationLogEntry represents a log entry (info, warning, error) during the registration process
type RegistrationLogEntry struct {
	RegistrationID string    `json:"registration_id"`
	Email          string    `json:"email"`
	VatID          string    `json:"vat_id"`
	Type           string    `json:"type"` // info, warning, error
	Message        string    `json:"message"`
	CreatedAt      time.Time `json:"created_at"`
}

// LogReg logs an entry in the registration log.
// regId and email are optional (they may be empty strings)
func (s *Service) LogReg(level SeverityLevel, vatId string, message string, regId string, email string) error {
	if vatId == "" || message == "" {
		return errl.Errorf("vatId and message must be non-empty")
	}
	query := `
	INSERT INTO registration_log (
		registration_id, email, vat_id, type, message, created_at
	) VALUES (:registration_id, :email, :vat_id, :type, :message, :created_at)`

	createdAt := time.Now()

	_, err := s.conn.Exec(query,
		sql.Named("registration_id", regId),
		sql.Named("email", email),
		sql.Named("vat_id", vatId),
		sql.Named("type", level.String()),
		sql.Named("message", message),
		sql.Named("created_at", createdAt),
	)
	if err != nil {
		return errl.Errorf("failed to insert registration log: %w", err)
	}
	return nil
}

// GetRegistrationLogs gets all registration logs for a specific vatID.
// If vatID is empty, it gets all registration logs.
func (s *Service) GetRegistrationLogs(vatID string, limit, offset int) ([]*RegistrationLogEntry, error) {

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

	var logsList []*RegistrationLogEntry
	for rows.Next() {
		var l RegistrationLogEntry
		err := rows.Scan(
			&l.RegistrationID, &l.Email, &l.VatID, &l.Type, &l.Message, &l.CreatedAt,
		)
		if err != nil {
			return nil, errl.Errorf("failed to scan registration log row: %w", err)
		}
		logsList = append(logsList, &l)
	}

	if err = rows.Err(); err != nil {
		return nil, errl.Errorf("failed to iterate over registration log rows: %w", err)
	}

	return logsList, nil
}
