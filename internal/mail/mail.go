package mail

import (
	"crypto/tls"
	"embed"
	"fmt"
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
	views              *MailViews
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

	debug := false
	if runtime.IsDevelopment() {
		debug = true
	}

	views, err := NewMailViews(debug)
	if err != nil {
		return nil, errl.Errorf("failed to load mail views: %w", err)
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
		views:              views,
	}, nil
}

// getOnboardTeamEmail returns the primary onboarding team email address,
// or an empty string if the slice is empty or contains only whitespace.
func (s *Service) getOnboardTeamEmail() string {
	for _, email := range s.onboardTeamEmail {
		trimmed := strings.TrimSpace(email)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

// SendFirstPhaseEmail sends the welcome email to the registered user, after completing first phase of Onboarding.
// It copies the interested internal people.
func (s *Service) SendFirstPhaseEmail(reg *db.RegistrationRecord) error {

	subject := "Welcome to DOME Marketplace!"

	// Send mail to the user who onboarded
	to := []string{reg.Email}
	// Copy the interested internal people
	cc := s.onboardTeamEmail
	bcc := s.ccTeamEmail

	htmlContent, err := s.views.RenderEmailFirstPhaseView(s.runtime, s.onboardTeamEmail, reg)
	if err != nil {
		return errl.Errorf("failed to render first phase email: %w", err)
	}

	return s.SendEmailNew(to, cc, bcc, subject, htmlContent)
}

// SendSecondPhaseEmail sends an email to the user to indicate that they need to complete the second phase of the onboarding process.
// The second phase is for the user to complete the onboarding process in the Credential Issuer.
func (s *Service) SendSecondPhaseEmail(reg *db.RegistrationRecord) error {

	subject := "Welcome to DOME Marketplace!"
	// Send mail to the user who onboarded
	to := []string{reg.Email}
	// Copy the interested internal people
	cc := s.onboardTeamEmail
	bcc := s.ccTeamEmail

	htmlContent, err := s.views.RenderEmailSecondPhaseView(s.runtime, s.onboardTeamEmail, reg)
	if err != nil {
		return errl.Errorf("failed to render second phase email: %w", err)
	}

	return s.SendEmailNew(to, cc, bcc, subject, htmlContent)
}

// SendReminderEmail sends an email to the user after a period of time to remind them to complete the onboarding process.
func (s *Service) SendReminderEmail(reg *db.RegistrationRecord) error {

	subject := "Reminder: Please complete your registration to DOME Marketplace!"
	// Send mail to the user who onboarded
	to := []string{reg.Email}
	// Copy the interested internal people
	cc := s.onboardTeamEmail
	bcc := s.ccTeamEmail

	htmlContent, err := s.views.RenderReminderEmailView(s.runtime, s.onboardTeamEmail, reg)
	if err != nil {
		return errl.Errorf("failed to render reminder email: %w", err)
	}

	return s.SendEmailNew(to, cc, bcc, subject, htmlContent)
}

// SendVerificationCodeEmail sends an email to the user with a verification code, to check that the user controls the email.
func (s *Service) SendVerificationCodeEmail(email string, code string) error {

	subject := "DOME Marketplace Verification Code"
	// Send mail to the user
	to := []string{email}
	// No cc and bcc
	cc := []string{}
	bcc := []string{}

	htmlContent, err := s.views.RenderVerificationCodeEmailView(s.runtime, s.onboardTeamEmail, code)
	if err != nil {
		return errl.Errorf("failed to render verification code email: %w", err)
	}

	return s.SendEmailNew(to, cc, bcc, subject, htmlContent)
}

// SendIssuerErrorEmail sends an email to the team in charge of the Issuer so they can handle the techinical error.
// The payload is the payload sent to the issuer.
func (s *Service) SendIssuerErrorEmail(reg *db.RegistrationRecord, payload string, errorMsg string) error {

	subject := "DOME: Error in Credential Issuer during customer registration"
	// Send mail to the team in charge of the Issuer so they can handle the techinical error
	to := s.issuerTeamEmail
	// No cc and bcc
	cc := []string{}
	bcc := []string{}

	htmlContent, err := s.views.RenderIssuerErrorView(s.runtime, errorMsg, payload, reg)
	if err != nil {
		return errl.Errorf("failed to render issuer error email: %w", err)
	}

	return s.SendEmailNew(to, cc, bcc, subject, htmlContent)
}

// SendCheckIssuerErrorEmail sends an email to the team in charge of the Issuer if an error was received during
// the self-checks that the serfer performs at startup and scheduled times.
func (s *Service) SendCheckIssuerErrorEmail(payload string, errorMsg string) error {

	subject := "DOME: Error in Credential Issuer during customer self-registration test"
	htmlContent, err := s.views.RenderCheckIssuerErrorView(s.runtime, errorMsg, payload)
	if err != nil {
		return errl.Errorf("failed to render check issuer error email: %w", err)
	}

	return s.SendEmailNew(s.issuerTeamEmail, []string{s.testRecipientEmail}, []string{}, subject, htmlContent)
}

// SendTestEmail sends an email to the test recipient to verify that the mail service is working.
func (s *Service) SendTestEmail() error {

	htmlContent, err := s.views.RenderTestEmailView(s.runtime, s.onboardTeamEmail, "123456")
	if err != nil {
		return errl.Errorf("failed to render test email: %w", err)
	}

	return s.SendEmailNew([]string{s.testRecipientEmail}, []string{}, []string{}, "DOME Marketplace Test Email", htmlContent)
}

// SendEmail compiles and sends the email using a specified template.
func (s *Service) SendEmailNew(to, cc, bcc []string, subject string, htmlContent string) error {
	if !s.smtpConfig.Enabled {
		return nil
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
		mimeType + htmlContent)

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
