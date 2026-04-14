package server

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"sync"

	"golang.org/x/time/rate"

	"github.com/hesusruiz/onboardng/credissuance"
	"github.com/hesusruiz/onboardng/internal/configuration"
	"github.com/hesusruiz/onboardng/internal/db"
)

// DBServiceProvider enables easy testing or replacing of the database implementation
type DBServiceProvider interface {
	SaveRegistration(reg *db.RegistrationRecord) error
	UpdateRegistrationStatus(reg *db.RegistrationRecord) error
	UpdateApproval(registrationID string, approved int) error
	UpdateRepresentativesByVatID(vatID string, rep *db.RegistrationRecord) error
	SaveRegistrationLog(logEntry *db.RegistrationLog) error
	GetRegistrationByVatID(vatID string) (*db.RegistrationRecord, error)
	GetRegistrationByEmail(email string) (*db.RegistrationRecord, error)
	GetRegistrationByEmailOrVatID(email string, vatID string) (*db.RegistrationRecord, error)
	GetRegistrations(limit, offset int) ([]db.RegistrationRecord, error)
	GetRegistrationLogs(vatID string, limit, offset int) ([]db.RegistrationLog, error)
	GetRegistrationFiles(vatID string, limit, offset int) ([]db.RegistrationFile, error)
	GetRegistrationFile(fileID string) (*db.RegistrationFile, error)
	GetRegistrationByID(registrationID string) (*db.RegistrationRecord, error)
}

// MailServiceProvider enables easy testing or replacing of the mail implementation
type MailServiceProvider interface {
	SendVerificationCode(email string, code string) error
	SendWelcomeEmail(reg *db.RegistrationRecord) error
	SendIssuerError(reg *db.RegistrationRecord, payload string, errorMsg string) error
}

// IssuanceServiceProvider enables easy testing or replacing of the issuance implementation
type IssuanceServiceProvider interface {
	GetAccessToken(ctx context.Context) (string, error)
	TMFGetOrganizationByELSI(ctx context.Context, accessToken string, elsi string) ([]credissuance.Organization, error)
	TMFDeleteOrganization(ctx context.Context, accessToken string, id string) error
	LEARIssuanceRequest(ctx context.Context, accessToken string, learCredData *credissuance.LEARIssuanceRequestBody) ([]byte, error)
	TMFCreateOrganization(ctx context.Context, accessToken string, org *credissuance.Organization_Create) (*credissuance.Organization, error)
	TMFUpdateOrganization(ctx context.Context, accessToken string, id string, org *credissuance.Organization_Update) (*credissuance.Organization, error)
}

type Server struct {
	Runtime           configuration.RuntimeEnv
	DB                DBServiceProvider
	Issuer            IssuanceServiceProvider
	Mail              MailServiceProvider
	EmailRateLimiter  map[string]*RateLimitEntry
	VerificationCodes map[string]*VerificationCodeEntry
	RateLimiterMu     sync.RWMutex
	CodesMu           sync.RWMutex
	IPLimiters        map[string]*rate.Limiter
	IPLimitersMu      sync.Mutex
	Handler           http.Handler
	AdminUser         string
	AdminPassword     string
	Features          configuration.Features
}

