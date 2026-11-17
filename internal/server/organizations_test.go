package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCanonicalizeVATID(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantCountry string
		wantNumber  string
		wantErr     string
	}{
		// Canonical format and comment examples
		{name: "canonical format", input: "VATIT-55555555555", wantCountry: "IT", wantNumber: "55555555555"},
		{name: "no VAT prefix", input: "IT55555555555", wantCountry: "IT", wantNumber: "55555555555"},
		{name: "no separator", input: "VATIT55555555555", wantCountry: "IT", wantNumber: "55555555555"},
		{name: "duplicated country code", input: "VATIT-IT55555555555", wantCountry: "IT", wantNumber: "55555555555"},
		{name: "spaces everywhere", input: "VAT IT 55555555555", wantCountry: "IT", wantNumber: "55555555555"},
		{name: "hyphens everywhere", input: "VAT-IT-55555555555", wantCountry: "IT", wantNumber: "55555555555"},
		{name: "space and hyphen", input: "VAT IT-55555555555", wantCountry: "IT", wantNumber: "55555555555"},
		{name: "lowercase canonical", input: "vatit-55555555555", wantCountry: "IT", wantNumber: "55555555555"},
		{name: "lowercase space and hyphen", input: "vat it-55555555555", wantCountry: "IT", wantNumber: "55555555555"},
		{name: "lowercase hyphens", input: "vat-it-55555555555", wantCountry: "IT", wantNumber: "55555555555"},
		{name: "no VAT prefix with space", input: "IT 55555555555", wantCountry: "IT", wantNumber: "55555555555"},
		{name: "no VAT prefix with hyphen", input: "IT-55555555555", wantCountry: "IT", wantNumber: "55555555555"},
		{name: "lowercase no VAT prefix", input: "it55555555555", wantCountry: "IT", wantNumber: "55555555555"},
		{name: "lowercase no VAT prefix with space", input: "it 55555555555", wantCountry: "IT", wantNumber: "55555555555"},
		{name: "lowercase no VAT prefix with hyphen", input: "it-55555555555", wantCountry: "IT", wantNumber: "55555555555"},
		{name: "leading and trailing whitespace", input: "  vat it-55555555555  ", wantCountry: "IT", wantNumber: "55555555555"},
		{name: "Spanish alphanumeric VAT", input: "VATES-G87936159", wantCountry: "ES", wantNumber: "G87936159"},
		{name: "lowercase Spanish without VAT prefix", input: "es b12345678", wantCountry: "ES", wantNumber: "B12345678"},

		// Greek GR / EL variants
		{name: "Greece EL canonical", input: "VATEL-123456789", wantCountry: "EL", wantNumber: "123456789"},
		{name: "Greece GR canonical", input: "VATGR-123456789", wantCountry: "EL", wantNumber: "123456789"},
		{name: "Greece GR without VAT prefix", input: "GR123456789", wantCountry: "EL", wantNumber: "123456789"},
		{name: "Greece EL without VAT prefix", input: "el 123456789", wantCountry: "EL", wantNumber: "123456789"},
		{name: "Greece GR with duplicated GR", input: "VATGR-GR123456789", wantCountry: "EL", wantNumber: "123456789"},
		{name: "Greece EL with duplicated GR", input: "VATEL-GR123456789", wantCountry: "EL", wantNumber: "123456789"},
		{name: "Greece GR with duplicated EL", input: "VATGR-EL123456789", wantCountry: "EL", wantNumber: "123456789"},

		// Error cases: too short
		{name: "empty input", input: "", wantErr: "too short"},
		{name: "too short without VAT prefix", input: "IT1234", wantErr: "too short"},
		{name: "too short with VAT prefix", input: "VATIT-1234", wantErr: "too short"},
		{name: "too short after stripping duplicated country code", input: "VATIT-IT1234", wantErr: "too short"},
		{name: "too short after stripping VAT and separators", input: "VAT-123", wantErr: "too short"},

		// Error cases: invalid / non-EU country codes
		{name: "non-EU US", input: "US123456789", wantErr: "invalid EU country code"},
		{name: "non-EU GB", input: "VATGB-123456789", wantErr: "invalid EU country code"},
		{name: "non-EU EFTA CH", input: "VATCH-123456789", wantErr: "invalid EU country code"},
		{name: "unknown country code XX", input: "VATXX-123456789", wantErr: "invalid EU country code"},
		{name: "digits instead of country code", input: "123456789", wantErr: "invalid EU country code"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			countryCode, vatNumber, err := CanonicalizeVATID(tc.input)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				assert.Empty(t, countryCode)
				assert.Empty(t, vatNumber)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.wantCountry, countryCode)
			assert.Equal(t, tc.wantNumber, vatNumber)
		})
	}
}
