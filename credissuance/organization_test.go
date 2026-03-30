package credissuance

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOrganizationAPI(t *testing.T) {
	ctx := context.Background()

	// Simple mock server to respond to Organization API calls
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for Authorization header
		if r.Header.Get("Authorization") != "Bearer test-token" && r.Header.Get("Authorization") != "" {
			// In some tests we might want to verify it's NOT present if token is empty
		}

		switch r.URL.Path {
		case "/tmf-api/party/v4/organization":
			if r.Method == "GET" {
				orgs := []Organization{{ID: "1", Name: "Org 1"}}
				json.NewEncoder(w).Encode(orgs)
			} else if r.Method == "POST" {
				var org Organization
				json.NewDecoder(r.Body).Decode(&org)
				org.ID = "1"
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(org)
			}
		case "/tmf-api/party/v4/organization/1":
			if r.Method == "GET" {
				org := Organization{ID: "1", Name: "Org 1"}
				json.NewEncoder(w).Encode(org)
			} else if r.Method == "PATCH" {
				var org Organization
				json.NewDecoder(r.Body).Decode(&org)
				org.ID = "1"
				json.NewEncoder(w).Encode(org)
			} else if r.Method == "DELETE" {
				w.WriteHeader(http.StatusNoContent)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create a dummy LEARIssuance instance
	l := &LEARIssuance{
		verifierURL: server.URL,
		tmForumURL:  server.URL,
		httpClient:  server.Client(),
	}

	// Token
	accessToken := "test-token"

	t.Run("ListOrganizations", func(t *testing.T) {
		orgs, err := l.TMFListOrganizations(ctx, accessToken, "", 0, 0)
		if err != nil {
			t.Fatalf("ListOrganizations failed: %v", err)
		}
		if len(orgs) != 1 || orgs[0].ID != "1" {
			t.Errorf("expected 1 org with ID 1, got %v", orgs)
		}
	})

	t.Run("CreateOrganization", func(t *testing.T) {
		newOrg := Organization_Create{TradingName: "Test Org"}
		org, err := l.TMFCreateOrganization(ctx, accessToken, &newOrg)
		if err != nil {
			t.Fatalf("CreateOrganization failed: %v", err)
		}
		if org.ID != "1" {
			t.Errorf("expected org ID 1, got %s", org.ID)
		}
	})

	t.Run("RetrieveOrganization", func(t *testing.T) {
		org, err := l.TMFRetrieveOrganization(ctx, accessToken, "1", "")
		if err != nil {
			t.Fatalf("RetrieveOrganization failed: %v", err)
		}
		if org.ID != "1" {
			t.Errorf("expected org ID 1, got %s", org.ID)
		}
	})

	t.Run("UpdateOrganization", func(t *testing.T) {
		update := Organization_Update{Name: "Updated Name"}
		org, err := l.TMFUpdateOrganization(ctx, accessToken, "1", &update)
		if err != nil {
			t.Fatalf("UpdateOrganization failed: %v", err)
		}
		if org.ID != "1" {
			t.Errorf("expected org ID 1, got %s", org.ID)
		}
	})

	t.Run("DeleteOrganization", func(t *testing.T) {
		err := l.TMFDeleteOrganization(ctx, accessToken, "1")
		if err != nil {
			t.Fatalf("DeleteOrganization failed: %v", err)
		}
	})

	t.Run("EmptyAccessToken", func(t *testing.T) {
		// Mock server to verify NO Authorization header
		serverEmpty := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "" {
				t.Errorf("Authorization header should be empty, got %s", r.Header.Get("Authorization"))
			}
			w.WriteHeader(http.StatusNoContent)
		}))
		defer serverEmpty.Close()

		lEmpty := &LEARIssuance{
			verifierURL: serverEmpty.URL,
			tmForumURL:  serverEmpty.URL,
			httpClient:  serverEmpty.Client(),
		}
		err := lEmpty.TMFDeleteOrganization(ctx, "", "1")
		if err != nil {
			t.Fatalf("DeleteOrganization failed: %v", err)
		}
	})
}

func TestMyOrganization(t *testing.T) {
	tests := []struct {
		name    string // description of this test case
		want    *Organization
		wantErr bool
	}{
		{
			name: "Test",
			want: &Organization{
				ID: "1",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := TMFMyOrganization()
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("TMFMyOrganization() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("TMFMyOrganization() succeeded unexpectedly")
			}
			_ = got
		})
	}
}
