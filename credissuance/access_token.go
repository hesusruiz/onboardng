package credissuance

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/hesusruiz/utils/errl"
)

// TokenRequest performs an OAuth 2.0 Access Token Request at the specified token endpoint.
// It generates a Client Assertion (a signed JWT) using the provided credentials and
// returns the access token string if successful.
//
// Parameters:
//   - tokenEndpoint: The URL of the OAuth 2.0 Token Endpoint.
//   - machineCredential: The raw Verifiable Credential (VC) string.
//   - didkey: The DID of the client used for identification (client_id).
//   - verifierURL: The audience URL for the assertion.
//   - privateKey: The ECDSA private key for signing the assertion.
func TokenRequest(
	ctx context.Context,
	tokenEndpoint string,
	machineCredential string,
	didkey string,
	verifierURL string,
	privateKey *ecdsa.PrivateKey,
	client *http.Client,
) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if tokenEndpoint == "" {
		return "", errl.Errorf("token endpoint is required")
	}
	if machineCredential == "" {
		return "", errl.Errorf("machine credential is required")
	}
	if didkey == "" {
		return "", errl.Errorf("didkey is required")
	}
	if verifierURL == "" {
		return "", errl.Errorf("verifier URL is required")
	}
	if privateKey == nil {
		return "", errl.Errorf("private key is required")
	}

	// 1. Generate the Client Assertion (the signed JWT) to authenticate with the Token Endpoint
	cliAssertion, err := NewCliAssertion(machineCredential, didkey, verifierURL, privateKey)
	if err != nil {
		return "", errl.Errorf("error creating client assertion: %w", err)
	}

	// 2. Build the x-www-form-urlencoded request body for the token endpoint (RFC 7523)
	var b bytes.Buffer
	b.WriteString("client_id=" + didkey + "&")
	b.WriteString("grant_type=client_credentials&")
	b.WriteString("client_assertion_type=urn%3Aietf%3Aparams%3Aoauth%3Aclient-assertion-type%3Ajwt-bearer&")
	b.WriteString("client_assertion=" + cliAssertion)

	// 3. Initialize the POST request to the token endpoint
	req, err := http.NewRequestWithContext(ctx, "POST", tokenEndpoint, &b)
	if err != nil {
		return "", errl.Errorf("error creating token request: %w", err)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	if client == nil {
		client = http.DefaultClient
	}

	// 4. Send the request and handle the response
	resp, err := client.Do(req)
	if err != nil {
		return "", errl.Errorf("error calling token endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 399 {
		fmt.Println("Error calling Token Endpoint:", resp.Status, req.Host, req.URL.String())
		return "", errl.Errorf("error calling Token Endpoint: %v", resp.Status)
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errl.Errorf("error reading response body: %w", err)
	}

	// 5. Unmarshal the response body to extract the access token
	type accessTokenResponse struct {
		AccessToken string `json:"access_token"`
	}

	at := &accessTokenResponse{}
	err = json.Unmarshal(responseBody, at)
	if err != nil {
		return "", errl.Errorf("error unmarshalling response body: %w", err)
	}

	return at.AccessToken, nil
}

// CliAssertion represents the claims for the outer JWT used in client authentication.
type CliAssertion struct {
	jwt.RegisteredClaims
	VpToken string `json:"vp_token"`
}

// NewCliAssertion generates a signed JWT (Client Assertion) used for client authentication
// at the Token Endpoint, following RFC 7523.
//
// Parameters:
//   - learCredential: The raw Verifiable Credential (VC) string, typically a LEARCredentialMachine.
//   - didkey: The DID of the client, used as issuer (iss), subject (sub), and key ID (kid).
//   - verifierURL: The audience (aud) of the assertion, usually the Token Endpoint.
//   - privateKey: The ECDSA private key used to sign the nested JWT structure.
//
// The function creates a nested JWT structure:
// 1. An inner VP Token containing a Verifiable Presentation.
// 2. An outer Client Assertion containing the Base64URL-encoded VP Token in the "vp_token" claim.
func NewCliAssertion(learCredential string, didkey string, verifierURL string, privateKey *ecdsa.PrivateKey) (string, error) {
	if !strings.HasPrefix(didkey, "did:key:") {
		return "", errl.Errorf("didkey must start with 'did:key:'")
	}
	if verifierURL == "" {
		return "", errl.Errorf("verifierURL cannot be empty")
	}

	// 1. Create the inner Verifiable Presentation (VP) Token
	vpStringToken, err := NewVPToken(string(learCredential), didkey, privateKey, verifierURL)
	if err != nil {
		return "", errl.Errorf("error creating VP Token: %w", err)
	}

	// 2. Initialize the Client Assertion claims with the Base64URL-encoded inner token
	claims := CliAssertion{
		VpToken: B64Encode([]byte(vpStringToken)),
	}

	setRegisteredClaims(&claims.RegisteredClaims, didkey, didkey, verifierURL, time.Hour)

	// 4. Generate the JWT using ES256 (ECDSA with P-256 and SHA-256)
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)

	// 5. Add the 'kid' (Key ID) header to identify the signing key
	token.Header["kid"] = didkey

	// 6. Sign the outer token with the client's private key
	return token.SignedString(privateKey)

}

