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

// SendFirstPhaseEmail sends the welcome email to the registered user, after completing first phase of Onboarding.
// It copies the interested internal people.
func (s *Service) SendFirstPhaseEmail(reg *db.RegistrationRecord) error {
	data := map[string]any{
		"RegistrationID":   reg.RegistrationID,
		"Email":            reg.Email,
		"FirstName":        reg.FirstName,
		"LastName":         reg.LastName,
		"CompanyName":      reg.CompanyName,
		"Country":          reg.Country,
		"VatID":            reg.VatID,
		"Runtime":          s.runtime,
		"OnboardTeamEmail": s.onboardTeamEmail,
	}

	subject := "Welcome to DOME Marketplace!"

	// Send mail to the user who onboarded
	to := []string{reg.Email}
	// Copy the interested internal people
	cc := s.onboardTeamEmail
	bcc := s.ccTeamEmail

	return s.SendEmail(to, cc, bcc, subject, "email_first_phase.html", data)
}

// SendSecondPhaseEmail sends an email to the user to indicate that they need to complete the second phase of the onboarding process.
// The second phase is for the user to complete the onboarding process in the Credential Issuer.
func (s *Service) SendSecondPhaseEmail(reg *db.RegistrationRecord) error {
	data := map[string]any{
		"Reg":              reg,
		"RegistrationID":   reg.RegistrationID,
		"Email":            reg.Email,
		"FirstName":        reg.FirstName,
		"LastName":         reg.LastName,
		"CompanyName":      reg.CompanyName,
		"Country":          reg.Country,
		"VatID":            reg.VatID,
		"Runtime":          s.runtime,
		"OnboardTeamEmail": s.onboardTeamEmail,
	}

	subject := "Welcome to DOME Marketplace!"
	// Send mail to the user who onboarded
	to := []string{reg.Email}
	// Copy the interested internal people
	cc := s.onboardTeamEmail
	bcc := s.ccTeamEmail

	return s.SendEmail(to, cc, bcc, subject, "email_second_phase.html", data)
}

// SendReminderEmail sends an email to the user after a period of time to remind them to complete the onboarding process.
func (s *Service) SendReminderEmail(reg *db.RegistrationRecord) error {
	data := map[string]any{
		"RegistrationID":   reg.RegistrationID,
		"Email":            reg.Email,
		"FirstName":        reg.FirstName,
		"LastName":         reg.LastName,
		"CompanyName":      reg.CompanyName,
		"Country":          reg.Country,
		"VatID":            reg.VatID,
		"Runtime":          s.runtime,
		"OnboardTeamEmail": s.onboardTeamEmail,
	}

	subject := "Reminder: Please complete your registration to DOME Marketplace!"
	// Send mail to the user who onboarded
	to := []string{reg.Email}
	// Copy the interested internal people
	cc := s.onboardTeamEmail
	bcc := s.ccTeamEmail

	return s.SendEmail(to, cc, bcc, subject, "email_reminder.html", data)
}

// SendVerificationCodeEmail sends an email to the user with a verification code, to check that the user controls the email.
func (s *Service) SendVerificationCodeEmail(email string, code string) error {
	data := map[string]any{
		"Code":             code,
		"Runtime":          s.runtime,
		"OnboardTeamEmail": s.onboardTeamEmail,
	}

	subject := "DOME Marketplace Verification Code"
	// Send mail to the user
	to := []string{email}
	// No cc and bcc
	cc := []string{}
	bcc := []string{}

	return s.SendEmail(to, cc, bcc, subject, "email_verification.html", data)
}

// SendIssuerErrorEmail sends an email to the team in charge of the Issuer so they can handle the techinical error.
// The payload is the payload sent to the issuer.
func (s *Service) SendIssuerErrorEmail(reg *db.RegistrationRecord, payload string, errorMsg string) error {
	data := map[string]any{
		"FirstName":      reg.FirstName,
		"CompanyName":    reg.CompanyName,
		"RegistrationID": reg.RegistrationID,
		"Payload":        payload,
		"ErrorMsg":       errorMsg,
		"Runtime":        s.runtime,
	}

	subject := "DOME: Error in Credential Issuer during customer registration"
	// Send mail to the team in charge of the Issuer so they can handle the techinical error
	to := s.issuerTeamEmail
	// No cc and bcc
	cc := []string{}
	bcc := []string{}

	return s.SendEmail(to, cc, bcc, subject, "issuer_error.html", data)
}

// SendCheckIssuerErrorEmail sends an email to the team in charge of the Issuer if an error was received during
// the self-checks that the serfer performs at startup and scheduled times.
func (s *Service) SendCheckIssuerErrorEmail(payload string, errorMsg string) error {
	data := map[string]any{
		"Payload":  payload,
		"ErrorMsg": errorMsg,
		"Runtime":  s.runtime,
	}

	subject := "DOME: Error in Credential Issuer during customer self-registration test"
	return s.SendEmail(s.issuerTeamEmail, []string{s.testRecipientEmail}, []string{}, subject, "check_issuer_error.html", data)
}

func (s *Service) SendTestEmail() error {
	data := map[string]any{
		"Email":            s.testRecipientEmail,
		"Code":             "123456",
		"Runtime":          s.runtime,
		"OnboardTeamEmail": s.onboardTeamEmail,
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
	if templateDir == "" && s.runtime.IsDevelopment() {
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

	// Build the email message. The bcc users will receive the message but will not be displayed in the message headers.
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
