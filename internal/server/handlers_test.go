package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hesusruiz/onboardng/credissuance"
	"github.com/hesusruiz/onboardng/internal/configuration"
	"github.com/hesusruiz/onboardng/internal/db"
	"github.com/stretchr/testify/assert"
)

func TestHandleValidateEmail(t *testing.T) {

	mockMail := &MockMail{}
	s := &Server{
		Mail:              mockMail,
		EmailRateLimiter:  make(map[string]*RateLimitEntry),
		VerificationCodes: make(map[string]*VerificationCodeEntry),
	}

	t.Run("Method Not Allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/validate-email", nil)
		w := httptest.NewRecorder()
		s.HandleSendEmailValidationCode(w, req)
		assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	})

	t.Run("Missing CSRF Header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/validate-email", nil)
		w := httptest.NewRecorder()
		s.HandleSendEmailValidationCode(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("Invalid Email", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"email": "invalid-email"})
		req := httptest.NewRequest(http.MethodPost, "/api/validate-email", bytes.NewBuffer(body))
		req.Header.Set("X-Requested-With", "XMLHttpRequest")
		w := httptest.NewRecorder()
		s.HandleSendEmailValidationCode(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp APIResponse
		json.NewDecoder(w.Body).Decode(&resp)
		assert.False(t, resp.Success)
		assert.Equal(t, "A valid email is required", resp.Message)
	})

	t.Run("Success", func(t *testing.T) {
		email := "test@example.com"
		body, _ := json.Marshal(map[string]string{"email": email})
		req := httptest.NewRequest(http.MethodPost, "/api/validate-email", bytes.NewBuffer(body))
		req.Header.Set("X-Requested-With", "XMLHttpRequest")
		w := httptest.NewRecorder()

		mockMail.SendVerificationCodeFunc = func(e string, code string) error {
			assert.Equal(t, email, e)
			assert.Len(t, code, 6)
			return nil
		}

		s.HandleSendEmailValidationCode(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		json.NewDecoder(w.Body).Decode(&resp)
		assert.True(t, resp.Success)
		assert.Equal(t, "Validation code sent to your email", resp.Message)

		// Map should contain the code
		dataMap := resp.Data.(map[string]any)
		code := dataMap["code"].(string)

		// Verify code is stored in server
		s.CodesMu.RLock()
		storedCode, exists := s.VerificationCodes[email]
		s.CodesMu.RUnlock()
		assert.True(t, exists)
		assert.Equal(t, code, storedCode.Code)
	})
}

func TestHandleVerifyCode(t *testing.T) {
	s := &Server{
		VerificationCodes: make(map[string]*VerificationCodeEntry),
		DB: &MockDB{
			GetRegistrationByEmailFunc: func(email string) (*db.RegistrationRecord, error) {
				return nil, fmt.Errorf("not found")
			},
		},
	}

	t.Run("Invalid Code", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"email": "test@example.com", "code": "123456"})
		req := httptest.NewRequest(http.MethodPost, "/api/verify-code", bytes.NewBuffer(body))
		req.Header.Set("X-Requested-With", "XMLHttpRequest")
		w := httptest.NewRecorder()

		s.HandleValidateEmailCode(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Success", func(t *testing.T) {
		email := "test@example.com"
		code := "654321"
		s.StoreVerificationCode(email, code)

		body, _ := json.Marshal(map[string]string{"email": email, "code": code})
		req := httptest.NewRequest(http.MethodPost, "/api/verify-code", bytes.NewBuffer(body))
		req.Header.Set("X-Requested-With", "XMLHttpRequest")
		w := httptest.NewRecorder()

		s.HandleValidateEmailCode(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Brute Force Protection", func(t *testing.T) {
		email := "brute@example.com"
		code := "111111"
		s.StoreVerificationCode(email, code)

		wrongBody, _ := json.Marshal(map[string]string{"email": email, "code": "wrong"})

		// 1st attempt
		req1 := httptest.NewRequest(http.MethodPost, "/api/verify-code", bytes.NewBuffer(wrongBody))
		req1.Header.Set("X-Requested-With", "XMLHttpRequest")
		w1 := httptest.NewRecorder()
		s.HandleValidateEmailCode(w1, req1)
		assert.Equal(t, http.StatusBadRequest, w1.Code)

		// 2nd attempt
		req2 := httptest.NewRequest(http.MethodPost, "/api/verify-code", bytes.NewBuffer(wrongBody))
		req2.Header.Set("X-Requested-With", "XMLHttpRequest")
		w2 := httptest.NewRecorder()
		s.HandleValidateEmailCode(w2, req2)
		assert.Equal(t, http.StatusBadRequest, w2.Code)

		// 3rd attempt - should fail and delete the code
		req3 := httptest.NewRequest(http.MethodPost, "/api/verify-code", bytes.NewBuffer(wrongBody))
		req3.Header.Set("X-Requested-With", "XMLHttpRequest")
		w3 := httptest.NewRecorder()
		s.HandleValidateEmailCode(w3, req3)
		assert.Equal(t, http.StatusBadRequest, w3.Code)

		// 4th attempt with CORRECT code - should fail because code was deleted
		correctBody, _ := json.Marshal(map[string]string{"email": email, "code": code})
		reqCorrect := httptest.NewRequest(http.MethodPost, "/api/verify-code", bytes.NewBuffer(correctBody))
		reqCorrect.Header.Set("X-Requested-With", "XMLHttpRequest")
		w4 := httptest.NewRecorder()
		s.HandleValidateEmailCode(w4, reqCorrect)
		assert.Equal(t, http.StatusBadRequest, w4.Code)
	})
}

