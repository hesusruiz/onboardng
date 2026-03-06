package server

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/hesusruiz/onboardng/common"
	"github.com/hesusruiz/onboardng/credissuance"
	"github.com/hesusruiz/onboardng/internal/db"
	"github.com/hesusruiz/utils/errl"
)

// APIResponse is the reply to the API calls
type APIResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type RegistrationRequest struct {
	FirstName     string `json:"firstName"`
	LastName      string `json:"lastName"`
	CompanyName   string `json:"companyName"`
	Country       string `json:"country"`
	VatId         string `json:"vatId"`
	StreetAddress string `json:"streetAddress"`
	PostalCode    string `json:"postalCode"`
	Email         string `json:"email"`
	Code          string `json:"code"`
	Website       string `json:"website"`
}

// SendJSON utility helper
func (s *Server) SendJSON(w http.ResponseWriter, status int, success bool, message string, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(APIResponse{
		Success: success,
		Message: message,
		Data:    data,
	})
}

// EnableCORS middleware to allow all origins
func (s *Server) EnableCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Requested-With, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next(w, r)
	}
}

// validateCSRF checks the X-Requested-With header
func validateCSRF(r *http.Request) bool {
	return r.Header.Get("X-Requested-With") != ""
}

func generateCode() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(1000000))
	return fmt.Sprintf("%06d", n)
}

