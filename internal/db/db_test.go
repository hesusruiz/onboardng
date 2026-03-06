package db

import (
	"os"
	"testing"

	"github.com/hesusruiz/onboardng/internal/configuration"
)

func TestRegistrationConstraints(t *testing.T) {
	dbPath := "data/onboarding_test.db"
	os.Remove(dbPath) // Start fresh
	defer os.Remove(dbPath)

	// Test with Production environment to check strict constraints
	svc, err := NewService(configuration.Production, dbPath)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}
	defer svc.Close()

	reg1 := &RegistrationRecord{
		RegistrationID: "reg-001",
		Email:          "user@example.com",
		VatID:          "VAT123",
		CompanyName:    "Company A",
	}

	// 1. First registration should succeed
	err = svc.SaveRegistration(reg1)
	if err != nil {
		t.Fatalf("first registration failed: %v", err)
	}

	// 2. Registering same company (VAT) with different email should FAIL in production
	reg2 := &RegistrationRecord{
		RegistrationID: "reg-002",
		Email:          "other@example.com",
		VatID:          "VAT123",
		CompanyName:    "Company A Again",
	}
	err = svc.SaveRegistration(reg2)
	if err == nil {
		t.Errorf("expected error for duplicate VAT ID in production, got nil")
	}

	// 3. Registering different company (VAT) with SAME email should SUCCEED
	reg3 := &RegistrationRecord{
		RegistrationID: "reg-003",
		Email:          "user@example.com",
		VatID:          "VAT456",
		CompanyName:    "Company B",
	}
	err = svc.SaveRegistration(reg3)
	if err != nil {
		t.Errorf("failed to register different company with same email: %v", err)
	}

	// Test with Development environment to check "one email per company" update logic
	svcDev, err := NewService(configuration.Development, dbPath)
	if err != nil {
		t.Fatalf("failed to create dev service: %v", err)
	}
	defer svcDev.Close()

	// 4. Update Company A (VAT123) with new email in Dev mode
	reg4 := &RegistrationRecord{
		RegistrationID: "reg-004",
		Email:          "new-email@example.com",
		VatID:          "VAT123",
		CompanyName:    "Company A Refined",
	}
	err = svcDev.SaveRegistration(reg4)
	if err != nil {
		t.Fatalf("failed to update company in dev mode: %v", err)
	}

	// Verify the update
	updatedReg, err := svcDev.GetRegistrationByVatID("VAT123")
	if err != nil {
		t.Fatalf("failed to get updated registration: %v", err)
	}
	if updatedReg.Email != "new-email@example.com" {
		t.Errorf("expected email to be updated to new-email@example.com, got %s", updatedReg.Email)
	}
}
