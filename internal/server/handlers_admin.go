package server

import (
	"crypto/x509"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"text/template"
	"time"

	"net/http"
	"net/url"
	"strconv"

	"github.com/hesusruiz/onboardng/internal/db"
	"github.com/hesusruiz/onboardng/internal/x509util"
	"github.com/hesusruiz/utils/errl"

	"github.com/go-sprout/sprout"
	"github.com/go-sprout/sprout/group/all"

	passkeys "github.com/hesusruiz/authgo"
)

const stdCertHeader = "tls-client-certificate"
const kubeCertHeader = "X-Amzn-Mtls-Clientcert"

//go:embed templates
var templatesFS embed.FS

var tplIndex, tplDetail *template.Template

func loadTemplates(dir string) error {

	// Register all functions from Sprout to the template handler
	sproutHandler := sprout.New()
	sproutHandler.AddGroups(all.RegistryGroup())
	funcs := sproutHandler.Build()

	fileSystem, err := getFileSystem(dir)
	if err != nil {
		return errl.Errorf("error getting filesystem: %v", err)
	}

	tplIndex, err = template.New("index.hbs").Funcs(funcs).ParseFS(fileSystem, "index.hbs")
	if err != nil {
		return errl.Errorf("error parsing index.hbs: %v", err)
	}

	tplDetail, err = template.New("detail.hbs").Funcs(funcs).ParseFS(fileSystem, "detail.hbs")
	if err != nil {
		return errl.Errorf("error parsing detail.hbs: %v", err)
	}
	return nil
}

// getFileSystem returns an fs.FS to serve frontend files. In development mode
// it serves directly from the 'templates' directory to allow live reloads. In production,
// it uses the embedded filesystem.
func getFileSystem(dir string) (fs.FS, error) {

	// Get the path to this specific .go file, and serve from the '/templates' subdirectory if it exists
	if _, thisFilePath, _, ok := runtime.Caller(0); ok {

		// The frontend files should be in the 'front' subdirectory
		thisFileDir := filepath.Dir(thisFilePath)
		frontendPath := filepath.Join(thisFileDir, dir)

		if _, err := os.Stat(frontendPath); err == nil {
			println("Development mode: Serving from", frontendPath)
			return os.DirFS(frontendPath), nil
		}
	}

	// Otherwise, serve from the embedded filesystem
	// We use fs.Sub to strip the "front" prefix from the embed paths
	f, err := fs.Sub(templatesFS, dir)
	if err != nil {
		return nil, errl.Errorf("error getting filesystem: %v", err)
	}
	return f, nil
}

func (s *Server) registerAdminRoutes(mux *http.ServeMux) {

	err := loadTemplates("templates")
	if err != nil {
		panic(err)
	}

	originsEnv := os.Getenv("AUTH_ORIGINS")
	if originsEnv == "" {
		originsEnv = "http://localhost:7777"
	}
	allowedOrigins := strings.Split(originsEnv, ",")

	urlOrigin, err := url.Parse(allowedOrigins[0])
	if err != nil {
		panic(err)
	}

	pk, err := passkeys.NewPasskeys(passkeys.Config{
		RPDisplayName: "Admin Panel",
		RPID:          urlOrigin.Hostname(),
		RPOrigins:     allowedOrigins,
		PathPrefix:    "/passkeys",
		HomePage:      "/admin/index",
	})
	if err != nil {
		panic(err)
	}

	fmt.Println("allowedOrigins, ", allowedOrigins)
	fmt.Println("RPID, ", urlOrigin.Hostname())
	slog.Info("RPID", "rpid", urlOrigin.Hostname())

	pk.RegisterHandlers(mux)

	// Middleware to wrap the Admin routes
	adminChain := func(next http.HandlerFunc) http.Handler {
		return s.LogRequest(s.EnableCORS(pk.RequirePasskey(next)))
	}

	// Admin routes for pages and APIs
	mux.Handle("/admin/index", adminChain(s.PageAdminIndex))
	mux.Handle("/admin/registration", adminChain(s.PageAdminDetailsByVatID))
	mux.Handle("/admin/api/registrations", adminChain(s.APIAdminGetRegistrations))
	mux.Handle("/admin/api/registration", adminChain(s.APIAdminGetRegistrationByVatID))
	mux.Handle("/admin/api/registration-logs", adminChain(s.APIAdminGetRegistrationLogs))
	mux.Handle("/admin/api/registration-files", adminChain(s.APIAdminGetRegistrationFiles))
	mux.Handle("/admin/api/file/{file_id}", adminChain(s.APIAdminGetFile))

}

