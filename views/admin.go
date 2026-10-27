package views

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

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
func (v *AdminViews) RenderDashboardView() (string, error) {

	// Retrieve all Organizations from the TMF server
	baseUrlSBX := "https://dome-marketplace-sbx.org/party/organization"
	offset := 0
	limit := 100
	allOrgs := make([]Organization, 0)

	for {
		orgs, err := RetrieveOrganizationsPage(baseUrlSBX, offset, limit)
		if err != nil {
			return "", err
		}

		allOrgs = append(allOrgs, orgs...)

		if len(orgs) < limit {
			break
		}
		offset += limit
	}

	ctx := pongo2.Context{
		"Organizations": allOrgs,
	}

	return executeTemplate(v.Set, "dashboard.html.j2", ctx)
}

type Organization struct {
	ID                         string         `json:"id"`
	Name                       string         `json:"name"`
	TradingName                string         `json:"tradingName"`
	EmailAddress               string         `json:"emailAddress"`
	OrganizationIdentification string         `json:"organizationIdentification"`
	Valid                      bool           `json:"valid"`
	Errors                     []string       `json:"errors"`
	RawData                    map[string]any `json:"rawData"`
}

func RetrieveOrganizationsPage(baseUrl string, offset int, limit int) ([]Organization, error) {
	url := fmt.Sprintf("%s?offset=%d&limit=%d", baseUrl, offset, limit)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var rawOrgs []map[string]any
	err = json.Unmarshal(body, &rawOrgs)
	if err != nil {
		return nil, err
	}

	var orgs []Organization
	for _, rawOrg := range rawOrgs {
		id, _ := rawOrg["id"].(string)
		name, _ := rawOrg["name"].(string)
		tradingName, _ := rawOrg["tradingName"].(string)

		org := Organization{
			ID:          id,
			Name:        name,
			TradingName: tradingName,
			Errors:      make([]string, 0),
			RawData:     rawOrg,
		}
		if org.TradingName == "" {
			org.Errors = append(org.Errors, "Trading name is empty")
		}
		orgs = append(orgs, org)
	}

	return orgs, nil
}
