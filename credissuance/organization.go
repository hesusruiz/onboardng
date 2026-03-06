package credissuance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/hesusruiz/utils/errl"
)

// Organization represents a group of people identified by shared interests or purpose.
type Organization struct {
	ID                             string                          `json:"id"`
	Href                           string                          `json:"href,omitempty"`
	IsHeadOffice                   bool                            `json:"isHeadOffice,omitempty"`
	IsLegalEntity                  bool                            `json:"isLegalEntity,omitempty"`
	Name                           string                          `json:"name,omitempty"`
	NameType                       string                          `json:"nameType,omitempty"`
	OrganizationType               string                          `json:"organizationType,omitempty"`
	TradingName                    string                          `json:"tradingName,omitempty"`
	ContactMedium                  []ContactMedium                 `json:"contactMedium,omitempty"`
	CreditRating                   []PartyCreditProfile            `json:"creditRating,omitempty"`
	ExistsDuring                   *TimePeriod                     `json:"existsDuring,omitempty"`
	ExternalReference              []ExternalReference             `json:"externalReference,omitempty"`
	OrganizationChildRelationship  []OrganizationChildRelationship `json:"organizationChildRelationship,omitempty"`
	OrganizationIdentification     []OrganizationIdentification    `json:"organizationIdentification,omitempty"`
	OrganizationParentRelationship *OrganizationParentRelationship `json:"organizationParentRelationship,omitempty"`
	OtherName                      []OtherNameOrganization         `json:"otherName,omitempty"`
	PartyCharacteristic            []Characteristic                `json:"partyCharacteristic,omitempty"`
	RelatedParty                   []RelatedParty                  `json:"relatedParty,omitempty"`
	Status                         OrganizationStateType           `json:"status,omitempty"`
	TaxExemptionCertificate        []TaxExemptionCertificate       `json:"taxExemptionCertificate,omitempty"`
	BaseType                       string                          `json:"@baseType,omitempty"`
	SchemaLocation                 string                          `json:"@schemaLocation,omitempty"`
	Type                           string                          `json:"@type,omitempty"`
}

type Organization_Create struct {
	IsHeadOffice                   bool                            `json:"isHeadOffice,omitempty"`
	IsLegalEntity                  bool                            `json:"isLegalEntity,omitempty"`
	Name                           string                          `json:"name,omitempty"`
	NameType                       string                          `json:"nameType,omitempty"`
	OrganizationType               string                          `json:"organizationType,omitempty"`
	TradingName                    string                          `json:"tradingName" binding:"required"`
	ContactMedium                  []ContactMedium                 `json:"contactMedium,omitempty"`
	CreditRating                   []PartyCreditProfile            `json:"creditRating,omitempty"`
	ExistsDuring                   *TimePeriod                     `json:"existsDuring,omitempty"`
	ExternalReference              []ExternalReference             `json:"externalReference,omitempty"`
	OrganizationChildRelationship  []OrganizationChildRelationship `json:"organizationChildRelationship,omitempty"`
	OrganizationIdentification     []OrganizationIdentification    `json:"organizationIdentification,omitempty"`
	OrganizationParentRelationship *OrganizationParentRelationship `json:"organizationParentRelationship,omitempty"`
	OtherName                      []OtherNameOrganization         `json:"otherName,omitempty"`
	PartyCharacteristic            []Characteristic                `json:"partyCharacteristic,omitempty"`
	RelatedParty                   []RelatedParty                  `json:"relatedParty,omitempty"`
	Status                         OrganizationStateType           `json:"status,omitempty"`
	TaxExemptionCertificate        []TaxExemptionCertificate       `json:"taxExemptionCertificate,omitempty"`
	BaseType                       string                          `json:"@baseType,omitempty"`
	SchemaLocation                 string                          `json:"@schemaLocation,omitempty"`
	Type                           string                          `json:"@type,omitempty"`
}

