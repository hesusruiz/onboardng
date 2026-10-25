package mail

import (
	"bufio"
	"fmt"
	"net"
	"net/textproto"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hesusruiz/onboardng/internal/configuration"
	"github.com/hesusruiz/onboardng/internal/db"
)

// mockSMTPServer is a very simple SMTP server for testing
type mockSMTPServer struct {
	addr     string
	listener net.Listener
	quit     chan struct{}
	received chan string
}

func newMockSMTPServer(addr string) (*mockSMTPServer, error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	return &mockSMTPServer{
		addr:     l.Addr().String(),
		listener: l,
		quit:     make(chan struct{}),
		received: make(chan string, 1),
	}, nil
}

func (s *mockSMTPServer) start() {
	go func() {
		for {
			conn, err := s.listener.Accept()
			if err != nil {
				select {
				case <-s.quit:
					return
				default:
					continue
				}
			}
			go s.handle(conn)
		}
	}()
}

func (s *mockSMTPServer) stop() {
	close(s.quit)
	if s.listener != nil {
		s.listener.Close()
	}
}

func (s *mockSMTPServer) handle(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	tp := textproto.NewReader(reader)

	conn.Write([]byte("220 Welcome to Mock SMTP\r\n"))

	for {
		line, err := tp.ReadLine()
		if err != nil {
			return
		}

		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		cmd := strings.ToUpper(fields[0])
		switch cmd {
		case "HELO", "EHLO":
			conn.Write([]byte("250-Hello\r\n250-AUTH PLAIN\r\n250 OK\r\n"))
		case "AUTH":
			conn.Write([]byte("235 Authentication succeeded\r\n"))
		case "MAIL":
			conn.Write([]byte("250 OK\r\n"))
		case "RCPT":
			conn.Write([]byte("250 OK\r\n"))
		case "DATA":
			conn.Write([]byte("354 Start mail input; end with <CRLF>.<CRLF>\r\n"))
			var message strings.Builder
			for {
				line, err := tp.ReadLine()
				if err != nil || line == "." {
					break
				}
				message.WriteString(line)
				message.WriteString("\n")
			}
			s.received <- message.String()
			conn.Write([]byte("250 OK\r\n"))
		case "QUIT":
			conn.Write([]byte("221 Bye\r\n"))
			return
		default:
			conn.Write([]byte("500 Unknown command\r\n"))
		}
	}
}

func TestSendFirstPhaseEmail(t *testing.T) {

	// Start mock SMTP server
	mockServer, err := newMockSMTPServer("127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start mock SMTP server: %v", err)
	}
	mockServer.start()
	defer mockServer.stop()

	// Get server host and port
	host, portStr, _ := net.SplitHostPort(mockServer.addr)
	var port int
	fmt.Sscanf(portStr, "%d", &port)

	// Create temporary password file
	tmpFile, err := os.CreateTemp("", "smtppassword")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString("testpassword")
	tmpFile.Close()

	// SMTP config
	cfg := configuration.SMTPConfig{
		Enabled:      true,
		Host:         host,
		Port:         port,
		TLS:          false, // Use normal SMTP for simple test
		Username:     "test@example.com",
		PasswordFile: tmpFile.Name(),
	}

	mailCfg := configuration.MailConfig{
		OnboardTeamEmail: []string{"onboard@example.com"},
		IssuerTeamEmail:  []string{"issuer@example.com"},
		CCTeamEmail:      []string{"admin1@example.com", "admin2@example.com"},
		SMTP:             cfg,
	}

	// Initialize Mail Service
	mailService, err := NewMailService(configuration.Development, mailCfg)
	if err != nil {
		t.Fatalf("failed to create mail service: %v", err)
	}

	// Mock registration data
	reg := &db.RegistrationRecord{
		FirstName:      "John",
		LastName:       "Doe",
		CompanyName:    "Acme Corp",
		Country:        "ES",
		VatID:          "ESA12345678",
		RegistrationID: "20260222-12345678",
		Email:          "recipient@example.com",
	}

	// Send email
	err = mailService.SendFirstPhaseEmail(reg)
	if err != nil {
		t.Fatalf("SendFirstPhaseEmail failed: %v", err)
	}

	// Verify received email
	select {
	case msg := <-mockServer.received:
		if !strings.Contains(msg, "Hello, John") {
			t.Errorf("expected email to contain 'Hello, John', got: %s", msg)
		}
		if !strings.Contains(msg, "Doe!") {
			t.Errorf("expected email to contain 'Doe!', got: %s", msg)
		}
		if !strings.Contains(msg, "onboard@example.com") {
			t.Errorf("expected email to contain 'onboard@example.com', got: %s", msg)
		}
		if !strings.Contains(msg, "Acme Corp") {
			t.Errorf("expected email to contain 'Acme Corp', got: %s", msg)
		}
		if !strings.Contains(msg, "20260222-12345678") {
			t.Errorf("expected email to contain registration ID, got: %s", msg)
		}
		if !strings.Contains(msg, "ES") {
			t.Errorf("expected email to contain country 'ES', got: %s", msg)
		}
		if !strings.Contains(msg, "ESA12345678") {
			t.Errorf("expected email to contain VAT ID 'ESA12345678', got: %s", msg)
		}
		if !strings.Contains(msg, "Test Environment:") || !strings.Contains(msg, "SBX") {
			t.Errorf("expected email to contain Development environment banner with SBX, got: %s", msg)
		}
	case <-time.After(2 * time.Second):
		t.Errorf("timeout waiting for email")
	}
}



