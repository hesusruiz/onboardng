package credissuance

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/hesusruiz/utils/errl"
)

// TMFListOrganizations lists or finds Organization objects.
func (l *LEARIssuance) TMFListOrganizations(ctx context.Context, accessToken string, fields string, offset, limit int) ([]Organization, error) {
	url := fmt.Sprintf("%s%s/organization?fields=%s&offset=%d&limit=%d", l.tmForumURL, partyPathPrefix, fields, offset, limit)

	orgs, err := l.doHTTPList(ctx, url, accessToken)
	if err != nil {
		return nil, err
	}

	return orgs, nil
}

// TMFGetOrganizationByELSI retrieves Organization objects by ELSI identifier.
// If the error is nil, the returned array has at least one element. Otherwise, the array is empty.
// The function accepts an ELSI identifier with or without "did:elsi:" prefix, and performs two searches to the server,
// one with the prefix and the second without, to make sure that it finds the Organization in the server.
func (l *LEARIssuance) TMFGetOrganizationByELSI(ctx context.Context, accessToken string, elsi string) ([]Organization, error) {

	// Strip the prefix "did:elsi:" if it exists
	elsi = strings.TrimPrefix(elsi, "did:elsi:")

	// First search with the prefix
	url := fmt.Sprintf("%s%s/organization?organizationIdentification.identificationId=did:elsi:%s", l.tmForumURL, partyPathPrefix, elsi)

	orgs, err := l.doHTTPList(ctx, url, accessToken)
	if err == nil && len(orgs) > 0 {
		return orgs, nil
	}

	// If not found, try again without the prefix
	url = fmt.Sprintf("%s%s/organization?organizationIdentification.identificationId=%s", l.tmForumURL, partyPathPrefix, elsi)

	orgs, err = l.doHTTPList(ctx, url, accessToken)
	if err == nil && len(orgs) > 0 {
		return orgs, nil
	}

	// And lastly, try the externalReference.name mechanism, as a legacy fallback
	url = fmt.Sprintf("%s%s/organization?externalReference.name=%s", l.tmForumURL, partyPathPrefix, elsi)

	orgs, err = l.doHTTPList(ctx, url, accessToken)
	if err == nil && len(orgs) > 0 {
		return orgs, nil
	}

	return nil, errl.Errorf("no organization with ELSI %s: %w", elsi, ErrorNotFound)
}

// doHTTPList retrieves Organization objects from the TM Forum API.
// If the error is nil, the returned array has at least one element. Otherwise, the array is empty.
func (l *LEARIssuance) doHTTPList(ctx context.Context, url string, accessToken string) ([]Organization, error) {

	httpClient := l.httpClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, errl.Errorf("error creating http request: %w", err)
	}

	if accessToken != "" {
		req.Header.Add("Authorization", "Bearer "+accessToken)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, errl.Errorf("error in http request: %w", err)
	}
	defer resp.Body.Close()

	// Return the organization(s) if it was found (StatusCode == StatusOK)
	if resp.StatusCode == http.StatusOK {
		var orgs []Organization
		if err := json.NewDecoder(resp.Body).Decode(&orgs); err != nil {
			return nil, errl.Errorf("error decoding response: %w", err)
		}

		// If no organization was found, return an error
		if len(orgs) == 0 {
			return nil, errl.Error(ErrorNotFound)
		}

		// It is OK to retrieve more than one organization with the same ELSI for this function.
		// This is an error in the backend that will be solved in another way. The caller will decide what to do.
		return orgs, nil
	}

	// If the organization was not found, return an error
	return nil, errl.Error(ErrorNotFound)
}