func TestHandleRegister(t *testing.T) {
	mockDB := &MockDB{
		GetRegistrationByEmailOrVatIDFunc: func(email, vatID string) (*db.RegistrationRecord, error) {
			return nil, fmt.Errorf("not found")
		},
	}
	mockMail := &MockMail{}
	mockIssuance := &MockIssuance{}
	s := &Server{
		Runtime:           configuration.Development,
		DB:                mockDB,
		Mail:              mockMail,
		Issuer:            mockIssuance,
		VerificationCodes: make(map[string]*VerificationCodeEntry),
	}

	validReq := RegistrationRequest{
		FirstName:     "John",
		LastName:      "Doe",
		CompanyName:   "ACME Corp",
		Country:       "ES",
		VatId:         "ESB12345678",
		StreetAddress: "Main St 1",
		City:          "Madrid",
		PostalCode:    "28001",
		Email:         "john@example.com",
		Code:          "123456",
	}

	t.Run("Invalid Body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/register", strings.NewReader("invalid"))
		req.Header.Set("X-Requested-With", "XMLHttpRequest")
		w := httptest.NewRecorder()
		s.HandleRegister(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Honeypot Bot", func(t *testing.T) {
		reqData := validReq
		reqData.Website = "http://bot.com"
		body, _ := json.Marshal(reqData)
		req := httptest.NewRequest(http.MethodPost, "/api/register", bytes.NewBuffer(body))
		req.Header.Set("X-Requested-With", "XMLHttpRequest")
		w := httptest.NewRecorder()
		s.HandleRegister(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Invalid Verification Code", func(t *testing.T) {
		body, _ := json.Marshal(validReq)
		req := httptest.NewRequest(http.MethodPost, "/api/register", bytes.NewBuffer(body))
		req.Header.Set("X-Requested-With", "XMLHttpRequest")
		w := httptest.NewRecorder()
		s.HandleRegister(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp APIResponse
		json.NewDecoder(w.Body).Decode(&resp)
		assert.Equal(t, "Invalid or expired verification code", resp.Message)
	})

	t.Run("Full Success Flow (Dev Mode)", func(t *testing.T) {
		email := validReq.Email
		code := validReq.Code
		s.StoreVerificationCode(email, code)

		body, _ := json.Marshal(validReq)
		req := httptest.NewRequest(http.MethodPost, "/api/register", bytes.NewBuffer(body))
		req.Header.Set("X-Requested-With", "XMLHttpRequest")
		w := httptest.NewRecorder()

		// Setup mocks
		mockIssuance.GetAccessTokenFunc = func(context.Context) (string, error) { return "token", nil }
		mockIssuance.TMFGetOrganizationByELSIFunc = func(context context.Context, token, elsi string) ([]credissuance.Organization, error) {
			return []credissuance.Organization{{ID: "org1"}}, nil
		}
		mockIssuance.TMFDeleteOrganizationFunc = func(context context.Context, token, id string) error { return nil }
		mockDB.SaveRegistrationFunc = func(reg *db.RegistrationRecord) error { return nil }
		mockMail.SendWelcomeEmailFunc = func(reg *db.RegistrationRecord) error { return nil }
		mockIssuance.LEARIssuanceRequestFunc = func(context context.Context, token string, data *credissuance.LEARIssuanceRequestBody) ([]byte, error) {
			return []byte("ok"), nil
		}
		mockDB.UpdateRegistrationStatusFunc = func(reg *db.RegistrationRecord) error { return nil }
		mockIssuance.TMFCreateOrganizationFunc = func(context context.Context, token string, org *credissuance.Organization_Create) (*credissuance.Organization, error) {
			return &credissuance.Organization{ID: "neworg"}, nil
		}
		mockIssuance.TMFUpdateOrganizationFunc = func(context context.Context, token string, id string, org *credissuance.Organization_Update) (*credissuance.Organization, error) {
			return &credissuance.Organization{ID: id}, nil
		}

		s.HandleRegister(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp APIResponse
		json.NewDecoder(w.Body).Decode(&resp)
		assert.True(t, resp.Success)

		// Code should still exist because user removed deletion in HandleRegister
		s.CodesMu.RLock()
		_, exists := s.VerificationCodes[email]
		s.CodesMu.RUnlock()
		assert.True(t, exists)
	})

	t.Run("Production Conflict", func(t *testing.T) {
		s.Runtime = configuration.Production
		email := validReq.Email
		code := validReq.Code
		s.StoreVerificationCode(email, code)

		body, _ := json.Marshal(validReq)
		req := httptest.NewRequest(http.MethodPost, "/api/register", bytes.NewBuffer(body))
		req.Header.Set("X-Requested-With", "XMLHttpRequest")
		w := httptest.NewRecorder()

		mockIssuance.GetAccessTokenFunc = func(context.Context) (string, error) { return "token", nil }
		mockDB.GetRegistrationByEmailOrVatIDFunc = func(email, vatID string) (*db.RegistrationRecord, error) {
			return &db.RegistrationRecord{
				Email: email,
				VatID: "DIFFERENT",
			}, nil
		}

		s.HandleRegister(w, req)
		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("Access Token Failure", func(t *testing.T) {
		mockDB.GetRegistrationByEmailOrVatIDFunc = func(email, vatID string) (*db.RegistrationRecord, error) {
			return nil, fmt.Errorf("not found")
		}
		s.Runtime = configuration.Development
		s.StoreVerificationCode(validReq.Email, validReq.Code)
		body, _ := json.Marshal(validReq)
		req := httptest.NewRequest(http.MethodPost, "/api/register", bytes.NewBuffer(body))
		req.Header.Set("X-Requested-With", "XMLHttpRequest")
		w := httptest.NewRecorder()

		mockIssuance.GetAccessTokenFunc = func(context.Context) (string, error) { return "", fmt.Errorf("token error") }

		s.HandleRegister(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("DB Save Failure", func(t *testing.T) {
		s.StoreVerificationCode(validReq.Email, validReq.Code)
		body, _ := json.Marshal(validReq)
		req := httptest.NewRequest(http.MethodPost, "/api/register", bytes.NewBuffer(body))
		req.Header.Set("X-Requested-With", "XMLHttpRequest")
		w := httptest.NewRecorder()

		mockIssuance.GetAccessTokenFunc = func(context.Context) (string, error) { return "token", nil }
		mockIssuance.TMFGetOrganizationByELSIFunc = func(context context.Context, token, elsi string) ([]credissuance.Organization, error) {
			return nil, nil
		}
		mockDB.SaveRegistrationFunc = func(reg *db.RegistrationRecord) error { return fmt.Errorf("db error") }

		s.HandleRegister(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestHandleHealth(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	s.HandleHealth(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
