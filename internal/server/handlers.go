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
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/hesusruiz/onboardng/common"
	"github.com/hesusruiz/onboardng/credissuance"
	"github.com/hesusruiz/onboardng/internal/configuration"
	"github.com/hesusruiz/onboardng/internal/db"
	"github.com/hesusruiz/onboardng/internal/vies"
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

	// Main API Routes
	mux.Handle("POST /api/validate-email", nonAdminChain(s.HandleSendEmailValidationCode))
	mux.Handle("POST /api/verify-code", nonAdminChain(s.HandleValidateEmailCode))
	mux.Handle("POST /api/register", nonAdminChain(s.HandleRegister))
	mux.Handle("POST /api/representatives", nonAdminChain(s.HandleUpdateRepresentatives))
	mux.Handle("GET /api/orgstatus", nonAdminChain(s.HandleOrgStatus))

	// The Health route does not log, to avoid polluting the logs
	mux.Handle("GET /health", s.EnableCORS(s.RateLimitIP(http.HandlerFunc(s.HandleHealth))))

	mux.HandleFunc("GET /qr", func(w http.ResponseWriter, r *http.Request) {

		// Get the credential_offer_uri from the parameter in the request URL
		credentialOfferURI := r.URL.Query().Get("credential_offer_uri")

		// Return error if credentialOfferURI is not provided
		if credentialOfferURI == "" {
			s.SendJSON(w, r, http.StatusBadRequest, false, "credential_offer_uri is required", nil)
			return
		}

		fmt.Printf("credentialOfferURI: %s\n", credentialOfferURI)

		//URLEncode the credentialOfferURI
		credentialOfferURI = url.QueryEscape(credentialOfferURI)

		// Add the prefix for the wallet credential issuance callback URL
		credentialOfferURI = s.WalletURL + "?credential_offer_uri=" + credentialOfferURI

		fmt.Printf("Displaying credentialOfferURI: %s\n", credentialOfferURI)

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

// HandleSendEmailValidationCode is the first step in the registration process.
// It receives an email address in the body of the request and sends a verification code
// to that email address.
// It does not check if the email address is already registered, to avoid email enumeration.
// It uses the rate limiter (RegisterEmailAttempt) to prevent too many requests using the same email address.
// The code will be received and validated by the HandleValidateEmailCode handler.
func (s *Server) HandleSendEmailValidationCode(w http.ResponseWriter, r *http.Request) {

	if !validateCSRF(r) {
		s.SendJSON(w, r, http.StatusForbidden, false, "Security check failed: missing CSRF header", nil)
		return
	}

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

	if !validateCSRF(r) {
		s.SendJSON(w, r, http.StatusForbidden, false, "Security check failed: missing CSRF header", nil)
		return
	}

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

	if !validateCSRF(r) {
		s.SendJSON(w, r, http.StatusForbidden, false, "Security check failed: missing CSRF header", nil)
		return
	}

	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		s.SendJSON(w, r, http.StatusBadRequest, false, "Invalid request body", err.Error())
		return
	}

	if requestData.Website != "" {
		slog.Warn("Bot detected via honeypot field")
		s.SendJSON(w, r, http.StatusOK, true, "Registration successful", nil)
		return
	}

	if err := requestData.Validate(s.Runtime); err != nil {
		s.SendJSON(w, r, http.StatusBadRequest, false, err.Error(), err.Error())
		return
	}

	// Verify the code again to prevent spurious calls
	if !s.VerifyCode(requestData.Email, requestData.Code) {
		s.SendJSON(w, r, http.StatusBadRequest, false, "Invalid or expired verification code", nil)
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
	reg, err := saveToDB(requestData, s)
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
		_ = s.DB.SaveRegistrationLog(&db.RegistrationLog{
			RegistrationID: reg.RegistrationID,
			Email:          reg.Email,
			VatID:          reg.VatID,
			Type:           "error",
			Message:        fmt.Sprintf("Welcome email error: %v", err.Error()),
		})
	} else {
		slog.Info("Welcome email sent", "email", reg.Email)
		reg.Notified = true
		if updateErr := s.DB.UpdateRegistrationStatus(reg); updateErr != nil {
			slog.Error("Error updating registration status with email result", "error", updateErr)
		}
	}

	if s.Features.DirectIssuance {
		slog.Info("Direct issuance enabled")
		// Perform the verifiable credential issuance
		redirectUri, err := performIssuanceNew(r.Context(), token, reg, s, requestData)
		if err != nil {
			slog.Error("Error performing issuance", "error", errl.Error(err))
		}
		// Instruct the client to perform the redirect
		s.SendJSON(w, r, http.StatusOK, true, "Redirecting...", redirectUri)

	} else {
		slog.Info("Direct issuance not enabled, using email delivery")
		// Perform the verifiable credential issuance
		err = performIssuance(r.Context(), token, reg, s, requestData)
		if err != nil {
			slog.Error("Error performing issuance", "error", errl.Error(err))
		}
		s.SendJSON(w, r, http.StatusOK, true, "Registration successful", nil)
	}

	// if s.Features.TMFServerEnabled {
	// 	if existingOrgID != "" {
	// 		err = updateTMForumOrganization(r.Context(), token, existingOrgID, requestData, reg, s)
	// 	} else {
	// 		err = createTMForumOrganization(r.Context(), token, requestData, reg, s)
	// 	}
	// 	if err != nil {
	// 		err = errl.Error(err)
	// 		slog.Error("Error updating TM Forum", "error", err)
	// 		// Record the error in the database
	// 		_ = s.DB.SaveRegistrationLog(&db.RegistrationLog{
	// 			RegistrationID: reg.RegistrationID,
	// 			Email:          reg.Email,
	// 			VatID:          reg.VatID,
	// 			Type:           "error",
	// 			Message:        fmt.Sprintf("TM Forum update error: %v", err),
	// 		})
	// 	}
	// }

	// s.SendJSON(w, r, http.StatusOK, true, "Registration successful", nil)
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
			slog.Error("Error saving initial registration", "error", err)
		}
		return nil, errl.Error(err)
	}
	return reg, nil
}

