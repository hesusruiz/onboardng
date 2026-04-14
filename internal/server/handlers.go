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

	"github.com/hesusruiz/onboardng/credissuance"
	"github.com/hesusruiz/onboardng/internal/configuration"
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
	City          string `json:"city"`
	PostalCode    string `json:"postalCode"`
	Email         string `json:"email"`
	Code          string `json:"code"`
	Website       string `json:"website"`
	Role          string `json:"role"`
}

// SendJSON utility helper
func (s *Server) SendJSON(w http.ResponseWriter, r *http.Request, status int, success bool, message string, data any) {
	slog.Info("Exit", "method", r.Method, "url", r.URL.Path, "status", status)
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

func (s *Server) HandleSendEmailValidationCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !validateCSRF(r) {
		s.SendJSON(w, r, http.StatusForbidden, false, "Security check failed: missing CSRF header", nil)
		return
	}

	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.SendJSON(w, r, http.StatusBadRequest, false, "Invalid request body", nil)
		return
	}

	if req.Email == "" || !isValidEmailFormat(req.Email) {
		s.SendJSON(w, r, http.StatusBadRequest, false, "A valid email is required", nil)
		return
	}

	// Rate limiting
	if !s.RegisterEmailAttempt(req.Email) {
		s.SendJSON(w, r, http.StatusTooManyRequests, false, "Too many requests. Please wait a few minutes.", nil)
		return
	}

	// Generate and store code
	code := generateCode()
	s.StoreVerificationCode(req.Email, code)

	// Send the code to the user in a separate goroutine
	go func() {
		if err := s.Mail.SendVerificationCode(req.Email, code); err != nil {
			slog.Error("❌ Error sending verification code", "error", err)
		}
	}()

	s.SendJSON(w, r, http.StatusOK, true, "Validation code sent to your email", map[string]string{"code": code})
}

func (s *Server) HandleValidateEmailCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !validateCSRF(r) {
		s.SendJSON(w, r, http.StatusForbidden, false, "Security check failed: missing CSRF header", nil)
		return
	}

	var req struct {
		Email string `json:"email"`
		Code  string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.SendJSON(w, r, http.StatusBadRequest, false, "Invalid request body", nil)
		return
	}

	if !s.VerifyCode(req.Email, req.Code) {
		s.SendJSON(w, r, http.StatusBadRequest, false, "Invalid verification code", nil)
		return
	}

	// Check if there is a registratrion for this email
	regRecord, err := s.DB.GetRegistrationByEmail(req.Email)
	if err != nil {
		// Just send a successful response to avoid leaking information
		s.SendJSON(w, r, http.StatusOK, true, "Email verified successfully", nil)
		return
	}

	// Send back a successful message including the registration data
	// Convert the registration record to a registratrion request
	regRequest := RegistrationRequest{
		FirstName:     regRecord.FirstName,
		LastName:      regRecord.LastName,
		CompanyName:   regRecord.CompanyName,
		Country:       regRecord.Country,
		VatId:         regRecord.VatID,
		StreetAddress: regRecord.StreetAddress,
		City:          regRecord.City,
		PostalCode:    regRecord.PostalCode,
		Email:         regRecord.Email,
	}

	s.SendJSON(w, r, http.StatusOK, true, "Email verified successfully", regRequest)
}

// HandleHealth returns a simple 200 OK response for health checks
func (s *Server) HandleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.SendJSON(w, r, http.StatusOK, true, "OK", nil)
}

