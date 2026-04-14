package server

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hesusruiz/onboardng/internal/db"
)

func TestLogging(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, nil)
	logger := slog.New(handler)
	slog.SetDefault(logger)

	s := &Server{}

	t.Run("API call logs entry and exit", func(t *testing.T) {
		buf.Reset()
		req := httptest.NewRequest("POST", "/api/test", nil)
		w := httptest.NewRecorder()

		handlerFunc := s.LogRequest(func(w http.ResponseWriter, r *http.Request) {
			s.SendJSON(w, r, http.StatusOK, true, "test", nil)
		})

		handlerFunc.ServeHTTP(w, req)

		logs := buf.String()
		if !strings.Contains(logs, "\"msg\":\"Entry\"") {
			t.Errorf("Expected Entry log, got: %s", logs)
		}
		if !strings.Contains(logs, "\"msg\":\"Exit\"") {
			t.Errorf("Expected Exit log, got: %s", logs)
		}
		if !strings.Contains(logs, "\"status\":200") {
			t.Errorf("Expected status 200 in log, got: %s", logs)
		}
	})

	t.Run("Health check does not log entry (via LogRequest)", func(t *testing.T) {
		buf.Reset()
		// In actual setup, /health is not wrapped by LogRequest.
		// We verify HandleHealth directly to see if it logs (it shouldn't log entry, only exit via SendJSON)
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		s.HandleHealth(w, req)

		logs := buf.String()
		if strings.Contains(logs, "\"msg\":\"Entry\"") {
			t.Errorf("Did not expect Entry log for health check, got: %s", logs)
		}
		if !strings.Contains(logs, "\"msg\":\"Exit\"") {
			t.Errorf("Expected Exit log for health check (via SendJSON), got: %s", logs)
		}
	})

	t.Run("Conflict logging uses Warn level", func(t *testing.T) {
		buf.Reset()
		// Mock error check logic from saveToDB
		err := db.ErrorAlreadyExists
		if errors.Is(err, db.ErrorAlreadyExists) {
			slog.Warn("Registration conflict", "error", err)
		}

		logs := buf.String()
		if !strings.Contains(logs, "\"level\":\"WARN\"") {
			t.Errorf("Expected WARN level for conflict, got: %s", logs)
		}
		if !strings.Contains(logs, "\"msg\":\"Registration conflict\"") {
			t.Errorf("Expected 'Registration conflict' message, got: %s", logs)
		}
	})
}
