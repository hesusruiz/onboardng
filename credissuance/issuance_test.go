package credissuance

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLEARIssuanceRequest(t *testing.T) {
	// Simple mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"access_token": "test-token"})
			return
		}
		if r.URL.Path == "/issuances" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"credential": "mock_credential"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// We need a dummy private key that matches the did:key encoding logic in NewLEARIssuance
	// Or we can just manually construct the LEARIssuance struct to avoid the complex NewLEARIssuance logic.
	issuer := &LEARIssuance{
		verifierURL:            server.URL,
		verifierTokenEndpoint:  server.URL + "/token",
		credentialIssuancePath: server.URL + "/issuances",
	}

	// Use a sample credential
	cred := &LEARIssuanceRequestBody{
		Payload: Payload{
			Mandator: Mandator{Organization: "Test Org"},
		},
	}

	// Perform the issuance request
	resp, err := issuer.LEARIssuanceRequest("test-token", cred)
	if err != nil {
		t.Fatalf("LEARIssuanceRequest failed: %v", err)
	}

	// Verify the response
	expectedResponse := `{"credential": "mock_credential"}`
	if string(resp) != expectedResponse {
		t.Errorf("expected response %s, got %s", expectedResponse, string(resp))
	}
}