func TestSendEmailNew(t *testing.T) {
	// Start mock SMTP server
	mockServer, err := newMockSMTPServer("127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start mock SMTP server: %v", err)
	}
	mockServer.start()
	defer mockServer.stop()

	host, portStr, _ := net.SplitHostPort(mockServer.addr)
	var port int
	fmt.Sscanf(portStr, "%d", &port)

	tmpFile, err := os.CreateTemp("", "smtppassword")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString("testpassword")
	tmpFile.Close()

	mailCfg := configuration.MailConfig{
		SMTP: configuration.SMTPConfig{
			Enabled:      true,
			Host:         host,
			Port:         port,
			TLS:          false,
			Username:     "test@example.com",
			PasswordFile: tmpFile.Name(),
		},
	}

	mailService, err := NewMailService(configuration.Development, mailCfg)
	if err != nil {
		t.Fatalf("failed to create mail service: %v", err)
	}

	// Test SendEmailNew with To, CC, and BCC
	to := []string{"recipient1@example.com", "recipient2@example.com"}
	cc := []string{"cc@example.com"}
	bcc := []string{"bcc@example.com"}
	subject := "Test SendEmailNew"
	htmlContent := "<h1>Hello from SendEmailNew</h1>"

	err = mailService.SendEmailNew(to, cc, bcc, subject, htmlContent)
	if err != nil {
		t.Fatalf("SendEmailNew failed: %v", err)
	}

	select {
	case msg := <-mockServer.received:
		if !strings.Contains(msg, "From: test@example.com") {
			t.Errorf("expected From header, got: %s", msg)
		}
		if !strings.Contains(msg, "To: recipient1@example.com, recipient2@example.com") {
			t.Errorf("expected To header, got: %s", msg)
		}
		if !strings.Contains(msg, "Cc: cc@example.com") {
			t.Errorf("expected Cc header, got: %s", msg)
		}
		// BCC should NOT be in the headers
		if strings.Contains(msg, "bcc@example.com") {
			t.Errorf("bcc recipient should not be visible in headers, got: %s", msg)
		}
		if !strings.Contains(msg, "Subject: Test SendEmailNew") {
			t.Errorf("expected Subject header, got: %s", msg)
		}
		if !strings.Contains(msg, "<h1>Hello from SendEmailNew</h1>") {
			t.Errorf("expected HTML content in body, got: %s", msg)
		}
	case <-time.After(2 * time.Second):
		t.Errorf("timeout waiting for email")
	}

	// Test SendEmailNew when SMTP is disabled
	disabledCfg := configuration.MailConfig{
		SMTP: configuration.SMTPConfig{
			Enabled: false,
		},
	}
	disabledService, err := NewMailService(configuration.Development, disabledCfg)
	if err != nil {
		t.Fatalf("failed to create disabled mail service: %v", err)
	}
	if err := disabledService.SendEmailNew([]string{"recipient@example.com"}, nil, nil, "Subj", "<p>hi</p>"); err != nil {
		t.Errorf("expected nil error when SMTP is disabled, got: %v", err)
	}
}



