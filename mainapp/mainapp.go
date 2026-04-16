package mainapp

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"filippo.io/age"
	"github.com/fsnotify/fsnotify"
	"github.com/hesusruiz/onboardng/credissuance"
	"github.com/hesusruiz/onboardng/internal/configuration"
	"github.com/hesusruiz/onboardng/internal/db"
	"github.com/hesusruiz/onboardng/internal/mail"
	"github.com/hesusruiz/onboardng/internal/maintenance"
	"github.com/hesusruiz/onboardng/internal/server"
	"github.com/hesusruiz/utils/errl"
	"gopkg.in/yaml.v3"
)

func Run() error {

	// Define the main serve command
	serveCmd := flag.NewFlagSet("serve", flag.ExitOnError)
	serveCfgPath := serveCmd.String("config", "config.age", "Path to config file (.yaml or .age)")
	watchFlag := serveCmd.Bool("watch", false, "watch for changes and start server")
	envFlag := serveCmd.String("env", "dev", "environment to serve (dev, pre or pro)")
	port := serveCmd.String("port", "7777", "port for the server")

	// Define the generate command
	generateCmd := flag.NewFlagSet("generate", flag.ExitOnError)

	// Define the seal command
	sealCmd := flag.NewFlagSet("seal", flag.ExitOnError)
	sealIn := sealCmd.String("in", "config/config.yaml", "Plaintext YAML file to encrypt")
	sealOut := sealCmd.String("out", "config.age", "Target encrypted file path")
	newKey := sealCmd.Bool("newkey", false, "Generate a new key to use for encryption")

	command := os.Getenv("ONBOARDNG_COMMAND")
	var cmdArgs []string

	if command != "" {
		// If command is from ENV, all CLI args are potential flags for that command
		if len(os.Args) > 1 {
			cmdArgs = os.Args[1:]
		}
	} else {
		// If no ENV command, use CLI arg as command
		if len(os.Args) < 2 {
			usage(serveCmd, generateCmd, sealCmd)
			return nil
		}
		command = os.Args[1]
		if len(os.Args) > 2 {
			cmdArgs = os.Args[2:]
		}
	}

	// Route the Command
	switch command {
	case "serve":
		// Start the server
		serveCmd.Parse(cmdArgs)

		// Environment variables take precedence over command-line flags
		if envConfig := os.Getenv("ONBOARDNG_CONFIG"); envConfig != "" {
			*serveCfgPath = envConfig
		}
		if envWatch := os.Getenv("ONBOARDNG_WATCH"); envWatch != "" {
			*watchFlag = (envWatch == "true" || envWatch == "1")
		}
		if runtimeEnv := os.Getenv("ONBOARDNG_ENV"); runtimeEnv != "" {
			*envFlag = runtimeEnv
		}
		if envPort := os.Getenv("ONBOARDNG_PORT"); envPort != "" {
			*port = envPort
		}

		// Read and immediately UNSET the secret key from the environment
		secretKey := os.Getenv("AGE_SECRET_KEY")
		if secretKey != "" {
			os.Unsetenv("AGE_SECRET_KEY")
			slog.Info("🔐 AGE_SECRET_KEY captured and removed from environment")
		}

		// Read secret key from file if not already set by env var
		if secretKey == "" {
			// Try reading from config/age_secret_key.txt
			if data, err := os.ReadFile("config/age_secret_key.txt"); err == nil {
				secretKey = strings.TrimSpace(string(data))
			}
		}

		cfg := LoadEncryptedConfig(*serveCfgPath, secretKey)

		return serve(cfg, *envFlag, *port, *watchFlag, secretKey)

	case "generate":
		// Generate the frontend
		generateCmd.Parse(cmdArgs)

		// Read and immediately UNSET the secret key from the environment
		secretKey := os.Getenv("AGE_SECRET_KEY")
		if secretKey != "" {
			os.Unsetenv("AGE_SECRET_KEY")
		}

		// Read secret key from file if not already set by env var
		if secretKey == "" {
			// Try reading from config/age_secret_key.txt
			if data, err := os.ReadFile("config/age_secret_key.txt"); err == nil {
				secretKey = strings.TrimSpace(string(data))
			}
		}

		cfg := LoadEncryptedConfig(*serveCfgPath, secretKey)
		return generate(cfg)

	case "seal":
		// Seal the config file
		sealCmd.Parse(cmdArgs)
		if err := sealConfig(*sealIn, *sealOut, *newKey); err != nil {
			log.Fatalf("Error sealing config: %v", err)
		}

	default:
		// Show usage
		usage(serveCmd, generateCmd, sealCmd)
	}

	return nil

}