type Organization_Update struct {
	IsHeadOffice                   bool                            `json:"isHeadOffice,omitempty"`
	IsLegalEntity                  bool                            `json:"isLegalEntity,omitempty"`
	Name                           string                          `json:"name,omitempty"`
	NameType                       string                          `json:"nameType,omitempty"`
	OrganizationType               string                          `json:"organizationType,omitempty"`
	TradingName                    string                          `json:"tradingName,omitempty"`
	ContactMedium                  []ContactMedium                 `json:"contactMedium,omitempty"`
	CreditRating                   []PartyCreditProfile            `json:"creditRating,omitempty"`
	ExistsDuring                   *TimePeriod                     `json:"existsDuring,omitempty"`
	ExternalReference              []ExternalReference             `json:"externalReference,omitempty"`
	OrganizationChildRelationship  []OrganizationChildRelationship `json:"organizationChildRelationship,omitempty"`
	OrganizationIdentification     []OrganizationIdentification    `json:"organizationIdentification,omitempty"`
	OrganizationParentRelationship *OrganizationParentRelationship `json:"organizationParentRelationship,omitempty"`
	OtherName                      []OtherNameOrganization         `json:"otherName,omitempty"`
	PartyCharacteristic            []Characteristic                `json:"partyCharacteristic,omitempty"`
	RelatedParty                   []RelatedParty                  `json:"relatedParty,omitempty"`
	Status                         OrganizationStateType           `json:"status,omitempty"`
	TaxExemptionCertificate        []TaxExemptionCertificate       `json:"taxExemptionCertificate,omitempty"`
	BaseType                       string                          `json:"@baseType,omitempty"`
	SchemaLocation                 string                          `json:"@schemaLocation,omitempty"`
	Type                           string                          `json:"@type,omitempty"`
}

type OrganizationChildRelationship struct {
	RelationshipType string           `json:"relationshipType,omitempty"`
	Organization     *OrganizationRef `json:"organization,omitempty"`
	BaseType         string           `json:"@baseType,omitempty"`
	SchemaLocation   string           `json:"@schemaLocation,omitempty"`
	Type             string           `json:"@type,omitempty"`
}

type OrganizationIdentification struct {
	IdentificationID   string                `json:"identificationId,omitempty"`
	IdentificationType string                `json:"identificationType,omitempty"`
	IssuingAuthority   string                `json:"issuingAuthority,omitempty"`
	IssuingDate        string                `json:"issuingDate,omitempty"`
	Attachment         *AttachmentRefOrValue `json:"attachment,omitempty"`
	ValidFor           *TimePeriod           `json:"validFor,omitempty"`
	BaseType           string                `json:"@baseType,omitempty"`
	SchemaLocation     string                `json:"@schemaLocation,omitempty"`
	Type               string                `json:"@type,omitempty"`
}

type OrganizationParentRelationship struct {
	RelationshipType string           `json:"relationshipType,omitempty"`
	Organization     *OrganizationRef `json:"organization,omitempty"`
	BaseType         string           `json:"@baseType,omitempty"`
	SchemaLocation   string           `json:"@schemaLocation,omitempty"`
	Type             string           `json:"@type,omitempty"`
}

type OrganizationRef struct {
	ID             string `json:"id"`
	Href           string `json:"href,omitempty"`
	Name           string `json:"name,omitempty"`
	BaseType       string `json:"@baseType,omitempty"`
	SchemaLocation string `json:"@schemaLocation,omitempty"`
	Type           string `json:"@type,omitempty"`
	ReferredType   string `json:"@referredType,omitempty"`
}

type OrganizationStateType string

const (
	OrganizationStateInitialized OrganizationStateType = "initialized"
	OrganizationStateValidated   OrganizationStateType = "validated"
	OrganizationStateClosed      OrganizationStateType = "closed"
)

type ContactMedium struct {
	MediumType     string                `json:"mediumType,omitempty"`
	Preferred      bool                  `json:"preferred,omitempty"`
	Characteristic *MediumCharacteristic `json:"characteristic,omitempty"`
	ValidFor       *TimePeriod           `json:"validFor,omitempty"`
	BaseType       string                `json:"@baseType,omitempty"`
	SchemaLocation string                `json:"@schemaLocation,omitempty"`
	Type           string                `json:"@type,omitempty"`
}