func TestSendPhaseEmailsAndReminders(t *testing.T) {
	mockServer, err := newMockSMTPServer("127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start mock SMTP server: %v", err)
	}
	mockServer.start()
	defer mockServer.stop()

	host, portStr, _ := net.SplitHostPort(mockServer.addr)
	var port int
	fmt.Sscanf(portStr, "%d", &port)

	tmpFile, err := os.CreateTemp("", "smtppassword")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString("testpassword")
	tmpFile.Close()

	mailCfg := configuration.MailConfig{
		OnboardTeamEmail: []string{"onboard@example.com"},
		IssuerTeamEmail:  []string{"issuer@example.com"},
		SMTP: configuration.SMTPConfig{
			Enabled:      true,
			Host:         host,
			Port:         port,
			TLS:          false,
			Username:     "test@example.com",
			PasswordFile: tmpFile.Name(),
		},
	}

	mailService, err := NewMailService(configuration.Development, mailCfg)
	if err != nil {
		t.Fatalf("failed to create mail service: %v", err)
	}

	reg := &db.RegistrationRecord{
		RegistrationID: "REG-2026-9999",
		Email:          "user@example.com",
		FirstName:      "Alice",
		LastName:       "Smith",
		CompanyName:    "Globex Corp",
		Country:        "ES",
		VatID:          "ESA12345678",
	}

	// 1. SendSecondPhaseEmail
	if err := mailService.SendSecondPhaseEmail(reg); err != nil {
		t.Fatalf("SendSecondPhaseEmail failed: %v", err)
	}
	select {
	case msg := <-mockServer.received:
		if !strings.Contains(msg, "Hello, Alice Smith!") {
			t.Errorf("expected email to contain 'Hello, Alice Smith!', got: %s", msg)
		}
		if !strings.Contains(msg, "REG-2026-9999") {
			t.Errorf("expected email to contain registration ID 'REG-2026-9999', got: %s", msg)
		}
		if !strings.Contains(msg, "user@example.com") {
			t.Errorf("expected email to contain 'user@example.com', got: %s", msg)
		}
		if !strings.Contains(msg, "mailto:onboard@example.com") {
			t.Errorf("expected mailto:onboard@example.com, got: %s", msg)
		}
		if strings.Contains(msg, "[onboard@example.com]") {
			t.Errorf("expected no brackets in onboard team email, got: %s", msg)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for second phase email")
	}

	// 2. SendReminderEmail
	if err := mailService.SendReminderEmail(reg); err != nil {
		t.Fatalf("SendReminderEmail failed: %v", err)
	}
	select {
	case msg := <-mockServer.received:
		if !strings.Contains(msg, "Hello, Alice Smith!") {
			t.Errorf("expected email to contain 'Hello, Alice Smith!', got: %s", msg)
		}
		if !strings.Contains(msg, "REG-2026-9999") {
			t.Errorf("expected email to contain registration ID, got: %s", msg)
		}
		if !strings.Contains(msg, "user@example.com") {
			t.Errorf("expected email to contain 'user@example.com', got: %s", msg)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for reminder email")
	}

	// 3. SendIssuerErrorEmail
	if err := mailService.SendIssuerErrorEmail(reg, `{"test":"payload"}`, "issuer connection failed"); err != nil {
		t.Fatalf("SendIssuerErrorEmail failed: %v", err)
	}
	select {
	case msg := <-mockServer.received:
		if !strings.Contains(msg, "REG-2026-9999") {
			t.Errorf("expected email to contain registration ID, got: %s", msg)
		}
		if !strings.Contains(msg, "payload") {
			t.Errorf("expected email to contain payload, got: %s", msg)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for issuer error email")
	}
}

func TestOnboardTeamEmailEmptyHandling(t *testing.T) {
	mockServer, err := newMockSMTPServer("127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start mock SMTP server: %v", err)
	}
	mockServer.start()
	defer mockServer.stop()

	host, portStr, _ := net.SplitHostPort(mockServer.addr)
	var port int
	fmt.Sscanf(portStr, "%d", &port)

	tmpFile, err := os.CreateTemp("", "smtppassword")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString("testpassword")
	tmpFile.Close()

	// Empty slice for OnboardTeamEmail
	mailCfg := configuration.MailConfig{
		OnboardTeamEmail: []string{},
		SMTP: configuration.SMTPConfig{
			Enabled:      true,
			Host:         host,
			Port:         port,
			TLS:          false,
			Username:     "test@example.com",
			PasswordFile: tmpFile.Name(),
		},
	}

	mailService, err := NewMailService(configuration.Development, mailCfg)
	if err != nil {
		t.Fatalf("failed to create mail service: %v", err)
	}

	reg := &db.RegistrationRecord{
		RegistrationID: "REG-EMPTY-0001",
		Email:          "emptytest@example.com",
		FirstName:      "Bob",
		LastName:       "Builder",
	}

	// SendFirstPhaseEmail with empty OnboardTeamEmail
	if err := mailService.SendFirstPhaseEmail(reg); err != nil {
		t.Fatalf("SendFirstPhaseEmail failed: %v", err)
	}

	select {
	case msg := <-mockServer.received:
		if strings.Contains(msg, "mailto:") {
			t.Errorf("expected no mailto link when OnboardTeamEmail is empty, got: %s", msg)
		}
		if strings.Contains(msg, "assistance, please contact our support team") {
			t.Errorf("expected support section to be omitted when OnboardTeamEmail is empty, got: %s", msg)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for email")
	}

	// Also verify SendVerificationCodeEmail omits mailto link when OnboardTeamEmail is empty
	if err := mailService.SendVerificationCodeEmail("emptytest@example.com", "111222"); err != nil {
		t.Fatalf("SendVerificationCodeEmail failed: %v", err)
	}
	select {
	case msg := <-mockServer.received:
		if strings.Contains(msg, "mailto:") {
			t.Errorf("expected no mailto link in verification email when OnboardTeamEmail is empty, got: %s", msg)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for verification email")
	}

	// Also verify SendSecondPhaseEmail omits mailto link when OnboardTeamEmail is empty
	if err := mailService.SendSecondPhaseEmail(reg); err != nil {
		t.Fatalf("SendSecondPhaseEmail failed: %v", err)
	}
	select {
	case msg := <-mockServer.received:
		if strings.Contains(msg, "mailto:") {
			t.Errorf("expected no mailto link in second phase email when OnboardTeamEmail is empty, got: %s", msg)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for second phase email")
	}
}

// setupTestMailServerAndService initializes a mock SMTP server and a configured mail Service for testing.
func setupTestMailServerAndService(t *testing.T, runtime configuration.RuntimeEnv, mailCfg configuration.MailConfig) (*Service, *mockSMTPServer, func()) {
	t.Helper()
	mockServer, err := newMockSMTPServer("127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start mock SMTP server: %v", err)
	}
	mockServer.start()

	host, portStr, _ := net.SplitHostPort(mockServer.addr)
	var port int
	fmt.Sscanf(portStr, "%d", &port)

	tmpFile, err := os.CreateTemp("", "smtppassword")
	if err != nil {
		mockServer.stop()
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpFile.WriteString("testpassword")
	tmpFile.Close()

	mailCfg.SMTP.Enabled = true
	mailCfg.SMTP.Host = host
	mailCfg.SMTP.Port = port
	mailCfg.SMTP.TLS = false
	mailCfg.SMTP.Username = "test@example.com"
	mailCfg.SMTP.PasswordFile = tmpFile.Name()

	mailService, err := NewMailService(runtime, mailCfg)
	if err != nil {
		os.Remove(tmpFile.Name())
		mockServer.stop()
		t.Fatalf("failed to create mail service: %v", err)
	}

	cleanup := func() {
		os.Remove(tmpFile.Name())
		mockServer.stop()
	}

	return mailService, mockServer, cleanup
}

func receiveEmail(t *testing.T, mockServer *mockSMTPServer) string {
	t.Helper()
	select {
	case msg := <-mockServer.received:
		return msg
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for email")
		return ""
	}
}

func TestSendVerificationCodeEmail(t *testing.T) {
	mailCfg := configuration.MailConfig{
		OnboardTeamEmail: []string{"onboard@example.com"},
	}
	service, mockServer, cleanup := setupTestMailServerAndService(t, configuration.Development, mailCfg)
	defer cleanup()

	err := service.SendVerificationCodeEmail("user@example.com", "987654")
	if err != nil {
		t.Fatalf("SendVerificationCodeEmail failed: %v", err)
	}

	msg := receiveEmail(t, mockServer)
	if !strings.Contains(msg, "To: user@example.com") {
		t.Errorf("expected To: user@example.com, got: %s", msg)
	}
	if !strings.Contains(msg, "Subject: DOME Marketplace Verification Code") {
		t.Errorf("expected subject 'DOME Marketplace Verification Code', got: %s", msg)
	}
	if !strings.Contains(msg, "987654") {
		t.Errorf("expected verification code '987654' in body, got: %s", msg)
	}
	if !strings.Contains(msg, "Identity Verification") {
		t.Errorf("expected 'Identity Verification' header in body, got: %s", msg)
	}
	if !strings.Contains(msg, "This code will expire in 15 minutes") {
		t.Errorf("expected expiration notice in body, got: %s", msg)
	}
	if !strings.Contains(msg, "mailto:onboard@example.com") {
		t.Errorf("expected support mailto link, got: %s", msg)
	}
}

func TestSendDiagnosticEmails(t *testing.T) {
	mailCfg := configuration.MailConfig{
		OnboardTeamEmail:   []string{"onboard@example.com"},
		IssuerTeamEmail:    []string{"issuer@example.com"},
		TestRecipientEmail: "testrecip@example.com",
	}
	service, mockServer, cleanup := setupTestMailServerAndService(t, configuration.Development, mailCfg)
	defer cleanup()

	// 1. SendCheckIssuerErrorEmail (check_issuer_error.html.j2)
	err := service.SendCheckIssuerErrorEmail(`{"orderId":"12345"}`, "issuer connection timeout")
	if err != nil {
		t.Fatalf("SendCheckIssuerErrorEmail failed: %v", err)
	}
	msg := receiveEmail(t, mockServer)
	if !strings.Contains(msg, "To: issuer@example.com") {
		t.Errorf("expected To: issuer@example.com, got: %s", msg)
	}
	if !strings.Contains(msg, "Cc: testrecip@example.com") {
		t.Errorf("expected Cc: testrecip@example.com, got: %s", msg)
	}
	if !strings.Contains(msg, "Subject: DOME: Error in Credential Issuer during customer self-registration test") {
		t.Errorf("expected check issuer error subject, got: %s", msg)
	}
	if !strings.Contains(msg, "Credential Issuance Failure") {
		t.Errorf("expected failure header in body, got: %s", msg)
	}
	if !strings.Contains(msg, "issuer connection timeout") {
		t.Errorf("expected error message in body, got: %s", msg)
	}
	if !strings.Contains(msg, "orderId") || !strings.Contains(msg, "12345") {
		t.Errorf("expected payload in body, got: %s", msg)
	}

	// 2. SendTestEmail (email_test.html.j2)
	err = service.SendTestEmail()
	if err != nil {
		t.Fatalf("SendTestEmail failed: %v", err)
	}
	testMsg := receiveEmail(t, mockServer)
	if !strings.Contains(testMsg, "To: testrecip@example.com") {
		t.Errorf("expected To: testrecip@example.com, got: %s", testMsg)
	}
	if !strings.Contains(testMsg, "Subject: DOME Marketplace Test Email") {
		t.Errorf("expected test email subject, got: %s", testMsg)
	}
	if !strings.Contains(testMsg, "This is a test email") || !strings.Contains(testMsg, "can ignore it") {
		t.Errorf("expected test warning banner in body, got: %s", testMsg)
	}
	if !strings.Contains(testMsg, "123456") {
		t.Errorf("expected verification code 123456 in test email, got: %s", testMsg)
	}
	if !strings.Contains(testMsg, "mailto:onboard@example.com") {
		t.Errorf("expected support mailto link in test email, got: %s", testMsg)
	}
}

func TestSendSecondPhaseEmail_FullDetails(t *testing.T) {
	mailCfg := configuration.MailConfig{
		OnboardTeamEmail: []string{"onboard@example.com"},
		CCTeamEmail:      []string{"ccadmin@example.com"},
	}
	service, mockServer, cleanup := setupTestMailServerAndService(t, configuration.Development, mailCfg)
	defer cleanup()

	reg := &db.RegistrationRecord{
		RegistrationID:   "REG-2026-PHASE2",
		Email:            "enterprise@example.com",
		FirstName:        "Carlos",
		LastName:         "Santana",
		CompanyName:      "SuperCorp S.L.",
		Country:          "ES",
		VatID:            "ESB12345678",
		LRFirstName:      "Laura",
		LRLastName:       "Rodriguez",
		LREmail:          "laura.lr@example.com",
		LRCountry:        "ES",
		LRIdCard:         "ID-LR-9988",
		LEARFirstName:    "Leo",
		LEARLastName:     "Admin",
		LEAREmail:        "leo.lear@example.com",
		LEARCountry:      "ES",
		LEARAddress:      "Av. Diagonal 456",
		LEARIdCard:       "ID-LEAR-1122",
		LEARMobileNumber: "+34699887766",
	}

	err := service.SendSecondPhaseEmail(reg)
	if err != nil {
		t.Fatalf("SendSecondPhaseEmail failed: %v", err)
	}

	msg := receiveEmail(t, mockServer)
	if !strings.Contains(msg, "Hello, Carlos Santana!") {
		t.Errorf("expected 'Hello, Carlos Santana!', got: %s", msg)
	}
	if !strings.Contains(msg, "REG-2026-PHASE2") {
		t.Errorf("expected registration ID, got: %s", msg)
	}
	if !strings.Contains(msg, "SuperCorp S.L.") {
		t.Errorf("expected company name, got: %s", msg)
	}
	if !strings.Contains(msg, "ESB12345678") {
		t.Errorf("expected VAT ID, got: %s", msg)
	}
	// Check Legal Representative fields
	if !strings.Contains(msg, "Laura") || !strings.Contains(msg, "Rodriguez") || !strings.Contains(msg, "laura.lr@example.com") || !strings.Contains(msg, "ID-LR-9988") {
		t.Errorf("expected Legal Representative details in second phase email, got: %s", msg)
	}
	// Check Account Administrator (LEAR) fields
	if !strings.Contains(msg, "Leo") || !strings.Contains(msg, "Admin") || !strings.Contains(msg, "leo.lear@example.com") || !strings.Contains(msg, "Av. Diagonal 456") || !strings.Contains(msg, "ID-LEAR-1122") || !strings.Contains(msg, "+34699887766") {
		t.Errorf("expected Account Administrator details in second phase email, got: %s", msg)
	}
}

func TestReminderEmail_LEARCompletedBranching(t *testing.T) {
	mailCfg := configuration.MailConfig{
		OnboardTeamEmail: []string{"onboard@example.com"},
	}
	service, mockServer, cleanup := setupTestMailServerAndService(t, configuration.Development, mailCfg)
	defer cleanup()

	baseReg := db.RegistrationRecord{
		RegistrationID: "REG-2026-REM",
		Email:          "user@example.com",
		FirstName:      "Elena",
		LastName:       "Naranjo",
		CompanyName:    "TestCo",
		Country:        "ES",
		VatID:          "ESA99999999",
		LRFirstName:    "Laura",
		LRLastName:     "Representative",
		LEARFirstName:  "Leo",
		LEARLastName:   "Administrator",
	}

	// Case 1: LEARCompleted = false -> LR and LEAR tables should be omitted
	regIncomplete := baseReg
	regIncomplete.LEARCompleted = false
	if err := service.SendReminderEmail(&regIncomplete); err != nil {
		t.Fatalf("SendReminderEmail failed: %v", err)
	}
	msgIncomplete := receiveEmail(t, mockServer)
	if !strings.Contains(msgIncomplete, "Hello, Elena Naranjo!") {
		t.Errorf("expected greeting, got: %s", msgIncomplete)
	}
	if !strings.Contains(msgIncomplete, "Company Information") {
		t.Errorf("expected company info, got: %s", msgIncomplete)
	}
	if strings.Contains(msgIncomplete, "Legal Representative") {
		t.Errorf("expected Legal Representative section to be omitted when LEARCompleted is false, got: %s", msgIncomplete)
	}
	if strings.Contains(msgIncomplete, "Legal Entity Authorized") {
		t.Errorf("expected LEAR section to be omitted when LEARCompleted is false, got: %s", msgIncomplete)
	}

	// Case 2: LEARCompleted = true -> LR and LEAR tables should be rendered
	regComplete := baseReg
	regComplete.LEARCompleted = true
	if err := service.SendReminderEmail(&regComplete); err != nil {
		t.Fatalf("SendReminderEmail failed: %v", err)
	}
	msgComplete := receiveEmail(t, mockServer)
	if !strings.Contains(msgComplete, "Legal Representative") {
		t.Errorf("expected Legal Representative table when LEARCompleted is true, got: %s", msgComplete)
	}
	if !strings.Contains(msgComplete, "Laura") {
		t.Errorf("expected LRFirstName in reminder email, got: %s", msgComplete)
	}
	if !strings.Contains(msgComplete, "Legal Entity Authorized") {
		t.Errorf("expected LEAR table when LEARCompleted is true, got: %s", msgComplete)
	}
	if !strings.Contains(msgComplete, "Leo") {
		t.Errorf("expected LEARFirstName in reminder email, got: %s", msgComplete)
	}
}

func TestTemplateEnvironmentBanner(t *testing.T) {
	reg := &db.RegistrationRecord{
		RegistrationID: "REG-2026-ENV",
		Email:          "envtest@example.com",
		FirstName:      "Env",
		LastName:       "Tester",
		CompanyName:    "Env Corp",
		Country:        "ES",
		VatID:          "ESA11112222",
	}

	// 1. Development: banner should be displayed with runtime string "SBX"
	{
		serviceDev, mockDev, cleanupDev := setupTestMailServerAndService(t, configuration.Development, configuration.MailConfig{})
		defer cleanupDev()

		if err := serviceDev.SendFirstPhaseEmail(reg); err != nil {
			t.Fatalf("SendFirstPhaseEmail failed in Development: %v", err)
		}
		msgDev := receiveEmail(t, mockDev)
		if !strings.Contains(msgDev, "Test Environment") {
			t.Errorf("expected Test Environment banner in Development, got: %s", msgDev)
		}
		if !strings.Contains(msgDev, "SBX") {
			t.Errorf("expected runtime string 'SBX' in Development banner, got: %s", msgDev)
		}
	}

	// 2. Production: banner should be hidden
	{
		serviceProd, mockProd, cleanupProd := setupTestMailServerAndService(t, configuration.Production, configuration.MailConfig{})
		defer cleanupProd()

		if err := serviceProd.SendFirstPhaseEmail(reg); err != nil {
			t.Fatalf("SendFirstPhaseEmail failed in Production: %v", err)
		}
		msgProd := receiveEmail(t, mockProd)
		if strings.Contains(msgProd, "⚠️") || strings.Contains(msgProd, "In production, this message is hidden") {
			t.Errorf("expected Test Environment banner to be hidden in Production, but found in output: %s", msgProd)
		}
	}
}