func (s *Server) HandleOrgStatus(w http.ResponseWriter, r *http.Request) {
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

	resp := struct {
		Role     string `json:"role"`
		Approved int    `json:"approved"`
	}{
		Role:     reg.Role,
		Approved: reg.Approved,
	}

	s.SendJSON(w, r, http.StatusOK, true, "Registration fetched successfully", resp)
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
	if s.VatId == "" {
		return fmt.Errorf("VAT ID is required")
	}
	if s.StreetAddress == "" {
		return fmt.Errorf("street address is required")
	}
	if s.City == "" {
		return fmt.Errorf("city is required")
	}
	if s.PostalCode == "" {
		return fmt.Errorf("postal code is required")
	}
	if s.Email == "" {
		return fmt.Errorf("email is required")
	}
	if !isValidEmailFormat(s.Email) {
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

	// Perform all validations on input data
	//

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !validateCSRF(r) {
		s.SendJSON(w, r, http.StatusForbidden, false, "Security check failed: missing CSRF header", nil)
		return
	}

	var requestData RegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		s.SendJSON(w, r, http.StatusBadRequest, false, "Invalid request body", nil)
		return
	}

	if requestData.Website != "" {
		slog.Info("🤖 Bot detected via honeypot field")
		s.SendJSON(w, r, http.StatusOK, true, "Registration successful", nil)
		return
	}

	// TODO: this is temporal
	if requestData.City == "" {
		requestData.City = "Las Rozas de Madrid"
	}

	if err := requestData.Validate(); err != nil {
		s.SendJSON(w, r, http.StatusBadRequest, false, err.Error(), nil)
		return
	}

	if requestData.Role == "" {
		requestData.Role = "Buyer"
	}

	// Verify the code again to prevent spurious calls
	if !s.VerifyCode(requestData.Email, requestData.Code) {
		s.SendJSON(w, r, http.StatusBadRequest, false, "Invalid or expired verification code", nil)
		return
	}

	slog.Info("Attempting to issue credential for registration", "email", requestData.Email, "vatID", requestData.VatId)

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
		slog.Error("❌ Error getting access token", "error", err)
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
						slog.Error("❌ Error deleting organization", "error", err)
					}
					slog.Info("Existing organization deleted from TM Forum server", "orgId", org.ID)
				}

			}
		}
	}

	// The first step is to save to the database the data provided by the user
	// This is a draft registration, subject to some verifications
	reg, err := saveToDB(requestData, s)
	if err != nil {
		// For any other error, registration stops with failure
		err = errl.Errorf("Failed to save registration: %w", err)
		slog.Error("❌ Error saving registration", "error", err)
		s.SendJSON(w, r, http.StatusInternalServerError, false, "Error saving to database", err.Error())
		return
	}

	// Once the data is saved, we continue with the registration process.
	// Even if we faind erros, we always return to the user with a success and a welcome message,
	// so they know the process is in motion.
	// The rest of the remaining steps, if any, will be performed manually by the internal team.

	// Send a welcome email to the user, with copy to the relevant onboarding team, so they can followup on the onboarding process
	if err := s.Mail.SendWelcomeEmail(reg); err != nil {
		err = errl.Errorf("Failed to send welcome email: %w", err)
		slog.Error("❌ Error sending welcome email", "error", err)
		// Record the error in the database
		_ = s.DB.SaveRegistrationLog(&db.RegistrationLog{
			RegistrationID: reg.RegistrationID,
			Email:          reg.Email,
			VatID:          reg.VatID,
			Type:           "error",
			Message:        fmt.Sprintf("Welcome email error: %v", err.Error()),
		})
	} else {
		slog.Info("📧 Welcome email sent", "email", reg.Email)
		reg.Notified = true
		if updateErr := s.DB.UpdateRegistrationStatus(reg); updateErr != nil {
			slog.Error("❌ Error updating registration status with email result", "error", updateErr)
		}
	}

	// Perform the verifiable credential issuance
	err = performIssuance(r.Context(), token, reg, s, requestData)
	if err != nil {
		slog.Error("❌ Error performing issuance", "error", err)
	}

	if s.Features.TMFServerEnabled {
		if existingOrgID != "" {
			err = updateTMForumOrganization(r.Context(), token, existingOrgID, requestData, reg, s)
		} else {
			err = createTMForumOrganization(r.Context(), token, requestData, reg, s)
		}
		if err != nil {
			slog.Error("❌ Error updating TM Forum", "error", err)
			// Record the error in the database
			_ = s.DB.SaveRegistrationLog(&db.RegistrationLog{
				RegistrationID: reg.RegistrationID,
				Email:          reg.Email,
				VatID:          reg.VatID,
				Type:           "error",
				Message:        fmt.Sprintf("TM Forum update error: %v", err),
			})
		}
	}

	s.SendJSON(w, r, http.StatusOK, true, "Registration successful", nil)
}

