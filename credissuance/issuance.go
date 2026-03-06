package credissuance

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"filippo.io/age"
	"github.com/hesusruiz/onboardng/internal/configuration"
	"github.com/hesusruiz/utils/errl"
	"github.com/mr-tron/base58/base58"
)

type LEARIssuanceRequestBody struct {
	Schema        string  `json:"schema,omitempty"`
	OperationMode string  `json:"operation_mode,omitempty"`
	Format        string  `json:"format,omitempty"`
	ResponseUri   string  `json:"response_uri,omitempty"`
	Payload       Payload `json:"payload"`
}

func ParseLEARIssuanceRequestBody(body []byte) (*LEARIssuanceRequestBody, error) {
	var req LEARIssuanceRequestBody
	err := json.Unmarshal(body, &req)
	if err != nil {
		return nil, err
	}
	return &req, nil
}

type Payload struct {
	Mandator Mandator `json:"mandator"`
	Mandatee Mandatee `json:"mandatee"`
	Power    []Power  `json:"power,omitempty"`
}

type Mandator struct {
	OrganizationIdentifier string `json:"organizationIdentifier,omitempty"`
	Organization           string `json:"organization,omitempty"`
	Country                string `json:"country,omitempty"`
	CommonName             string `json:"commonName,omitempty"`
	EmailAddress           string `json:"emailAddress,omitempty"`
	SerialNumber           string `json:"serialNumber,omitempty"`
}

type Mandatee struct {
	FirstName   string `json:"firstName,omitempty"`
	LastName    string `json:"lastName,omitempty"`
	Nationality string `json:"nationality,omitempty"`
	Email       string `json:"email,omitempty"`
}

type Power struct {
	Type     string  `json:"type,omitempty"`
	Domain   string  `json:"domain,omitempty"`
	Function string  `json:"function,omitempty"`
	Action   Strings `json:"action,omitempty"`
}

// The "action" claim can either be a single string or an array.
// We need to serialize the claim as a single string if the array has only one element
type Strings []string

func (s Strings) MarshalJSON() (b []byte, err error) {
	if len(s) == 1 {
		return json.Marshal(s[0])
	}

	return json.Marshal([]string(s))
}

type LEARIssuance struct {
	privateKey        *ecdsa.PrivateKey
	machineCredential string

	verifierTokenEndpoint  string
	verifierURL            string
	myDidkey               string
	credentialIssuancePath string
	tmForumURL             string
}

func (l *LEARIssuance) GetAccessToken() (string, error) {
	return TokenRequest(
		l.verifierTokenEndpoint,
		l.machineCredential,
		l.myDidkey,
		l.verifierURL,
		l.privateKey,
	)
}

func NewLEARIssuance(config configuration.EnvConfig) (*LEARIssuance, error) {

	// Read the private key
	var pemBytesRaw []byte
	if config.PrivateKey != "" {
		// Try to decrypt the embedded key
		if config.AgeSecretKey == "" {
			return nil, errl.Errorf("AgeSecretKey is missing but required for embedded private key")
		}

		identity, err := age.ParseHybridIdentity(config.AgeSecretKey)
		if err != nil {
			return nil, errl.Errorf("invalid identity key: %w", err)
		}

		ageReader, err := age.Decrypt(strings.NewReader(config.PrivateKey), identity)
		if err != nil {
			return nil, errl.Errorf("failed to decrypt embedded private key: %w", err)
		}

		pemBytesRaw, err = io.ReadAll(ageReader)
		if err != nil {
			return nil, errl.Errorf("error reading decrypted private key: %w", err)
		}
	} else {
		slog.Warn("private key not encrypted in config file, reading from file")
		// Read the private key from file
		var err error
		pemBytesRaw, err = os.ReadFile(config.PrivateKeyFile)
		if err != nil {
			return nil, errl.Errorf("error reading private key from file: %w", err)
		}
	}

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

	// For safety, we are going to derive the associated did:key and compare to the one in the config
	// We have to represent the public key as a compressed array of bytes,
	// and then apply the encoding for did:key.

	// This is the uncompressed public key
	uncompressed, err := privateKey.PublicKey.Bytes()
	if err != nil {
		return nil, errl.Errorf("error getting public key: %w", err)
	}

	// Extract X and Y from the slice
	// X is bytes [1:33], Y is bytes [33:65]
	xBytes := uncompressed[1:33]
	yLastByte := uncompressed[64]

	// Determine the compressedPrefix (0x02 if Y is even, 0x03 if Y is odd)
	var compressedPrefix byte = 0x02
	if yLastByte%2 != 0 {
		compressedPrefix = 0x03
	}

	// Construct the 33-byte compressed key
	compressedBytes := append([]byte{compressedPrefix}, xBytes...)

	// Compress the public key for the DID
	varintPrefix := []byte{0x80, 0x24} // Varint for P-256
	didKey := "did:key:z" + base58.Encode(append(varintPrefix, compressedBytes...))

	if didKey != config.MyDidkey {
		return nil, errl.Errorf("the private key does not correspond to the did:key in the configuration")
	}

	// Read the LEARCredentialMachine
	var machineCredential string
	if config.MachineCredential != "" {
		// Try to decrypt the embedded machine credential
		if config.AgeSecretKey == "" {
			return nil, errl.Errorf("AgeSecretKey is missing but required for embedded machine credential")
		}

		identity, err := age.ParseHybridIdentity(config.AgeSecretKey)
		if err != nil {
			return nil, errl.Errorf("invalid identity key: %w", err)
		}

		ageReader, err := age.Decrypt(strings.NewReader(config.MachineCredential), identity)
		if err != nil {
			return nil, errl.Errorf("failed to decrypt embedded machine credential: %w", err)
		}

		buf, err := io.ReadAll(ageReader)
		if err != nil {
			return nil, errl.Errorf("error reading decrypted machine credential: %w", err)
		}
		machineCredential = string(buf)
	} else {
		slog.Warn("machine credential not encrypted in config file, reading from file")
		buf, err := os.ReadFile(config.MachineCredentialFile)
		if err != nil {
			return nil, errl.Errorf("error reading machine credential from file: %w", err)
		}
		machineCredential = string(buf)
	}

	l := &LEARIssuance{
		privateKey:        privateKey,
		machineCredential: machineCredential,
	}

	l.verifierTokenEndpoint = config.Verifier.TokenEndpoint
	l.verifierURL = config.Verifier.URL
	l.myDidkey = config.MyDidkey
	l.credentialIssuancePath = config.Issuer.CredentialIssuancePath
	l.tmForumURL = config.TMForum.BaseURL

	return l, nil

}

func (l *LEARIssuance) LEARIssuanceRequest(accessToken string, learCredData *LEARIssuanceRequestBody) ([]byte, error) {

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
	req, _ := http.NewRequest("POST", l.credentialIssuancePath, requestBody)
	req.Header.Add("Content-Type", "application/json")
	if accessToken != "" {
		req.Header.Add("Authorization", "Bearer "+accessToken)
	}

	resp, err := http.DefaultClient.Do(req)
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
