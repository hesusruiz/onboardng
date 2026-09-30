package mail

import (
	"bytes"
	"crypto/tls"
	"embed"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net/smtp"
	"os"
	"path/filepath"
	"strings"

	"filippo.io/age"
	"github.com/hesusruiz/onboardng/internal/configuration"
	"github.com/hesusruiz/onboardng/internal/db"
	"github.com/hesusruiz/utils/errl"
)

//go:embed templates
var templatesFS embed.FS

type MailSender interface {
	SendWelcomeEmail(reg *db.RegistrationRecord) error
}

type Service struct {
	runtime            configuration.RuntimeEnv
	onboardTeamEmail   []string
	issuerTeamEmail    []string
	ccTeamEmail        []string
	testRecipientEmail string
	smtpConfig         configuration.SMTPConfig
	password           string
	templateDir        string
}

const mimeType = "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"

func NewMailService(runtime configuration.RuntimeEnv, cfg configuration.MailConfig) (*Service, error) {
	if !cfg.SMTP.Enabled {
		return &Service{
			runtime:     runtime,
			smtpConfig:  cfg.SMTP,
			templateDir: cfg.TemplateDir,
		}, nil
	}

	var password string
	if cfg.SMTP.Password != "" {
		// Decrypt the password from configuration
		if cfg.AgeSecretKey == "" {
			return nil, fmt.Errorf("AgeSecretKey is missing but required for encrypted SMTP password")
		}

		identity, err := age.ParseHybridIdentity(cfg.AgeSecretKey)
		if err != nil {
			return nil, errl.Errorf("invalid identity key: %w", err)
		}

		ageReader, err := age.Decrypt(strings.NewReader(cfg.SMTP.Password), identity)
		if err != nil {
			return nil, errl.Errorf("failed to decrypt SMTP password: %w", err)
		}

		passBytes, err := io.ReadAll(ageReader)
		if err != nil {
			return nil, errl.Errorf("failed to read decrypted SMTP password: %w", err)
		}
		password = strings.TrimSpace(string(passBytes))
	} else {
		slog.Warn("SMTP password not found in configuration, reading from file")
		// Read the password from a file
		passwordBytes, err := os.ReadFile(cfg.SMTP.PasswordFile)
		if err != nil {
			return nil, errl.Errorf("failed to read SMTP password file: %w", err)
		}
		password = strings.TrimSpace(string(passwordBytes))
	}

	return &Service{
		runtime:            runtime,
		onboardTeamEmail:   cfg.OnboardTeamEmail,
		issuerTeamEmail:    cfg.IssuerTeamEmail,
		ccTeamEmail:        cfg.CCTeamEmail,
		testRecipientEmail: cfg.TestRecipientEmail,
		smtpConfig:         cfg.SMTP,
		password:           password,
		templateDir:        cfg.TemplateDir,
	}, nil
}

func (s *Service) SendWelcomeEmail(reg *db.RegistrationRecord) error {
	data := map[string]any{
		"RegistrationID":   reg.RegistrationID,
		"Email":            reg.Email,
		"FirstName":        reg.FirstName,
		"LastName":         reg.LastName,
		"CompanyName":      reg.CompanyName,
		"Country":          reg.Country,
		"VatID":            reg.VatID,
		"Runtime":          s.runtime.String(),
		"OnboardTeamEmail": s.onboardTeamEmail[0],
	}

	to := []string{reg.Email}
	cc := s.onboardTeamEmail
	bcc := s.ccTeamEmail
	subject := "Welcome to DOME Marketplace!"

	return s.SendEmail(to, cc, bcc, subject, "email_welcome.html", data)
}

func (s *Service) SendSecondPhaseEmail(reg *db.RegistrationRecord) error {
	data := map[string]any{
		"RegistrationID":   reg.RegistrationID,
		"Email":            reg.Email,
		"FirstName":        reg.FirstName,
		"LastName":         reg.LastName,
		"CompanyName":      reg.CompanyName,
		"Country":          reg.Country,
		"VatID":            reg.VatID,
		"Runtime":          s.runtime.String(),
		"OnboardTeamEmail": s.onboardTeamEmail[0],
	}

	to := []string{reg.Email}
	cc := s.onboardTeamEmail
	bcc := s.ccTeamEmail
	subject := "Welcome to DOME Marketplace!"

	return s.SendEmail(to, cc, bcc, subject, "email_welcome.html", data)
}

func (s *Service) SendVerificationCodeEmail(email string, code string) error {
	data := map[string]any{
		"Code":             code,
		"Runtime":          s.runtime.String(),
		"OnboardTeamEmail": s.onboardTeamEmail[0],
	}

	return s.SendEmail([]string{email}, []string{}, []string{}, "DOME Marketplace Verification Code", "email_verification.html", data)
}

