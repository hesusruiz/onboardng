package credissuance

import "errors"

const partyPathPrefix = "/tmf-api/party/v4"

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

var ErrorNotFound = errors.New("not found")

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
	Role          string `json:"role"`
}