// TMFMyOrganization is a helper function to create a sample organization (used for testing).
// It currently uses global accessToken and hardcoded URL as per user instructions.
func TMFMyOrganization(ctx context.Context) (*Organization, error) {

	org := Organization_Create{}
	org.Type = "organization"
	org.Name = "My Company"
	org.Status = "initialized"

	org.IsHeadOffice = true
	org.IsLegalEntity = true
	org.TradingName = "My Company"
	org.OrganizationType = "company"

	org.OrganizationIdentification = []OrganizationIdentification{
		{
			Type:               "OrganizationIdentification",
			IdentificationID:   "VAT-TEST-777777J",
			IdentificationType: "elsi", // ETSI Legal person Semantic Identifier, as in eIDAS certificates
			IssuingAuthority:   "eIDAS",
		},
	}

	org.ExternalReference = []ExternalReference{
		{
			ExternalReferenceType: "idm_id",
			Name:                  "VAT-TEST-777777J",
		},
	}

	org.ContactMedium = []ContactMedium{
		{
			MediumType: "email",
			Preferred:  true,
			Characteristic: &MediumCharacteristic{
				EmailAddress: "perico@pepe.com",
			},
		},
	}

	org.PartyCharacteristic = []Characteristic{
		{
			Name:      "country",
			Value:     "ES",
			ValueType: "string",
		},
	}

	buf, err := json.Marshal(org)
	if err != nil {
		return nil, errl.Errorf("error marshalling request body: %w", err)
	}

	// NOTE: Hardcoded URL and global accessToken as per user instruction
	url := "https://tmf.dome-marketplace-sbx.org/tmf-api/party/v4/organization"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(buf))
	if err != nil {
		return nil, errl.Errorf("error creating http request: %w", err)
	}

	req.Header.Add("Content-Type", "application/json")
	if accessToken != "" {
		req.Header.Add("Authorization", "Bearer "+accessToken)
	}

	slog.Info("Creating organization", "url", url)
	fmt.Println(string(buf))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errl.Errorf("error calling CreateOrganization at %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		// Read the body because contains the error
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, errl.Errorf("error reading CreateOrganization response body: %w", err)
		}
		return nil, errl.Errorf("error calling CreateOrganization at %s: %v - %s", url, resp.Status, string(body))
	}

	var createdOrg Organization
	if err := json.NewDecoder(resp.Body).Decode(&createdOrg); err != nil {
		return nil, errl.Errorf("error decoding CreateOrganization response: %w", err)
	}

	return &createdOrg, nil

}

func TMFOrganizationFromRequest(requestData RegistrationRequest) *Organization_Create {
	org := Organization_Create{}
	org.Type = "organization"
	org.Name = requestData.CompanyName
	org.Status = "initialized"

	org.IsHeadOffice = true
	org.IsLegalEntity = true
	org.TradingName = requestData.CompanyName
	org.OrganizationType = "company"

	// Determine the organizationIdentifier from the VatId.
	// We have to add the prefix 'did:elsi:' if it already doe snot have it
	organizationIdentifier := requestData.VatId
	if !strings.HasPrefix(requestData.VatId, "did:elsi:") {
		organizationIdentifier = "did:elsi:" + requestData.VatId
	}

	org.OrganizationIdentification = []OrganizationIdentification{
		{
			Type:               "organizationIdentification",
			IdentificationID:   organizationIdentifier,
			IdentificationType: "elsi", // ETSI Legal person Semantic Identifier, as in eIDAS certificates
			IssuingAuthority:   "eIDAS",
		},
	}

	org.ExternalReference = []ExternalReference{
		{
			ExternalReferenceType: "idm_id",
			Name:                  requestData.VatId,
		},
	}

	org.ContactMedium = []ContactMedium{
		{
			MediumType: "email",
			Preferred:  true,
			Characteristic: &MediumCharacteristic{
				EmailAddress: requestData.Email,
			},
		},
		{
			MediumType: "postalAddress",
			Characteristic: &MediumCharacteristic{
				Street1:  requestData.StreetAddress,
				City:     requestData.City,
				PostCode: requestData.PostalCode,
				Country:  requestData.Country,
			},
		},
	}

	org.PartyCharacteristic = []Characteristic{
		{
			Name:      "country",
			Value:     requestData.Country,
			ValueType: "string",
		},
		{
			Name:      "role",
			Value:     requestData.Role,
			ValueType: "string",
		},
	}

	return &org
}

