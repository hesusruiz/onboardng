package server

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/hesusruiz/onboardng/internal/db"
	"github.com/hesusruiz/utils/errl"
)

// RemindSellers is a maintenance function that is called by the maintenance service to remind sellers to complete the onboarding process
// It is scheduled to run every Monday at 8:00AM.
func (s *Server) RemindSellers(ctx context.Context) error {

	// We iterate throigh all registrations one page (100 items) at a time.
	limit := 100
	offset := 0
	for {
		// Retrieve one page at the current offset
		registrations, _, err := s.DB.GetRegistrations(limit, offset)
		if err != nil {
			return errl.Errorf("failed to get pending registrations: %v", err)
		}

		// Check each registration
		for _, reg := range registrations {

			// We are interested only in Sellers
			if reg.Role != "Seller" {
				continue
			}

			// Only registrations which did not complete Step 2: LEARCompleted is false
			if reg.LEARCompleted {
				continue
			}

			// And only registrations that were updated more than seven days ago
			if time.Since(reg.UpdatedAt) < 7*24*time.Hour {
				continue
			}

			// We will send a reminder email
			err := s.Mail.SendReminderEmail(reg)
			if err != nil {
				slog.Error("Error sending reminder email", "error", err)
				// Record the error in the database
				_ = s.DB.SaveRegistrationLog(&db.RegistrationLog{
					RegistrationID: reg.RegistrationID,
					Email:          reg.Email,
					VatID:          reg.VatID,
					Type:           "error",
					Message:        fmt.Sprintf("Reminder email error: %v", err.Error()),
				})
			} else {
				slog.Info("Reminder email sent", "email", reg.Email)
			}

		}

		// If the number of registrations is less than the limit, we have retrieved all registrations
		if len(registrations) < limit {
			break
		}
		offset += limit
	}

	return nil
}