type MediumCharacteristic struct {
	City            string `json:"city,omitempty"`
	ContactType     string `json:"contactType,omitempty"`
	Country         string `json:"country,omitempty"`
	EmailAddress    string `json:"emailAddress,omitempty"`
	FaxNumber       string `json:"faxNumber,omitempty"`
	PhoneNumber     string `json:"phoneNumber,omitempty"`
	PostCode        string `json:"postCode,omitempty"`
	SocialNetworkID string `json:"socialNetworkId,omitempty"`
	StateOrProvince string `json:"stateOrProvince,omitempty"`
	Street1         string `json:"street1,omitempty"`
	Street2         string `json:"street2,omitempty"`
	BaseType        string `json:"@baseType,omitempty"`
	SchemaLocation  string `json:"@schemaLocation,omitempty"`
	Type            string `json:"@type,omitempty"`
}

type TimePeriod struct {
	EndDateTime   string `json:"endDateTime,omitempty"`
	StartDateTime string `json:"startDateTime,omitempty"`
}

type Characteristic struct {
	Name           string      `json:"name"`
	ValueType      string      `json:"valueType,omitempty"`
	Value          interface{} `json:"value"`
	BaseType       string      `json:"@baseType,omitempty"`
	SchemaLocation string      `json:"@schemaLocation,omitempty"`
	Type           string      `json:"@type,omitempty"`
}

type PartyCreditProfile struct {
	CreditAgencyName string      `json:"creditAgencyName,omitempty"`
	CreditAgencyType string      `json:"creditAgencyType,omitempty"`
	RatingReference  string      `json:"ratingReference,omitempty"`
	RatingScore      int         `json:"ratingScore,omitempty"`
	ValidFor         *TimePeriod `json:"validFor,omitempty"`
	BaseType         string      `json:"@baseType,omitempty"`
	SchemaLocation   string      `json:"@schemaLocation,omitempty"`
	Type             string      `json:"@type,omitempty"`
}

type ExternalReference struct {
	ExternalReferenceType string `json:"externalReferenceType,omitempty"`
	Name                  string `json:"name,omitempty"`
	BaseType              string `json:"@baseType,omitempty"`
	SchemaLocation        string `json:"@schemaLocation,omitempty"`
	Type                  string `json:"@type,omitempty"`
}

type RelatedParty struct {
	ID             string `json:"id"`
	Href           string `json:"href,omitempty"`
	Name           string `json:"name,omitempty"`
	Role           string `json:"role,omitempty"`
	BaseType       string `json:"@baseType,omitempty"`
	SchemaLocation string `json:"@schemaLocation,omitempty"`
	Type           string `json:"@type,omitempty"`
	ReferredType   string `json:"@referredType"`
}

type TaxExemptionCertificate struct {
	ID             string          `json:"id,omitempty"`
	Attachment     *AttachmentRef  `json:"attachment,omitempty"`
	TaxDefinition  []TaxDefinition `json:"taxDefinition,omitempty"`
	ValidFor       *TimePeriod     `json:"validFor,omitempty"`
	BaseType       string          `json:"@baseType,omitempty"`
	SchemaLocation string          `json:"@schemaLocation,omitempty"`
	Type           string          `json:"@type,omitempty"`
}

type TaxDefinition struct {
	ID             string `json:"id,omitempty"`
	Name           string `json:"name,omitempty"`
	TaxType        string `json:"taxType,omitempty"`
	BaseType       string `json:"@baseType,omitempty"`
	SchemaLocation string `json:"@schemaLocation,omitempty"`
	Type           string `json:"@type,omitempty"`
	ReferredType   string `json:"@referredType,omitempty"`
}

