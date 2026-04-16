package server

import (
	"context"

	"github.com/hesusruiz/onboardng/credissuance"
	"github.com/hesusruiz/onboardng/internal/db"
)

// DBServiceProvider enables easy testing or replacing of the database implementation
type DBServiceProvider interface {
	SaveRegistration(reg *db.RegistrationRecord) error
	UpdateRegistrationStatus(reg *db.RegistrationRecord) error
	UpdateApproval(registrationID string, approved int) error
	UpdateRepresentativesByVatID(vatID string, rep *db.RegistrationRecord) error
	SaveRegistrationLog(logEntry *db.RegistrationLog) error
	GetRegistrationByVatID(vatID string) (*db.RegistrationRecord, error)
	GetRegistrationByEmail(email string) (*db.RegistrationRecord, error)
	GetRegistrationByEmailOrVatID(email string, vatID string) (*db.RegistrationRecord, error)
	GetRegistrations(limit, offset int) ([]db.RegistrationRecord, error)
	GetRegistrationLogs(vatID string, limit, offset int) ([]db.RegistrationLog, error)
	GetRegistrationFiles(vatID string, limit, offset int) ([]db.RegistrationFile, error)
	GetRegistrationFile(fileID string) (*db.RegistrationFile, error)
	GetRegistrationByID(registrationID string) (*db.RegistrationRecord, error)
}

// MailServiceProvider enables easy testing or replacing of the mail implementation
type MailServiceProvider interface {
	SendVerificationCode(email string, code string) error
	SendTestEmail() error
	SendWelcomeEmail(reg *db.RegistrationRecord) error
	SendIssuerError(reg *db.RegistrationRecord, payload string, errorMsg string) error
}

// IssuanceServiceProvider enables easy testing or replacing of the issuance implementation
type IssuanceServiceProvider interface {
	GetAccessToken(ctx context.Context) (string, error)
	TMFGetOrganizationByELSI(ctx context.Context, accessToken string, elsi string) ([]credissuance.Organization, error)
	TMFDeleteOrganization(ctx context.Context, accessToken string, id string) error

	// LEARIssuanceRequest initiates the second step of the process: requesting the issuance
	// of a new Verifiable Credential using the previously obtained access token.
	LEARIssuanceRequest(ctx context.Context, accessToken string, learCredData *credissuance.LEARIssuanceRequestBody) ([]byte, error)

	TMFCreateOrganization(ctx context.Context, accessToken string, org *credissuance.Organization_Create) (*credissuance.Organization, error)
	TMFUpdateOrganization(ctx context.Context, accessToken string, id string, org *credissuance.Organization_Update) (*credissuance.Organization, error)
}
