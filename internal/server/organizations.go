package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/hesusruiz/onboardng/common"
	"github.com/hesusruiz/onboardng/internal/jpath"
	"github.com/hesusruiz/onboardng/internal/vies"
)

type Organization struct {
	ID                         string         `json:"id"`
	Name                       string         `json:"name"`
	TradingName                string         `json:"tradingName"`
	EmailAddress               string         `json:"emailAddress"`
	VATID                      string         `json:"vatId"`
	VATIDCorrected             string         `json:"vatIdCorrected"`
	VIESError                  string         `json:"viesError"`
	HREF                       string         `json:"href"`
	OrganizationIdentification string         `json:"organizationIdentification"`
	Country                    string         `json:"country"`
	Valid                      bool           `json:"valid"`
	Errors                     []string       `json:"errors"`
	Warnings                   []string       `json:"warnings"`
	rawData                    map[string]any `json:"-"`
}

// TMFRetrieveOrganizations retrieves from the remote TMF server up to limit organizations
func TMFRetrieveOrganizations(baseUrl string, offset int, limit int) ([]Organization, error) {

	// Retrieve limit objects starting from offset
	url := fmt.Sprintf("%s?offset=%d&limit=%d&fields=externalReference,tradingName", baseUrl, offset, limit)
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

	// Now we process each object trying to find the VAT-ID of the retrieved organization
	var orgs []Organization
	for _, rawOrg := range rawOrgs {
		id, _ := rawOrg["id"].(string)
		tradingName, _ := rawOrg["tradingName"].(string)

		org := Organization{
			ID:          id,
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

		// Try to get the VAT ID
		externalReferences, err := jpath.GetListStrict(rawOrg, "externalReference")
		if err != nil {
			org.Errors = append(org.Errors, "externalReference missing or not a list")
			org.VIESError = org.VIESError + "<div style='color: red;'>externalReference missing or not a list</div>"
			continue
		}

		for _, extRef := range externalReferences {
			extRefType, _ := jpath.GetStringStrict(extRef, "externalReferenceType")
			if extRefType != "idm_id" {
				continue
			}

			vatId, err := jpath.GetStringStrict(extRef, "name")
			if err != nil {
				// This is a severe error on this entry: it is not worth continuing
				org.Errors = append(org.Errors, "VATID missing in externalReference")
				org.VIESError = org.VIESError + "<div style='color: red;'>VATID missing in externalReference</div>"
				break
			}

			org.VATID = vatId

			// Canonicalize the VAT ID
			countryCode, vatNumber, err := CanonicalizeVATID(vatId)
			if err != nil {
				org.Errors = append(org.Errors, "VATID invalid format")
				org.VIESError = org.VIESError + fmt.Sprintf("<div style='color: red;'>%s</div>", err.Error())
				slog.Error("VATID invalid format", "VATID", vatId, "OID", org.ID)
				break
			}

			org.VATIDCorrected = fmt.Sprintf("VAT%s-%s", countryCode, vatNumber)
			org.Country = countryCode

			if org.VATIDCorrected != org.VATID {
				org.Warnings = append(org.Warnings, fmt.Sprintf("VATID changed to %s", org.VATIDCorrected))
				org.VIESError = org.VIESError + fmt.Sprintf("<div style='color: orange;'>%s</div>", "Corrected to: "+org.VATIDCorrected)
			}

			// Check VIES database
			slog.Info("Checking VAT ID in VIES", "VATID", org.VATID, "CC", countryCode, "Num", vatNumber)
			startTime := time.Now()
			viesResponse, err := vies.CheckVAT(context.Background(), false, countryCode, vatNumber)
			elapsed := time.Since(startTime)
			if err != nil {
				// This is a technical error
				slog.Error("VIES call failed", "elapsed", elapsed, "VATID", org.VATID, "CC", countryCode, "Num", vatNumber, "Error", err.Error())
				org.Errors = append(org.Errors, "VIES call failed")
				org.VIESError = org.VIESError + "<div style='color: red;'>VIES call failed</div>"
				break
			}

			// Check for validation errors
			if !viesResponse.Valid {
				slog.Error("VATID invalid in VIES", "elapsed", elapsed, "VATID", org.VATID, "CC", countryCode, "Num", vatNumber, "OID", org.ID)
				org.Errors = append(org.Errors, "VATID invalid in VIES")
				org.VIESError = org.VIESError + "<div style='color: red;'>NOT VALID in VIES</div>"
				break
			}
			slog.Info("VIES response", "valid", viesResponse.Valid, "elapsed", elapsed)

			org.VIESError = org.VIESError + "<div style='color: green;'>Valid in VIES</div>"

		}

		if org.VATID == "" {
			org.Errors = append(org.Errors, "VATID not found in externalReference")
			org.VIESError = org.VIESError + "<div style='color: red;'>VAT ID not found in externalReference</div>"
		}

		if len(org.Errors) == 0 {
			org.Valid = true
		}

		orgs = append(orgs, org)
	}

	return orgs, nil
}

// CanonicalizeVATID normalizes a possibly malformed EU VAT ID into a 2-letter EU
// country code (converting "GR" to "EL") and the national VAT number.
func CanonicalizeVATID(vatID string) (countryCode string, vatNumber string, err error) {
	// Normalize case and remove spaces and hyphens
	vatID = strings.TrimSpace(vatID)
	vatID = strings.ToUpper(vatID)
	vatID = strings.ReplaceAll(vatID, " ", "")
	vatID = strings.ReplaceAll(vatID, "-", "")

	// Strip optional leading "VAT" prefix
	vatID = strings.TrimPrefix(vatID, "VAT")

	// Check if VAT ID is long enough (2-char country code + at least 5-char VAT number)
	if len(vatID) < 7 {
		return "", "", fmt.Errorf("VAT ID is too short")
	}

	countryCode = vatID[:2]
	vatNumber = vatID[2:]

	// Greece uses "EL" in EU VAT IDs rather than ISO "GR"
	if countryCode == "GR" {
		countryCode = "EL"
	}

	if !common.IsValidEUCountry(countryCode) {
		return "", "", fmt.Errorf("invalid EU country code: %s", countryCode)
	}

	// Strip a duplicated country code prefix inside the national VAT number (e.g. VATIT-IT555...)
	if strings.HasPrefix(vatNumber, countryCode) || (countryCode == "EL" && strings.HasPrefix(vatNumber, "GR")) {
		vatNumber = vatNumber[2:]
	}

	if len(vatNumber) < 5 {
		return "", "", fmt.Errorf("VAT ID is too short")
	}

	return countryCode, vatNumber, nil
}
