package views

import (
	"testing"

	"github.com/flosch/pongo2/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewViews(t *testing.T) {
	v, err := NewAppViews(true)
	require.NoError(t, err)
	require.NotNil(t, v)
}

const domeTestUrl = "https://dome-marketplace-sbx.org/"

func TestViews_ExecuteTemplate(t *testing.T) {
	v, err := NewAppViews(true)
	require.NoError(t, err)
	require.NotNil(t, v)

	t.Run("renders existing template with context and default globals", func(t *testing.T) {
		out, err := executeTemplate(v.Set, "qrcode.html.j2", pongo2.Context{
			"CredentialOfferUri": "openid-credential-offer://test-uri",
		})
		require.NoError(t, err)
		assert.Contains(t, out, "DOME Marketplace")
		assert.Contains(t, out, "openid-credential-offer://test-uri")
	})

	t.Run("allows context to override global AppName", func(t *testing.T) {
		out, err := executeTemplate(v.Set, "qrcode.html.j2", pongo2.Context{
			"AppName":            "Custom Marketplace",
			"CredentialOfferUri": "openid-credential-offer://custom-uri",
		})
		require.NoError(t, err)
		assert.Contains(t, out, "Custom Marketplace")
		assert.Contains(t, out, "openid-credential-offer://custom-uri")
	})

	t.Run("returns error when template does not exist", func(t *testing.T) {
		out, err := executeTemplate(v.Set, "nonexistent_template.html", nil)
		assert.Error(t, err)
		assert.Empty(t, out)
	})
}

func TestAppViews_RenderQRCodeView(t *testing.T) {
	v, err := NewAppViews(true)
	require.NoError(t, err)
	require.NotNil(t, v)

	t.Run("renders qr code view successfully", func(t *testing.T) {
		out, err := v.RenderQRCodeView("https://example.com/offer", domeTestUrl)
		require.NoError(t, err)
		assert.Contains(t, out, "DOME Marketplace")
		assert.Contains(t, out, "https://example.com/offer")
	})

	t.Run("returns error when credentialOfferUri is empty", func(t *testing.T) {
		out, err := v.RenderQRCodeView("", domeTestUrl)
		assert.Error(t, err)
		assert.Empty(t, out)
	})
}
