package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/hesusruiz/onboardng/credissuance"
	"github.com/hesusruiz/onboardng/internal/db"
	"github.com/hesusruiz/utils/errl"
)

func performIssuanceDirect(ctx context.Context, token string, reg *db.RegistrationRecord, s *Server, requestData RegistrationRequest) (string, error) {

	// Add the VATxx- prefix to the VAT ID if the user has not provided it
	organizationIdentifierPrefix := "VAT" + requestData.Country
	organizationIdentifier := requestData.VatId
	if !strings.HasPrefix(requestData.VatId, organizationIdentifierPrefix) {
		organizationIdentifier = organizationIdentifierPrefix + "-" + requestData.VatId
	}

	// Create the mandate data to include in the Verifiable Credential for a LEAR
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
		_ = s.DB.LogRegError(reg.VatID, errIssuance.Error(), reg.RegistrationID, reg.Email)

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
		} else {
			_ = s.DB.LogRegWarn(reg.VatID, "Internal team notified of Issuer error", reg.RegistrationID, reg.Email)
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
	_ = s.DB.LogRegInfo(reg.VatID, "Credential issued via QR code - Displaying QR code for mobile wallet", reg.RegistrationID, reg.Email)

	return redirectUri, nil
}

func performIssuanceEmail(ctx context.Context, token string, reg *db.RegistrationRecord, s *Server, requestData RegistrationRequest) error {

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
		_ = s.DB.LogRegError(reg.VatID, fmt.Sprintf("Issuance error (in performIssuance): %v", err), reg.RegistrationID, reg.Email)

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
		} else {
			_ = s.DB.LogRegWarn(reg.VatID, "Internal team notified of Issuer error", reg.RegistrationID, reg.Email)
		}

		return errl.Error(err)
	}

	// If the credential issuance was ok, update the database
	reg.Issued = true
	if err := s.DB.UpdateRegistrationStatus(reg); err != nil {
		slog.Error("Error updating registration status with issuance success", "error", err)
	} else {
		_ = s.DB.LogRegInfo(reg.VatID, "Credential issued via email", reg.RegistrationID, reg.Email)
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
		_ = s.DB.LogRegError(reg.VatID, fmt.Sprintf("TM Forum create organization error: %v", err), reg.RegistrationID, reg.Email)
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
		_ = s.DB.LogRegError(reg.VatID, fmt.Sprintf("TM Forum update organization error: %v", err), reg.RegistrationID, reg.Email)
		return err
	}
	reg.TMFRegistered = true
	if err := s.DB.UpdateRegistrationStatus(reg); err != nil {
		slog.Error("Error updating registration status with TMF success", "error", err)
	}
	return nil
}
