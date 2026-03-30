package credissuance

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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

// TMFMyOrganization returns a sample TM Forum organization object used for testing.
func TMFMyOrganization() (*Organization, error) {
	org := Organization{
		Type:             "organization",
		Name:             "My Company",
		Status:           OrganizationStateInitialized,
		IsHeadOffice:     true,
		IsLegalEntity:    true,
		TradingName:      "My Company",
		OrganizationType: "company",
		OrganizationIdentification: []OrganizationIdentification{
			{
				Type:               "OrganizationIdentification",
				IdentificationID:   "VAT-TEST-777777J",
				IdentificationType: "elsi",
				IssuingAuthority:   "eIDAS",
			},
		},
		ExternalReference: []ExternalReference{
			{
				ExternalReferenceType: "idm_id",
				Name:                  "VAT-TEST-777777J",
			},
		},
		ContactMedium: []ContactMedium{
			{
				MediumType: "email",
				Preferred:  true,
				Characteristic: &MediumCharacteristic{
					EmailAddress: "perico@pepe.com",
				},
			},
		},
		PartyCharacteristic: []Characteristic{
			{
				Name:      "country",
				Value:     "ES",
				ValueType: "string",
			},
		},
	}

	return &org, nil
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