func usage(runCmd, generateCmd, sealCmd *flag.FlagSet) {
	fmt.Printf("Usage: %s <command> [options]\n\n", filepath.Base(os.Args[0]))

	fmt.Println("Commands:")

	fmt.Println("serve       Run the server")
	runCmd.PrintDefaults()
	fmt.Println()

	fmt.Println("generate  Generate the frontend")
	generateCmd.PrintDefaults()
	fmt.Println()

	fmt.Println("seal      Seal the config file")
	sealCmd.PrintDefaults()
	fmt.Println()
}

func serve(cfg configuration.Config, envFlag string, port string, watchFlag bool, secretKey string) error {

	runtimeEnv := configuration.RuntimeEnv(envFlag)

	slog.Info("Starting server", "env", envFlag, "port", port, "watch", watchFlag)

	// Get the configuration corresponding to the runtime environment
	srvConfig, ok := cfg.Environments[envFlag]
	if !ok {
		return errl.Errorf("environment %s not found in config", envFlag)
	}

	srvConfig.Runtime = runtimeEnv
	fmt.Println(srvConfig.String())

	// Pass the secret key to the environment configuration
	srvConfig.AgeSecretKey = secretKey
	srvConfig.Mail.AgeSecretKey = secretKey
	cfg.Environments[envFlag] = srvConfig

	// Setup issuer
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
		Features: srvConfig.Features,
	}
	issuanceService, err := credissuance.NewLEARIssuance(issuerCfg)
	if err != nil {
		slog.Error("❌ Error creating issuance service", "error", errl.Error(err))
		os.Exit(1)
	}

	// Initialize Database service
	dbService, err := db.NewDBService(runtimeEnv, "data/onboarding.db")
	if err != nil {
		slog.Error("❌ Error initializing database service", "error", errl.Error(err))
		return err
	}
	defer dbService.Close()

	// Initialize Mail service
	mailService, err := mail.NewMailService(runtimeEnv, srvConfig.Mail)
	if err != nil {
		slog.Error("❌ Error initializing mail service", "error", errl.Error(err))
		return err
	}

	// Initialize and start automated Maintenance service
	maintenanceService := maintenance.NewMaintenanceService()
	// Schedule database maintenance everyday at 03:00
	maintenanceService.AddTask("Database Maintenance", maintenance.Schedule{Hour: 3, Minute: 0}, dbService.RunMaintenance)
	maintenanceService.Start()

	// Run the database maintenance once at startup
	dbService.RunMaintenance(context.Background())

	adminUser := os.Getenv("ADMIN_USER")
	if adminUser == "" {
		if runtimeEnv != configuration.Production {
			adminUser = "admin"
		} else {
			log.Fatalf("Error: ADMIN_USER environment variable is not set")
		}
	}
	adminPassword := os.Getenv("ADMIN_PASSWORD")
	if adminPassword == "" {
		if runtimeEnv != configuration.Production {
			adminPassword = "pepe"
		} else {
			log.Fatalf("Error: ADMIN_PASSWORD environment variable is not set")
		}
	}

	srv, err := server.NewServer(runtimeEnv, dbService, issuanceService, mailService, cfg.DestDir, adminUser, adminPassword, srvConfig.Features)
	if err != nil {
		log.Fatalf("Error: Could not create server: %v", err)
	}

	// Start Watcher if requested
	if watchFlag {
		go startWatcher(cfg)
	}

	// Start Server
	slog.Info("🚀 Server running", "env", envFlag, "dir", cfg.DestDir, "url", "https://onboarddev.dome.mycredential.eu")
	if err := http.ListenAndServe(":"+port, srv.Handler); err != nil {
		newerr := errl.Error(err)
		slog.Error("Server failed", "error", newerr)
		return newerr
	}

	return nil
}

