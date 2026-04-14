package server

import (
	"context"

	"github.com/hesusruiz/onboardng/credissuance"
	"github.com/hesusruiz/onboardng/internal/db"
)

type MockDB struct {
	SaveRegistrationFunc              func(reg *db.RegistrationRecord) error
	UpdateRegistrationStatusFunc      func(reg *db.RegistrationRecord) error
	UpdateApprovalFunc                func(registrationID string, approved int) error
	SaveRegistrationLogFunc           func(logEntry *db.RegistrationLog) error
	GetRegistrationByVatIDFunc        func(vatID string) (*db.RegistrationRecord, error)
	GetRegistrationByEmailFunc        func(email string) (*db.RegistrationRecord, error)
	GetRegistrationByEmailOrVatIDFunc func(email, vatID string) (*db.RegistrationRecord, error)
	GetRegistrationsFunc              func(limit, offset int) ([]db.RegistrationRecord, error)
	GetRegistrationLogsFunc           func(vatID string, limit, offset int) ([]db.RegistrationLog, error)
	GetRegistrationFilesFunc          func(vatID string, limit, offset int) ([]db.RegistrationFile, error)
	GetRegistrationFileFunc           func(fileID string) (*db.RegistrationFile, error)
	GetRegistrationByIDFunc           func(registrationID string) (*db.RegistrationRecord, error)
	UpdateRepresentativesByVatIDFunc  func(vatID string, rep *db.RegistrationRecord) error
}

func (m *MockDB) SaveRegistration(reg *db.RegistrationRecord) error {
	return m.SaveRegistrationFunc(reg)
}
func (m *MockDB) UpdateRegistrationStatus(reg *db.RegistrationRecord) error {
	return m.UpdateRegistrationStatusFunc(reg)
}
func (m *MockDB) UpdateApproval(registrationID string, approved int) error {
	return m.UpdateApprovalFunc(registrationID, approved)
}
func (m *MockDB) SaveRegistrationLog(logEntry *db.RegistrationLog) error {
	return m.SaveRegistrationLogFunc(logEntry)
}
func (m *MockDB) GetRegistrationByVatID(vatID string) (*db.RegistrationRecord, error) {
	return m.GetRegistrationByVatIDFunc(vatID)
}
func (m *MockDB) GetRegistrationByEmail(email string) (*db.RegistrationRecord, error) {
	return m.GetRegistrationByEmailFunc(email)
}
func (m *MockDB) GetRegistrationByEmailOrVatID(email, vatID string) (*db.RegistrationRecord, error) {
	return m.GetRegistrationByEmailOrVatIDFunc(email, vatID)
}
func (m *MockDB) GetRegistrations(limit, offset int) ([]db.RegistrationRecord, error) {
	return m.GetRegistrationsFunc(limit, offset)
}
func (m *MockDB) GetRegistrationLogs(vatID string, limit, offset int) ([]db.RegistrationLog, error) {
	return m.GetRegistrationLogsFunc(vatID, limit, offset)
}
func (m *MockDB) GetRegistrationFiles(vatID string, limit, offset int) ([]db.RegistrationFile, error) {
	return m.GetRegistrationFilesFunc(vatID, limit, offset)
}
func (m *MockDB) GetRegistrationFile(fileID string) (*db.RegistrationFile, error) {
	return m.GetRegistrationFileFunc(fileID)
}
func (m *MockDB) GetRegistrationByID(registrationID string) (*db.RegistrationRecord, error) {
	return m.GetRegistrationByIDFunc(registrationID)
}
func (m *MockDB) UpdateRepresentativesByVatID(vatID string, rep *db.RegistrationRecord) error {
	return m.UpdateRepresentativesByVatIDFunc(vatID, rep)
}

type MockMail struct {
	SendVerificationCodeFunc func(email string, code string) error
	SendWelcomeEmailFunc     func(reg *db.RegistrationRecord) error
	SendIssuerErrorFunc      func(reg *db.RegistrationRecord, payload string, errorMsg string) error
}

func (m *MockMail) SendVerificationCode(email string, code string) error {
	return m.SendVerificationCodeFunc(email, code)
}
func (m *MockMail) SendWelcomeEmail(reg *db.RegistrationRecord) error {
	return m.SendWelcomeEmailFunc(reg)
}
func (m *MockMail) SendIssuerError(reg *db.RegistrationRecord, payload string, errorMsg string) error {
	return m.SendIssuerErrorFunc(reg, payload, errorMsg)
}

type MockIssuance struct {
	GetAccessTokenFunc           func(context.Context) (string, error)
	TMFGetOrganizationByELSIFunc func(context context.Context, accessToken string, elsi string) ([]credissuance.Organization, error)
	TMFDeleteOrganizationFunc    func(context context.Context, accessToken string, id string) error
	LEARIssuanceRequestFunc      func(context context.Context, accessToken string, learCredData *credissuance.LEARIssuanceRequestBody) ([]byte, error)
	TMFCreateOrganizationFunc    func(context context.Context, accessToken string, org *credissuance.Organization_Create) (*credissuance.Organization, error)
	TMFUpdateOrganizationFunc    func(context context.Context, accessToken string, id string, org *credissuance.Organization_Update) (*credissuance.Organization, error)
}

func (m *MockIssuance) GetAccessToken(ctx context.Context) (string, error) {
	return m.GetAccessTokenFunc(ctx)
}
func (m *MockIssuance) TMFGetOrganizationByELSI(context context.Context, accessToken string, elsi string) ([]credissuance.Organization, error) {
	return m.TMFGetOrganizationByELSIFunc(context, accessToken, elsi)
}
func (m *MockIssuance) TMFDeleteOrganization(context context.Context, accessToken string, id string) error {
	return m.TMFDeleteOrganizationFunc(context, accessToken, id)
}
func (m *MockIssuance) LEARIssuanceRequest(context context.Context, accessToken string, learCredData *credissuance.LEARIssuanceRequestBody) ([]byte, error) {
	return m.LEARIssuanceRequestFunc(context, accessToken, learCredData)
}
func (m *MockIssuance) TMFCreateOrganization(context context.Context, accessToken string, org *credissuance.Organization_Create) (*credissuance.Organization, error) {
	return m.TMFCreateOrganizationFunc(context, accessToken, org)
}
func (m *MockIssuance) TMFUpdateOrganization(context context.Context, accessToken string, id string, org *credissuance.Organization_Update) (*credissuance.Organization, error) {
	return m.TMFUpdateOrganizationFunc(context, accessToken, id, org)
}
