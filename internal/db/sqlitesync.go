package db

import (
	"bytes"
	"os/exec"

	"github.com/hesusruiz/utils/errl"
)

// SQLiteRsync executes the sqlite3_rsync binary to synchronize an origin database to a destination database.
// It assumes the sqlite3_rsync binary is present in the path.
// This function blocks until the command completes.
func SQLiteRsync(origin, destination string) error {

	binaryPath := "sqlite3_rsync"
	cmd := exec.Command(binaryPath, origin, destination)

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute the command
	err := cmd.Run()
	if err != nil {
		return errl.Errorf("sqlite3_rsync failed: %w, stderr: %s, stdout: %s", err, stderr.String(), stdout.String())
	}

	return nil
}