type AttachmentRefOrValue struct {
	ID             string      `json:"id,omitempty"`
	Href           string      `json:"href,omitempty"`
	AttachmentType string      `json:"attachmentType,omitempty"`
	Content        string      `json:"content,omitempty"`
	Description    string      `json:"description,omitempty"`
	MimeType       string      `json:"mimeType,omitempty"`
	Name           string      `json:"name,omitempty"`
	URL            string      `json:"url,omitempty"`
	Size           *Quantity   `json:"size,omitempty"`
	ValidFor       *TimePeriod `json:"validFor,omitempty"`
	BaseType       string      `json:"@baseType,omitempty"`
	SchemaLocation string      `json:"@schemaLocation,omitempty"`
	Type           string      `json:"@type,omitempty"`
	ReferredType   string      `json:"@referredType,omitempty"`
}

type AttachmentRef struct {
	ID             string      `json:"id,omitempty"`
	Href           string      `json:"href,omitempty"`
	AttachmentType string      `json:"attachmentType,omitempty"`
	Content        string      `json:"content,omitempty"`
	Description    string      `json:"description,omitempty"`
	MimeType       string      `json:"mimeType,omitempty"`
	Name           string      `json:"name,omitempty"`
	URL            string      `json:"url,omitempty"`
	Size           *Quantity   `json:"size,omitempty"`
	ValidFor       *TimePeriod `json:"validFor,omitempty"`
	BaseType       string      `json:"@baseType,omitempty"`
	SchemaLocation string      `json:"@schemaLocation,omitempty"`
	Type           string      `json:"@type,omitempty"`
	ReferredType   string      `json:"@referredType,omitempty"`
}

type Quantity struct {
	Amount float64 `json:"amount,omitempty"`
	Units  string  `json:"units,omitempty"`
}

type OtherNameOrganization struct {
	Name           string      `json:"name,omitempty"`
	NameType       string      `json:"nameType,omitempty"`
	TradingName    string      `json:"tradingName,omitempty"`
	ValidFor       *TimePeriod `json:"validFor,omitempty"`
	BaseType       string      `json:"@baseType,omitempty"`
	SchemaLocation string      `json:"@schemaLocation,omitempty"`
	Type           string      `json:"@type,omitempty"`
}

