package db

import (
	"database/sql"
	"time"

	"github.com/hesusruiz/utils/errl"
)

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

func (s *Service) GetRegistrationFilesByVatID(vatID string) ([]*RegistrationFile, error) {
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

	var files []*RegistrationFile
	for rows.Next() {
		var f RegistrationFile
		err := rows.Scan(
			&f.FileID, &f.RegistrationID, &f.VatID, &f.Name, &f.MimeType, &f.Size, &f.Status, &f.Content, &f.CreatedAt, &f.UpdatedAt,
		)
		if err != nil {
			return nil, errl.Errorf("failed to scan registration file row: %w", err)
		}
		files = append(files, &f)
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

func (s *Service) GetRegistrationFiles(vatID string, limit, offset int) ([]*RegistrationFile, error) {

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

	var files []*RegistrationFile
	for rows.Next() {
		var f RegistrationFile
		err := rows.Scan(
			&f.FileID, &f.RegistrationID, &f.VatID, &f.Name, &f.MimeType, &f.Size, &f.Status, &f.Content, &f.CreatedAt, &f.UpdatedAt,
		)
		if err != nil {
			return nil, errl.Errorf("failed to scan registration file row: %w", err)
		}
		files = append(files, &f)
	}

	if err = rows.Err(); err != nil {
		return nil, errl.Errorf("failed to iterate over registration file rows: %w", err)
	}

	return files, nil
}