func NewServer(runtime configuration.RuntimeEnv,
	dbService DBServiceProvider,
	issuer IssuanceServiceProvider,
	mailService MailServiceProvider,
	staticFilesDir string,
	adminUser string,
	adminPassword string,
	features configuration.Features) *Server {

	s := &Server{
		Runtime:           runtime,
		DB:                dbService,
		Issuer:            issuer,
		Mail:              mailService,
		EmailRateLimiter:  make(map[string]*RateLimitEntry),
		VerificationCodes: make(map[string]*VerificationCodeEntry),
		IPLimiters:        make(map[string]*rate.Limiter),
		AdminUser:         adminUser,
		AdminPassword:     adminPassword,
		Features:          features,
	}

	mux := http.NewServeMux()

	// Static file serving
	// fileServer := http.FileServer(http.Dir(staticFilesDir))
	fileServer := http.FileServer(http.Dir("dist/browser"))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Only intercept exactly "/"
		if r.URL.Path == "/" {
			page := r.URL.Query().Get("page")
			switch page {
			case "buyer":
				http.Redirect(w, r, "/register-customer", http.StatusMovedPermanently)
				return
			case "seller":
				http.Redirect(w, r, "/register-provider", http.StatusMovedPermanently)
				return
			}
		}

		// Otherwise serve static files
		fileServer.ServeHTTP(w, r)
	})

	// // Admin dashboard static files
	// adminFileServer := http.FileServer(http.Dir("docs/admin"))
	// mux.Handle("/admin/", http.StripPrefix("/admin/", adminFileServer))

	// Main API Routes
	mux.HandleFunc("/api/validate-email", s.LogRequest(s.EnableCORS(s.RateLimitIP(s.HandleSendEmailValidationCode))))
	mux.HandleFunc("/api/verify-code", s.LogRequest(s.EnableCORS(s.HandleValidateEmailCode)))
	mux.HandleFunc("/api/register", s.LogRequest(s.EnableCORS(s.HandleRegister)))
	mux.HandleFunc("/api/representatives", s.LogRequest(s.EnableCORS(s.HandleUpdateRepresentatives)))
	mux.HandleFunc("/api/orgstatus", s.LogRequest(s.EnableCORS(s.HandleOrgStatus)))
	mux.HandleFunc("/health", s.HandleHealth)

	// Admin routes
	adminChain := func(next http.HandlerFunc) http.HandlerFunc {
		return s.LogRequest(s.EnableCORS(s.BasicAuth(next)))
	}

	// Admin routes for pages and APIs
	mux.HandleFunc("/admin/index", adminChain(s.PageAdminIndex))
	mux.HandleFunc("/admin/registration", adminChain(s.PageAdminDetailsByVatID))
	mux.HandleFunc("/admin/api/registrations", adminChain(s.APIAdminGetRegistrations))
	mux.HandleFunc("/admin/api/registration", adminChain(s.APIAdminGetRegistrationByVatID))
	mux.HandleFunc("/admin/api/registration-logs", adminChain(s.APIAdminGetRegistrationLogs))
	mux.HandleFunc("/admin/api/registration-files", adminChain(s.APIAdminGetRegistrationFiles))
	mux.HandleFunc("/admin/api/file/{file_id}", adminChain(s.APIAdminGetFile))

	// Serve Angular SPA routes
	serveIndex := func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "dist/browser/index.html")
	}
	mux.HandleFunc("/register-customer", serveIndex)
	mux.HandleFunc("/register-provider", serveIndex)
	mux.HandleFunc("/onboard-provider", serveIndex)

	s.Handler = mux
	return s
}

func (s *Server) LogRequest(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slog.Info("Entry", "method", r.Method, "url", r.URL.Path)
		next(w, r)
	}
}

func (s *Server) BasicAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != s.AdminUser || pass != s.AdminPassword {
			w.Header().Set("WWW-Authenticate", `Basic realm="Admin"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func (s *Server) getIPLimiter(ip string) *rate.Limiter {
	s.IPLimitersMu.Lock()
	defer s.IPLimitersMu.Unlock()

	limiter, exists := s.IPLimiters[ip]
	if !exists {
		// Allow 1 request per second with a burst of 5
		limiter = rate.NewLimiter(1, 5)
		s.IPLimiters[ip] = limiter
	}

	return limiter
}

func (s *Server) RateLimitIP(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}

		limiter := s.getIPLimiter(ip)
		if !limiter.Allow() {
			s.SendJSON(w, r, http.StatusTooManyRequests, false, "Too many requests", nil)
			return
		}

		next(w, r)
	}
}