// ListOrganizations lists or finds Organization objects.
func (l *LEARIssuance) ListOrganizations(accessToken string, fields string, offset, limit int) ([]Organization, error) {
	url := fmt.Sprintf("%s/organization?fields=%s&offset=%d&limit=%d", l.tmForumURL, fields, offset, limit)

	req, _ := http.NewRequest("GET", url, nil)
	if accessToken != "" {
		req.Header.Add("Authorization", "Bearer "+accessToken)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errl.Errorf("error calling ListOrganizations: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errl.Errorf("error calling ListOrganizations: %v", resp.Status)
	}

	var orgs []Organization
	if err := json.NewDecoder(resp.Body).Decode(&orgs); err != nil {
		return nil, errl.Errorf("error decoding ListOrganizations response: %w", err)
	}

	return orgs, nil
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
}

var accessToken = "eyJraWQiOiJkaWQ6a2V5OnpEbmFldk44NVo3VkpnY0JvUWVxUVU3ZDhrWnB1VmhEU2RtOGhRdEpZV2p2ZWszVkwiLCJ0eXAiOiJKV1QiLCJhbGciOiJFUzI1NiJ9.eyJhdWQiOiJodHRwczovL3ZlcmlmaWVyLmRvbWUtbWFya2V0cGxhY2Utc2J4Lm9yZyIsInN1YiI6ImRpZDprZXk6ekRuYWVhanczRm1NZ3NHSnhXZ2dNTGJYRmdyN3lvZVRCS0JzUEFkRXJiTHBTRkxadCIsInNjb3BlIjoibWFjaGluZSBsZWFyY3JlZGVudGlhbCIsImlzcyI6Imh0dHBzOi8vdmVyaWZpZXIuZG9tZS1tYXJrZXRwbGFjZS1zYngub3JnIiwiZXhwIjoxNzcyNzc4NjQ2LCJpYXQiOjE3NzI3NzUwNDYsInZjIjp7IkBjb250ZXh0IjpbImh0dHBzOi8vd3d3LnczLm9yZy9ucy9jcmVkZW50aWFscy92MiIsImh0dHBzOi8vY3JlZGVudGlhbHMuZXVkaXN0YWNrLmV1Ly53ZWxsLWtub3duL2NyZWRlbnRpYWxzL2xlYXJfY3JlZGVudGlhbF9tYWNoaW5lL3czYy92MiJdLCJjcmVkZW50aWFsU3RhdHVzIjp7ImlkIjoiaHR0cHM6Ly9pc3N1ZXIuZG9tZS1tYXJrZXRwbGFjZS1zYngub3JnL3czYy92MS9jcmVkZW50aWFscy9zdGF0dXMvMSM5NDMyMyIsInN0YXR1c0xpc3RDcmVkZW50aWFsIjoiaHR0cHM6Ly9pc3N1ZXIuZG9tZS1tYXJrZXRwbGFjZS1zYngub3JnL3czYy92MS9jcmVkZW50aWFscy9zdGF0dXMvMSIsInN0YXR1c0xpc3RJbmRleCI6Ijk0MzIzIiwic3RhdHVzUHVycG9zZSI6InJldm9jYXRpb24iLCJ0eXBlIjoiQml0c3RyaW5nU3RhdHVzTGlzdEVudHJ5In0sImNyZWRlbnRpYWxTdWJqZWN0Ijp7ImlkIjoiZGlkOmtleTp6RG5hZWFqdzNGbU1nc0dKeFdnZ01MYlhGZ3I3eW9lVEJLQnNQQWRFcmJMcFNGTFp0IiwibWFuZGF0ZSI6eyJtYW5kYXRlZSI6eyJkb21haW4iOiJ0bWYuc2J4LmV2aWRlbmNlbGVkZ2VyLmV1IiwiaWQiOiJkaWQ6a2V5OnpEbmFlYWp3M0ZtTWdzR0p4V2dnTUxiWEZncjd5b2VUQktCc1BBZEVyYkxwU0ZMWnQiLCJpcEFkZHJlc3MiOiIyMTIuMjI3LjYxLjIwNiJ9LCJtYW5kYXRvciI6eyJjb21tb25OYW1lIjoiQ29uc3RhbnRpbm8gRmVybsOhbmRleiIsImNvdW50cnkiOiJFUyIsImVtYWlsIjoiZXhhbXBsZUBleGFtcGxlLm9yZyIsImlkIjoiZGlkOmVsc2k6VkFURVMtQTE1NDU2NTg1Iiwib3JnYW5pemF0aW9uIjoiQUxUSUEgQ09OU1VMVE9SRVMgU0EiLCJvcmdhbml6YXRpb25JZGVudGlmaWVyIjoiVkFURVMtQTE1NDU2NTg1Iiwic2VyaWFsTnVtYmVyIjoiMzI3NzEzODVMIn0sInBvd2VyIjpbeyJhY3Rpb24iOlsiRXhlY3V0ZSJdLCJkb21haW4iOiJET01FIiwiZnVuY3Rpb24iOiJPbmJvYXJkaW5nIiwidHlwZSI6ImRvbWFpbiJ9XX19LCJpZCI6InVybjp1dWlkOjg5MzEzOWU0LTE2YjgtNGFjZS1hN2QzLWZjNWNhMjkyYWY3ZCIsImlzc3VlciI6eyJjb21tb25OYW1lIjoiU2VhbCBTaWduYXR1cmUgQ3JlZGVudGlhbHMgaW4gU0JYIGZvciB0ZXN0aW5nIiwiY291bnRyeSI6IkVTIiwiaWQiOiJkaWQ6ZWxzaTpWQVRFUy1CNjA2NDU5MDAiLCJvcmdhbml6YXRpb24iOiJJTjIiLCJvcmdhbml6YXRpb25JZGVudGlmaWVyIjoiVkFURVMtQjYwNjQ1OTAwIiwic2VyaWFsTnVtYmVyIjoiQjQ3NDQ3NTYwIn0sInR5cGUiOlsiTEVBUkNyZWRlbnRpYWxNYWNoaW5lIiwiVmVyaWZpYWJsZUNyZWRlbnRpYWwiXSwidmFsaWRGcm9tIjoiMjAyNi0wMi0yMFQxMzowNjo1My4yMDMzMzQ3NDNaIiwidmFsaWRVbnRpbCI6IjIwMjctMDItMjBUMTM6MDY6NTMuMjAzMzM0NzQzWiJ9LCJqdGkiOiJjMWM1ZTVjZS04YzE2LTRmMzMtYTdkYy05NTQ3YjZkNTI1NTEiLCJjbGllbnRfaWQiOiJodHRwczovL3ZlcmlmaWVyLmRvbWUtbWFya2V0cGxhY2Utc2J4Lm9yZyJ9.AmlnDNComKFujzyuqwXCE_JkpE7r0gGsO-3pCtp3CNK2tcRcFYPkyzej8DR9LCpGtoDg8fytDkVOjwqn0QwNBw"

func MyOrganization() (*Organization, error) {

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

	url := "https://tmf.dome-marketplace-sbx.org/tmf-api/party/v4/organization"
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(buf))
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

func OrganizationFromRequest(requestData RegistrationRequest) *Organization_Create {
	org := Organization_Create{}
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
	}

	return &org
}

