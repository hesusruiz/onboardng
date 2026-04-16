package configuration

import (
	"fmt"
	"strings"
)

type RuntimeEnv string

const (
	Development   RuntimeEnv = "dev"
	Preproduction RuntimeEnv = "pre"
	Production    RuntimeEnv = "pro"
)

// Config holds the configuration for the application.
type Config struct {
	// The directory where the generated frontend will be placed.
	DestDir string `yaml:"dest_dir"`
	// The directory where the source code for the frontend is located.
	SrcDir string `yaml:"src_dir"`
	// The name of the application.
	AppName string `yaml:"app_name"`
	// The environments configuration.
	Environments map[string]EnvConfig `yaml:"environments"`
}

// EnvConfig holds the configuration for a specific environment.
type EnvConfig struct {
	// The runtime environment.
	Runtime RuntimeEnv

	// The decrypted AGE secret key used for decryption of embedded files.
	// This is not stored in the YAML config file.
	AgeSecretKey string `yaml:"-"`

	// Whether the environment is in debug mode.
	Debug bool `yaml:"debug"`

	// The DID key of the issuer.
	MyDidkey string `yaml:"mydidkey,omitempty"`

	// The path to the private key file corresponding to the didKey, which must be placed in a secure place.
	PrivateKeyFile string `yaml:"privateKeyFile,omitempty"`
	PrivateKey     string `yaml:"privateKey,omitempty"`

	// The path to the machine credential file, which must be placed in a secure place.
	MachineCredentialFile string `yaml:"machineCredentialFile,omitempty"`
	MachineCredential     string `yaml:"machineCredential,omitempty"`

	// The verifier configuration.
	Verifier VerifierConfig `yaml:"verifier"`
	// The issuer configuration.
	Issuer IssuerConfig `yaml:"issuer"`
	// The TMForum configuration.
	TMForum TMForumConfig `yaml:"tmforum"`
	// The mail configuration.
	Mail MailConfig `yaml:"mail"`

	// The features configuration.
	Features Features `yaml:"features"`
}

// Features defines a set of feature flags which may depend on the environment at a given time
type Features struct {
	TMFServerEnabled bool `yaml:"tmfServerEnabled"`
}

func (e *EnvConfig) String() string {

	var b strings.Builder

	fmt.Fprintf(&b, "Runtime: %s\n", e.Runtime)
	fmt.Fprintf(&b, "Debug: %t\n", e.Debug)
	fmt.Fprintf(&b, "MyDidkey: %s\n", e.MyDidkey)
	fmt.Fprintf(&b, "PrivateKeyFile: %s\n", e.PrivateKeyFile)
	fmt.Fprintf(&b, "MachineCredentialFile: %s\n", e.MachineCredentialFile)
	fmt.Fprintf(&b, "Verifier:\n")
	fmt.Fprintf(&b, "  URL: %s\n", e.Verifier.URL)
	fmt.Fprintf(&b, "  TokenEndpoint: %s\n", e.Verifier.TokenEndpoint)
	fmt.Fprintf(&b, "Issuer:\n")
	fmt.Fprintf(&b, "  CredentialIssuancePath: %s\n", e.Issuer.CredentialIssuancePath)
	fmt.Fprintf(&b, "TMForum:\n")
	fmt.Fprintf(&b, "  BaseURL: %s\n", e.TMForum.BaseURL)
	fmt.Fprintf(&b, "Mail:\n")
	fmt.Fprintf(&b, "  OnboardTeamEmail: %v\n", e.Mail.OnboardTeamEmail)
	fmt.Fprintf(&b, "  IssuerTeamEmail: %v\n", e.Mail.IssuerTeamEmail)
	fmt.Fprintf(&b, "  CCTeamEmail: %v\n", e.Mail.CCTeamEmail)
	fmt.Fprintf(&b, "  TestRecipientEmail: %s\n", e.Mail.TestRecipientEmail)
	fmt.Fprintf(&b, "  SMTP:\n")
	fmt.Fprintf(&b, "    Enabled: %t\n", e.Mail.SMTP.Enabled)
	fmt.Fprintf(&b, "    Host: %s\n", e.Mail.SMTP.Host)
	fmt.Fprintf(&b, "    Port: %d\n", e.Mail.SMTP.Port)
	fmt.Fprintf(&b, "    TLS: %t\n", e.Mail.SMTP.TLS)
	fmt.Fprintf(&b, "    Username: %s\n", e.Mail.SMTP.Username)
	fmt.Fprintf(&b, "    PasswordFile: %s\n", e.Mail.SMTP.PasswordFile)

	return b.String()
}

type VerifierConfig struct {
	URL           string `yaml:"url,omitempty"`
	TokenEndpoint string `yaml:"token_endpoint,omitempty"`
}

type IssuerConfig struct {
	CredentialIssuancePath string `yaml:"credentialIssuancePath,omitempty"`
}

type TMForumConfig struct {
	BaseURL string `yaml:"baseUrl,omitempty"`
}

type MailConfig struct {
	AgeSecretKey       string   `yaml:"-"`
	OnboardTeamEmail   []string `yaml:"onboard_team_email"`
	IssuerTeamEmail    []string `yaml:"issuer_team_email"`
	CCTeamEmail        []string `yaml:"cc_list_email"`
	TestRecipientEmail string   `yaml:"test_recipient_email"`
	SMTP               SMTPConfig
}

type SMTPConfig struct {
	Enabled      bool   `json:"enabled,omitempty" yaml:"enabled"`
	Host         string `json:"host,omitempty" yaml:"host"`
	Port         int    `json:"port,omitempty" yaml:"port"`
	TLS          bool   `json:"tls,omitempty" yaml:"tls"`
	Username     string `json:"username,omitempty" yaml:"username"`
	PasswordFile string `json:"passwordFile,omitempty" yaml:"passwordFile"`
	Password     string `json:"password,omitempty" yaml:"password"`
}
