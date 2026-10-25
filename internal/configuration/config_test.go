package configuration_test

import (
	"os"
	"testing"

	"github.com/hesusruiz/onboardng/internal/configuration"
	"gopkg.in/yaml.v3"
)

func TestLoadEnvironmentConfigs(t *testing.T) {
	configs := []struct {
		envName          string
		filePath         string
		expectedHost     string
		expectedFeatures bool
	}{
		{
			envName:          "dev",
			filePath:         "../../config/config.dev.yaml",
			expectedHost:     "smtp.ionos.de",
			expectedFeatures: true,
		},
		{
			envName:          "pre",
			filePath:         "../../config/config.pre.yaml",
			expectedHost:     "smtp.ionos.de",
			expectedFeatures: false,
		},
		{
			envName:          "pro",
			filePath:         "../../config/config.pro.yaml",
			expectedHost:     "smtp.ionos.de",
			expectedFeatures: false,
		},
	}

	for _, tc := range configs {
		t.Run(tc.envName, func(t *testing.T) {
			data, err := os.ReadFile(tc.filePath)
			if err != nil {
				t.Fatalf("Failed to read %s: %v", tc.filePath, err)
			}

			var cfg configuration.EnvConfig
			if err := yaml.Unmarshal(data, &cfg); err != nil {
				t.Fatalf("Failed to parse YAML for %s: %v", tc.filePath, err)
			}

			if cfg.AppName != "Onboarding" {
				t.Errorf("expected AppName 'Onboarding', got '%s'", cfg.AppName)
			}
			if cfg.DestDir != "docs" {
				t.Errorf("expected DestDir 'docs', got '%s'", cfg.DestDir)
			}
			if cfg.SrcDir != "src" {
				t.Errorf("expected SrcDir 'src', got '%s'", cfg.SrcDir)
			}
			if cfg.Mail.SMTP.Host != tc.expectedHost {
				t.Errorf("expected Host '%s', got '%s'", tc.expectedHost, cfg.Mail.SMTP.Host)
			}
			if cfg.Features.TMFServerEnabled != tc.expectedFeatures {
				t.Errorf("expected TMFServerEnabled %v, got %v", tc.expectedFeatures, cfg.Features.TMFServerEnabled)
			}
			if cfg.Verifier.URL == "" {
				t.Errorf("expected verifier URL to be non-empty")
			}
			if cfg.Issuer.CredentialIssuancePath == "" {
				t.Errorf("expected issuer credentialIssuancePath to be non-empty")
			}
			if cfg.TMForum.BaseURL == "" {
				t.Errorf("expected TMForum BaseURL to be non-empty")
			}
		})
	}
}

func TestRuntimeEnvMethods(t *testing.T) {
	tests := []struct {
		env           configuration.RuntimeEnv
		isDev         bool
		isPre         bool
		isPro         bool
		displayString string
	}{
		{configuration.Development, true, false, false, "SBX"},
		{configuration.Preproduction, false, true, false, "DEV"},
		{configuration.Preproduction2, false, true, false, "DEV2"},
		{configuration.Production, false, false, true, "PRO"},
	}

	for _, tc := range tests {
		t.Run(string(tc.env), func(t *testing.T) {
			if tc.env.IsDevelopment() != tc.isDev {
				t.Errorf("%s: expected IsDevelopment=%v, got %v", tc.env, tc.isDev, tc.env.IsDevelopment())
			}
			if tc.env.IsPreproduction() != tc.isPre {
				t.Errorf("%s: expected IsPreproduction=%v, got %v", tc.env, tc.isPre, tc.env.IsPreproduction())
			}
			if tc.env.IsProduction() != tc.isPro {
				t.Errorf("%s: expected IsProduction=%v, got %v", tc.env, tc.isPro, tc.env.IsProduction())
			}
			if tc.env.String() != tc.displayString {
				t.Errorf("%s: expected String()=%s, got %s", tc.env, tc.displayString, tc.env.String())
			}
		})
	}
}
