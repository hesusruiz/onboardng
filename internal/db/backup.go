package db

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hesusruiz/utils/errl"
)

// RunMaintenance performs some database maintenance operations
func (s *Service) RunMaintenance(ctx context.Context) error {

	slog.Info("Starting database maintenance")
	start := time.Now()
	defer func() {
		slog.Info("Database maintenance completed", "elapsed", time.Since(start))
	}()

	// Compact the database
	if err := s.Compact(ctx); err != nil {
		return errl.Errorf("failed to compact database: %w", err)
	}

	// Backup the database
	if err := s.Backup(ctx); err != nil {
		return errl.Errorf("failed to backup database: %w", err)
	}

	return nil
}

func (s *Service) Compact(ctx context.Context) error {
	// Perform VACUUM
	start := time.Now()
	if _, err := s.conn.ExecContext(ctx, "VACUUM"); err != nil {
		slog.Error("failed to vacuum database", "error", err, "elapsed", time.Since(start))
		return errl.Errorf("failed to vacuum database: %w", err)
	}

	slog.Info("Database vacuumed successfully", "elapsed", time.Since(start))
	return nil
}

// Backup creates a backup of the origin database in a "backups" subdirectory.
// The backup filename includes the day of the week (1=Monday, ..., 7=Sunday).
func (s *Service) Backup(ctx context.Context) error {
	// Get absolute path of origin to safely manipulate directories
	absOrigin, err := filepath.Abs(s.dbPath)
	if err != nil {
		return errl.Errorf("failed to get absolute path of origin: %w", err)
	}

	dir := filepath.Dir(absOrigin)
	filename := filepath.Base(absOrigin)
	ext := filepath.Ext(filename)
	name := strings.TrimSuffix(filename, ext)

	// Create backups directory
	backupDir := filepath.Join(dir, "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return errl.Errorf("failed to create backup directory: %w", err)
	}

	// Determine day of week (1=Monday, ..., 7=Sunday)
	weekday := time.Now().Weekday()
	dayNum := int(weekday)
	if dayNum == 0 {
		dayNum = 7
	}

	// Construct destination filename
	backupFilename := fmt.Sprintf("%s_%d%s", name, dayNum, ext)
	destination := filepath.Join(backupDir, backupFilename)

	// Perform sync
	slog.Info("Performing backup", slog.String("origin", absOrigin), slog.String("destination", destination))
	if err := SQLiteRsync(absOrigin, destination); err != nil {
		return errl.Errorf("failed to perform sync: %w", err)
	}

	return nil
}
