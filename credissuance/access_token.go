package credissuance

import (
	"bytes"
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

func TokenRequest(
	tokenEndpoint string,
	machineCredential string,
	didkey string,
	verifierURL string,
	privateKey *ecdsa.PrivateKey,
) (string, error) {

	fmt.Println("Machine Credential:")
	fmt.Println(machineCredential)
	fmt.Println("===")
	fmt.Println("Didkey:")
	fmt.Println(didkey)
	fmt.Println("===")

	// The assertion to authenticate to the token endpoint
	cliAssertion, err := NewCliAssertion(machineCredential, didkey, verifierURL, privateKey)
	if err != nil {
		return "", errl.Errorf("error creating client assertion: %w", err)
	}

	// Build the request body for calling the token endpoint
	var b bytes.Buffer
	b.WriteString("client_id=" + didkey + "&")
	b.WriteString("grant_type=client_credentials&")
	b.WriteString("client_assertion_type=urn%3Aietf%3Aparams%3Aoauth%3Aclient-assertion-type%3Ajwt-bearer&")
	b.WriteString("client_assertion=" + cliAssertion)

	requestBody := b.String()
	_ = requestBody
	fmt.Println(requestBody)

	// Initialize the request to the token endpoint
	req, _ := http.NewRequest("POST", tokenEndpoint, &b)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	// Send the request to the token endpoint and get the response
	resp, err := http.DefaultClient.Do(req)
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

	// The access token is one of the fields in the response body
	type accessTokenResponse struct {
		AccessToken string `json:"access_token"`
	}

	at := &accessTokenResponse{}
	err = json.Unmarshal(responseBody, at)
	if err != nil {
		return "", errl.Errorf("error unmarshalling response body: %w", err)
	}
	accessToken := at.AccessToken

	return accessToken, nil
}

type CliAssertion struct {
	jwt.RegisteredClaims
	VpToken string `json:"vp_token"`
}

func NewCliAssertion(learCredential string, didkey string, verifierURL string, privateKey *ecdsa.PrivateKey) (string, error) {

	vpStringToken, err := NewVPToken(string(learCredential), didkey, privateKey, verifierURL)
	if err != nil {
		return "", errl.Errorf("error creating VP Token: %w", err)
	}

	// This is the object to create the Client Assertion
	claims := CliAssertion{
		VpToken: B64Encode([]byte(vpStringToken)),
	}

	// Set the claims with timestamps
	now := time.Now()
	claims.ExpiresAt = jwt.NewNumericDate(now.Add(24 * 30 * time.Hour))
	claims.IssuedAt = jwt.NewNumericDate(now)
	claims.NotBefore = jwt.NewNumericDate(now)

	// I am the issuer of this token
	claims.Issuer = didkey
	claims.Subject = didkey

	// The audience is the Verifier, and we set the aud field as a single string
	jwt.MarshalSingleStringAsArray = false
	claims.Audience = jwt.ClaimStrings{verifierURL}

	// The nonce for the token
	claims.ID = GenerateNonce()

	// Generate and sign the token
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)

	// Add the kid header
	token.Header["kid"] = didkey

	return token.SignedString(privateKey)

}

type VPToken struct {
	jwt.RegisteredClaims
	VP    VP     `json:"vp"`
	Nonce string `json:"nonce,omitempty"`
}

func (o *VPToken) String() string {
	out, _ := json.MarshalIndent(o, "", "  ")
	return string(out)
}

type VP struct {
	Context              []string `json:"@context"`
	Type                 []string `json:"type"`
	Holder               string   `json:"holder,omitempty"`
	Id                   string   `json:"id,omitempty"`
	VerifiableCredential []string `json:"verifiableCredential"`
}

func (o *VP) String() string {
	out, _ := json.Marshal(o)
	return string(out)
}

func NewVPToken(vcStringToken string, didkey string, privateKey *ecdsa.PrivateKey, verifierSBX string) (string, error) {

	// This is the Verifiable Presentation object
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

	// This is the object to create the vp_token
	claims := VPToken{
		VP:    vp,
		Nonce: GenerateNonce(),
	}

	// Set the claims with timestamps
	now := time.Now()
	claims.ExpiresAt = jwt.NewNumericDate(now.Add(24 * 30 * time.Hour))
	claims.IssuedAt = jwt.NewNumericDate(now)
	claims.NotBefore = jwt.NewNumericDate(now)

	// I am the issuer of this token
	claims.Issuer = didkey

	// The audience is the Verifier, and we set the aud field as a single string
	jwt.MarshalSingleStringAsArray = false
	claims.Audience = jwt.ClaimStrings{verifierSBX}

	// The nonce for the token
	claims.ID = GenerateNonce()

	// Generate and sign the token
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)

	// Add the kid header
	token.Header["kid"] = didkey

	signed, err := token.SignedString(privateKey)
	if err != nil {
		return "", errl.Errorf("error signing VP Token: %w", err)
	}

	return signed, nil

}

func GenerateNonce() string {
	b := make([]byte, 16)
	io.ReadFull(rand.Reader, b)
	nonce := base64.RawURLEncoding.EncodeToString(b)
	return nonce
}

func B64Encode(data []byte) string {
	result := base64.StdEncoding.EncodeToString(data)
	result = strings.Replace(result, "+", "-", -1) // 62nd char of encoding
	result = strings.Replace(result, "/", "_", -1) // 63rd char of encoding
	result = strings.Replace(result, "=", "", -1)  // Remove any trailing '='s

	return result
}
