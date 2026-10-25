package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/hesusruiz/onboardng/credissuance"
	"github.com/hesusruiz/onboardng/internal/configuration"
	"github.com/hesusruiz/onboardng/mainapp"
)

func main() {
	configPath := flag.String("config", "config.age", "Path to config file (.yaml or .age)")
	envFlag := flag.String("env", "dev", "environment to use (dev, pre or pro)")
	flag.Parse()

	// Read secret key from environment or file
	secretKey := os.Getenv("AGE_SECRET_KEY")
	if secretKey == "" {
		// Try reading from config/age_secret_key.txt
		if data, err := os.ReadFile("config/age_secret_key.txt"); err == nil {
			secretKey = strings.TrimSpace(string(data))
		}
	}

	resolvedConfigPath := *configPath
	if resolvedConfigPath == "config.age" {
		envAge := fmt.Sprintf("config.%s.age", *envFlag)
		envYaml := filepath.Join("config", fmt.Sprintf("config.%s.yaml", *envFlag))
		if _, err := os.Stat(envAge); err == nil {
			resolvedConfigPath = envAge
		} else if _, err := os.Stat(envYaml); err == nil {
			resolvedConfigPath = envYaml
		}
	}

	// Load encrypted config
	cfg := mainapp.LoadEncryptedConfig(resolvedConfigPath, secretKey)

	// Pass the runtime environment
	runtimeEnv := configuration.RuntimeEnv(*envFlag)
	cfg.Runtime = runtimeEnv

	// Configuration for the Issuer service
	issuerCfg := cfg

	// Create the Issuer service (acting as client of the Issuer server)
	issuanceService, err := credissuance.NewLEARIssuance(issuerCfg)
	if err != nil {
		log.Fatalf("Error creating issuance service: %v", err)
	}

	// Get an access token from the Verifier
	token, err := issuanceService.GetAccessToken(context.Background())
	if err != nil {
		log.Fatalf("Error obtaining access token: %v", err)
	}

	fmt.Println("Access Token:")
	fmt.Println(token)
}
