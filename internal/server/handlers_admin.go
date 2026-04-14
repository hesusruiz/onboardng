package server

import (
	"crypto/x509"
	"embed"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"text/template"
	"time"

	"net/http"
	"net/url"
	"strconv"

	"github.com/hesusruiz/onboardng/internal/configuration"
	"github.com/hesusruiz/onboardng/internal/db"
	"github.com/hesusruiz/onboardng/internal/x509util"
	"github.com/hesusruiz/utils/errl"

	"github.com/go-sprout/sprout"
	"github.com/go-sprout/sprout/group/all"
)

const stdCertHeader = "tls-client-certificate"
const kubeCertHeader = "X-Amzn-Mtls-Clientcert"

//go:embed templates
var templatesFS embed.FS

var tplIndex, tplDetail *template.Template

func init() {

	// 1. Initialize Sprout handler
	sproutHandler := sprout.New()

	// Add all built-in registries to the handler
	sproutHandler.AddGroups(all.RegistryGroup())

	// 2. Build the Sprout function map
	funcs := sproutHandler.Build()

	var err error
	tplIndex, err = template.New("index.hbs").Funcs(funcs).ParseFS(templatesFS, "templates/index.hbs")
	if err != nil {
		panic(err)
	}

	tplDetail, err = template.New("detail.hbs").Funcs(funcs).ParseFS(templatesFS, "templates/detail.hbs")
	if err != nil {
		panic(err)
	}

}

// PageAdminIndex returns the list of registrations, displaying the template index.html
func (s *Server) PageAdminIndex(w http.ResponseWriter, r *http.Request) {

	err := tplIndex.Execute(w, nil)
	if err != nil {
		http.Error(w, "Error executing template", http.StatusInternalServerError)
		return
	}
}

func (s *Server) PageAdminDetailsByVatID(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	vatID := r.FormValue("vat_id")
	if vatID == "" {
		http.Error(w, "Missing vat_id", http.StatusBadRequest)
		return
	}

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

	// Append a test file entry, to see the template working
	files = append(files, db.RegistrationFile{
		FileID:         "test-file-id",
		RegistrationID: registration.RegistrationID,
		Name:           "test-file.pdf",
		MimeType:       "application/pdf",
		Size:           1024,
		Status:         "uploaded",
		Content:        []byte("test-file-content"),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	})

	tplData := struct {
		Reg   *db.RegistrationRecord
		Files []db.RegistrationFile
		Logs  []db.RegistrationLog
	}{
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

	regs, err := s.DB.GetRegistrations(limit, offset)
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

	if s.Runtime != configuration.Production && fileID == "test-file-id" { // This is for testing purposes only
		file := db.RegistrationFile{
			FileID:         "test-file-id",
			RegistrationID: "1234",
			Name:           "test-file.txt",
			MimeType:       "text/plain",
			Size:           1024,
			Status:         "uploaded",
			Content:        []byte("Hello, how are you?\nThis is a test file."),
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		w.Header().Set("Content-Type", file.MimeType)
		w.Header().Set("Content-Disposition", "inline; filename=\""+file.Name+"\"")
		w.Write(file.Content)
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