// saveToDB converts a RegistrationRequest received from the API to a database RegistrationRecord and saves it to the database.
// It assumes that the request data has been validated
func saveToDB(requestData RegistrationRequest, s *Server) (*db.RegistrationRecord, error) {
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
			slog.Error("❌ Error saving initial registration", "error", err)
		}
		return nil, err
	}
	return reg, nil
}

func performIssuance(ctx context.Context, token string, reg *db.RegistrationRecord, s *Server, requestData RegistrationRequest) error {

	organizationIdentifierPrefix := "VAT" + requestData.Country
	organizationIdentifier := requestData.VatId
	if !strings.HasPrefix(requestData.VatId, organizationIdentifierPrefix) {
		organizationIdentifier = organizationIdentifierPrefix + "-" + requestData.VatId
	}

	// Create the struct needed by the Issuer API for a credential for the self-registration
	soloCredential := &credissuance.LEARIssuanceRequestBody{
		Schema:        "LEARCredentialEmployee",
		OperationMode: "S",
		Format:        "jwt_vc_json",
		Payload: credissuance.Payload{
			Mandator: credissuance.Mandator{
				OrganizationIdentifier: organizationIdentifier,
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
					Action:   credissuance.Strings{"Execute"},
				},
				{
					Type:     "domain",
					Domain:   "DOME",
					Function: "ProductOffering",
					Action:   credissuance.Strings{"Create", "Update", "Delete"},
				},
			},
		},
	}

	if _, err := s.Issuer.LEARIssuanceRequest(ctx, token, soloCredential); err != nil {
		slog.Error("❌ Error calling issuance service", "error", err)

		// Record the error in the database
		_ = s.DB.SaveRegistrationLog(&db.RegistrationLog{
			RegistrationID: reg.RegistrationID,
			Email:          reg.Email,
			VatID:          reg.VatID,
			Type:           "error",
			Message:        fmt.Sprintf("Issuance error (in performIssuance): %v", err),
		})

		// Format the information we sent to the Issuer
		buf, errformat := json.MarshalIndent(soloCredential, "", "  ")
		if errformat != nil {
			slog.Error("❌ Error marshalling credential data", "error", errformat)
			buf = []byte("Error marshalling credential data")
		}

		// Send an email to the internal team informing of the error, including the information that we wanted to issue
		// They can perform the rest of the registration steps manually, transparently to the user.

		if err := s.Mail.SendIssuerError(reg, string(buf), err.Error()); err != nil {
			slog.Error("❌ Error sending issuer error email", "error", err)
		}

		return err
	}

	// If the credential issuance was ok, update the database and send an email informing of the success
	reg.Issued = true
	if err := s.DB.UpdateRegistrationStatus(reg); err != nil {
		slog.Error("❌ Error updating registration status with issuance success", "error", err)
	}
	return nil
}

func createTMForumOrganization(ctx context.Context, token string, requestData RegistrationRequest, reg *db.RegistrationRecord, s *Server) error {

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
	newOrg := credissuance.TMFOrganizationFromRequest(orgForm)

	_, err := s.Issuer.TMFCreateOrganization(ctx, token, newOrg)
	if err != nil {
		err = errl.Errorf("Failed to create organization for registration: %v", err)
		slog.Error("❌ Error creating organization", "error", err)
		// Record the error in the database
		_ = s.DB.SaveRegistrationLog(&db.RegistrationLog{
			RegistrationID: reg.RegistrationID,
			Type:           "error",
			Message:        fmt.Sprintf("TM Forum create organization error: %v", err),
		})
		return err
	}
	reg.TMFRegistered = true
	if err := s.DB.UpdateRegistrationStatus(reg); err != nil {
		slog.Error("❌ Error updating registration status with TMF success", "error", err)
	}
	return nil
}

func updateTMForumOrganization(ctx context.Context, token string, id string, requestData RegistrationRequest, reg *db.RegistrationRecord, s *Server) error {

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
	updatedOrg := credissuance.TMFOrganizationUpdateFromRequest(orgForm)

	_, err := s.Issuer.TMFUpdateOrganization(ctx, token, id, updatedOrg)
	if err != nil {
		err = errl.Errorf("Failed to update organization for registration: %v", err)
		slog.Error("❌ Error updating organization", "error", err)
		// Record the error in the database
		_ = s.DB.SaveRegistrationLog(&db.RegistrationLog{
			RegistrationID: reg.RegistrationID,
			Type:           "error",
			Message:        fmt.Sprintf("TM Forum update organization error: %v", err),
		})
		return err
	}
	reg.TMFRegistered = true
	if err := s.DB.UpdateRegistrationStatus(reg); err != nil {
		slog.Error("❌ Error updating registration status with TMF success", "error", err)
	}
	return nil
}

