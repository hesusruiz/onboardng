package server

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"strings"

	"github.com/flosch/pongo2/v7"
	"github.com/hesusruiz/onboardng/common"
	"github.com/hesusruiz/onboardng/internal/db"
	"github.com/hesusruiz/onboardng/internal/jpath"
	"github.com/hesusruiz/onboardng/internal/vies"
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

func (v *AdminViews) RenderRegistrationsView(tmfUrl string) (string, error) {

	return executeTemplate(v.Set, "registrations.html.j2", pongo2.Context{"APIJson": tmfUrl})
}

func (v *AdminViews) RenderRegistrationsJSON(regs []*db.RegistrationRecord) ([]byte, error) {

	jsonData, err := json.Marshal(regs)
	if err != nil {
		fmt.Println("Error marshalling registrations to JSON:", err)
		return nil, err
	}

	return jsonData, nil

}

func (v *AdminViews) RenderOrganizationsView(tmfUrl string) (string, error) {

	return executeTemplate(v.Set, "organizations.html.j2", pongo2.Context{"APIJson": tmfUrl})
}

func (v *AdminViews) RenderOrganizationsJSON() ([]byte, error) {

	// Retrieve all Organizations from the TMF server
	baseUrlSBX := v.TMFUrl
	offset := 0
	limit := 200
	allOrgs := make([]Organization, 0)

	for {
		orgs, err := RetrieveOrganizationsPage(baseUrlSBX, offset, limit)
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

type Organization struct {
	ID                         string         `json:"id"`
	Name                       string         `json:"name"`
	TradingName                string         `json:"tradingName"`
	EmailAddress               string         `json:"emailAddress"`
	VATID                      string         `json:"vatId"`
	VIESError                  string         `json:"viesError"`
	HREF                       string         `json:"href"`
	OrganizationIdentification string         `json:"organizationIdentification"`
	Valid                      bool           `json:"valid"`
	Errors                     []string       `json:"errors"`
	Warnings                   []string       `json:"warnings"`
	rawData                    map[string]any `json:"-"`
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
			Warnings:    make([]string, 0),
			rawData:     rawOrg,
			HREF:        fmt.Sprintf("%s/%s", baseUrl, id),
		}

		// tradingName is required
		if org.TradingName == "" {
			org.Errors = append(org.Errors, "Trading name is empty")
		}

		// Try to get the email address from the raw data
		mediums, err := jpath.GetListStrict(rawOrg, "contactMedium")
		if err != nil {
			org.Warnings = append(org.Warnings, "contactMedium is missing or not a list")
		} else {
			foundEmail := false
			for _, medium := range mediums {
				mediumType, _ := jpath.GetStringStrict(medium, "mediumType")
				if strings.ToUpper(mediumType) == "EMAIL" {
					email, err := jpath.GetStringStrict(medium, "characteristic.emailAddress")
					if err != nil {
						org.Warnings = append(org.Warnings, "emailAddress is missing in contactMedium")
					} else {
						org.EmailAddress = email
					}
					foundEmail = true
					break
				}
			}
			if !foundEmail {
				org.Warnings = append(org.Warnings, "Email address not found in contactMedium")
			}
		}

		// Try to get the VAT ID
		externalReferences, err := jpath.GetListStrict(rawOrg, "externalReference")
		if err != nil {
			org.Errors = append(org.Errors, "externalReference is missing or not a list")
		} else {
			foundVATID := false
			for _, extRef := range externalReferences {
				extRefType, _ := jpath.GetStringStrict(extRef, "externalReferenceType")
				if extRefType == "idm_id" {
					vatId, err := jpath.GetStringStrict(extRef, "name")
					if err != nil {
						org.Errors = append(org.Errors, "VAT ID is missing in externalReference")
						break
					}

					// The VAT ID must be long enough
					if len(vatId) < 7 {
						org.Errors = append(org.Errors, "VAT ID too short")
						fmt.Println("VAT ID too short:", "OID:", org.ID, "VAT ID:", org.VATID)
						org.VIESError = org.VIESError + "<div style='color: red;'>Error: VAT ID too short</div>"
						break
					}

					org.VATID = vatId
					foundVATID = true

					// Now we are going to handle a series of cases for badly formatted VATIDs, and will try to correct them if possible
					var countryCode string
					var vatNumber string

					if !strings.HasPrefix(org.VATID, "VAT") {
						// No VAT prefix. Check if it starts with a valid EU country code (2 letters)
						countryCode = org.VATID[0:2]

						// Handle the 'GR' case for Greece, which should be 'EL'
						if countryCode == "GR" {
							countryCode = "EL"
							org.VIESError = org.VIESError + "<div style='color: orange;'>Greece should be 'EL'</div>"
						}

						if !common.IsValidEUCountry(countryCode) {
							org.Errors = append(org.Errors, "Invalid EU country code in VAT ID")
							fmt.Println("Invalid EU country code in VAT ID:", "OID:", org.ID, "VAT ID:", org.VATID, "CountryCode:", countryCode)
							org.VIESError = org.VIESError + "<div style='color: orange;'>Non EU country code</div>"
							break
						}

						vatNumber = org.VATID[2:]
						// Add warning to VIESError field, but continue to check in VIES
						org.VIESError = org.VIESError + "<div style='color: orange;'>Missing VAT prefix</div>"

						// Check VIES database
						viesResponse, err := vies.CheckVAT(context.Background(), nil, countryCode, vatNumber)
						if err != nil {
							org.Errors = append(org.Errors, "Failed to check VAT ID in VIES")
							org.VIESError = org.VIESError + "<div style='color: red;'>VIES call failed</div>"
							break
						} else {
							if !viesResponse.Valid {
								org.Errors = append(org.Errors, "VAT ID is not valid according to VIES")
								org.VIESError = org.VIESError + "<div style='color: red;'>NOT VALID in VIES</div>"
								fmt.Println("VATID invalid:", org.VATID, "CC:", countryCode, "Num:", vatNumber, "OID:", org.ID)
								break
							} else {
								org.VIESError = org.VIESError + "<div style='color: green;'>Valid in VIES</div>"
								break
							}
						}
					}

					// Get the country code and VAT number from the VAT ID
					countryCode = org.VATID[3:5]
					vatNumber = org.VATID[6:]

					// Handle the 'GR' case for Greece, which should be 'EL'
					if countryCode == "GR" {
						countryCode = "EL"
						org.VIESError = org.VIESError + "<div style='color: orange;'>Greece should be 'EL'</div>"
					}

					// Check if country is a valid EU country code (2 letters)
					if !common.IsValidEUCountry(countryCode) {
						org.Errors = append(org.Errors, "Invalid EU country code in VAT ID")
						fmt.Println("Invalid EU country code in VAT ID:", "OID:", org.ID, "VAT ID:", org.VATID, "CountryCode:", countryCode, "VAT number:", vatNumber)
						org.VIESError = org.VIESError + "<div style='color: orange;'>Warning: non EU country code</div>"
						break
					}

					// Now check in the VIES database
					viesResponse, err := vies.CheckVAT(context.Background(), nil, countryCode, vatNumber)
					if err != nil {
						org.Errors = append(org.Errors, "Failed to check VAT ID in VIES")
						org.VIESError = org.VIESError + "<div style='color: red;'>Error: VIES call failed</div>"
						break
					}

					if viesResponse.Valid {
						org.VIESError = org.VIESError + "<div style='color: green;'>Valid in VIES</div>"
						break
					}

					// Check to see if the first 2 chars of the VAT number is the country code, and try again if so
					countryCode = org.VATID[3:5]

					if len(vatNumber) >= 2 && strings.ToUpper(vatNumber[0:2]) == countryCode {
						if countryCode == "GR" {
							countryCode = "EL"
						}
						vatNumber = vatNumber[2:]
						viesResponse, err := vies.CheckVAT(context.Background(), nil, countryCode, vatNumber)
						if err != nil {
							org.Errors = append(org.Errors, "Failed to check VAT ID in VIES (second attempt)")
							org.VIESError = org.VIESError + "<div style='color: red;'>Error: VIES call failed (second attempt)</div>"
							break
						}

						if viesResponse.Valid {
							org.VIESError = org.VIESError + "<div style='color: green;'>Valid in VIES (second attempt)</div>"
							break
						}
					}

					org.Errors = append(org.Errors, "VAT ID is not valid according to VIES")
					org.VIESError = org.VIESError + "<div style='color: red;'>Error: not valid in VIES</div>"
					fmt.Println("VATID invalid:", org.VATID, "CC:", countryCode, "Num:", vatNumber, "OID:", org.ID)
					break
				}
			}
			if !foundVATID {
				org.Errors = append(org.Errors, "VAT ID not found in externalReference")
			}
		}

		if len(org.Errors) == 0 {
			org.Valid = true
		}

		orgs = append(orgs, org)
	}

	return orgs, nil
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
