package server

import (
	"embed"
	"log/slog"
	"text/template"
	"time"

	"net/http"
	"strconv"

	"github.com/hesusruiz/onboardng/internal/db"
	"github.com/hesusruiz/utils/errl"

	"github.com/go-sprout/sprout"
	"github.com/go-sprout/sprout/group/all"
)

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

	vatID := r.URL.Query().Get("vat_id")
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

	fileID := r.URL.Path[len("/api/admin/file/"):]
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