const pathPrefix = "/tmf-api/party/v4"

// CreateOrganization creates a Organization.
func (l *LEARIssuance) CreateOrganization(accessToken string, org *Organization_Create) (*Organization, error) {
	buf, err := json.Marshal(org)
	if err != nil {
		return nil, errl.Errorf("error marshalling request body: %w", err)
	}

	url := fmt.Sprintf("%s%s/organization", l.tmForumURL, pathPrefix)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(buf))
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
		return nil, errl.Errorf("error calling CreateOrganization at %s: %v", url, resp.Status)
	}

	var createdOrg Organization
	if err := json.NewDecoder(resp.Body).Decode(&createdOrg); err != nil {
		return nil, errl.Errorf("error decoding CreateOrganization response: %w", err)
	}

	return &createdOrg, nil
}

// RetrieveOrganization retrieves a Organization by ID.
func (l *LEARIssuance) RetrieveOrganization(accessToken string, id string, fields string) (*Organization, error) {
	url := fmt.Sprintf("%s%s/organization/%s?fields=%s", l.tmForumURL, pathPrefix, id, fields)
	req, _ := http.NewRequest("GET", url, nil)
	if accessToken != "" {
		req.Header.Add("Authorization", "Bearer "+accessToken)
	}

	resp, err := http.DefaultClient.Do(req)
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

// PatchOrganization partially updates a Organization.
func (l *LEARIssuance) PatchOrganization(accessToken string, id string, org *Organization_Update) (*Organization, error) {
	buf, err := json.Marshal(org)
	if err != nil {
		return nil, errl.Errorf("error marshalling request body: %w", err)
	}

	url := fmt.Sprintf("%s%s/organization/%s", l.tmForumURL, pathPrefix, id)
	req, _ := http.NewRequest("PATCH", url, bytes.NewBuffer(buf))
	req.Header.Add("Content-Type", "application/json")
	if accessToken != "" {
		req.Header.Add("Authorization", "Bearer "+accessToken)
	}

	resp, err := http.DefaultClient.Do(req)
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

// DeleteOrganization deletes a Organization.
func (l *LEARIssuance) DeleteOrganization(accessToken string, id string) error {
	url := fmt.Sprintf("%s%s/organization/%s", l.tmForumURL, pathPrefix, id)
	req, _ := http.NewRequest("DELETE", url, nil)
	if accessToken != "" {
		req.Header.Add("Authorization", "Bearer "+accessToken)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errl.Errorf("error calling DeleteOrganization: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return errl.Errorf("error calling DeleteOrganization: %v", resp.Status)
	}

	return nil
}