// PageAdminIndex returns the list of registrations, displaying the template index.html
func (s *Server) PageAdminIndex(w http.ResponseWriter, r *http.Request) {

	// Retrieve the user from the request context
	user := passkeys.FromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	// Prepare the email to be used in the template
	email := user.Email()

	err := loadTemplates("templates")
	if err != nil {
		err = errl.Error(err)
		slog.Error("loading templates", "error", err.Error())
		http.Error(w, "Error loading templates", http.StatusInternalServerError)
		return
	}

	// Pagination parameters if the caller does not specify them
	limit := 100
	offset := 0

	if l := r.URL.Query().Get("limit"); l != "" {
		if parsedLimit, err := strconv.Atoi(l); err == nil {
			limit = parsedLimit
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsedOffset, err := strconv.Atoi(o); err == nil {
			offset = parsedOffset
		}
	}

	regs, total, err := s.DB.GetRegistrations(limit, offset)
	if err != nil {
		err = errl.Error(err)
		slog.Error("fetching registrations", "error", err.Error())
		http.Error(w, "Error fetching registrations", http.StatusInternalServerError)
		return
	}

	tplData := struct {
		Email      string
		Regs       []db.RegistrationRecord
		Limit      int
		Offset     int
		NumEntries int
		Total      int
	}{
		Email:      email,
		Regs:       regs,
		Limit:      limit,
		Offset:     offset,
		NumEntries: len(regs),
		Total:      total,
	}

	err = tplIndex.Execute(w, tplData)
	if err != nil {
		err = errl.Error(err)
		slog.Error("executing template", "error", err.Error())
		http.Error(w, "Error executing template", http.StatusInternalServerError)
		return
	}
}

func (s *Server) PageAdminDetailsByVatID(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Retrieve the user from the request context
	user := passkeys.FromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	// Prepare the email to be used in the template
	email := user.Email()

	vatID := r.FormValue("vat_id")
	if vatID == "" {
		http.Error(w, "Missing vat_id", http.StatusBadRequest)
		return
	}

	// If the user clicked a button in the details screen
	if r.Method == http.MethodPost {
		status := r.FormValue("status")
		if status == "" {
			http.Error(w, "Missing status", http.StatusBadRequest)
			return
		}
		var approved int
		switch status {
		case "pending":
			approved = 0
		case "approved":
			approved = 1
		case "rejected":
			approved = 2
		default:
			http.Error(w, "Invalid status", http.StatusBadRequest)
			return
		}
		err := s.DB.UpdateApproval(vatID, approved)
		if err != nil {
			http.Error(w, "Error updating registration status", http.StatusInternalServerError)
			return
		}
	}

	// If we are retrieving and presenting the details

	err := loadTemplates("templates")
	if err != nil {
		http.Error(w, "Error loading templates", http.StatusInternalServerError)
		return
	}

	registration, err := s.DB.GetRegistrationByVatID(vatID)
	if err != nil {
		http.Error(w, "Registration not found", http.StatusNotFound)
		return
	}

	logs, err := s.DB.GetRegistrationLogs(vatID, 100, 0)
	if err != nil {
		http.Error(w, "Error fetching registration logs", http.StatusInternalServerError)
		return
	}

	files, err := s.DB.GetRegistrationFiles(vatID, 100, 0)
	if err != nil {
		http.Error(w, "Error fetching registration files", http.StatusInternalServerError)
		return
	}

	// TODO: eliminate the things for testing
	//

	// registration.LEARCompleted = true

	// // Append a test file entry, to see the template working
	// files = append(files, db.RegistrationFile{
	// 	FileID:         "test-file-id",
	// 	RegistrationID: registration.RegistrationID,
	// 	Name:           "test-file.pdf",
	// 	MimeType:       "application/pdf",
	// 	Size:           1024,
	// 	Status:         "uploaded",
	// 	Content:        []byte("test-file-content"),
	// 	CreatedAt:      time.Now(),
	// 	UpdatedAt:      time.Now(),
	// })

	// registration.FilesUploaded = true

	//
	// TODO: eliminate the things for testing

	tplData := struct {
		Email string
		Reg   *db.RegistrationRecord
		Files []db.RegistrationFile
		Logs  []db.RegistrationLog
	}{
		Email: email,
		Reg:   registration,
		Files: files,
		Logs:  logs,
	}

	err = tplDetail.Execute(w, tplData)
	if err != nil {
		slog.Error("Error executing template", "err", errl.Error(err))
		return
	}
}

// APIAdminGetRegistrations returns the list of registrations
func (s *Server) APIAdminGetRegistrations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	limit := 100
	offset := 0

	if l := r.URL.Query().Get("limit"); l != "" {
		if parsedLimit, err := strconv.Atoi(l); err == nil {
			limit = parsedLimit
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsedOffset, err := strconv.Atoi(o); err == nil {
			offset = parsedOffset
		}
	}

	regs, _, err := s.DB.GetRegistrations(limit, offset)
	if err != nil {
		s.SendJSON(w, r, http.StatusInternalServerError, false, "Error fetching registrations", err.Error())
		return
	}

	s.SendJSON(w, r, http.StatusOK, true, "Registrations fetched successfully", regs)
}

// APIAdminGetRegistrationByVatID returns the registration with the given VAT ID
func (s *Server) APIAdminGetRegistrationByVatID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	vatID := r.FormValue("vat_id")
	if vatID == "" {
		http.Error(w, "Missing vat_id", http.StatusBadRequest)
		return
	}

	reg, err := s.DB.GetRegistrationByVatID(vatID)
	if err != nil {
		http.Error(w, "Registration not found", http.StatusNotFound)
		return
	}

	s.SendJSON(w, r, http.StatusOK, true, "Registration fetched successfully", reg)
}

