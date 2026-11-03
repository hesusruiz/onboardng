package vies

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestCheckVAT(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "vies_test.db")
	defer os.Remove(dbPath)

	calls := 0
	client := &http.Client{
		Timeout: 15 * time.Second,
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			calls++
			return http.DefaultTransport.RoundTrip(req)
		}),
	}

	svc, err := NewService(dbPath, client)
	require.NoError(t, err)
	defer svc.Close()

	t.Run("Valid VAT", func(t *testing.T) {
		resp, err := svc.CheckVAT(context.Background(), "ES", "G87936159")
		if assert.NoError(t, err) && assert.NotNil(t, resp) {
			assert.True(t, resp.Valid)
			assert.Equal(t, "ES", resp.CountryCode)
			assert.Equal(t, "G87936159", resp.VATNumber)
			assert.False(t, resp.RequestDate.IsZero())
		}
		assert.Equal(t, 1, calls)

		// Second call should be served from SQLite cache without network call
		respCached, err := svc.CheckVAT(context.Background(), "ES", "G87936159")
		if assert.NoError(t, err) && assert.NotNil(t, respCached) {
			assert.True(t, respCached.Valid)
			assert.Equal(t, "ES", respCached.CountryCode)
			assert.Equal(t, "G87936159", respCached.VATNumber)
		}
		assert.Equal(t, 1, calls)
	})

	t.Run("Invalid VAT", func(t *testing.T) {
		resp, err := svc.CheckVAT(context.Background(), "ES", "G87936159FF")
		if assert.NoError(t, err) && assert.NotNil(t, resp) {
			assert.False(t, resp.Valid)
			assert.Equal(t, "ES", resp.CountryCode)
			assert.Equal(t, "G87936159FF", resp.VATNumber)
		}
		assert.Equal(t, 2, calls)

		// Second call for invalid VAT should also be served from SQLite cache
		respCached, err := svc.CheckVAT(context.Background(), "ES", "G87936159FF")
		if assert.NoError(t, err) && assert.NotNil(t, respCached) {
			assert.False(t, respCached.Valid)
		}
		assert.Equal(t, 2, calls)
	})

	t.Run("Expired Entry Refreshes From API", func(t *testing.T) {
		// Backdate the cached entry for ES / G87936159 to more than 1 month ago
		expiredDate := time.Now().AddDate(0, -1, -2)
		_, err := svc.db.Exec(
			`UPDATE vat_validations SET request_date = ? WHERE country_code = ? AND vat_number = ?`,
			expiredDate, "ES", "G87936159",
		)
		require.NoError(t, err)

		resp, err := svc.CheckVAT(context.Background(), "ES", "G87936159")
		if assert.NoError(t, err) && assert.NotNil(t, resp) {
			assert.True(t, resp.Valid)
			assert.True(t, resp.RequestDate.After(expiredDate))
		}
		assert.Equal(t, 3, calls)
	})
}
