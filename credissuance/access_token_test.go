package credissuance

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"testing"
)

func TestNewCliAssertion(t *testing.T) {
	privateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	learCredential := "mock-vc"
	didkey := "did:key:zQ3shokFTS3brHcDQrn82rm"
	verifierURL := "https://verifier.example.com"

	t.Run("ValidParameters", func(t *testing.T) {
		assertion, err := NewCliAssertion(learCredential, didkey, verifierURL, privateKey)
		if err != nil {
			t.Fatalf("NewCliAssertion failed with valid parameters: %v", err)
		}
		if assertion == "" {
			t.Error("expected non-empty assertion string")
		}
	})

	t.Run("InvalidDidKey", func(t *testing.T) {
		_, err := NewCliAssertion(learCredential, "invalid-did", verifierURL, privateKey)
		if err == nil {
			t.Error("expected error for invalid didkey, got nil")
		}
	})

	t.Run("EmptyVerifierURL", func(t *testing.T) {
		_, err := NewCliAssertion(learCredential, didkey, "", privateKey)
		if err == nil {
			t.Error("expected error for empty verifierURL, got nil")
		}
	})
}

func TestNewVPToken(t *testing.T) {
	privateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	vcStringToken := "mock-vc"
	didkey := "did:key:zQ3shokFTS3brHcDQrn82rm"
	verifierSBX := "https://verifier.example.com"

	t.Run("ValidParameters", func(t *testing.T) {
		token, err := NewVPToken(vcStringToken, didkey, privateKey, verifierSBX)
		if err != nil {
			t.Fatalf("NewVPToken failed with valid parameters: %v", err)
		}
		if token == "" {
			t.Error("expected non-empty token string")
		}
	})

	t.Run("InvalidDidKey", func(t *testing.T) {
		_, err := NewVPToken(vcStringToken, "invalid-did", privateKey, verifierSBX)
		if err == nil {
			t.Error("expected error for invalid didkey, got nil")
		}
	})

	t.Run("EmptyVerifierSBX", func(t *testing.T) {
		_, err := NewVPToken(vcStringToken, didkey, privateKey, "")
		if err == nil {
			t.Error("expected error for empty verifierSBX, got nil")
		}
	})
}
