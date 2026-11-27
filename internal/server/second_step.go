package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/hesusruiz/utils/errl"
)

// RepresentativesRequest payload for updating LR and LEAR details
type RepresentativesRequest struct {
	VatID            string `json:"vatId"`
	Email            string `json:"email"`
	Code             string `json:"code"`
	LRFirstName      string `json:"lr_first_name"`
	LRLastName       string `json:"lr_last_name"`
	LREmail          string `json:"lr_email"`
	LRCountry        string `json:"lr_country"`
	LRIdCard         string `json:"lr_id_card"`
	LEARFirstName    string `json:"lear_first_name"`
	LEARLastName     string `json:"lear_last_name"`
	LEAREmail        string `json:"lear_email"`
	LEARCountry      string `json:"lear_country"`
	LEARAddress      string `json:"lear_address"`
	LEARIdCard       string `json:"lear_id_card"`
	LEARMobileNumber string `json:"lear_mobile_number"`
	LEARCompleted    bool   `json:"lear_completed"`
}

// HandleUpdateRepresentatives handles updating the LR and LEAR info
func (s *Server) HandleUpdateRepresentatives(w http.ResponseWriter, r *http.Request) {

	var req RepresentativesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.SendJSON(w, r, http.StatusBadRequest, false, "Invalid request body", errl.Error(err))
		return
	}
	out, _ := json.MarshalIndent(req, "", "  ")
	slog.Info("Representatives request", "request", string(out))

	if req.VatID == "" {
		s.SendJSON(w, r, http.StatusBadRequest, false, "VAT ID is required", nil)
		return
	}
	if req.Email == "" {
		s.SendJSON(w, r, http.StatusBadRequest, false, "Email is required", nil)
		return
	}
	if req.Code == "" {
		s.SendJSON(w, r, http.StatusBadRequest, false, "Verification code is required", nil)
		return
	}

	// Verify the code
	if !s.VerifyCode(req.Email, req.Code) {
		s.SendJSON(w, r, http.StatusBadRequest, false, "Invalid or expired verification code", nil)
		return
	}

	// Fetch registration and check if immutable
	current, err := s.DB.GetRegistrationByEmail(req.Email)
	if err != nil {
		s.SendJSON(w, r, http.StatusNotFound, false, "Registration not found", errl.Error(err))
		return
	}
	slog.Info("Registration found", "registration", current.String())

	// Check if the email matches the registration email
	if current.Email != req.Email {
		s.SendJSON(w, r, http.StatusForbidden, false, "Email does not match the registration", nil)
		return
	}

	// Build the VATID with the proper prefix of VATCC-
	country := current.Country

	vatPrefix := "VAT" + country + "-"

	if !strings.HasPrefix(req.VatID, vatPrefix) {
		req.VatID = vatPrefix + req.VatID
	}

	if current.VatID != req.VatID {
		s.SendJSON(w, r, http.StatusForbidden, false, "VATID does not match the registration", nil)
		return
	}

	// Check if the VATid matches the registration VATID
	if current.Approved > 0 {
		s.SendJSON(w, r, http.StatusForbidden, false, "Registration is already approved or denied and immutable", nil)
		return
	}

	// Update DB
	current.LRFirstName = req.LRFirstName
	current.LRLastName = req.LRLastName
	current.LREmail = req.LREmail
	current.LRCountry = req.LRCountry
	current.LRIdCard = req.LRIdCard
	current.LEARFirstName = req.LEARFirstName
	current.LEARLastName = req.LEARLastName
	current.LEAREmail = req.LEAREmail
	current.LEARCountry = req.LEARCountry
	current.LEARAddress = req.LEARAddress
	current.LEARIdCard = req.LEARIdCard
	current.LEARMobileNumber = req.LEARMobileNumber
	current.LEARCompleted = true
	current.Role = "Seller"

	err = s.DB.UpdateRepresentativesByVatID(req.VatID, current)
	if err != nil {
		s.SendJSON(w, r, http.StatusInternalServerError, false, "Error updating registration", errl.Error(err))
		return
	}

	if err := s.Mail.SendSecondPhaseEmail(current); err != nil {
		err = errl.Errorf("Failed to send second phase email: %w", err)
		slog.Error("Error sending second phase email", "error", err)
		// Record the error in the database
		_ = s.DB.LogRegError(current.VatID, err.Error(), current.RegistrationID, current.Email)
	} else {
		slog.Info("Second phase email sent", "email", current.Email)
		current.Notified = true
		if updateErr := s.DB.UpdateRegistrationStatus(current); updateErr != nil {
			slog.Error("Error updating registration status with email result", "error", errl.Error(updateErr))
		}
	}

	s.SendJSON(w, r, http.StatusOK, true, "Representatives updated successfully", nil)
}

type AppointmentLetterRequest struct {
	VatID string `json:"vatId"`
	Email string `json:"email"`
	Code  string `json:"code"`
}

// HandleAppointmentLetterPDF handles the LR and LEAR info
func (s *Server) HandleAppointmentLetterPDF(w http.ResponseWriter, r *http.Request) {

	var req AppointmentLetterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.SendJSON(w, r, http.StatusBadRequest, false, "Invalid request body", errl.Error(err))
		return
	}

	if req.VatID == "" {
		s.SendJSON(w, r, http.StatusBadRequest, false, "VAT ID is required", nil)
		return
	}
	if req.Email == "" {
		s.SendJSON(w, r, http.StatusBadRequest, false, "Email is required", nil)
		return
	}
	if req.Code == "" {
		s.SendJSON(w, r, http.StatusBadRequest, false, "Verification code is required", nil)
		return
	}

	// Verify the email/code pair
	if !s.VerifyCode(req.Email, req.Code) {
		s.SendJSON(w, r, http.StatusBadRequest, false, "Invalid or expired verification code", nil)
		return
	}

	// Fetch registration
	current, err := s.DB.GetRegistrationByEmail(req.Email)
	if err != nil {
		s.SendJSON(w, r, http.StatusNotFound, false, "Registration not found", errl.Error(err))
		return
	}
	slog.Info("Registration found", "registration", current.String())

	// Check if the email matches the registration email
	if current.Email != req.Email {
		s.SendJSON(w, r, http.StatusForbidden, false, "Email does not match the registration", nil)
		return
	}

	// Build the VATID with the proper prefix of VATCC-
	country := current.Country

	vatPrefix := "VAT" + country + "-"

	if !strings.HasPrefix(req.VatID, vatPrefix) {
		req.VatID = vatPrefix + req.VatID
	}

	if current.VatID != req.VatID {
		s.SendJSON(w, r, http.StatusForbidden, false, "VATID does not match the registration", nil)
		return
	}

	// Create appointment letter PDF
	pdf, err := s.generateAppointmentLetterPDF(current)
	if err != nil {
		s.SendJSON(w, r, http.StatusInternalServerError, false, "Error generating appointment letter PDF", errl.Error(err))
		return
	}

	// Send appointment letter PDF as response
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "attachment; filename=appointment_letter.pdf")
	_, err = w.Write(pdf)
	if err != nil {
		s.SendJSON(w, r, http.StatusInternalServerError, false, "Error sending appointment letter PDF", errl.Error(err))
		return
	}

}
