package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"github.com/go-webauthn/webauthn/webauthn"
	_ "modernc.org/sqlite"
)

var DB *sql.DB

func InitDB() {
	var err error
	DB, err = sql.Open("sqlite", "./authn.db")
	if err != nil {
		log.Fatal(err)
	}

	_, err = DB.ExecContext(context.Background(), `
	CREATE TABLE users (
		email TEXT PRIMARY KEY NOT NULL,
		display_name TEXT,
		invitation_token TEXT,
		token_expiry DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
		
	CREATE TABLE credentials (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_email TEXT NOT NULL,
		cred_type TEXT NOT NULL, 
		cred_id TEXT UNIQUE NOT NULL, 
		public_data BLOB NOT NULL,     
		sign_count INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(user_email) REFERENCES users(email) ON DELETE CASCADE
	);
	`)
	if err != nil {
		log.Fatal(err)
	}
}

func GetUserByEmail(email string) (*User, error) {
	var u User
	err := DB.QueryRow("SELECT id, email FROM users WHERE email = ?", email).Scan(&u.ID, &u.Email)
	return &u, err
}

// GetUserWithCredentials retrieves a user by email along with all their registered
// WebAuthn credentials. Each credential's public_data BLOB is expected to be a
// JSON-encoded webauthn.Credential. Returns sql.ErrNoRows if the user is not found.
// Both queries run inside a single read-only transaction for a consistent snapshot.
func GetUserWithCredentials(email string) (*User, error) {
	ctx := context.Background()

	// Open a read-only transaction so both queries see the same DB snapshot.
	tx, err := DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("GetUserWithCredentials: begin tx: %w", err)
	}
	// A read-only transaction only needs a rollback (commit would work too, but
	// rollback is the safe default when we might return early on error).
	defer tx.Rollback() //nolint:errcheck

	// 1. Fetch the user row.
	var u User
	err = tx.QueryRowContext(ctx,
		`SELECT email FROM users WHERE email = ?`,
		email,
	).Scan(&u.Email)
	if err != nil {
		return nil, fmt.Errorf("GetUserWithCredentials: user lookup: %w", err)
	}

	// 2. Fetch all credentials for this user, oldest first.
	rows, err := tx.QueryContext(ctx,
		`SELECT public_data FROM credentials WHERE user_email = ? ORDER BY created_at ASC`,
		email,
	)
	if err != nil {
		return nil, fmt.Errorf("GetUserWithCredentials: credentials query: %w", err)
	}
	defer rows.Close()

	// 3. Deserialize each credential from its JSON blob.
	for rows.Next() {
		var blob []byte
		if err := rows.Scan(&blob); err != nil {
			return nil, fmt.Errorf("GetUserWithCredentials: scan: %w", err)
		}
		var cred webauthn.Credential
		if err := json.Unmarshal(blob, &cred); err != nil {
			return nil, fmt.Errorf("GetUserWithCredentials: unmarshal: %w", err)
		}
		u.Credentials = append(u.Credentials, cred)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("GetUserWithCredentials: rows: %w", err)
	}

	return &u, nil
}

func InsertUser(email string) (*User, error) {
	result, err := DB.Exec("INSERT INTO users (email) VALUES (?)", email)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()
	return &User{ID: id, Email: email}, nil
}
