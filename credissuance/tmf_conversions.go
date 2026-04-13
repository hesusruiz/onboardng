package credissuance

import "strings"

// NewSampleTMFOrganization returns a sample TM Forum organization object used for testing.
func NewSampleTMFOrganization() (*Organization, error) {
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
