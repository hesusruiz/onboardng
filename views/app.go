package views

import (
	"embed"
	"errors"

	"github.com/flosch/pongo2/v7"
)

//go:embed app/*
var appFS embed.FS

// Use pongo2 V7 (https://pkg.go.dev/github.com/flosch/pongo2/v7@v7.0.0-alpha.1)
//

type AppViews struct {
	Set *pongo2.TemplateSet
}

func NewAppViews(debug bool) (*AppViews, error) {
	set, err := loadSet("app", appFS, debug)
	if err != nil {
		return nil, err
	}

	return &AppViews{Set: set}, nil
}

func (v *AppViews) RenderQRCodeView(credentialOfferUri string, domeURL string) (string, error) {
	if credentialOfferUri == "" {
		return "", errors.New("credentialOfferUri is required")
	}

	ctx := pongo2.Context{
		"CredentialOfferUri": credentialOfferUri,
		"DOMEURL":            domeURL,
	}

	return executeTemplate(v.Set, "qrcode.html.j2", ctx)
}
