package server

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/hesusruiz/onboardng/common"
	"github.com/hesusruiz/onboardng/internal/configuration"
	"github.com/hesusruiz/onboardng/internal/db"
	"github.com/hesusruiz/onboardng/internal/vies"
	"github.com/hesusruiz/utils/errl"
)

// HandleSendEmailValidationCode is the first step in the registration process.
// It receives an email address in the body of the request and sends a verification code
// to that email address.
// It does not check if the email address is already registered, to avoid email enumeration.
// It uses the rate limiter (RegisterEmailAttempt) to prevent too many requests using the same email address.
// The code will be received and validated by the HandleValidateEmailCode handler.
func (s *Server) HandleSendEmailValidationCode(w http.ResponseWriter, r *http.Request) {

	var req struct {
		Email  string `json:"email"`
		Resend bool   `json:"resend"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.SendJSON(w, r, http.StatusBadRequest, false, "Invalid request body", err.Error())
		return
	}

	if req.Email == "" || !isValidEmailFormat(req.Email) {
		s.SendJSON(w, r, http.StatusBadRequest, false, "A valid email is required", nil)
		return
	}

	// If not an explicit resend request, reuse the code in memory if the email was already verified
	if !req.Resend {
		if code, ok := s.GetVerifiedCode(req.Email); ok {
			slog.Info("Email already verified, reusing code", "email", req.Email)
			s.SendJSON(w, r, http.StatusOK, true, "Email already verified", map[string]any{
				"code":            code,
				"alreadyVerified": true,
			})
			return
		}
	}

	// Rate limiting
	if !s.RegisterEmailAttempt(req.Email) {
		s.SendJSON(w, r, http.StatusTooManyRequests, false, "Too many requests. Please wait a few minutes.", nil)
		return
	}

	// Generate and store code
	code := generateCode()
	s.StoreVerificationCode(req.Email, code)

	slog.Info("Sending validation code to email", "email", req.Email, "code", code)

	// Send the code to the user via email
	if err := s.Mail.SendVerificationCodeEmail(req.Email, code); err != nil {
		s.SendJSON(w, r, http.StatusInternalServerError, false, "Error sending verification code", err.Error())
		return
	}

	s.SendJSON(w, r, http.StatusOK, true, "Validation code sent to your email", map[string]any{
		"code":            code,
		"alreadyVerified": false,
	})
}

// HandleValidateEmailCode is the second step in the registration process.
// It receives an email address and a code in the body of the request and validates them.
// The received code must be the same as the one sent by the HandleSendEmailValidationCode handler.
// It then checks if the email address is already registered.
// - If the email address is registered, it returns the registration data.
// - If the email address is not registered, it just returns OK with empty registration data.
func (s *Server) HandleValidateEmailCode(w http.ResponseWriter, r *http.Request) {

	var req struct {
		Email string `json:"email"`
		Code  string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.SendJSON(w, r, http.StatusBadRequest, false, "Invalid request body", err.Error())
		return
	}

	if !s.VerifyCode(req.Email, req.Code) {
		s.SendJSON(w, r, http.StatusBadRequest, false, "Invalid verification code", nil)
		return
	}

	// Check if there is a registration for this email
	regRecord, err := s.DB.GetRegistrationByEmail(req.Email)
	if err != nil {
		// Just send a successful response to avoid leaking information
		s.SendJSON(w, r, http.StatusOK, true, "Email verified successfully", err.Error())
		return
	}

	// Strip the prefix "VATxx-" from the VAT ID if it exists, to display a clean VAT ID to the user in the frontend.
	// The full VAT ID will be reconstructed when received in the registration request
	vatPrefix := "VAT" + regRecord.Country + "-"
	vatID := strings.TrimPrefix(regRecord.VatID, vatPrefix)

	// Send back a successful message including the registration data, converting the registration record to a registration request
	regRequest := RegistrationRequest{
		FirstName:     regRecord.FirstName,
		LastName:      regRecord.LastName,
		CompanyName:   regRecord.CompanyName,
		Country:       regRecord.Country,
		VatId:         vatID,
		StreetAddress: regRecord.StreetAddress,
		City:          regRecord.City,
		PostalCode:    regRecord.PostalCode,
		Email:         regRecord.Email,
	}

	s.SendJSON(w, r, http.StatusOK, true, "Email verified successfully", regRequest)
}

// The frontend sends this data when completing the first step of the registration process.
type RegistrationRequest struct {
	FirstName     string `json:"firstName"`
	LastName      string `json:"lastName"`
	CompanyName   string `json:"companyName"`
	Country       string `json:"country"`
	VatId         string `json:"vatId"`
	StreetAddress string `json:"streetAddress"`
	City          string `json:"city"`
	PostalCode    string `json:"postalCode"`
	Email         string `json:"email"`
	Code          string `json:"code"`
	Website       string `json:"website"`
	Role          string `json:"role"`
}

// HandleRegister is the third and last step in the first phase of the registration process.
// It receives registration data, validates it and stores it in the database.
// Then it proceeds to the automatic processes:
// - Send a welcome email to the user
// - Send a notification email to the onboarding team
// - Perform the verifiable credential issuance
// - Update the TM Forum server with the organization data
// - Send a notification email to the onboarding team with the result of the registration
func (s *Server) HandleRegister(w http.ResponseWriter, r *http.Request) {
	var requestData RegistrationRequest

	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		s.SendJSON(w, r, http.StatusBadRequest, false, "Invalid request body", err.Error())
		return
	}

	if requestData.Website != "" {
		slog.Warn("Bot detected via honeypot field")
		s.SendJSON(w, r, http.StatusOK, true, "Registration successful", nil)
		return
	}

	// Verify the pair email/code again to prevent spurious calls
	if !s.VerifyCode(requestData.Email, requestData.Code) {
		s.SendJSON(w, r, http.StatusBadRequest, false, "Invalid or expired verification code", nil)
		return
	}

	if err := requestData.Validate(s.Runtime); err != nil {
		s.SendJSON(w, r, http.StatusBadRequest, false, err.Error(), err.Error())
		return
	}

	slog.Info("Attempting registration", "email", requestData.Email, "vatID", requestData.VatId)

	// Check if there is a registration for this email or VAT ID
	regRecord, err := s.DB.GetRegistrationByEmailOrVatID(requestData.Email, requestData.VatId)

	// There are several possibilities:
	// 1. If new email == old email AND new vatID != old vatID, reject with message "email already registered"
	// 2. If new vatID == old vatID AND new email != old email, reject with message "company already registered"
	// 3. If new email == old email AND new vatID == old vatID, continue with the request
	// 4. If new email != old email AND new vatID != old vatID, continue with the request

	if err == nil {
		if regRecord.Email == requestData.Email && regRecord.VatID != requestData.VatId {
			s.SendJSON(w, r, http.StatusConflict, false, "Email already registered", nil)
			return
		}
		if regRecord.VatID == requestData.VatId && regRecord.Email != requestData.Email {
			s.SendJSON(w, r, http.StatusConflict, false, "Company already registered", nil)
			return
		}
	}

	// Get an access token to authenticate in the Issuer and TM Forum APIs
	token, err := s.Issuer.GetAccessToken(r.Context())
	if err != nil {
		err = errl.Errorf("Failed to get access token for credential issuance: %v", err)
		s.SendJSON(w, r, http.StatusInternalServerError, false, "Error getting access token", err.Error())
		return
	}

	// Check in the TMF server if the organization already exists.
	// In PRO, we use the first organization found and continue with the registration.
	// In other environments, we continue with the registration, deleting all orgs except the first.
	var existingOrgID string
	if s.Features.TMFServerEnabled {
		existingOrgs, _ := s.Issuer.TMFGetOrganizationByELSI(r.Context(), token, requestData.VatId)
		if len(existingOrgs) > 0 {
			slog.Info("Organization already exists in TMF server", "vatId", requestData.VatId)

			switch s.Runtime {

			case configuration.Production:
				slog.Info("Using the first organization found and continuing", "environment", s.Runtime)
				existingOrgID = existingOrgs[0].ID

			case configuration.Development, configuration.Preproduction:
				slog.Info("Deleting all organizations except the first and continuing", "environment", s.Runtime)
				existingOrgID = existingOrgs[0].ID
				// Delete all organizations except the first one
				for i := 1; i < len(existingOrgs); i++ {
					org := existingOrgs[i]
					if err := s.Issuer.TMFDeleteOrganization(r.Context(), token, org.ID); err != nil {
						err = errl.Errorf("Failed to delete organization for registration: %v", err)
						slog.Error("Error deleting organization", "error", err)
					}
					slog.Info("Existing organization deleted from TM Forum server", "orgId", org.ID)
				}

			}
		}
	}
	_ = existingOrgID

	// Before doing anything else, store the data provided by the user in the database
	// This is a draft registration, subject to some verifications
	reg, err := saveRegistration(requestData, s)
	if err != nil {
		err = errl.Errorf("Failed to save registration: %w", err)
		s.SendJSON(w, r, http.StatusInternalServerError, false, "Error saving to database", err.Error())
		return
	}

	// Once the data is saved, we continue with the registration process.
	// Even if we find erros, we always return to the user with a success and a welcome message,
	// so they know the process is in motion. The internal team will handle the errors to complete the process.
	// The rest of the remaining steps, if any, will be performed manually by the internal team.

	// Send a welcome email to the user, with copy to the relevant onboarding team, so they can followup on the onboarding process
	if err := s.Mail.SendFirstPhaseEmail(reg); err != nil {
		err = errl.Errorf("Failed to send welcome email: %w", err)
		slog.Error("Error sending welcome email", "error", err)
		// Record the error in the database
		_ = s.DB.LogRegError(reg.VatID, fmt.Sprintf("Welcome email error: %v", err.Error()), reg.RegistrationID, reg.Email)
	} else {
		slog.Info("Welcome email sent", "email", reg.Email)
		_ = s.DB.LogRegInfo(reg.VatID, "Welcome email sent", reg.RegistrationID, reg.Email)
		reg.Notified = true
		if updateErr := s.DB.UpdateRegistrationStatus(reg); updateErr != nil {
			slog.Error("Error updating registration status with email result", "error", updateErr)
			_ = s.DB.LogRegError(reg.VatID, fmt.Sprintf("Update registration status error: %v", updateErr.Error()), reg.RegistrationID, reg.Email)
		}
	}

	if s.Features.DirectIssuance {
		slog.Info("Direct issuance enabled")
		// Perform the verifiable credential issuance
		redirectUri, err := performIssuanceDirect(r.Context(), token, reg, s, requestData)
		if err != nil {
			slog.Error("Error performing issuance", "error", errl.Error(err))
		}
		// Instruct the client to perform the redirect
		s.SendJSON(w, r, http.StatusOK, true, "Redirecting...", redirectUri)

	} else {
		slog.Info("Direct issuance not enabled, using email delivery")
		// Perform the verifiable credential issuance
		err = performIssuanceEmail(r.Context(), token, reg, s, requestData)
		if err != nil {
			slog.Error("Error performing issuance", "error", errl.Error(err))
		}
		s.SendJSON(w, r, http.StatusOK, true, "Registration successful", nil)
	}

}

func (r *RegistrationRequest) Validate(runtime configuration.RuntimeEnv) error {
	if r.FirstName == "" {
		return errl.Errorf("first name is required")
	}
	if r.LastName == "" {
		return errl.Errorf("last name is required")
	}
	if r.CompanyName == "" {
		return errl.Errorf("company name is required")
	}

	// Check that Role is either "Buyer" or "Seller"
	if r.Role != "Buyer" && r.Role != "Seller" {
		return errl.Errorf("invalid role")
	}

	if r.Country == "" {
		return errl.Errorf("country is required")
	}

	// Convert to uppercase to avoid case sensitivity issues
	r.Country = strings.ToUpper(r.Country)

	if r.StreetAddress == "" {
		return errl.Errorf("street address is required")
	}
	if r.City == "" {
		return errl.Errorf("city is required")
	}
	if r.PostalCode == "" {
		return errl.Errorf("postal code is required")
	}
	if r.Email == "" {
		return errl.Errorf("email is required")
	}
	if !isValidEmailFormat(r.Email) {
		return errl.Errorf("invalid email address format")
	}
	if r.Code == "" {
		return errl.Errorf("verification code is required")
	}

	if r.VatId == "" {
		return errl.Errorf("VAT ID is required")
	}

	// The prefix must be "VAT" followed by the country code and a dash
	expectedPrefix := "VAT" + r.Country + "-"

	// Handle non-EU countries
	if !common.IsValidEUCountry(r.Country) {
		if !strings.HasPrefix(r.VatId, expectedPrefix) {
			// Add the expected prefix to the VAT ID
			r.VatId = expectedPrefix + r.VatId
		}
		return nil
	}

	// If country is an EU country, then the VAT ID must comply with the ETSI EN 319 412-1 format for organizationIdentifier
	// That is, it must start with "VAT" followed by the country code and a dash, e.g. "VATES-12345678"

	vatNumber, err := VerifyFormalEUVATNumber(r.VatId, r.Country)
	if err != nil {
		return err
	}

	// Do not perform VIES validation in development, but log for developers
	if runtime == configuration.Development {
		slog.Warn("VIES in Development", "country", r.Country, "VAT ID", r.VatId)
		// Add the expected prefix to the VAT number
		r.VatId = expectedPrefix + vatNumber
		return nil
	}

	// Check in the VIES database if the VAT number is valid for the country provided
	viesResp, err := vies.CheckVAT(context.Background(), true, r.Country, vatNumber)
	if err != nil {
		return errl.Errorf("error checking VAT ID %s with VIES API: %w", r.VatId, err)
	}
	if !viesResp.Valid {
		return errl.Errorf("invalid VAT ID %s for country %s", r.VatId, r.Country)
	}

	// Add the expected prefix to the VAT number
	r.VatId = expectedPrefix + vatNumber

	return nil
}

// VerifyFormalEUVATNumber verifies vatNumber for a EU country
// The vatNumber is the last part of a VATID, which has the format VAT{country}-{vatNumber} where country is the 2-digit country code
// This function is intended for verifying the vatNumber introduced by a user in a form
func VerifyFormalEUVATNumber(vatNumber string, country string) (string, error) {
	// Trim blanks at the beginning and end
	vatNumber = strings.TrimSpace(vatNumber)
	// Convert to uppercase to avoid case sensitivity issues
	vatNumber = strings.ToUpper(vatNumber)

	// The prefix must be "VAT" followed by the country code and a dash
	expectedPrefix := "VAT" + country + "-"

	if strings.HasPrefix(vatNumber, expectedPrefix) {
		return "", fmt.Errorf("enter the VAT number without the prefix %s", expectedPrefix)
	}

	if strings.HasPrefix(vatNumber, "VAT") {
		return "", fmt.Errorf("enter the VAT number without the prefix VAT")
	}

	return vatNumber, nil
}

// saveRegistration converts a RegistrationRequest received from the API to a database RegistrationRecord and saves it to the database.
// It assumes that the request data has been validated
func saveRegistration(requestData RegistrationRequest, s *Server) (*db.RegistrationRecord, error) {
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
		City:           requestData.City,
		PostalCode:     requestData.PostalCode,
		Role:           requestData.Role,
	}

	// Create an initial registration in the database, updated with error and status later
	if err := s.DB.SaveRegistration(reg); err != nil {
		if errors.Is(err, db.ErrorAlreadyExists) {
			slog.Error("Registration conflict", "error", err)
		} else {
			slog.Error("Error saving initial registration", "error", err)
		}
		return nil, errl.Error(err)
	}
	return reg, nil
}

func generateCode() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(1000000))
	return fmt.Sprintf("%06d", n)
}

// isValidEmailFormat checks if the email address provided has a valid format
func isValidEmailFormat(email string) bool {
	re := regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}$`)
	return re.MatchString(strings.ToLower(email))
}

// generateRegistrationID creates a human-readable but unguessable ID in the format YYYYMMDD-{8-digit}
func generateRegistrationID() string {
	dateStr := time.Now().Format("20060102")
	n, _ := rand.Int(rand.Reader, big.NewInt(100000000)) // 8 digits
	return fmt.Sprintf("%s-%08d", dateStr, n)
}
