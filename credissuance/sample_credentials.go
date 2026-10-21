package credissuance

func Cred1() *LEARIssuanceRequestBody {
	return &LEARIssuanceRequestBody{
		Schema:        "LEARCredentialEmployee",
		OperationMode: "S",
		Format:        "jwt_vc_json",
		Payload: Payload{
			Mandator: Mandator{
				OrganizationIdentifier: "VATES-123456",
				Organization:           "Test Organization",
				Country:                "ES",
				CommonName:             "Test User",
				Email:                  "jesus@alastria.io",
			},
			Mandatee: Mandatee{
				FirstName: "Test",
				LastName:  "User",
				// Nationality: "ES",
				Email: "jesus@alastria.io",
			},
			Power: []Power{
				{
					Type:     "domain",
					Domain:   "DOME",
					Function: "Onboarding",
					Action:   Strings{"execute"},
				},
			},
		},
	}
}

func Cred2() *LEARIssuanceRequestBody {
	return &LEARIssuanceRequestBody{
		Schema:        "LEARCredentialEmployee",
		OperationMode: "S",
		Format:        "jwt_vc_json",
		Payload: Payload{
			Mandator: Mandator{
				OrganizationIdentifier: "",
				Organization:           "",
				Country:                "",
				CommonName:             "",
				Email:                  "",
			},
			Mandatee: Mandatee{
				FirstName: "",
				LastName:  "",
				// Nationality: "",
				Email: "",
			},
			Power: []Power{
				{
					Type:     "domain",
					Domain:   "DOME",
					Function: "Onboarding",
					Action:   Strings{"execute"},
				},
			},
		},
	}
}
