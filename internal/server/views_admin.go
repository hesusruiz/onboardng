package server

import (
	"embed"
	"encoding/json"
	"io/fs"
	"os"

	"github.com/flosch/pongo2/v7"
	"github.com/hesusruiz/onboardng/internal/db"
)

//go:embed admintpl/*
var adminFS embed.FS

const templateDir = "internal/server"

type AdminViews struct {
	Set    *pongo2.TemplateSet
	TMFUrl string
}

func NewAdminViews(debug bool, tmfUrl string) (*AdminViews, error) {
	set, err := loadSet("admintpl", adminFS, debug)
	if err != nil {
		return nil, err
	}

	return &AdminViews{Set: set, TMFUrl: tmfUrl}, nil
}

func (v *AdminViews) PageIndex(email string, regs []db.RegistrationRecord, limit int, offset int, numEntries int, total int) (string, error) {

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

func (v *AdminViews) PageRegistrations(apiJSON string) (string, error) {

	return executeTemplate(v.Set, "registrations.html.j2", pongo2.Context{"APIJson": apiJSON})
}

func (v *AdminViews) PageRegsOrgs(apiJSON string, tmfOrganizationUrl string) (string, error) {

	return executeTemplate(v.Set, "regs_orgs.html.j2", pongo2.Context{
		"APIJson":            apiJSON,
		"tmfOrganizationUrl": tmfOrganizationUrl,
	})
}

func (v *AdminViews) PageOneRegistration(registration *db.RegistrationRecord) (string, error) {

	return executeTemplate(v.Set, "oneregistration.html.j2", pongo2.Context{"Reg": registration})
}

func (v *AdminViews) PageOrganizations(tmfUrl string) (string, error) {

	return executeTemplate(v.Set, "organizations.html.j2", pongo2.Context{"APIJson": tmfUrl})
}

func (v *AdminViews) APIOrganizationsJSON() ([]byte, error) {

	// Retrieve all Organizations from the TMF server
	baseUrlSBX := v.TMFUrl
	offset := 0
	limit := 200
	allOrgs := make([]Organization, 0)

	for {
		orgs, err := TMFRetrieveOrganizations(baseUrlSBX, offset, limit)
		if err != nil {
			return nil, err
		}

		allOrgs = append(allOrgs, orgs...)

		if len(orgs) < limit {
			break
		}
		offset += limit
	}

	// Convert to JSON
	orgsJSON, err := json.Marshal(allOrgs)
	if err != nil {
		return nil, err
	}

	return orgsJSON, nil
}

func (v *AdminViews) AllOrganizations() ([]Organization, error) {

	// Retrieve all Organizations from the TMF server
	baseUrlSBX := v.TMFUrl
	offset := 0
	limit := 200
	allOrgs := make([]Organization, 0)

	for {
		orgs, err := TMFRetrieveOrganizations(baseUrlSBX, offset, limit)
		if err != nil {
			return nil, err
		}

		allOrgs = append(allOrgs, orgs...)

		if len(orgs) < limit {
			break
		}
		offset += limit
	}

	return allOrgs, nil
}

func executeTemplate(set *pongo2.TemplateSet, templateName string, context pongo2.Context) (string, error) {
	tpl, err := set.FromCache(templateName)
	if err != nil {
		return "", err
	}

	out, err := tpl.Execute(context)
	if err != nil {
		return "", err
	}

	return out, nil
}

func loadSet(subapp string, embeddedFS embed.FS, debug bool) (*pongo2.TemplateSet, error) {
	var templateFS fs.FS
	if _, err := os.Stat(subapp); err == nil {
		templateFS = os.DirFS(subapp)
	} else if _, err := os.Stat(templateDir + "/" + subapp); err == nil {
		templateFS = os.DirFS(templateDir + "/" + subapp)
	} else {
		sub, err := fs.Sub(embeddedFS, subapp)
		if err != nil {
			return nil, err
		}
		templateFS = sub
	}

	loader := pongo2.NewFSLoader(templateFS)
	set := pongo2.NewSet(subapp, loader)
	set.Debug = debug

	set.Globals = pongo2.Context{
		"AppName": "DOME Marketplace",
	}

	return set, nil
}