// RepresentativesRequest payload for updating LR and LEAR details
type RepresentativesRequest struct {
	VatID            string `json:"vatId"`
	Email            string `json:"email"`
	Code             string `json:"code"`
	LRFirstName      string `json:"lr_first_name"`
	LRLastName       string `json:"lr_last_name"`
	LREmail          string `json:"lr_email"`
	LRCountry        string `json:"lr_country"`
	LRIdCard         string `json:"lr_id_card"`
	LEARFirstName    string `json:"lear_first_name"`
	LEARLastName     string `json:"lear_last_name"`
	LEAREmail        string `json:"lear_email"`
	LEARCountry      string `json:"lear_country"`
	LEARAddress      string `json:"lear_address"`
	LEARIdCard       string `json:"lear_id_card"`
	LEARMobileNumber string `json:"lear_mobile_number"`
	LEARCompleted    bool   `json:"lear_completed"`
}

// HandleUpdateRepresentatives handles updating the LR and LEAR info
func (s *Server) HandleUpdateRepresentatives(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !validateCSRF(r) {
		s.SendJSON(w, r, http.StatusForbidden, false, "Security check failed: missing CSRF header", nil)
		return
	}

	var req RepresentativesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.SendJSON(w, r, http.StatusBadRequest, false, "Invalid request body", nil)
		return
	}
	out, _ := json.MarshalIndent(req, "", "  ")
	slog.Info("Representatives request", "request", string(out))

	if req.VatID == "" {
		s.SendJSON(w, r, http.StatusBadRequest, false, "VAT ID is required", nil)
		return
	}
	if req.Email == "" {
		s.SendJSON(w, r, http.StatusBadRequest, false, "Email is required", nil)
		return
	}
	if req.Code == "" {
		s.SendJSON(w, r, http.StatusBadRequest, false, "Verification code is required", nil)
		return
	}

	// Verify the code
	if !s.VerifyCode(req.Email, req.Code) {
		s.SendJSON(w, r, http.StatusBadRequest, false, "Invalid or expired verification code", nil)
		return
	}

	// Delete verification code after successful use. In any case, it will expire in 1 hour.
	s.DeleteVerificationCode(req.Email)

	// Fetch registration and check if immutable
	current, err := s.DB.GetRegistrationByVatID(req.VatID)
	if err != nil {
		s.SendJSON(w, r, http.StatusNotFound, false, "Registration not found", nil)
		return
	}
	slog.Info("Registration found", "registration", current)

	// Check if the email matches the registration email
	if current.Email != strings.ToLower(req.Email) {
		s.SendJSON(w, r, http.StatusForbidden, false, "Email does not match the registration", nil)
		return
	}
	if current.Approved > 0 {
		s.SendJSON(w, r, http.StatusForbidden, false, "Registration is already approved or denied and immutable", nil)
		return
	}

	// Update DB
	repParams := &db.RegistrationRecord{
		LRFirstName:      req.LRFirstName,
		LRLastName:       req.LRLastName,
		LREmail:          req.LREmail,
		LRCountry:        req.LRCountry,
		LRIdCard:         req.LRIdCard,
		LEARFirstName:    req.LEARFirstName,
		LEARLastName:     req.LEARLastName,
		LEAREmail:        req.LEAREmail,
		LEARCountry:      req.LEARCountry,
		LEARAddress:      req.LEARAddress,
		LEARIdCard:       req.LEARIdCard,
		LEARMobileNumber: req.LEARMobileNumber,
		LEARCompleted:    true,
		Role:             "Seller",
	}

	err = s.DB.UpdateRepresentativesByVatID(req.VatID, repParams)
	if err != nil {
		slog.Error("❌ Error updating representatives", "error", err)
		s.SendJSON(w, r, http.StatusInternalServerError, false, "Error updating registration", err.Error())
		return
	}

	s.SendJSON(w, r, http.StatusOK, true, "Representatives updated successfully", nil)
}
