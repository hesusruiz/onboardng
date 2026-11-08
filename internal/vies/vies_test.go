package vies

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckVAT(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "vies_test.db")
	defer os.Remove(dbPath)

	svc, err := NewService(dbPath)
	require.NoError(t, err)
	defer svc.Close()

	t.Run("Valid VAT", func(t *testing.T) {
		resp, err := svc.CheckVAT(context.Background(), false, "ES", "G87936159")
		if assert.NoError(t, err) && assert.NotNil(t, resp) {
			assert.True(t, resp.Valid)
			assert.Equal(t, "ES", resp.CountryCode)
			assert.Equal(t, "G87936159", resp.VATNumber)
			assert.False(t, resp.RequestDate.IsZero())
		}

		// Set a known recent timestamp in the cache to verify the second call comes from SQLite
		cachedTime := time.Now().UTC().Add(-24 * time.Hour).Truncate(time.Second)
		_, err = svc.db.Exec(
			`UPDATE vat_validations SET request_date = ? WHERE country_code = ? AND vat_number = ?`,
			cachedTime, "ES", "G87936159",
		)
		require.NoError(t, err)

		// Second call should be served from SQLite cache with the cached RequestDate
		respCached, err := svc.CheckVAT(context.Background(), false, "ES", "G87936159")
		if assert.NoError(t, err) && assert.NotNil(t, respCached) {
			assert.True(t, respCached.Valid)
			assert.Equal(t, "ES", respCached.CountryCode)
			assert.Equal(t, "G87936159", respCached.VATNumber)
			assert.True(t, respCached.RequestDate.Equal(cachedTime))
		}
	})

	t.Run("Invalid VAT", func(t *testing.T) {
		resp, err := svc.CheckVAT(context.Background(), false, "ES", "G87936159FF")
		if assert.NoError(t, err) && assert.NotNil(t, resp) {
			assert.False(t, resp.Valid)
			assert.Equal(t, "ES", resp.CountryCode)
			assert.Equal(t, "G87936159FF", resp.VATNumber)
		}

		// Second call for invalid VAT should also be served from SQLite cache
		respCached, err := svc.CheckVAT(context.Background(), false, "ES", "G87936159FF")
		if assert.NoError(t, err) && assert.NotNil(t, respCached) {
			assert.False(t, respCached.Valid)
			assert.Equal(t, "ES", respCached.CountryCode)
			assert.Equal(t, "G87936159FF", respCached.VATNumber)
		}
	})

	t.Run("Expired Entry Refreshes From API", func(t *testing.T) {
		// Backdate the cached entry for ES / G87936159 to more than 1 month ago
		expiredDate := time.Now().AddDate(0, -1, -2)
		_, err := svc.db.Exec(
			`UPDATE vat_validations SET request_date = ? WHERE country_code = ? AND vat_number = ?`,
			expiredDate, "ES", "G87936159",
		)
		require.NoError(t, err)

		resp, err := svc.CheckVAT(context.Background(), false, "ES", "G87936159")
		if assert.NoError(t, err) && assert.NotNil(t, resp) {
			assert.True(t, resp.Valid)
			assert.True(t, resp.RequestDate.After(expiredDate))
		}
	})
}
