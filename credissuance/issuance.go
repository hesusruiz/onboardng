// Package credissuance implements the logic for requesting and issuing Verifiable Credentials.
// It follows a two-step process:
// 1. Authenticate with a Token Endpoint using a LEARCredentialMachine to obtain an access token.
// 2. Use the access token to request a new Verifiable Credential from the Issuance Endpoint.
package credissuance

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/hesusruiz/onboardng/internal/configuration"
	"github.com/hesusruiz/onboardng/internal/crypto"
	"github.com/hesusruiz/utils/errl"
)

// LEARIssuanceRequestBody represents the JSON structure for a credential issuance request to the Issuer machine.
type LEARIssuanceRequestBody struct {
	Schema        string  `json:"schema,omitempty"`
	OperationMode string  `json:"operation_mode,omitempty"`
	Format        string  `json:"format,omitempty"`
	ResponseUri   string  `json:"response_uri,omitempty"`
	Payload       Payload `json:"payload"`
}

// ParseLEARIssuanceRequestBody unmarshals a JSON byte slice into a LEARIssuanceRequestBody.
func ParseLEARIssuanceRequestBody(body []byte) (*LEARIssuanceRequestBody, error) {
	var req LEARIssuanceRequestBody
	err := json.Unmarshal(body, &req)
	if err != nil {
		return nil, err
	}
	return &req, nil
}

// Payload contains the details of the mandator, mandatee, and the powers being delegated.
type Payload struct {
	Mandator Mandator `json:"mandator"`
	Mandatee Mandatee `json:"mandatee"`
	Power    []Power  `json:"power,omitempty"`
}

// Mandator represents the legal entity (organization) that holds the original authority.
type Mandator struct {
	OrganizationIdentifier string `json:"organizationIdentifier,omitempty"`
	Organization           string `json:"organization,omitempty"`
	Country                string `json:"country,omitempty"`
	CommonName             string `json:"commonName,omitempty"`
	EmailAddress           string `json:"emailAddress,omitempty"`
	SerialNumber           string `json:"serialNumber,omitempty"`
}

// Mandatee represents the natural person or machine receiving the delegated powers.
type Mandatee struct {
	FirstName   string `json:"firstName,omitempty"`
	LastName    string `json:"lastName,omitempty"`
	Nationality string `json:"nationality,omitempty"`
	Email       string `json:"email,omitempty"`
}

// Power specifies the delegated capability (e.g., Execute Onboarding in the DOME domain).
type Power struct {
	Type     string  `json:"type,omitempty"`
	Domain   string  `json:"domain,omitempty"`
	Function string  `json:"function,omitempty"`
	Action   Strings `json:"action,omitempty"`
}

// Strings is a helper type to handle JSON marshaling of single strings vs arrays for the "action" claim.
type Strings []string

func (s Strings) MarshalJSON() (b []byte, err error) {
	if len(s) == 1 {
		return json.Marshal(s[0])
	}

	return json.Marshal([]string(s))
}

// LEARIssuance orchestrates the credential issuance workflow.
type LEARIssuance struct {
	privateKey        *ecdsa.PrivateKey
	machineCredential string

	verifierTokenEndpoint  string
	verifierURL            string
	myDidkey               string
	credentialIssuancePath string
	tmForumURL             string
	httpClient             *http.Client
}

// NewLEARIssuance initializes a new LEARIssuance instance from the provided configuration.
// It decodes the private key, derives the associated DID, and validates the setup.
func NewLEARIssuance(config configuration.EnvConfig) (*LEARIssuance, error) {

	// Read the private key
	if config.PrivateKey == "" {
		return nil, errl.Errorf("private key is missing in configuration")
	}
	pemBytesRaw := []byte(config.PrivateKey)

	// Strip any '0x' or '0X' prefix from the key and decode it
	hexKey := strings.TrimPrefix(string(pemBytesRaw), "0x")
	hexKey = strings.TrimPrefix(hexKey, "0X")
	hexKey = strings.TrimSpace(hexKey)
	dBytes, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, errl.Errorf("failed to decode private key hex: %w", err)
	}

	// Create ECDSA Private Key from the raw private key
	curve := elliptic.P256()
	privateKey, err := ecdsa.ParseRawPrivateKey(curve, dBytes)
	if err != nil {
		return nil, errl.Errorf("error parsing private key: %w", err)
	}

	// For safety, derive the associated did:key from the private key and compare it to the configured value.
	didKey, err := crypto.DeriveDidKeyFromPrivateKey(privateKey)
	if err != nil {
		return nil, errl.Errorf("error deriving did:key from private key: %w", err)
	}

	if didKey != config.MyDidkey {
		return nil, errl.Errorf("the private key does not correspond to the did:key in the configuration")
	}

	// Read the LEARCredentialMachine
	if config.MachineCredential == "" {
		return nil, errl.Errorf("machine credential is missing in configuration")
	}
	machineCredential := config.MachineCredential

	l := &LEARIssuance{
		privateKey:        privateKey,
		machineCredential: machineCredential,
	}

	l.verifierTokenEndpoint = config.Verifier.TokenEndpoint
	l.verifierURL = config.Verifier.URL
	l.myDidkey = config.MyDidkey
	l.credentialIssuancePath = config.Issuer.CredentialIssuancePath

	// Store the TM Forum Base URL, stripping the trailing slash if present
	l.tmForumURL = strings.TrimSuffix(config.TMForum.BaseURL, "/")

	// Create a HTTP Client with a timeout
	l.httpClient = &http.Client{
		Timeout: 10 * time.Second,
	}

	return l, nil

}

// GetAccessToken initiates the first step of the process: obtaining an OAuth 2.0 access token
// by presenting a Client Assertion (containing a LEARCredentialMachine).
func (l *LEARIssuance) GetAccessToken(ctx context.Context) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	return TokenRequest(
		ctx,
		l.verifierTokenEndpoint,
		l.machineCredential,
		l.myDidkey,
		l.verifierURL,
		l.privateKey,
		l.httpClient,
	)
}

// LEARIssuanceRequest initiates the second step of the process: requesting the issuance
// of a new Verifiable Credential using the previously obtained access token.
func (l *LEARIssuance) LEARIssuanceRequest(ctx context.Context, accessToken string, learCredData *LEARIssuanceRequestBody) ([]byte, error) {

	if accessToken == "" {
		return nil, errl.Errorf("access token is required for LEARIssuanceRequest")
	}

	fmt.Printf("Access Token: %v\n", accessToken)
	fmt.Printf("Issuance Endpoint: %v\n", l.credentialIssuancePath)

	// The request buffer
	buf, err := json.Marshal(learCredData)
	if err != nil {
		return nil, errl.Errorf("error marshalling request body: %w", err)
	}
	requestBody := bytes.NewBuffer(buf)

	// The request to send
	req, err := http.NewRequestWithContext(ctx, "POST", l.credentialIssuancePath, requestBody)
	if err != nil {
		return nil, errl.Errorf("error creating http request: %w", err)
	}

	req.Header.Add("Content-Type", "application/json")
	if accessToken != "" {
		req.Header.Add("Authorization", "Bearer "+accessToken)
	}

	resp, err := l.httpClient.Do(req)
	if err != nil {
		return nil, errl.Errorf("error calling LEAR Issuance Endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 399 {
		fmt.Println("Error calling LEAR Issuance Endpoint:", resp.Status)
		return nil, errl.Errorf("error calling LEAR Issuance Endpoint: %v", resp.Status)
	}

	ResponseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errl.Errorf("error reading response body: %w", err)
	}

	return ResponseBody, nil
}
