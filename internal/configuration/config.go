package configuration

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

	// The path to the private key file, which must be placed in a secure place.
	PrivateKeyFile string `yaml:"privateKeyFile,omitempty"`
	PrivateKey     string `yaml:"privateKey,omitempty"`

	// The path to the machine credential file, which must be placed in a secure place.
	MachineCredentialFile string `yaml:"machineCredentialFile,omitempty"`
	MachineCredential     string `yaml:"machineCredential,omitempty"`

	// The DID key of the issuer.
	MyDidkey string `yaml:"mydidkey,omitempty"`
	// The verifier configuration.
	Verifier VerifierConfig `yaml:"verifier"`
	// The issuer configuration.
	Issuer IssuerConfig `yaml:"issuer"`
	// The TMForum configuration.
	TMForum TMForumConfig `yaml:"tmforum"`
	// The mail configuration.
	Mail MailConfig `yaml:"mail"`
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
	AgeSecretKey     string   `yaml:"-"`
	OnboardTeamEmail []string `yaml:"onboard_team_email"`
	IssuerTeamEmail  []string `yaml:"issuer_team_email"`
	CCTeamEmail      []string `yaml:"cc_list_email"`
	SMTP             SMTPConfig
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