func TMFOrganizationUpdateFromRequest(requestData RegistrationRequest) *Organization_Update {
	org := Organization_Update{}
	org.Type = "organization"
	org.Name = requestData.CompanyName
	org.Status = "initialized"

	org.IsHeadOffice = true
	org.IsLegalEntity = true
	org.TradingName = requestData.CompanyName
	org.OrganizationType = "company"

	org.OrganizationIdentification = []OrganizationIdentification{
		{
			Type:               "organizationIdentification",
			IdentificationID:   requestData.VatId,
			IdentificationType: "elsi",
			IssuingAuthority:   "eIDAS",
		},
	}

	org.ExternalReference = []ExternalReference{
		{
			ExternalReferenceType: "idm_id",
			Name:                  requestData.VatId,
		},
	}

	org.ContactMedium = []ContactMedium{
		{
			MediumType: "email",
			Preferred:  true,
			Characteristic: &MediumCharacteristic{
				EmailAddress: requestData.Email,
			},
		},
		{
			MediumType: "postalAddress",
			Characteristic: &MediumCharacteristic{
				Street1:  requestData.StreetAddress,
				City:     requestData.City,
				PostCode: requestData.PostalCode,
				Country:  requestData.Country,
			},
		},
	}

	org.PartyCharacteristic = []Characteristic{
		{
			Name:      "country",
			Value:     requestData.Country,
			ValueType: "string",
		},
		{
			Name:      "role",
			Value:     requestData.Role,
			ValueType: "string",
		},
	}

	return &org
}

// TMFCreateOrganization creates a Organization.
func (l *LEARIssuance) TMFCreateOrganization(ctx context.Context, accessToken string, org *Organization_Create) (*Organization, error) {
	buf, err := json.Marshal(org)
	if err != nil {
		return nil, errl.Errorf("error marshalling request body: %w", err)
	}

	url := fmt.Sprintf("%s%s/organization", l.tmForumURL, partyPathPrefix)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(buf))
	if err != nil {
		return nil, errl.Errorf("error creating http request: %w", err)
	}

	req.Header.Add("Content-Type", "application/json")
	if accessToken != "" {
		req.Header.Add("Authorization", "Bearer "+accessToken)
	}

	slog.Info("Creating organization", "url", url)

	resp, err := l.httpClient.Do(req)
	if err != nil {
		return nil, errl.Errorf("error calling CreateOrganization at %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, errl.Errorf("error calling CreateOrganization at %s: %v", url, resp.Status)
	}

	var createdOrg Organization
	if err := json.NewDecoder(resp.Body).Decode(&createdOrg); err != nil {
		return nil, errl.Errorf("error decoding CreateOrganization response: %w", err)
	}

	return &createdOrg, nil
}

// TMFRetrieveOrganization retrieves a Organization by ID.
func (l *LEARIssuance) TMFRetrieveOrganization(ctx context.Context, accessToken string, id string, fields string) (*Organization, error) {
	url := fmt.Sprintf("%s%s/organization/%s?fields=%s", l.tmForumURL, partyPathPrefix, id, fields)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, errl.Errorf("error creating http request: %w", err)
	}

	if accessToken != "" {
		req.Header.Add("Authorization", "Bearer "+accessToken)
	}

	resp, err := l.httpClient.Do(req)
	if err != nil {
		return nil, errl.Errorf("error calling RetrieveOrganization: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errl.Errorf("error calling RetrieveOrganization: %v", resp.Status)
	}

	var org Organization
	if err := json.NewDecoder(resp.Body).Decode(&org); err != nil {
		return nil, errl.Errorf("error decoding RetrieveOrganization response: %w", err)
	}

	return &org, nil
}

