package mainapp

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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

	// Define the main run command
	runCmd := flag.NewFlagSet("run", flag.ExitOnError)
	runCfgPath := runCmd.String("config", "config.age", "Path to config file (.yaml or .age)")
	watchFlag := runCmd.Bool("watch", false, "watch for changes and start server")
	envFlag := runCmd.String("env", "dev", "environment to serve (dev, pre or pro)")
	port := runCmd.String("port", "7777", "port for the server")

	// Define the generate command
	generateCmd := flag.NewFlagSet("generate", flag.ExitOnError)

	// Define the seal command
	sealCmd := flag.NewFlagSet("seal", flag.ExitOnError)
	sealIn := sealCmd.String("in", "config/config.yaml", "Plaintext YAML file to encrypt")
	sealOut := sealCmd.String("out", "config.age", "Target encrypted file path")

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
			usage(runCmd, generateCmd, sealCmd)
			return nil
		}
		command = os.Args[1]
		if len(os.Args) > 2 {
			cmdArgs = os.Args[2:]
		}
	}

	// Route the Command
	switch command {
	case "run":
		// Start the server
		runCmd.Parse(cmdArgs)

		// Environment variables take precedence over command-line flags
		if envVal := os.Getenv("ONBOARDNG_CONFIG"); envVal != "" {
			*runCfgPath = envVal
		}
		if envVal := os.Getenv("ONBOARDNG_WATCH"); envVal != "" {
			*watchFlag = (envVal == "true" || envVal == "1")
		}
		if envVal := os.Getenv("ONBOARDNG_ENV"); envVal != "" {
			*envFlag = envVal
		}
		if envVal := os.Getenv("ONBOARDNG_PORT"); envVal != "" {
			*port = envVal
		}

		// Read and immediately UNSET the secret key from the environment
		secretKey := os.Getenv("AGE_SECRET_KEY")
		if secretKey != "" {
			os.Unsetenv("AGE_SECRET_KEY")
			slog.Info("🔐 AGE_SECRET_KEY captured and removed from environment")
		}

		cfg := loadEncryptedConfig(*runCfgPath, secretKey)
		return run(cfg, *envFlag, *port, *watchFlag, secretKey)

	case "generate":
		// Generate the frontend
		generateCmd.Parse(cmdArgs)

		// Read and immediately UNSET the secret key from the environment
		secretKey := os.Getenv("AGE_SECRET_KEY")
		if secretKey != "" {
			os.Unsetenv("AGE_SECRET_KEY")
		}

		cfg := loadEncryptedConfig(*runCfgPath, secretKey)
		return generate(cfg)

	case "seal":
		// Seal the config file
		sealCmd.Parse(cmdArgs)
		if err := sealConfig(*sealIn, *sealOut); err != nil {
			log.Fatalf("Error sealing config: %v", err)
		}

	default:
		// Show usage
		usage(runCmd, generateCmd, sealCmd)
	}

	return nil

}

func usage(runCmd, generateCmd, sealCmd *flag.FlagSet) {
	fmt.Printf("Usage: %s <command> [options]\n\n", filepath.Base(os.Args[0]))

	fmt.Println("Commands:")

	fmt.Println("run       Run the server")
	runCmd.PrintDefaults()
	fmt.Println()

	fmt.Println("generate  Generate the frontend")
	generateCmd.PrintDefaults()
	fmt.Println()

	fmt.Println("seal      Seal the config file")
	sealCmd.PrintDefaults()
	fmt.Println()
}

func run(cfg configuration.Config, envFlag string, port string, watchFlag bool, secretKey string) error {
	// Get the environment config
	srvConfig, ok := cfg.Environments[envFlag]
	if !ok {
		slog.Error("❌ Environment not found in config", "env", envFlag)
		return errl.Errorf("environment %s not found", envFlag)
	}

	// Pass the secret key to the environment configuration
	srvConfig.AgeSecretKey = secretKey
	srvConfig.Mail.AgeSecretKey = secretKey
	cfg.Environments[envFlag] = srvConfig

	runtimeEnv := configuration.RuntimeEnv(envFlag)

	// Setup issuer
	issuerCfg := configuration.EnvConfig{
		Runtime:               runtimeEnv,
		AgeSecretKey:          srvConfig.AgeSecretKey,
		Debug:                 srvConfig.Debug,
		PrivateKeyFile:        srvConfig.PrivateKeyFile,
		PrivateKey:            srvConfig.PrivateKey,
		MachineCredentialFile: srvConfig.MachineCredentialFile,
		MachineCredential:     srvConfig.MachineCredential,
		MyDidkey:              srvConfig.MyDidkey,
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
	issuanceService, err := credissuance.NewLEARIssuance(issuerCfg)
	if err != nil {
		slog.Error("❌ Error creating issuance service", "error", errl.Error(err))
		os.Exit(1)
	}

	// Initialize Database service
	dbService, err := db.NewService(runtimeEnv, "data/onboarding.db")
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

	dbService.RunMaintenance(context.Background())

	srv := server.NewServer(dbService, issuanceService, mailService, cfg.DestDir)

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

func loadEncryptedConfig(path string, secretKey string) configuration.Config {
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

	// 2. Parse YAML
	var cfg configuration.Config
	if err := yaml.NewDecoder(reader).Decode(&cfg); err != nil {
		log.Fatalf("Error: Failed to parse YAML: %v", err)
	}

	// 3. Start Application - Pretty print the config in JSON format
	jsonConfig, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		log.Fatalf("Error: Failed to marshal config: %v", err)
	}
	fmt.Println(string(jsonConfig))

	return cfg
}

// sealConfig encrypts a plain config file using age encryption
// It also encrypts the private key and machine credential, reading from the file paths specified in the config.
// The encrypted private key and machine credential are stored in the corresponding fields in the config.
func sealConfig(inputPath, outputPath string) error {
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

	// Generate new identity (Private + Public)
	identity, err := age.GenerateHybridIdentity()
	if err != nil {
		log.Fatalf("Error: Key generation failed: %v", err)
	}
	publicKey := identity.Recipient()

	// Save the private key to a file in the config/ directory
	// This directory is ignored by git
	secretKeyPath := filepath.Join("config", "age_secret_key.txt")
	if err := os.WriteFile(secretKeyPath, []byte(identity.String()), 0600); err != nil {
		return errl.Errorf("failed to save private key to %s: %w", secretKeyPath, err)
	}

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
