package main

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/hesusruiz/onboardng/internal/configuration"
	"github.com/hesusruiz/onboardng/internal/db"
)

func main() {
	// Initialize Database service
	dbPath := "data/onboarding.db"
	dbService, err := db.NewDBService(configuration.Development, dbPath)
	if err != nil {
		slog.Error("❌ Error initializing database service", "error", err)
		os.Exit(1)
	}
	defer dbService.Close()

	fmt.Println("🚀 Starting to generate test records...")

	for i := 1; i <= 20; i++ {
		regID := uuid.New().String()
		email := fmt.Sprintf("testuser%d@example.com", i)
		vatID := fmt.Sprintf("VAT%d", 1000+i)

		// Simulate sequential steps with random interruptions
		// 0: none, 1: notified, 2: issued, 3: tmf_registered
		progress := i % 4
		notified := progress >= 1
		issued := progress >= 2
		tmfRegistered := progress >= 3

		reg := &db.RegistrationRecord{
			RegistrationID: regID,
			Email:          email,
			FirstName:      fmt.Sprintf("First%d", i),
			LastName:       fmt.Sprintf("Last%d", i),
			CompanyName:    fmt.Sprintf("Company%d", i),
			Country:        "ES",
			VatID:          vatID,
			StreetAddress:  fmt.Sprintf("Street %d", i),
			PostalCode:     fmt.Sprintf("2800%d", i%10),
			Notified:       notified,
			Issued:         issued,
			TMFRegistered:  tmfRegistered,
		}

		err = dbService.SaveRegistration(reg)
		if err != nil {
			slog.Error("Failed to save registration", "email", email, "error", err)
			continue
		}

		// SaveRegistration resets flags. We need to explicitly set them back and update.
		reg.Notified = notified
		reg.Issued = issued
		reg.TMFRegistered = tmfRegistered
		err = dbService.UpdateRegistrationStatus(reg)
		if err != nil {
			slog.Error("Failed to update registration status", "email", email, "error", err)
			continue
		}

		// Generate Some Errors
		// Only registrations that have not completed all steps should have errors
		if !tmfRegistered {
			// Add random number of errors
			numErrors := (i % 2) + 1
			for j := 1; j <= numErrors; j++ {
				logEntry := &db.RegistrationLog{
					RegistrationID: regID,
					Email:          email,
					VatID:          vatID,
					Type:           "error",
					Message:        fmt.Sprintf("Sample error %d during processing for user %d", j, i),
				}
				_ = dbService.SaveRegistrationLog(logEntry)
			}
		}

		// Generate Some Files
		if i%2 == 0 {
			numFiles := (i % 3) + 1
			for j := 1; j <= numFiles; j++ {
				fileID := uuid.New().String()
				f := &db.RegistrationFile{
					FileID:         fileID,
					RegistrationID: regID,
					VatID:          vatID,
					Name:           fmt.Sprintf("document_%d_%d.txt", i, j),
					MimeType:       "text/plain",
					Size:           int64(1024 * j),
					Status:         "uploaded",
					Content:        []byte(fmt.Sprintf("This is the content of document %d for user %d.", j, i)),
				}
				_ = dbService.SaveRegistrationFile(f)
			}
		}

		// Simulate time difference
		time.Sleep(10 * time.Millisecond)
	}

	fmt.Println("✅ Successfully generated test records in", dbPath)
}