func setRegisteredClaims(claims *jwt.RegisteredClaims, issuer, subject, audience string, duration time.Duration) {
	now := time.Now().UTC()
	claims.Issuer = issuer
	claims.Subject = subject
	claims.ExpiresAt = jwt.NewNumericDate(now.Add(duration))
	claims.IssuedAt = jwt.NewNumericDate(now)
	claims.NotBefore = jwt.NewNumericDate(now)
	jwt.MarshalSingleStringAsArray = false
	claims.Audience = jwt.ClaimStrings{audience}
	claims.ID = GenerateNonce()
}

// VPToken represents the structure for the inner JWT claims, containing the Verifiable Presentation.
type VPToken struct {
	jwt.RegisteredClaims
	VP    VP     `json:"vp"`
	Nonce string `json:"nonce,omitempty"`
}

// String returns a pretty-printed JSON representation of the VPToken.
func (o *VPToken) String() string {
	out, _ := json.MarshalIndent(o, "", "  ")
	return string(out)
}

// VP represents a Verifiable Presentation structure as defined in the W3C VC Data Model.
type VP struct {
	Context              []string `json:"@context"`
	Type                 []string `json:"type"`
	Holder               string   `json:"holder,omitempty"`
	Id                   string   `json:"id,omitempty"`
	VerifiableCredential []string `json:"verifiableCredential"`
}

// String returns a compact JSON representation of the VP.
func (o *VP) String() string {
	out, _ := json.Marshal(o)
	return string(out)
}

// NewVPToken creates a signed JWT containing a Verifiable Presentation (VP).
// This JWT is then encoded and placed within the 'vp_token' claim of the outer Client Assertion.
func NewVPToken(vcStringToken string, didkey string, privateKey *ecdsa.PrivateKey, verifierSBX string) (string, error) {
	if !strings.HasPrefix(didkey, "did:key:") {
		return "", errl.Errorf("didkey must start with 'did:key:'")
	}
	if verifierSBX == "" {
		return "", errl.Errorf("verifierSBX cannot be empty")
	}

	// 1. Construct the Verifiable Presentation object
	vp := VP{
		Context: []string{
			"https://www.w3.org/2018/credentials/v1",
		},
		Type: []string{
			"VerifiablePresentation",
		},
		Holder:               didkey,
		Id:                   GenerateNonce(),
		VerifiableCredential: []string{vcStringToken},
	}

	// 2. Initialize the JWT claims for the VP Token
	claims := VPToken{
		VP:    vp,
		Nonce: GenerateNonce(),
	}

	setRegisteredClaims(&claims.RegisteredClaims, didkey, didkey, verifierSBX, time.Hour)

	// 3. Generate and sign the inner token using ES256
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["kid"] = didkey

	signed, err := token.SignedString(privateKey)
	if err != nil {
		return "", errl.Errorf("error signing VP Token: %w", err)
	}

	return signed, nil

}

// GenerateNonce creates a random 16-byte nonce, encoded as a Base64 RawURL string.
func GenerateNonce() string {
	b := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		panic("failed to generate random nonce: " + err.Error())
	}
	nonce := base64.RawURLEncoding.EncodeToString(b)
	return nonce
}

// B64Encode performs a custom Base64URL encoding (RFC 4648) without padding characters.
func B64Encode(data []byte) string {
	result := base64.StdEncoding.EncodeToString(data)
	result = strings.Replace(result, "+", "-", -1) // 62nd char of encoding
	result = strings.Replace(result, "/", "_", -1) // 63rd char of encoding
	result = strings.Replace(result, "=", "", -1)  // Remove any trailing '='s

	return result
}
