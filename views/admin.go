package views

import (
	"embed"

	"github.com/flosch/pongo2/v7"
	"github.com/hesusruiz/onboardng/internal/db"
)

//go:embed admin/*
var adminFS embed.FS

type AdminViews struct {
	Set *pongo2.TemplateSet
}

func NewAdminViews(debug bool) (*AdminViews, error) {
	set, err := loadSet("admin", adminFS, debug)
	if err != nil {
		return nil, err
	}

	return &AdminViews{Set: set}, nil
}

func (v *AdminViews) RenderIndexView(email string, regs []db.RegistrationRecord, limit int, offset int, numEntries int, total int) (string, error) {

	ctx := pongo2.Context{
		"Email":      email,
		"Regs":       regs,
		"Limit":      limit,
		"Offset":     offset,
		"NumEntries": numEntries,
		"Total":      total,
	}

	return executeTemplate(v.Set, "index.html.j2", ctx)
}
