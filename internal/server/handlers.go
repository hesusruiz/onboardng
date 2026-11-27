package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/hesusruiz/utils/errl"
)

// registerAppRoutes registers the application routes in the provided ServeMux
func (s *Server) registerAppRoutes(mux *http.ServeMux) error {

	// Create the App views, which will be used to render the QR code page
	appViews, err := NewAppViews(true)
	if err != nil {
		return errl.Errorf("Failed to create App views: %w", err)
	}

	// Middleware to wrap the non Admin routes
	nonAdminChain := func(next http.HandlerFunc) http.Handler {
		return s.LogRequest(s.EnableCORS(s.RateLimitIP(next)))
	}

	//------ Email authentication, used in both steps of onboarding ------

	// The frontend calls this to validate the email of the user
	mux.Handle("POST /api/validate-email", nonAdminChain(s.HandleSendEmailValidationCode))

	// The frontend calls this to validate the code sent to the user's email
	mux.Handle("POST /api/verify-code", nonAdminChain(s.HandleValidateEmailCode))

	//------ First Step of onboarding ------

	// The frontend calls this to register the user driving the onboarding
	mux.Handle("POST /api/register", nonAdminChain(s.HandleRegister))

	//------ Second Step of onboarding ------

	// The frontend calls this to update the representatives of the organization
	mux.Handle("POST /api/representatives", nonAdminChain(s.HandleUpdateRepresentatives))

	// The frontend calls this to get the status of the organization
	mux.Handle("GET /api/orgstatus", nonAdminChain(s.HandleOrgStatus))

	//----- Utility functions -----

	// The Health route does not log, to avoid polluting the logs, and does not require CSRF
	mux.Handle("GET /health", s.RateLimitIP(http.HandlerFunc(s.HandleHealth)))

	// Present the QR code for the wallet to scan
	mux.HandleFunc("GET /qr", func(w http.ResponseWriter, r *http.Request) {

		// Strip from the query the prefix "openid-credential-offer://?"
		query := r.URL.RawQuery

		// Check that it has the prefix
		if !strings.HasPrefix(query, "openid-credential-offer://?") {
			s.SendJSON(w, r, http.StatusBadRequest, false, "Invalid query", "Missing prefix openid-credential-offer://?")
			return
		}

		query = strings.TrimPrefix(query, "openid-credential-offer://?")

		// Check that query starts with "credential_offer_uri="
		if !strings.HasPrefix(query, "credential_offer_uri=") {
			s.SendJSON(w, r, http.StatusBadRequest, false, "Invalid query", "Missing credential_offer_uri")
			return
		}

		// Add the prefix for the wallet credential issuance callback URL
		credentialOfferURI := s.WalletURL + "?" + query
		fmt.Printf("credentialOfferURI: %s\n", query)

		html, err := appViews.RenderQRCodeView(credentialOfferURI, s.Marketplace)
		if err != nil {
			s.SendJSON(w, r, http.StatusInternalServerError, false, "Error rendering QR code view", err.Error())
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	})

	return nil

}

// HandleHealth returns a simple 200 OK response for health checks
func (s *Server) HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Message: "OK",
		Data:    nil,
	})
}

func (s *Server) HandleOrgStatus(w http.ResponseWriter, r *http.Request) {

	vatID := r.URL.Query().Get("vat_id")
	if vatID == "" {
		s.SendJSON(w, r, http.StatusBadRequest, false, "Missing vat_id", nil)
		return
	}

	reg, err := s.DB.GetRegistrationByVatID(vatID)
	if err != nil {
		s.SendJSON(w, r, http.StatusNotFound, false, "Registration not found", err.Error())
		return
	}

	resp := struct {
		Role     string `json:"role"`
		Approved int    `json:"approved"`
	}{
		Role:     reg.Role,
		Approved: reg.Approved,
	}

	s.SendJSON(w, r, http.StatusOK, true, "Registration fetched successfully", resp)
}

// APIResponse is the reply to the API calls
type APIResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// SendJSON utility helper
func (s *Server) SendJSON(w http.ResponseWriter, r *http.Request, status int, success bool, message string, data any) {
	// Format data as JSON if not nil, for logging purposes
	var b []byte
	if data != nil {
		b, _ = json.Marshal(data)
	}

	if status < http.StatusOK || status >= http.StatusMultipleChoices {
		slog.Error("Exit", "method", r.Method, "url", r.URL.Path, "status", status, "message", message, "data", string(b))
	} else {
		slog.Info("Exit", "method", r.Method, "url", r.URL.Path, "status", status, "message", message, "data", string(b))
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(APIResponse{
		Success: success,
		Message: message,
		Data:    data,
	})
}

// EnableCORS middleware enables permissive CORS headers only in development
// and enforces CSRF protection via the X-Requested-With header on all non-preflight requests.
// In preproduction and production the frontend is served from the same origin,
// so CORS headers are omitted and cross-origin requests are blocked by the browser.
func (s *Server) EnableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.Runtime.IsDevelopment() {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Requested-With, Authorization")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}

		if !validateCSRF(r) {
			s.SendJSON(w, r, http.StatusForbidden, false, "Security check failed: missing CSRF header", nil)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// validateCSRF checks the X-Requested-With header
func validateCSRF(r *http.Request) bool {
	return r.Header.Get("X-Requested-With") != ""
}
