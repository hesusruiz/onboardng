package mail

import (
	"bytes"
	"crypto/tls"
	_ "embed"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net/smtp"
	"os"
	"strings"

	"filippo.io/age"
	"github.com/hesusruiz/onboardng/internal/configuration"
	"github.com/hesusruiz/onboardng/internal/db"
	"github.com/hesusruiz/utils/errl"
)

//go:embed templates/email_welcome.html
var emailWelcomeTemplate string

//go:embed templates/issuer_error.html
var issuerErrorTemplate string

//go:embed templates/email_verification.html
var emailVerificationTemplate string

type MailSender interface {
	SendWelcomeEmail(reg *db.RegistrationRecord) error
}

type Service struct {
	runtime          configuration.RuntimeEnv
	onboardTeamEmail []string
	issuerTeamEmail  []string
	ccTeamEmail      []string
	smtpConfig       configuration.SMTPConfig
	password         string
}

func NewMailService(runtime configuration.RuntimeEnv, cfg configuration.MailConfig) (*Service, error) {
	if !cfg.SMTP.Enabled {
		return &Service{runtime: runtime, smtpConfig: cfg.SMTP}, nil
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
		runtime:          runtime,
		onboardTeamEmail: cfg.OnboardTeamEmail,
		issuerTeamEmail:  cfg.IssuerTeamEmail,
		ccTeamEmail:      cfg.CCTeamEmail,
		smtpConfig:       cfg.SMTP,
		password:         password,
	}, nil
}

func (s *Service) SendWelcomeEmail(reg *db.RegistrationRecord) error {
	if !s.smtpConfig.Enabled {
		return nil
	}

	data := map[string]any{
		"RegistrationID":   reg.RegistrationID,
		"Email":            reg.Email,
		"FirstName":        reg.FirstName,
		"LastName":         reg.LastName,
		"CompanyName":      reg.CompanyName,
		"Country":          reg.Country,
		"VatID":            reg.VatID,
		"Runtime":          s.runtime,
		"OnboardTeamEmail": s.onboardTeamEmail[0],
	}

	tmpl, err := template.New("email_welcome.html").Parse(emailWelcomeTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse email template: %w", err)
	}

	var emailBody bytes.Buffer
	if err := tmpl.ExecuteTemplate(&emailBody, "content", data); err != nil {
		return fmt.Errorf("failed to execute email template: %w", err)
	}

	// The email is sent from the same email as used to authenticate to the email server
	from := s.smtpConfig.Username

	// The email is sent to the customer and the list of corresponding internal emails
	messageVisibleTo := []string{reg.Email}
	messageVisibleCC := s.onboardTeamEmail
	messageInvisibleBCC := s.ccTeamEmail
	allRecipients := append(messageVisibleTo, messageVisibleCC...)
	allRecipients = append(allRecipients, messageInvisibleBCC...)

	subject := "Welcome to DOME Marketplace!"
	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"

	// Build the email message
	msg := []byte("From: " + from + "\n" +
		"To: " + strings.Join(messageVisibleTo, ", ") + "\n" +
		"Cc: " + strings.Join(messageVisibleCC, ", ") + "\n" +
		"Subject: " + subject + "\n" +
		mime + emailBody.String())

	return s.send(from, allRecipients, msg)
}

func (s *Service) SendVerificationCode(email string, code string) error {
	if !s.smtpConfig.Enabled {
		return nil
	}

	data := map[string]any{
		"Code":             code,
		"Runtime":          s.runtime,
		"OnboardTeamEmail": s.onboardTeamEmail[0],
	}

	tmpl, err := template.New("email_verification.html").Parse(emailVerificationTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse email template: %w", err)
	}

	var body bytes.Buffer
	if err := tmpl.ExecuteTemplate(&body, "content", data); err != nil {
		return fmt.Errorf("failed to execute email template: %w", err)
	}

	from := s.smtpConfig.Username
	to := []string{email}
	subject := "DOME Marketplace Verification Code"
	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	msg := []byte("From: " + from + "\n" +
		"To: " + strings.Join(to, ", ") + "\n" +
		"Subject: " + subject + "\n" +
		mime + body.String())

	return s.send(from, to, msg)
}

func (s *Service) SendIssuerError(reg *db.RegistrationRecord, payload string, errorMsg string) error {
	if !s.smtpConfig.Enabled {
		return nil
	}

	data := map[string]any{
		"FirstName":      reg.FirstName,
		"CompanyName":    reg.CompanyName,
		"RegistrationID": reg.RegistrationID,
		"Payload":        payload,
		"ErrorMsg":       errorMsg,
		"Runtime":        s.runtime,
	}

	tmpl, err := template.New("issuer_error.html").Parse(issuerErrorTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse email template: %w", err)
	}

	var body bytes.Buffer
	if err := tmpl.ExecuteTemplate(&body, "content", data); err != nil {
		return fmt.Errorf("failed to execute email template: %w", err)
	}

	from := s.smtpConfig.Username
	to := s.issuerTeamEmail
	subject := "DOME: Error in Credential Issuer during customer registration"
	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	msg := []byte("From: " + from + "\n" +
		"To: " + strings.Join(to, ", ") + "\n" +
		"Subject: " + subject + "\n" +
		mime + body.String())

	return s.send(from, to, msg)
}

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