// APIAdminGetRegistrationLogs returns the list of registration logs
func (s *Server) APIAdminGetRegistrationLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	limit := 100
	offset := 0

	if l := r.FormValue("limit"); l != "" {
		if parsedLimit, err := strconv.Atoi(l); err == nil {
			limit = parsedLimit
		}
	}
	if o := r.FormValue("offset"); o != "" {
		if parsedOffset, err := strconv.Atoi(o); err == nil {
			offset = parsedOffset
		}
	}

	logsList, err := s.DB.GetRegistrationLogs("", limit, offset)
	if err != nil {
		s.SendJSON(w, r, http.StatusInternalServerError, false, "Error fetching registration logs", err.Error())
		return
	}

	s.SendJSON(w, r, http.StatusOK, true, "Registration logs fetched successfully", logsList)
}

// APIAdminGetRegistrationFiles returns the list of registration files
func (s *Server) APIAdminGetRegistrationFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	limit := 100
	offset := 0

	if l := r.FormValue("limit"); l != "" {
		if parsedLimit, err := strconv.Atoi(l); err == nil {
			limit = parsedLimit
		}
	}
	if o := r.FormValue("offset"); o != "" {
		if parsedOffset, err := strconv.Atoi(o); err == nil {
			offset = parsedOffset
		}
	}

	files, err := s.DB.GetRegistrationFiles("", limit, offset)
	if err != nil {
		s.SendJSON(w, r, http.StatusInternalServerError, false, "Error fetching registration files", err.Error())
		return
	}

	for i := range files {
		files[i].Content = nil
	}

	s.SendJSON(w, r, http.StatusOK, true, "Registration files fetched successfully", files)
}

// APIAdminGetFile returns a single file for inline viewing
func (s *Server) APIAdminGetFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	fileID := r.PathValue("file_id")
	if fileID == "" {
		http.Error(w, "Missing file ID", http.StatusBadRequest)
		return
	}

	file, err := s.DB.GetRegistrationFile(fileID)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", file.MimeType)
	w.Header().Set("Content-Disposition", "inline; filename=\""+file.Name+"\"")
	w.Write(file.Content)
}

var adminIssuerOrganizationIdentifiers = []string{
	"VATES-G87936159", // Alastria
	"VATES-11111111K", // Fake ISBE Foundation
}

