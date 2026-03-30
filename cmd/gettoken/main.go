package main

import (
	"flag"
	"fmt"
	"log"
	"os"
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

	//Load encrypted config
	cfg := mainapp.LoadEncryptedConfig(*configPath, secretKey)

	// Get the environment config for the runtime environment
	srvConfig, ok := cfg.Environments[*envFlag]
	if !ok {
		log.Fatalf("Environment %s not found in config", *envFlag)
	}

	// Pass the runtime environment
	runtimeEnv := configuration.RuntimeEnv(*envFlag)

	// Configuration for the Issuer service
	issuerCfg := configuration.EnvConfig{
		Runtime:           runtimeEnv,
		Debug:             srvConfig.Debug,
		PrivateKey:        srvConfig.PrivateKey,
		MachineCredential: srvConfig.MachineCredential,
		MyDidkey:          srvConfig.MyDidkey,
		Verifier: configuration.VerifierConfig{
			URL:           srvConfig.Verifier.URL,
			TokenEndpoint: srvConfig.Verifier.TokenEndpoint,
		},
		Issuer: configuration.IssuerConfig{
			CredentialIssuancePath: srvConfig.Issuer.CredentialIssuancePath,
		},
		TMForum: configuration.TMForumConfig{
			BaseURL: srvConfig.TMForum.BaseURL,
		},
	}

	// Create the Issuer service (acting as client of the Issuer server)
	issuanceService, err := credissuance.NewLEARIssuance(issuerCfg)
	if err != nil {
		log.Fatalf("Error creating issuance service: %v", err)
	}

	// Get an access token from the Verifier
	token, err := issuanceService.GetAccessToken()
	if err != nil {
		log.Fatalf("Error obtaining access token: %v", err)
	}

	fmt.Println("Access Token:")
	fmt.Println(token)
}