// TMFUpdateOrganization partially updates a Organization.
func (l *LEARIssuance) TMFUpdateOrganization(ctx context.Context, accessToken string, id string, org *Organization_Update) (*Organization, error) {
	buf, err := json.Marshal(org)
	if err != nil {
		return nil, errl.Errorf("error marshalling request body: %w", err)
	}

	url := fmt.Sprintf("%s%s/organization/%s", l.tmForumURL, partyPathPrefix, id)
	req, err := http.NewRequestWithContext(ctx, "PATCH", url, bytes.NewBuffer(buf))
	if err != nil {
		return nil, errl.Errorf("error creating http request: %w", err)
	}

	req.Header.Add("Content-Type", "application/json")
	if accessToken != "" {
		req.Header.Add("Authorization", "Bearer "+accessToken)
	}

	resp, err := l.httpClient.Do(req)
	if err != nil {
		return nil, errl.Errorf("error calling PatchOrganization: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errl.Errorf("error calling PatchOrganization: %v", resp.Status)
	}

	var updatedOrg Organization
	if err := json.NewDecoder(resp.Body).Decode(&updatedOrg); err != nil {
		return nil, errl.Errorf("error decoding PatchOrganization response: %w", err)
	}

	return &updatedOrg, nil
}

// TMFDeleteOrganization deletes a Organization.
func (l *LEARIssuance) TMFDeleteOrganization(ctx context.Context, accessToken string, id string) error {
	url := fmt.Sprintf("%s%s/organization/%s", l.tmForumURL, partyPathPrefix, id)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return errl.Errorf("error creating http request: %w", err)
	}

	if accessToken != "" {
		req.Header.Add("Authorization", "Bearer "+accessToken)
	}

	resp, err := l.httpClient.Do(req)
	if err != nil {
		return errl.Errorf("error calling DeleteOrganization: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return errl.Errorf("error calling DeleteOrganization: %v", resp.Status)
	}

	return nil
}

// TMFDeleteAllOrganizationsByELSI deletes all organizations matching a given ELSI.
func (l *LEARIssuance) TMFDeleteAllOrganizationsByELSI(ctx context.Context, accessToken string, elsi string) error {
	// Check in the TMF server if the organization already exists.
	// In PRO, we reject registration if the organization already exists.
	// In other environments, we continue with the registration, deleting the existing orgs.
	existingOrgs, _ := l.TMFGetOrganizationByELSI(ctx, accessToken, elsi)
	if len(existingOrgs) > 0 {
		slog.Info("Organization already exists in TMF server", "vatId", elsi)

		// Delete all the organizations from the TMF server
		for _, org := range existingOrgs {
			if err := l.TMFDeleteOrganization(ctx, accessToken, org.ID); err != nil {
				err = errl.Errorf("Failed to delete organization for registration: %v", err)
				slog.Error("❌ Error deleting organization", "error", err)
			}
			slog.Info("Existing organization deleted from TM Forum server", "orgId", org.ID)
		}

	}

	return nil

}

// NOTE: Retaining global state as requested by the user
var accessToken = "eyJraWQiOiJkaWQ6a2V5OnpEbmFldk44NVo3VkpnY0JvUWVxUVU3ZDhrWnB1VmhEU2RtOGhRdEpZV2p2ZWszVkwiLCJ0eXAiOiJKV1QiLCJhbGciOiJFUzI1NiJ9.eyJhdWQiOiJodHRwczovL3ZlcmlmaWVyLmRvbWUtbWFya2V0cGxhY2Utc2J4Lm9yZyIsInN1YiI6ImRpZDprZXk6ekRuYWVhanczRm1NZ3NHSnhXZ2dNTGJYRmdyN3lvZVRCS0JzUEFkRXJiTHBTRkxadCIsInNjb3BlIjoibWFjaGluZSBsZWFyY3JlZGVudGlhbCIsImlzcyI6Imh0dHBzOi8vdmVyaWZpZXIuZG9tZS1tYXJrZXRwbGFjZS1zYngub3JnIiwiZXhwIjoxNzcyNzc4NjQ2LCJpYXQiOjE3NzI3NzUwNDYsInZjIjp7IkBjb250ZXh0IjpbImh0dHBzOi8vd3d3LnczLm9yZy9ucy9jcmVkZW50aWFscy92MiIsImh0dHBzOi8vY3JlZGVudGlhbHMuZXVkaXN0YWNrLmV1Ly53ZWxsLWtub3duL2NyZWRlbnRpYWxzL2xlYXJfY3JlZGVudGlhbF9tYWNoaW5lL3czYy92MiJdLCJjcmVkZW50aWFsU3RhdHVzIjp7ImlkIjoiaHR0cHM6Ly9pc3N1ZXIuZG9tZS1tYXJrZXRwbGFjZS1zYngub3JnL3czYy92MS9jcmVkZW50aWFscy9zdGF0dXMvMSM5NDMyMyIsInN0YXR1c0xpc3RDcmVkZW50aWFsIjoiaHR0cHM6Ly9pc3N1ZXIuZG9tZS1tYXJrZXRwbGFjZS1zYngub3JnL3czYy92MS9jcmVkZW50aWFscy9zdGF0dXMvMSIsInN0YXR1c0xpc3RJbmRleCI6Ijk0MzIzIiwic3RhdHVzUHVycG9zZSI6InJldm9jYXRpb24iLCJ0eXBlIjoiQml0c3RyaW5nU3RhdHVzTGlzdEVudHJ5In0sImNyZWRlbnRpYWxTdWJqZWN0Ijp7ImlkIjoiZGlkOmtleTp6RG5hZWFqdzNGbU1nc0dKeFdnZ01MYlhGZ3I3eW9lVEJLQnNQQWRFcmJMcFNGTFp0IiwibWFuZGF0ZSI6eyJtYW5kYXRlZSI6eyJkb21haW4iOiJ0bWYuc2J4LmV2aWRlbmNlbGVkZ2VyLmV1IiwiaWQiOiJkaWQ6a2V5OnpEbmFlYWp3M0ZtTWdzR0p4V2dnTUxiWEZncjd5b2VUQktCc1BBZEVyYkxwU0ZMWnQiLCJpcEFkZHJlc3MiOiIyMTIuMjI3LjYxLjIwNiJ9LCJtYW5kYXRvciI6eyJjb21tb25OYW1lIjoiQ29uc3RhbnRpbm8gRmVybsOhbmRleiIsImNvdW50cnkiOiJFUyIsImVtYWlsIjoiZXhhbXBsZUBleGFtcGxlLm9yZyIsImlkIjoiZGlkOmVsc2k6VkFURVMtQTE1NDU2NTg1Iiwib3JnYW5pemF0aW9uIjoiQUxUSUEgQ09OU1VMVE9SRVMgU0EiLCJvcmdhbml6YXRpb25JZGVudGlmaWVyIjoiVkFURVMtQTE1NDU2NTg1Iiwic2VyaWFsTnVtYmVyIjoiMzI3NzEzODVMIn0sInBvd2VyIjpbeyJhY3Rpb24iOlsiRXhlY3V0ZSJdLCJkb21haW4iOiJET01FIiwiZnVuY3Rpb24iOiJPbmJvYXJkaW5nIiwidHlwZSI6ImRvbWFpbiJ9XX19LCJpZCI6InVybjp1dWlkOjg5MzEzOWU0LTE2YjgtNGFjZS1hN2QzLWZjNWNhMjkyYWY3ZCIsImlzc3VlciI6eyJjb21tb25OYW1lIjoiU2VhbCBTaWduYXR1cmUgQ3JlZGVudGlhbHMgaW4gU0JYIGZvciB0ZXN0aW5nIiwiY291bnRyeSI6IkVTIiwiaWQiOiJkaWQ6ZWxzaTpWQVRF..."