var adminSubjectSerialNumbers = []string{
	"IDCES-21442837Y",
	"A12345678",
}

// isAdmin checks if the certificate is issued by any organization which is an authorized issuer.
// It additionally checks for a specific user.
func isAdmin(issuer *x509util.ELSIName, subject *x509util.ELSIName) bool {
	return slices.Contains(adminIssuerOrganizationIdentifiers, issuer.OrganizationIdentifier) || slices.Contains(adminSubjectSerialNumbers, subject.SerialNumber)
}

func (s *Server) checkAdminAuthentication(r *http.Request) (*x509util.ELSIName, error) {

	_, issuer, subject, _, err := s.retrieveCertificate(r)
	if err != nil {
		return nil, errl.Errorf("retrieving certificate: %w", err)
	}

	// Check for admin
	if !isAdmin(issuer, subject) {
		return nil, errl.Errorf("Certificate serial number '%s' or issuer.organizationIdentifier '%s' is invalid", subject.SerialNumber, issuer.OrganizationIdentifier)
	}

	return subject, nil
}

func (s *Server) retrieveCertificate(r *http.Request) (
	cert *x509.Certificate,
	issuer *x509util.ELSIName,
	subject *x509util.ELSIName,
	b64der string,
	err error) {
	// Check both the std and kube cert headers to see if we received a certificate
	certFromHeader := r.Header.Get(stdCertHeader)
	if certFromHeader != "" {
		slog.Debug("Certificate data found in standard header", "header", stdCertHeader, "cert_length", len(certFromHeader))
	} else {
		certFromHeader = r.Header.Get(kubeCertHeader)
		if certFromHeader != "" {
			slog.Debug("Certificate data found in kube header", "header", kubeCertHeader, "cert_length", len(certFromHeader))
		} else {
			return nil, nil, nil, "", errl.Errorf("No certificate provided, neither in %s nor in %s", stdCertHeader, kubeCertHeader)
		}
	}

	// Parse the certificate, which may come as DER or PEM format
	// First, detect if it seems PEM
	if strings.HasPrefix(certFromHeader, "-----BEGIN") {
		// It's PEM, so decode it from base64 and then PEM decode it

		// This header contains the URL-encoded PEM format of the entire client certificate chain presented in the connection, with +=/ as safe characters.
		// We have to first decode
		certFromHeaderDecoded, err := url.PathUnescape(certFromHeader)
		if err != nil {
			fmt.Printf("Failed to decode base64url certificate from header: %s\n", certFromHeader)
			return nil, nil, nil, "", errl.Errorf("Failed to decode base64url certificate from header: %w", err)
		}

		cert, issuer, subject, b64der, err = x509util.ParseCertificateFromPEM([]byte(certFromHeaderDecoded))
		if err != nil {
			fmt.Printf("Bad PEM certificate: %s\n", certFromHeader)
			return nil, nil, nil, "", errl.Errorf("Failed to parse certificate from PEM: %w", err)
		}
	} else {
		// Assume it is DER, so decode it directly
		cert, issuer, subject, err = x509util.ParseEIDASCertB64Der(certFromHeader)
		if err != nil {
			fmt.Printf("Bad DER certificate: %s\n", certFromHeader)
			return nil, nil, nil, "", errl.Errorf("Failed to parse certificate: %w", err)
		}
		b64der = certFromHeader
	}

	// For testing we accept personal certificates, but we do not accept that both
	// the organizationIdentifier and the serialNumber are empty.
	if subject.OrganizationIdentifier == "" && subject.SerialNumber == "" {
		return nil, nil, nil, "", errl.Errorf("Both organizationIdentifier and serialNumber are empty")
	}

	// Check certificate expiration
	now := time.Now()
	if now.Before(cert.NotBefore) {
		return nil, nil, nil, "", errl.Errorf("Certificate not yet valid, not_before: %s", cert.NotBefore.Format(time.RFC3339))
	}
	if now.After(cert.NotAfter) {
		return nil, nil, nil, "", errl.Errorf("Certificate expired not_after: %s", cert.NotAfter.Format(time.RFC3339))
	}

	return cert, issuer, subject, b64der, nil
}
