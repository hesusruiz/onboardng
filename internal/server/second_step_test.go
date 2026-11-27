package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hesusruiz/onboardng/internal/db"
	"github.com/hesusruiz/pdf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleAppointmentLetterPDF(t *testing.T) {
	newTestServer := func(mockDB *MockDB) *Server {
		return &Server{
			DB:                mockDB,
			EmailRateLimiter:  make(map[string]*RateLimitEntry),
			VerificationCodes: make(map[string]*VerificationCodeEntry),
		}
	}

	decodeAPIResp := func(t *testing.T, body []byte) APIResponse {
		t.Helper()
		var resp APIResponse
		require.NoError(t, json.Unmarshal(body, &resp))
		return resp
	}

	sampleRecord := func() *db.RegistrationRecord {
		return &db.RegistrationRecord{
			RegistrationID:   "20261125-12345678",
			Email:            "user@example.com",
			FirstName:        "Alice",
			LastName:         "Smith",
			CompanyName:      "Acme Cloud SL",
			Country:          "ES",
			VatID:            "VATES-B12345678",
			StreetAddress:    "Calle Mayor 10",
			City:             "Madrid",
			PostalCode:       "28013",
			LRFirstName:      "Laura",
			LRLastName:       "Gomez",
			LREmail:          "laura.gomez@example.com",
			LRCountry:        "ES",
			LRIdCard:         "12345678A",
			LEARFirstName:    "Carlos",
			LEARLastName:     "Ruiz",
			LEAREmail:        "carlos.ruiz@example.com",
			LEARCountry:      "ES",
			LEARAddress:      "Paseo de la Castellana 50, 28046 Madrid",
			LEARIdCard:       "87654321B",
			LEARMobileNumber: "+34600112233",
		}
	}

	t.Run("Invalid JSON body", func(t *testing.T) {
		s := newTestServer(&MockDB{})
		req := httptest.NewRequest(http.MethodPost, "/api/appointment-letter", strings.NewReader("{invalid"))
		w := httptest.NewRecorder()

		s.HandleAppointmentLetterPDF(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		resp := decodeAPIResp(t, w.Body.Bytes())
		assert.False(t, resp.Success)
		assert.Equal(t, "Invalid request body", resp.Message)
	})

	missingFieldCases := []struct {
		name    string
		payload AppointmentLetterRequest
		wantMsg string
	}{
		{
			name:    "Missing vatId",
			payload: AppointmentLetterRequest{Email: "user@example.com", Code: "123456"},
			wantMsg: "VAT ID is required",
		},
		{
			name:    "Missing email",
			payload: AppointmentLetterRequest{VatID: "B12345678", Code: "123456"},
			wantMsg: "Email is required",
		},
		{
			name:    "Missing code",
			payload: AppointmentLetterRequest{VatID: "B12345678", Email: "user@example.com"},
			wantMsg: "Verification code is required",
		},
	}

	for _, tc := range missingFieldCases {
		t.Run(tc.name, func(t *testing.T) {
			s := newTestServer(&MockDB{})
			body, err := json.Marshal(tc.payload)
			require.NoError(t, err)
			req := httptest.NewRequest(http.MethodPost, "/api/appointment-letter", bytes.NewReader(body))
			w := httptest.NewRecorder()

			s.HandleAppointmentLetterPDF(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)
			resp := decodeAPIResp(t, w.Body.Bytes())
			assert.False(t, resp.Success)
			assert.Equal(t, tc.wantMsg, resp.Message)
		})
	}

	t.Run("Invalid verification code", func(t *testing.T) {
		s := newTestServer(&MockDB{})
		s.StoreVerificationCode("user@example.com", "123456")

		body, err := json.Marshal(AppointmentLetterRequest{
			VatID: "B12345678",
			Email: "user@example.com",
			Code:  "999999",
		})
		require.NoError(t, err)
		req := httptest.NewRequest(http.MethodPost, "/api/appointment-letter", bytes.NewReader(body))
		w := httptest.NewRecorder()

		s.HandleAppointmentLetterPDF(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		resp := decodeAPIResp(t, w.Body.Bytes())
		assert.False(t, resp.Success)
		assert.Equal(t, "Invalid or expired verification code", resp.Message)
	})

	t.Run("Expired verification code", func(t *testing.T) {
		s := newTestServer(&MockDB{})
		s.CodesMu.Lock()
		s.VerificationCodes["user@example.com"] = &VerificationCodeEntry{
			Code:      "123456",
			CreatedAt: time.Now().Add(-2 * time.Hour),
		}
		s.CodesMu.Unlock()

		body, err := json.Marshal(AppointmentLetterRequest{
			VatID: "B12345678",
			Email: "user@example.com",
			Code:  "123456",
		})
		require.NoError(t, err)
		req := httptest.NewRequest(http.MethodPost, "/api/appointment-letter", bytes.NewReader(body))
		w := httptest.NewRecorder()

		s.HandleAppointmentLetterPDF(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		resp := decodeAPIResp(t, w.Body.Bytes())
		assert.False(t, resp.Success)
		assert.Equal(t, "Invalid or expired verification code", resp.Message)
	})

	t.Run("Registration not found", func(t *testing.T) {
		mockDB := &MockDB{
			GetRegistrationByEmailFunc: func(email string) (*db.RegistrationRecord, error) {
				return nil, errors.New("not found")
			},
		}
		s := newTestServer(mockDB)
		s.StoreVerificationCode("user@example.com", "123456")

		body, err := json.Marshal(AppointmentLetterRequest{
			VatID: "B12345678",
			Email: "user@example.com",
			Code:  "123456",
		})
		require.NoError(t, err)
		req := httptest.NewRequest(http.MethodPost, "/api/appointment-letter", bytes.NewReader(body))
		w := httptest.NewRecorder()

		s.HandleAppointmentLetterPDF(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		resp := decodeAPIResp(t, w.Body.Bytes())
		assert.False(t, resp.Success)
		assert.Equal(t, "Registration not found", resp.Message)
	})

	t.Run("Email mismatch", func(t *testing.T) {
		rec := sampleRecord()
		rec.Email = "other@example.com"
		mockDB := &MockDB{
			GetRegistrationByEmailFunc: func(email string) (*db.RegistrationRecord, error) {
				return rec, nil
			},
		}
		s := newTestServer(mockDB)
		s.StoreVerificationCode("user@example.com", "123456")

		body, err := json.Marshal(AppointmentLetterRequest{
			VatID: "B12345678",
			Email: "user@example.com",
			Code:  "123456",
		})
		require.NoError(t, err)
		req := httptest.NewRequest(http.MethodPost, "/api/appointment-letter", bytes.NewReader(body))
		w := httptest.NewRecorder()

		s.HandleAppointmentLetterPDF(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		resp := decodeAPIResp(t, w.Body.Bytes())
		assert.False(t, resp.Success)
		assert.Equal(t, "Email does not match the registration", resp.Message)
	})

	t.Run("VAT ID mismatch", func(t *testing.T) {
		rec := sampleRecord()
		rec.Country = "ES"
		rec.VatID = "VATES-99999999"
		mockDB := &MockDB{
			GetRegistrationByEmailFunc: func(email string) (*db.RegistrationRecord, error) {
				return rec, nil
			},
		}
		s := newTestServer(mockDB)
		s.StoreVerificationCode("user@example.com", "123456")

		body, err := json.Marshal(AppointmentLetterRequest{
			VatID: "12345678",
			Email: "user@example.com",
			Code:  "123456",
		})
		require.NoError(t, err)
		req := httptest.NewRequest(http.MethodPost, "/api/appointment-letter", bytes.NewReader(body))
		w := httptest.NewRecorder()

		s.HandleAppointmentLetterPDF(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		resp := decodeAPIResp(t, w.Body.Bytes())
		assert.False(t, resp.Success)
		assert.Equal(t, "VATID does not match the registration", resp.Message)
	})

	t.Run("Success with unprefixed VAT ID", func(t *testing.T) {
		rec := sampleRecord()
		mockDB := &MockDB{
			GetRegistrationByEmailFunc: func(email string) (*db.RegistrationRecord, error) {
				assert.Equal(t, "user@example.com", email)
				return rec, nil
			},
		}
		s := newTestServer(mockDB)
		s.StoreVerificationCode("user@example.com", "123456")

		body, err := json.Marshal(AppointmentLetterRequest{
			VatID: "B12345678",
			Email: "user@example.com",
			Code:  "123456",
		})
		require.NoError(t, err)
		req := httptest.NewRequest(http.MethodPost, "/api/appointment-letter", bytes.NewReader(body))
		w := httptest.NewRecorder()

		s.HandleAppointmentLetterPDF(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/pdf", w.Header().Get("Content-Type"))
		assert.Equal(t, "attachment; filename=appointment_letter.pdf", w.Header().Get("Content-Disposition"))
		pdfBytes := w.Body.Bytes()
		require.True(t, bytes.HasPrefix(pdfBytes, []byte("%PDF-")), "expected PDF header")

		res, err := pdf.Extract(pdfBytes, nil)
		require.NoError(t, err)
		assert.Equal(t, pdf.SourceAttachment, res.Source)
		assert.Empty(t, res.VisualMismatches)

		var extracted appointmentData
		require.NoError(t, json.Unmarshal(res.JSON, &extracted))
		assert.Equal(t, time.Now().Format("2006-01-02"), extracted.Date)
		assert.Equal(t, legalRepData{
			Name:     rec.LRFirstName,
			Surname:  rec.LRLastName,
			IDNumber: rec.LRIdCard,
			Email:    rec.LREmail,
		}, extracted.LegalRep)
		assert.Equal(t, companyData{
			LegalName:         rec.CompanyName,
			RegisteredAddress: rec.StreetAddress + ", " + rec.PostalCode + " - " + rec.City + ", " + rec.Country,
			CountryCode:       rec.Country,
			TaxID:             rec.VatID,
		}, extracted.CompanyData)
		assert.Equal(t, accountAdminData{
			Name:          rec.LEARFirstName,
			Surname:       rec.LEARLastName,
			IDNumber:      rec.LEARIdCard,
			PostalAddress: rec.LEARAddress,
			Email:         rec.LEAREmail,
		}, extracted.AccountAdmin)

		// Verification code should remain stored (not consumed)
		s.CodesMu.RLock()
		_, stillExists := s.VerificationCodes["user@example.com"]
		s.CodesMu.RUnlock()
		assert.True(t, stillExists)
	})

	t.Run("Success with already-prefixed VAT ID", func(t *testing.T) {
		rec := sampleRecord()
		mockDB := &MockDB{
			GetRegistrationByEmailFunc: func(email string) (*db.RegistrationRecord, error) {
				return rec, nil
			},
		}
		s := newTestServer(mockDB)
		s.StoreVerificationCode("user@example.com", "123456")

		body, err := json.Marshal(AppointmentLetterRequest{
			VatID: "VATES-B12345678",
			Email: "user@example.com",
			Code:  "123456",
		})
		require.NoError(t, err)
		req := httptest.NewRequest(http.MethodPost, "/api/appointment-letter", bytes.NewReader(body))
		w := httptest.NewRecorder()

		s.HandleAppointmentLetterPDF(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/pdf", w.Header().Get("Content-Type"))
		pdfBytes := w.Body.Bytes()
		require.True(t, bytes.HasPrefix(pdfBytes, []byte("%PDF-")), "expected PDF header")

		res, err := pdf.Extract(pdfBytes, nil)
		require.NoError(t, err)
		assert.Empty(t, res.VisualMismatches)

		var extracted appointmentData
		require.NoError(t, json.Unmarshal(res.JSON, &extracted))
		assert.Equal(t, "VATES-B12345678", extracted.CompanyData.TaxID)
	})
}
