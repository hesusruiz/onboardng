package server

import (
	"context"
	"log/slog"
	"strings"

	"github.com/hesusruiz/onboardng/credissuance"
	"github.com/hesusruiz/utils/errl"
)

// The functions below define a series or run-time checks that can be performed to assess if the server is working perfectly,
// including its dependencies.
// They can also be used to debug issues with the configuration files of the server, when deployed in a new environment.
// The checks are intented to be useful even on the production environment, so they have to be non-destructive, or at least
// to be able to revert the environment to the situation before the check was performed.
// For example, we may insert a record in the database, and then update it and the delete it, so that the database is back to the
// original state.

func (s *Server) CheckAll() error {

	slog.Info("Self-Diagnostics: running all checks")

	ctx := context.Background()

	token, err := s.Issuer.GetAccessToken(ctx)
	if err != nil {
		err = errl.Errorf("Self-Diagnostics: failed to get access token for credential issuance: %v", err)
		slog.Error(err.Error())
		return err
	}

	vatID := "B12345678"

	err = s.CheckIssuer(ctx, token, vatID)
	if err != nil {
		err = errl.Errorf("Self-Diagnostics: failed to check Issuer: %v", err)
		slog.Error(err.Error())
		return err
	}

	err = s.CheckTMForum(ctx, token, vatID)
	if err != nil {
		err = errl.Errorf("Self-Diagnostics: failed to check TM Forum: %v", err)
		slog.Error(err.Error())
		return err
	}

	err = s.CheckMail()
	if err != nil {
		err = errl.Errorf("Self-Diagnostics: failed to check Mail: %v", err)
		slog.Error(err.Error())
		return err
	}

	err = s.CheckDatabase()
	if err != nil {
		err = errl.Errorf("Self-Diagnostics: failed to check Database: %v", err)
		slog.Error(err.Error())
		return err
	}

	slog.Info("Self-Diagnostics: all checks passed")
	return nil

}

func (s *Server) CheckIssuer(ctx context.Context, token string, vatID string) error {

	slog.Info("Self-Diagnostics Issuer: checking the Issuer with a test credential")

	if ctx == nil {
		ctx = context.Background()
	}

	// Get an access token to authenticate in the Issuer and TM Forum APIs¡
	if token == "" {
		var err error
		token, err = s.Issuer.GetAccessToken(ctx)
		if err != nil {
			err = errl.Errorf("Self-Diagnostics Issuer: failed to get access token for credential issuance: %v", err)
			slog.Error(err.Error())
			return err
		}
	}

	// Create a test credential, to be used even in production to check if the issuance process works
	requestData := RegistrationRequest{
		FirstName:     "John",
		LastName:      "Doe",
		CompanyName:   "TESTING Corp",
		Country:       "ES",
		VatId:         vatID,
		StreetAddress: "Main St 1",
		City:          "Madrid",
		PostalCode:    "28001",
		Email:         "hesus.ruiz@gmail.com",
		Code:          "123456",
		Role:          "Seller",
	}

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

	_, err := s.Issuer.LEARIssuanceRequest(ctx, token, soloCredential)
	if err != nil {
		err = errl.Errorf("Self-Diagnostics Issuer: failed to issue credential: %v", err)
		slog.Error(err.Error())
		return err
	}

	slog.Info("Self-Diagnostics Issuer: successfully issued a test credential")

	return nil
}

func (s *Server) CheckTMForum(ctx context.Context, token string, vatID string) error {

	slog.Info("Self-Diagnostics TM Forum: checking the TM Forum API")

	if !s.Features.TMFServerEnabled {
		slog.Info("Self-Diagnostics TM Forum: TM Forum server is not enabled")
		return nil
	}

	if ctx == nil {
		ctx = context.Background()
	}

	if token == "" {
		// Get an access token to authenticate in the Issuer and TM Forum APIs
		var err error
		token, err = s.Issuer.GetAccessToken(ctx)
		if err != nil {
			err = errl.Errorf("Self-Diagnostics TM Forum: failed to get access token for TM Forum API: %v", err)
			slog.Error(err.Error())
			return err
		}
	}

	if vatID == "" {
		vatID = "B12345678"
	}

	// Check in the TMF server if the organization already exists.

	existingOrgs, err := s.Issuer.TMFGetOrganizationByELSI(ctx, token, vatID)
	if err != nil {
		err = errl.Errorf("Self-Diagnostics TM Forum: failed to get organization by ELSI: %v", err)
		slog.Error(err.Error())
		return err
	}

	if len(existingOrgs) > 0 {
		slog.Info("Self-Diagnostics TM Forum: success and Organization already exists in TMF server", "vatId", vatID)
	} else {
		slog.Info("Self-Diagnostics TM Forum: success but Organization does not exist in TMF server", "vatId", vatID)
	}

	return nil
}

func (s *Server) CheckMail() error {
	return nil
}

func (s *Server) CheckDatabase() error {
	return nil
}