func LoadEncryptedConfig(path string, secretKey string) configuration.Config {
	var source io.ReadCloser

	// Handle both remote and local files
	if strings.HasPrefix(path, "https://") {
		fmt.Printf("Fetching remote config from %s\n", path)
		resp, err := http.Get(path)
		if err != nil {
			log.Fatalf("Error: Could not fetch remote config: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			log.Fatalf("Error: Remote config returned status %d", resp.StatusCode)
		}
		source = resp.Body
	} else {
		file, err := os.Open(path)
		if err != nil {
			log.Fatalf("Error: Could not open config file: %v", err)
		}
		source = file
	}
	defer source.Close()

	// Read the entire source into memory to calculate the hash
	configBytes, err := io.ReadAll(source)
	if err != nil {
		log.Fatalf("Error: Could not read config source: %v", err)
	}

	// Calculate and print SHA256 hash
	hash := sha256.Sum256(configBytes)
	fmt.Printf("Config file SHA256: %s\n", hex.EncodeToString(hash[:]))

	var reader io.Reader = bytes.NewReader(configBytes)
	if strings.HasSuffix(path, ".age") {
		// Decryption mode
		if secretKey == "" {
			log.Fatal("Error: AGE_SECRET_KEY environment variable is missing but required for .age files")
		}

		identity, err := age.ParseHybridIdentity(secretKey)
		if err != nil {
			log.Fatalf("Error: Invalid identity key: %v", err)
		}

		ageReader, err := age.Decrypt(reader, identity)
		if err != nil {
			log.Fatalf("Error: Failed to decrypt file: %v", err)
		}
		reader = ageReader
	} else {
		// Standard YAML mode (Development)
		slog.Warn("Running in Development Mode (Unencrypted YAML)")
	}

	// Parse YAML
	var cfg configuration.Config
	if err := yaml.NewDecoder(reader).Decode(&cfg); err != nil {
		log.Fatalf("Error: Failed to parse YAML: %v", err)
	}

	// Prepare each environment: decrypt or read internal credentials
	for name, env := range cfg.Environments {
		env.AgeSecretKey = secretKey

		// Prepare Private Key
		priv, err := decryptInternalCredential(env.PrivateKey, env.PrivateKeyFile, secretKey)
		if err != nil {
			log.Fatalf("Error preparing private key for environment %s: %v", name, err)
		}
		env.PrivateKey = priv

		// Prepare Machine Credential
		machine, err := decryptInternalCredential(env.MachineCredential, env.MachineCredentialFile, secretKey)
		if err != nil {
			log.Fatalf("Error preparing machine credential for environment %s: %v", name, err)
		}
		env.MachineCredential = machine

		cfg.Environments[name] = env
	}

	return cfg
}

// decryptInternalCredential handles the logic of either decrypting an embedded credential or reading it from a file.
func decryptInternalCredential(encryptedData, filePath, secretKey string) (string, error) {
	if encryptedData != "" {
		if secretKey == "" {
			return "", errl.Errorf("secret key is missing but required for decryption")
		}
		identity, err := age.ParseHybridIdentity(secretKey)
		if err != nil {
			return "", errl.Errorf("invalid identity key: %w", err)
		}
		ageReader, err := age.Decrypt(strings.NewReader(encryptedData), identity)
		if err != nil {
			return "", errl.Errorf("failed to decrypt: %w", err)
		}
		buf, err := io.ReadAll(ageReader)
		if err != nil {
			return "", errl.Errorf("error reading decrypted data: %w", err)
		}
		return string(buf), nil
	}

	if filePath != "" {
		buf, err := os.ReadFile(filePath)
		if err != nil {
			return "", errl.Errorf("error reading from file %s: %w", filePath, err)
		}
		return string(buf), nil
	}

	return "", nil
}

// sealConfig encrypts a plain config file using age encryption
// It also encrypts the private key and machine credential, reading from the file paths specified in the config.
// The encrypted private key and machine credential are stored in the corresponding fields in the config.
func sealConfig(inputPath, outputPath string, newKey bool) error {
	// Load plain unencrypted config file
	plainBytes, err := os.ReadFile(inputPath)
	if err != nil {
		return err
	}

	// Parse YAML
	var cfg configuration.Config
	if err := yaml.NewDecoder(bytes.NewReader(plainBytes)).Decode(&cfg); err != nil {
		return err
	}

	// If newKey is true, generate a new key
	// Otherwise, use the one in the config/age_secret_key.txt file
	var identity *age.HybridIdentity
	var publicKey age.Recipient
	var secretKeyPath string
	if newKey {

		// Generate new identity (Private + Public)
		identity, err = age.GenerateHybridIdentity()
		if err != nil {
			log.Fatalf("Error: Key generation failed: %v", err)
		}
		publicKey = identity.Recipient()

		// Save the private key to a file in the config/ directory
		// This directory is ignored by git
		secretKeyPath = filepath.Join("config", "age_secret_key.txt")
		if err := os.WriteFile(secretKeyPath, []byte(identity.String()), 0600); err != nil {
			return errl.Errorf("failed to save private key to %s: %w", secretKeyPath, err)
		}
	} else {
		// Read the public key from the existing secret key file
		secretKeyPath = filepath.Join("config", "age_secret_key.txt")
		secretKeyBytes, err := os.ReadFile(secretKeyPath)
		if err != nil {
			return errl.Errorf("failed to read private key from %s: %w", secretKeyPath, err)
		}
		identity, err = age.ParseHybridIdentity(string(secretKeyBytes))
		if err != nil {
			return errl.Errorf("failed to parse private key from %s: %w", secretKeyPath, err)
		}
		publicKey = identity.Recipient()
	} // End of if newKey

	// Encrypt private key and machine credential for each environment
	// Encrypt also the SMTP password
	for name, env := range cfg.Environments {
		privateKeyEncrypted, err := sealFile(env.PrivateKeyFile, publicKey)
		if err != nil {
			return err
		}
		machineCredentialEncrypted, err := sealFile(env.MachineCredentialFile, publicKey)
		if err != nil {
			return err
		}
		smtpPasswordEncrypted, err := sealFile(env.Mail.SMTP.PasswordFile, publicKey)
		if err != nil {
			return err
		}
		env.PrivateKey = string(privateKeyEncrypted)
		env.MachineCredential = string(machineCredentialEncrypted)
		env.Mail.SMTP.Password = string(smtpPasswordEncrypted)
		cfg.Environments[name] = env
	}

	// Marshall to YAML
	yamlBytes, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	ageWriter, err := age.Encrypt(&buf, publicKey)
	if err != nil {
		return err
	}
	if _, err := ageWriter.Write(yamlBytes); err != nil {
		return err
	}
	ageWriter.Close()

	// Write config file
	if err := os.WriteFile(outputPath, buf.Bytes(), 0600); err != nil {
		return err
	}

	// Print a different message depending on whether the key was generated or read
	if newKey {

		// Present the credentials to the user
		fmt.Println("=======================================================================")
		fmt.Println("✅ CONFIGURATION SEALED WITH POST-QUANTUM ENCRYPTION (MLKEM768-X25519)")
		fmt.Println("=======================================================================")
		fmt.Printf("Encrypted File: %s\n", outputPath)
		fmt.Printf("Private Key:    %s\n", identity.String())
		fmt.Printf("Key saved in:   %s (GIT-IGNORED)\n", secretKeyPath)
		fmt.Println("=======================================================================")
		fmt.Println("ACTION REQUIRED:")
		fmt.Println("1. Commit the .age file to your repository.")
		fmt.Println("2. Set the Private Key as AGE_SECRET_KEY in your environment.")
		fmt.Printf("   (Or use the saved key in %s)\n", secretKeyPath)
		fmt.Println("3. DO NOT LOSE THIS KEY. It cannot be recovered.")
		fmt.Println("=======================================================================")
	} else {
		// Present the credentials to the user
		fmt.Println("=======================================================================")
		fmt.Println("✅ CONFIGURATION SEALED WITH POST-QUANTUM ENCRYPTION (MLKEM768-X25519)")
		fmt.Println("=======================================================================")
		fmt.Printf("Encrypted File: %s\n", outputPath)
		fmt.Printf("Private Key:    %s\n", identity.String())
		fmt.Printf("Key read from:  %s\n", secretKeyPath)
		fmt.Println("=======================================================================")
		fmt.Println("ACTION REQUIRED:")
		fmt.Println("1. Commit the .age file to your repository.")
		fmt.Println("2. Set the Private Key as AGE_SECRET_KEY in your environment.")
		fmt.Println("3. DO NOT LOSE THIS KEY. It cannot be recovered.")
		fmt.Println("=======================================================================")
	}

	return nil
}

// sealFile encrypts a file using age encryption
// It returns the encrypted file as a byte array
func sealFile(inputPath string, publicKey age.Recipient) ([]byte, error) {
	inputFile, err := os.Open(inputPath)
	if err != nil {
		return nil, err
	}
	defer inputFile.Close()

	var buf bytes.Buffer
	ageWriter, err := age.Encrypt(&buf, publicKey)
	if err != nil {
		return nil, err
	}

	if _, err := io.Copy(ageWriter, inputFile); err != nil {
		return nil, err
	}
	if err := ageWriter.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func startWatcher(cfg configuration.Config) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		slog.Error("❌ Watcher Error", "error", err)
		return
	}
	defer watcher.Close()

	watchPaths := []string{
		cfg.SrcDir,
	}

	for _, path := range watchPaths {
		err = filepath.Walk(path, func(walkPath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return watcher.Add(walkPath)
			}
			return nil
		})
		if err != nil {
			slog.Warn("⚠️ Error walking path", "path", path, "error", err)
		}
	}

	slog.Info("👀 Watching for changes...")

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) != 0 {
				slog.Info("📝 File updated. Regenerating...", "file", event.Name)
				generate(cfg)
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			slog.Error("❌ Watcher error", "error", err)
		}
	}
}
