package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

//go:embed front/*
var embeddedFiles embed.FS

type User struct {
	ID          int64
	Email       string
	Credentials []webauthn.Credential
}

// WebAuthn User Interface implementation
func (u *User) WebAuthnID() []byte                         { return []byte(strconv.FormatInt(u.ID, 10)) }
func (u *User) WebAuthnName() string                       { return u.Email }
func (u *User) WebAuthnDisplayName() string                { return u.Email }
func (u *User) WebAuthnCredentials() []webauthn.Credential { return u.Credentials }
func (u *User) WebAuthnIcon() string                       { return "" }

// --- 2. GLOBAL STATE (In-memory mocks for this example) ---

var (
	waInstance *webauthn.WebAuthn
	// In production, use a secure cookie or Redis for this!
	sessionStore = make(map[string]*webauthn.SessionData)
)

func main() {
	requireResidentKey := true

	var err error
	waInstance, err = webauthn.New(&webauthn.Config{
		RPDisplayName:         "Admin Panel",
		RPID:                  "localhost",
		RPOrigins:             []string{"http://localhost:8080"},
		AttestationPreference: protocol.PreferNoAttestation,
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			AuthenticatorAttachment: protocol.Platform,
			UserVerification:        protocol.VerificationRequired,
			ResidentKey:             protocol.ResidentKeyRequirementRequired,
			RequireResidentKey:      &requireResidentKey,
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()

	// Serve the frontend
	fileSystem := getFileSystem()
	mux.Handle("/", http.FileServer(http.FS(fileSystem)))

	mux.HandleFunc("/api/register/begin", handleRegisterBegin)
	mux.HandleFunc("/api/register/finish", handleRegisterFinish)

	println("Server running on http://localhost:8080")
	http.ListenAndServe(":8080", mux)
}

func handleRegisterBegin(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("t")
	// TODO: Get user from DB via token
	user := &User{ID: 1, Email: "admin@example.com"}

	options, session, _ := waInstance.BeginRegistration(user)

	// Store session temporarily to verify the next request
	sessionStore[token] = session

	json.NewEncoder(w).Encode(options)
}

func handleRegisterFinish(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("t")
	session := sessionStore[token]

	// TODO: Load user from DB
	user := &User{ID: 1, Email: "admin@example.com"}

	_, err := waInstance.FinishRegistration(user, *session, r)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	// TODO: Save credential.ID and credential.PublicKey to SQLite
	fmt.Printf("Successfully registered key for %s\n", user.Email)
	delete(sessionStore, token)
	w.WriteHeader(http.StatusOK)
}

func getFileSystem() fs.FS {
	// Set to false when deploying to production
	useLive := true

	// Get the path to this specific .go file
	_, b, _, ok := runtime.Caller(0)

	if useLive && ok {
		// Get the directory (cmd/prueba)
		projectRoot := filepath.Dir(b)
		// Join with the frontend folder
		frontendPath := filepath.Join(projectRoot, "front")

		println("Development mode: Serving from", frontendPath)
		return os.DirFS(frontendPath)
	}

	// Production: Use embedded files
	// We use fs.Sub to strip the "front" prefix from the embed paths
	f, _ := fs.Sub(embeddedFiles, "front")
	return f
}
