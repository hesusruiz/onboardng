package mail

import (
	"bufio"
	"fmt"
	"net"
	"net/textproto"
	"os"
	"path/filepath"
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

func TestSendWelcomeEmail(t *testing.T) {

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
		CompanyName:    "Acme Corp",
		RegistrationID: "20260222-12345678",
		Email:          "recipient@example.com",
	}

	// Send email
	err = mailService.SendWelcomeEmail(reg)
	if err != nil {
		t.Fatalf("SendWelcomeEmail failed: %v", err)
	}

	// Verify received email
	select {
	case msg := <-mockServer.received:
		if !strings.Contains(msg, "Hello, John") {
			t.Errorf("expected email to contain 'Hello, John', got: %s", msg)
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
	case <-time.After(2 * time.Second):
		t.Errorf("timeout waiting for email")
	}
}

func TestSendEmailWithCustomTemplate(t *testing.T) {
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

	// Create temporary template directory
	tempDir, err := os.MkdirTemp("", "templates")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Write a custom template file
	customTmpl := `<p>Custom Template Body: {{.Val}}</p>`
	err = os.WriteFile(filepath.Join(tempDir, "custom.html"), []byte(customTmpl), 0644)
	if err != nil {
		t.Fatalf("failed to write temp template file: %v", err)
	}

	// Create temporary password file
	tmpFile, err := os.CreateTemp("", "smtppassword")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString("testpassword")
	tmpFile.Close()

	mailCfg := configuration.MailConfig{
		OnboardTeamEmail: []string{"onboard@example.com"},
		TemplateDir:      tempDir,
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

	// 1. Verify custom template is loaded from disk
	data := map[string]any{"Val": "Hello dynamic templates!"}
	err = mailService.SendEmail([]string{"recipient@example.com"}, nil, nil, "Subject", "custom.html", data)
	if err != nil {
		t.Fatalf("failed to send custom email: %v", err)
	}

	select {
	case msg := <-mockServer.received:
		if !strings.Contains(msg, "Custom Template Body: Hello dynamic templates!") {
			t.Errorf("expected email to contain custom body, got: %s", msg)
		}
	case <-time.After(2 * time.Second):
		t.Errorf("timeout waiting for email")
	}

	// 2. Verify fallback to embedded template (e.g. email_test.html)
	testData := struct {
		Email            string
		Code             string
		Runtime          configuration.RuntimeEnv
		OnboardTeamEmail string
	}{
		Email:            "recipient@example.com",
		Code:             "987654",
		Runtime:          configuration.Development,
		OnboardTeamEmail: "support@example.com",
	}
	err = mailService.SendEmail([]string{"recipient@example.com"}, nil, nil, "Test Fallback", "email_test.html", testData)
	if err != nil {
		t.Fatalf("failed to send fallback email: %v", err)
	}

	select {
	case msg := <-mockServer.received:
		if !strings.Contains(msg, "987654") {
			t.Errorf("expected email to contain embedded code 987654, got: %s", msg)
		}
	case <-time.After(2 * time.Second):
		t.Errorf("timeout waiting for email")
	}

	// 3. Verify directory traversal protection (filepath.Base sanitization)
	err = mailService.SendEmail([]string{"recipient@example.com"}, nil, nil, "Traversal Test", "../../../secret.txt", nil)
	if err == nil {
		t.Error("expected error when trying to request path with directory traversal, but got none")
	} else if !strings.Contains(err.Error(), "secret.txt") {
		t.Errorf("expected error message to refer to the sanitized base name 'secret.txt', got: %v", err)
	}
}