func (s *Service) SendIssuerError(reg *db.RegistrationRecord, payload string, errorMsg string) error {
	data := map[string]any{
		"FirstName":      reg.FirstName,
		"CompanyName":    reg.CompanyName,
		"RegistrationID": reg.RegistrationID,
		"Payload":        payload,
		"ErrorMsg":       errorMsg,
		"Runtime":        s.runtime.String(),
	}

	return s.SendEmail(s.issuerTeamEmail, []string{}, []string{}, "DOME: Error in Credential Issuer during customer registration", "issuer_error.html", data)
}

func (s *Service) SendTestEmail() error {
	data := struct {
		Email            string
		Code             string
		Runtime          configuration.RuntimeEnv
		OnboardTeamEmail string
	}{
		Email:            s.testRecipientEmail,
		Code:             "123456",
		Runtime:          s.runtime,
		OnboardTeamEmail: s.onboardTeamEmail[0],
	}

	return s.SendEmail([]string{s.testRecipientEmail}, []string{}, []string{}, "DOME Marketplace Test Email", "email_test.html", data)
}

// getTemplate loads and parses the template by name.
// First it tries to read the template from disk (for hot-reloading during development),
// then falls back to the embedded templatesFS.
func (s *Service) getTemplate(name string) (*template.Template, error) {
	// Restrict to single filename to prevent path traversal
	name = filepath.Base(name)
	if !strings.HasSuffix(name, ".html") {
		name = name + ".html"
	}

	var content []byte
	var err error

	// 1. Try loading from disk if TemplateDir is configured or in Development mode
	templateDir := s.templateDir
	if templateDir == "" && s.runtime == configuration.Development {
		templateDir = "internal/mail/templates"
	}

	if templateDir != "" {
		filePath := filepath.Join(templateDir, name)
		content, err = os.ReadFile(filePath)
		if err == nil {
			slog.Debug("Loaded email template from disk", "path", filePath)
		}
	}

	// 2. Fall back to embedded FS if not loaded from disk
	if content == nil {
		embedPath := "templates/" + name
		content, err = templatesFS.ReadFile(embedPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read template %q from embed: %w", name, err)
		}
		slog.Debug("Loaded email template from embed", "path", embedPath)
	}

	// 3. Parse the template individually (so "content" definitions don't conflict)
	tmpl, err := template.New(name).Parse(string(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse template %q: %w", name, err)
	}

	return tmpl, nil
}

// SendEmail compiles and sends the email using a specified template.
func (s *Service) SendEmail(to, cc, bcc []string, subject string, templateName string, data any) error {
	if !s.smtpConfig.Enabled {
		return nil
	}

	tmpl, err := s.getTemplate(templateName)
	if err != nil {
		return err
	}

	var body bytes.Buffer
	if err := tmpl.Execute(&body, data); err != nil {
		return fmt.Errorf("failed to execute email template: %w", err)
	}

	// The email is sent from the same email as used to authenticate to the email server
	from := s.smtpConfig.Username

	var allRecipients []string
	allRecipients = append(allRecipients, to...)
	allRecipients = append(allRecipients, cc...)
	allRecipients = append(allRecipients, bcc...)

	// Build the email message
	msg := []byte("From: " + from + "\n" +
		"To: " + strings.Join(to, ", ") + "\n" +
		"Cc: " + strings.Join(cc, ", ") + "\n" +
		"Subject: " + subject + "\n" +
		mimeType + body.String())

	return s.send(from, allRecipients, msg)
}

// send sends the email using the SMTP server.
func (s *Service) send(from string, to []string, msg []byte) error {
	addr := fmt.Sprintf("%s:%d", s.smtpConfig.Host, s.smtpConfig.Port)
	auth := smtp.PlainAuth("", s.smtpConfig.Username, s.password, s.smtpConfig.Host)

	if s.smtpConfig.TLS && s.smtpConfig.Port == 465 {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: false,
			ServerName:         s.smtpConfig.Host,
		}

		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("failed to dial TLS: %w", err)
		}
		defer conn.Close()

		c, err := smtp.NewClient(conn, s.smtpConfig.Host)
		if err != nil {
			return fmt.Errorf("failed to create SMTP client: %w", err)
		}
		defer c.Quit()

		if err = c.Auth(auth); err != nil {
			return fmt.Errorf("failed to authenticate: %w", err)
		}

		if err = c.Mail(from); err != nil {
			return fmt.Errorf("failed to set sender: %w", err)
		}

		for _, addr := range to {
			if err = c.Rcpt(addr); err != nil {
				return fmt.Errorf("failed to add recipient: %w", err)
			}
		}

		w, err := c.Data()
		if err != nil {
			return fmt.Errorf("failed to open data writer: %w", err)
		}

		_, err = w.Write(msg)
		if err != nil {
			return fmt.Errorf("failed to write message: %w", err)
		}

		err = w.Close()
		if err != nil {
			return fmt.Errorf("failed to close data writer: %w", err)
		}

		return nil
	}

	return smtp.SendMail(addr, auth, from, to, msg)
}