// isValidEmail checks if the email address provided has a valid format
func isValidEmail(email string) bool {
	re := regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}$`)
	return re.MatchString(strings.ToLower(email))
}

// generateRegistrationID creates a human-readable but unguessable ID in the format YYYYMMDD-{8-digit}
func generateRegistrationID() string {
	dateStr := time.Now().Format("20060102")
	n, _ := rand.Int(rand.Reader, big.NewInt(100000000)) // 8 digits
	return fmt.Sprintf("%s-%08d", dateStr, n)
}

func (s *Server) HandleValidateEmail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !validateCSRF(r) {
		s.SendJSON(w, http.StatusForbidden, false, "Security check failed: missing CSRF header", nil)
		return
	}

	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.SendJSON(w, http.StatusBadRequest, false, "Invalid request body", nil)
		return
	}

	if req.Email == "" || !isValidEmail(req.Email) {
		s.SendJSON(w, http.StatusBadRequest, false, "A valid email is required", nil)
		return
	}

	// Rate limiting
	if !s.RegisterEmailAttempt(req.Email) {
		s.SendJSON(w, http.StatusTooManyRequests, false, "Too many requests. Please wait a few minutes.", nil)
		return
	}

	// Generate and store code
	code := generateCode()
	s.StoreVerificationCode(req.Email, code)

	// Send the code to the user
	go func() {
		if err := s.Mail.SendVerificationCode(req.Email, code); err != nil {
			slog.Error("❌ Error sending verification code", "error", err)
		}
	}()

	s.SendJSON(w, http.StatusOK, true, "Validation code sent to your email", map[string]string{"code": code})
}

func (s *Server) HandleVerifyCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !validateCSRF(r) {
		s.SendJSON(w, http.StatusForbidden, false, "Security check failed: missing CSRF header", nil)
		return
	}

	var req struct {
		Email string `json:"email"`
		Code  string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.SendJSON(w, http.StatusBadRequest, false, "Invalid request body", nil)
		return
	}

	if !s.VerifyCode(req.Email, req.Code) {
		s.SendJSON(w, http.StatusBadRequest, false, "Invalid verification code", nil)
		return
	}

	s.SendJSON(w, http.StatusOK, true, "Email verified successfully", nil)
}

// HandleHealth returns a simple 200 OK response for health checks
func (s *Server) HandleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.SendJSON(w, http.StatusOK, true, "OK", nil)
}

func (s *RegistrationRequest) Validate() error {
	if s.FirstName == "" {
		return fmt.Errorf("first name is required")
	}
	if s.LastName == "" {
		return fmt.Errorf("last name is required")
	}
	if s.CompanyName == "" {
		return fmt.Errorf("company name is required")
	}
	if s.Country == "" {
		return fmt.Errorf("country is required")
	}
	if !common.IsValidCountry(s.Country) {
		return fmt.Errorf("invalid country code")
	}
	if s.VatId == "" {
		return fmt.Errorf("VAT ID is required")
	}
	if s.StreetAddress == "" {
		return fmt.Errorf("street address is required")
	}
	if s.PostalCode == "" {
		return fmt.Errorf("postal code is required")
	}
	if s.Email == "" {
		return fmt.Errorf("email is required")
	}
	if !isValidEmail(s.Email) {
		return fmt.Errorf("invalid email address format")
	}
	if s.Code == "" {
		return fmt.Errorf("verification code is required")
	}
	return nil
}

// HandleRegister handles the registration process
// It validates the request data, generates a registration ID, and sends an email to the user
func (s *Server) HandleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !validateCSRF(r) {
		s.SendJSON(w, http.StatusForbidden, false, "Security check failed: missing CSRF header", nil)
		return
	}

	var requestData RegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		s.SendJSON(w, http.StatusBadRequest, false, "Invalid request body", nil)
		return
	}

	if requestData.Website != "" {
		slog.Info("🤖 Bot detected via honeypot field")
		s.SendJSON(w, http.StatusOK, true, "Registration successful", nil)
		return
	}

	if err := requestData.Validate(); err != nil {
		s.SendJSON(w, http.StatusBadRequest, false, err.Error(), nil)
		return
	}

	// Verify the code again to prevent spurious calls
	if !s.VerifyCode(requestData.Email, requestData.Code) {
		s.SendJSON(w, http.StatusBadRequest, false, "Invalid or expired verification code", nil)
		return
	}
	s.DeleteVerificationCode(requestData.Email)

	slog.Info("Attempting to issue credential for registration", "email", requestData.Email, "vatID", requestData.VatId)

	cred := &credissuance.LEARIssuanceRequestBody{
		Schema:        "LEARCredentialEmployee",
		OperationMode: "S",
		Format:        "jwt_vc_json",
		Payload: credissuance.Payload{
			Mandator: credissuance.Mandator{
				OrganizationIdentifier: requestData.Country + "-" + requestData.VatId,
				Organization:           requestData.CompanyName,
				Country:                requestData.Country,
				CommonName:             requestData.FirstName + " " + requestData.LastName,
				EmailAddress:           requestData.Email,
			},
			Mandatee: credissuance.Mandatee{
				FirstName:   requestData.FirstName,
				LastName:    requestData.LastName,
				Nationality: requestData.Country,
				Email:       requestData.Email,
			},
			Power: []credissuance.Power{
				{
					Type:     "domain",
					Domain:   "DOME",
					Function: "Onboarding",
					Action:   credissuance.Strings{"execute", "verify"},
				},
			},
		},
	}

	regID := generateRegistrationID()
	reg := &db.RegistrationRecord{
		RegistrationID: regID,
		Email:          requestData.Email,
		FirstName:      requestData.FirstName,
		LastName:       requestData.LastName,
		CompanyName:    requestData.CompanyName,
		Country:        requestData.Country,
		VatID:          requestData.VatId,
		StreetAddress:  requestData.StreetAddress,
		PostalCode:     requestData.PostalCode,
	}

	// Create an initial registration in the database, updated with error and status later
	if err := s.DB.SaveRegistration(reg); err != nil {
		slog.Error("❌ Error saving initial registration", "error", err)
		s.SendJSON(w, http.StatusInternalServerError, false, "Failed to save registration", err.Error())
		return
	}

	reg.IssuanceAt = time.Now()
	token, err := s.Issuer.GetAccessToken()
	if err != nil {
		err = errl.Errorf("Failed to get access token for credential issuance: %v", err)
		slog.Error("❌ Error getting access token", "error", err)
		reg.IssuanceError = err.Error()
		if updateErr := s.DB.UpdateRegistrationStatus(reg); updateErr != nil {
			slog.Error("❌ Error updating registration status with access token error", "error", updateErr)
		}
		s.SendJSON(w, http.StatusInternalServerError, false, "Failed to get access token for credential issuance", err.Error())
		return

	}

	_, issError := s.Issuer.LEARIssuanceRequest(token, cred)
	if issError != nil {
		// There was an error, update the register and send an email informing of the error

		slog.Error("❌ Error calling issuance service", "error", issError)
		reg.IssuanceError = issError.Error()
		if updateErr := s.DB.UpdateRegistrationStatus(reg); updateErr != nil {
			slog.Error("❌ Error updating registration status with issuance error", "error", updateErr)
		}

		// Format the information we sent to the Issuer
		buf, err := json.MarshalIndent(cred, "", "  ")
		if err != nil {
			slog.Error("❌ Error marshalling credential data", "error", err)
		}

		// Send an email informing of the error, including the information that we wanted to issue
		err = s.Mail.SendIssuerError(reg, string(buf), reg.IssuanceError)
		if err != nil {
			slog.Error("❌ Error sending issuer error email", "error", err)
		}

		// Send a welcome email to the user, as if no error happened
		err = s.Mail.SendWelcomeEmail(reg)
		if err != nil {
			slog.Error("❌ Error sending welcome email", "error", err)
			reg.NotifEmailError = err.Error()
		} else {
			slog.Info("📧 Welcome email sent", "email", reg.Email)
			reg.NotifEmailAt = time.Now()
			reg.NotifEmailError = ""
		}
		if updateErr := s.DB.UpdateRegistrationStatus(reg); updateErr != nil {
			slog.Error("❌ Error updating registration status with email result", "error", updateErr)
		}

		s.SendJSON(w, http.StatusOK, true, "Registration successful", nil)
		return
	}

	// Issuance correct, update the register and send an email informing of the success
	reg.IssuanceError = ""
	if err := s.DB.UpdateRegistrationStatus(reg); err != nil {
		slog.Error("❌ Error updating registration status with issuance success", "error", err)
	}

	err = s.Mail.SendWelcomeEmail(reg)
	if err != nil {
		slog.Error("❌ Error sending welcome email", "error", err)
		reg.NotifEmailError = err.Error()
	} else {
		slog.Info("📧 Welcome email sent", "email", reg.Email)
		reg.NotifEmailAt = time.Now()
		reg.NotifEmailError = ""
	}
	if updateErr := s.DB.UpdateRegistrationStatus(reg); updateErr != nil {
		slog.Error("❌ Error updating registration status with email result", "error", updateErr)
	}

	// Create the object in the TM Forum server
	orgForm := credissuance.RegistrationRequest{
		CompanyName:   requestData.CompanyName,
		FirstName:     requestData.FirstName,
		LastName:      requestData.LastName,
		Email:         requestData.Email,
		Country:       requestData.Country,
		VatId:         requestData.VatId,
		StreetAddress: requestData.StreetAddress,
		PostalCode:    requestData.PostalCode,
	}
	newOrg := credissuance.OrganizationFromRequest(orgForm)

	_, err = s.Issuer.CreateOrganization(token, newOrg)
	if err != nil {
		err = errl.Errorf("Failed to create organization for registration: %v", err)
		slog.Error("❌ Error creating organization", "error", err)
		reg.IssuanceError = err.Error()
		if updateErr := s.DB.UpdateRegistrationStatus(reg); updateErr != nil {
			slog.Error("❌ Error updating registration status with organization creation error", "error", updateErr)
		}
		s.SendJSON(w, http.StatusInternalServerError, false, "Failed to create organization for registration", err.Error())
		return
	}

	s.SendJSON(w, http.StatusOK, true, "Registration successful", nil)
}