func performIssuanceNew(ctx context.Context, token string, reg *db.RegistrationRecord, s *Server, requestData RegistrationRequest) (string, error) {

	// Add the VATxx- prefix to the VAT ID if the user has not provided it
	organizationIdentifierPrefix := "VAT" + requestData.Country
	organizationIdentifier := requestData.VatId
	if !strings.HasPrefix(requestData.VatId, organizationIdentifierPrefix) {
		organizationIdentifier = organizationIdentifierPrefix + "-" + requestData.VatId
	}

	// Create the mandate data to incluse in the Verifiable Credential for a LEAR
	payload := credissuance.Payload{
		Mandator: credissuance.Mandator{
			OrganizationIdentifier: organizationIdentifier,
			Organization:           requestData.CompanyName,
			Country:                requestData.Country,
			CommonName:             requestData.FirstName + " " + requestData.LastName,
			Email:                  requestData.Email,
		},
		Mandatee: credissuance.Mandatee{
			FirstName: requestData.FirstName,
			LastName:  requestData.LastName,
			// Nationality: requestData.Country,
			Email: requestData.Email,
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
	}

	newCredential := &credissuance.LEARIssuanceRequestBodyNew{
		CredentialConfigurationID: "learcredential.employee.w3c.4",
		Delivery:                  "ui",
		Email:                     requestData.Email,
		Payload:                   payload,
	}

	issuanceResponse, err := s.Issuer.LEARIssuanceRequestNew(ctx, token, newCredential)
	if err != nil {
		errIssuance := errl.Errorf("Failed to perform issuance: %v", err)
		slog.Error("Error calling issuance service", "error", errl.Error(err))

		// Record the error in the database
		_ = s.DB.SaveRegistrationLog(&db.RegistrationLog{
			RegistrationID: reg.RegistrationID,
			Email:          reg.Email,
			VatID:          reg.VatID,
			Type:           "error",
			Message:        errIssuance.Error(),
		})

		// Format the information we sent to the Issuer
		buf, errformat := json.MarshalIndent(newCredential, "", "  ")
		if errformat != nil {
			slog.Error("Error marshalling credential data", "error", errl.Error(errformat))
			buf = []byte("Error marshalling credential data")
		}

		// Send an email to the internal team informing of the error, including the information that we wanted to issue
		// They can perform the rest of the registration steps manually, transparently to the user.

		if err := s.Mail.SendIssuerErrorEmail(reg, string(buf), errIssuance.Error()); err != nil {
			slog.Error("Error sending issuer error email", "error", errl.Error(err))
		}

		return "", errIssuance
	}

	// Get the credential_offer_uri from the issuance response
	credentialOfferURI := issuanceResponse.Responses[0].Body.CredentialOfferURI

	// Strip the prefix, including the "?" character, from the credential offer URI, to get only the query parameters
	walletCallbackPrefix := s.WalletURL + "?"
	credentialOfferURI = strings.TrimPrefix(credentialOfferURI, walletCallbackPrefix)

	// Build the redirect Uri with the credential offer URI
	redirectUri := "/qr?" + credentialOfferURI

	slog.Info("Redirecting to QR page", "redirectUri", redirectUri)

	// // If the credential issuance was ok, update the database
	// reg.Issued = true
	// if err := s.DB.UpdateRegistrationStatus(reg); err != nil {
	// 	slog.Error("Error updating registration status with issuance success", "error", err)
	// }

	return redirectUri, nil
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
				Email:                  requestData.Email,
				SerialNumber:           "1234567890", // TODO: this is temporary because the Issuer requires a serialNumber
			},
			Mandatee: credissuance.Mandatee{
				FirstName: requestData.FirstName,
				LastName:  requestData.LastName,
				// Nationality: requestData.Country,
				Email: requestData.Email,
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
		slog.Error("Error calling issuance service", "error", err)

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
			slog.Error("Error marshalling credential data", "error", errformat)
			buf = []byte("Error marshalling credential data")
		}

		// Send an email to the internal team informing of the error, including the information that we wanted to issue
		// They can perform the rest of the registration steps manually, transparently to the user.

		if err := s.Mail.SendIssuerErrorEmail(reg, string(buf), err.Error()); err != nil {
			slog.Error("Error sending issuer error email", "error", err)
		}

		return errl.Error(err)
	}

	// If the credential issuance was ok, update the database
	reg.Issued = true
	if err := s.DB.UpdateRegistrationStatus(reg); err != nil {
		slog.Error("Error updating registration status with issuance success", "error", err)
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
		slog.Error("Error creating organization", "error", err)
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
		slog.Error("Error updating registration status with TMF success", "error", err)
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
		slog.Error("Error updating organization", "error", err)
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
		slog.Error("Error updating registration status with TMF success", "error", err)
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

	if !validateCSRF(r) {
		s.SendJSON(w, r, http.StatusForbidden, false, "Security check failed: missing CSRF header", nil)
		return
	}

	var req RepresentativesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.SendJSON(w, r, http.StatusBadRequest, false, "Invalid request body", errl.Error(err))
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
		s.SendJSON(w, r, http.StatusNotFound, false, "Registration not found", errl.Error(err))
		return
	}
	slog.Info("Registration found", "registration", current.String())

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
	current.LRFirstName = req.LRFirstName
	current.LRLastName = req.LRLastName
	current.LREmail = req.LREmail
	current.LRCountry = req.LRCountry
	current.LRIdCard = req.LRIdCard
	current.LEARFirstName = req.LEARFirstName
	current.LEARLastName = req.LEARLastName
	current.LEAREmail = req.LEAREmail
	current.LEARCountry = req.LEARCountry
	current.LEARAddress = req.LEARAddress
	current.LEARIdCard = req.LEARIdCard
	current.LEARMobileNumber = req.LEARMobileNumber
	current.LEARCompleted = true
	current.Role = "Seller"

	err = s.DB.UpdateRepresentativesByVatID(req.VatID, current)
	if err != nil {
		s.SendJSON(w, r, http.StatusInternalServerError, false, "Error updating registration", errl.Error(err))
		return
	}

	if err := s.Mail.SendSecondPhaseEmail(current); err != nil {
		err = errl.Errorf("Failed to send second phase email: %w", err)
		slog.Error("Error sending second phase email", "error", err)
		// Record the error in the database
		_ = s.DB.SaveRegistrationLog(&db.RegistrationLog{
			RegistrationID: current.RegistrationID,
			Email:          current.Email,
			VatID:          current.VatID,
			Type:           "error",
			Message:        err.Error(),
		})
	} else {
		slog.Info("Second phase email sent", "email", current.Email)
		current.Notified = true
		if updateErr := s.DB.UpdateRegistrationStatus(current); updateErr != nil {
			slog.Error("Error updating registration status with email result", "error", errl.Error(updateErr))
		}
	}

	s.SendJSON(w, r, http.StatusOK, true, "Representatives updated successfully", nil)
}

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

	// Trim blanks at the beginning and end
	r.VatId = strings.TrimSpace(r.VatId)
	// Convert to uppercase to avoid case sensitivity issues
	r.VatId = strings.ToUpper(r.VatId)

	// The prefix must be "VAT" followed by the country code and a dash
	expectedPrefix := "VAT" + r.Country + "-"

	// If country is an EU country, then the VAT ID must comply with the ETSI EN 319 412-1 format for organizationIdentifier
	// That is, it must start with "VAT" followed by the country code and a dash, e.g. "VATES-12345678"
	if common.IsValidEUCountry(r.Country) {

		// The user may have provided the VAT ID with or without the prefix, so we check if it starts with the expected prefix
		// If it does not have the prefix, we add it to the VAT ID and check if the rest of the string is a valid VAT number for the country provided

		if !strings.HasPrefix(r.VatId, expectedPrefix) {

			// It the provided VAT includes only the "VAT" prefix, it is an error
			if strings.HasPrefix(r.VatId, "VAT") {
				return errl.Errorf("invalid VAT ID format %s for country %s", r.VatId, r.Country)
			}

			// Add the expected prefix to the VAT ID
			r.VatId = expectedPrefix + strings.TrimPrefix(r.VatId, "VAT")
		}

		// At this moment, the VAT ID has the prefix
		vatNumber := strings.TrimPrefix(r.VatId, expectedPrefix)

		// Do not perform VIES validation in development, but log for developers
		if runtime == configuration.Development {
			slog.Warn("VIES in Development", "country", r.Country, "VAT number", r.VatId)
			return nil
		}

		// Check in the VIES database if the VAT ID is valid for the country provided
		viesResp, err := vies.CheckVAT(context.Background(), true, r.Country, vatNumber)
		if err != nil {
			return errl.Errorf("error checking VAT ID %s with VIES API: %w", r.VatId, err)
		}
		if !viesResp.Valid {
			return errl.Errorf("invalid VAT ID %s for country %s", r.VatId, r.Country)
		}

	} else {

		if !strings.HasPrefix(r.VatId, expectedPrefix) {
			// Add the expected prefix to the VAT ID
			r.VatId = expectedPrefix + r.VatId
		}
	}

	return nil
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

// EnableCORS middleware to allow all origins
func (s *Server) EnableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Requested-With, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
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
