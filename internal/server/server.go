package server

import (
	"log/slog"
	"net"
	"net/http"
	"sync"

	"golang.org/x/time/rate"

	"github.com/hesusruiz/onboardng/internal/configuration"
	"github.com/hesusruiz/utils/errl"
)

type Server struct {
	// The runtime environment where the server runs (dev, pre, pre2, pro)
	Runtime configuration.RuntimeEnv

	// The database service provider
	DB DBServiceProvider

	// The Issuer service provider
	Issuer IssuanceServiceProvider

	// The mail service provider
	Mail MailServiceProvider

	// We implement a rate limit on the emails and verification codes sent, using the source IPs
	EmailRateLimiter  map[string]*RateLimitEntry
	VerificationCodes map[string]*VerificationCodeEntry
	IPLimiters        map[string]*rate.Limiter
	RateLimiterMu     sync.RWMutex
	CodesMu           sync.RWMutex
	IPLimitersMu      sync.Mutex

	// The HTTP web server handler
	Handler http.Handler

	// Admin credentials
	AdminUser     string
	AdminPassword string

	// Optional feature flags implemented
	Features configuration.Features
}

func NewServer(runtime configuration.RuntimeEnv,
	dbService DBServiceProvider,
	issuer IssuanceServiceProvider,
	mailService MailServiceProvider,
	staticFilesDir string,
	adminUser string,
	adminPassword string,
	features configuration.Features) (*Server, error) {

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

	// Middleware to wrap the Admin routes
	nonAdminChain := func(next http.HandlerFunc) http.Handler {
		return s.LogRequest(s.EnableCORS(s.RateLimitIP(next)))
	}

	// Main API Routes
	mux.Handle("/api/validate-email", nonAdminChain(s.HandleSendEmailValidationCode))
	mux.Handle("/api/verify-code", nonAdminChain(s.HandleValidateEmailCode))
	mux.Handle("/api/register", nonAdminChain(s.HandleRegister))
	mux.Handle("/api/representatives", nonAdminChain(s.HandleUpdateRepresentatives))
	mux.Handle("/api/orgstatus", nonAdminChain(s.HandleOrgStatus))
	mux.Handle("/health", s.EnableCORS(s.RateLimitIP(http.HandlerFunc(s.HandleHealth))))

	// Admin routes
	s.registerAdminRoutes(mux)

	// Serve Angular SPA routes
	serveIndex := func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "dist/browser/index.html")
	}
	mux.HandleFunc("/register-customer", serveIndex)
	mux.HandleFunc("/register-provider", serveIndex)
	mux.HandleFunc("/onboard-provider", serveIndex)

	s.Handler = mux

	// Perform the run-time checks to see if everything works as expected
	err := s.CheckAll()
	if err != nil {
		err = errl.Errorf("Self-Diagnostics: failed to check all: %v", err)
		slog.Error(err.Error())
		return nil, errl.Error(err)
	}

	return s, nil
}

func (s *Server) LogRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info("Entry", "method", r.Method, "url", r.URL.Path)
		next.ServeHTTP(w, r)
	})
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

func (s *Server) RateLimitIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}

		limiter := s.getIPLimiter(ip)
		if !limiter.Allow() {
			s.SendJSON(w, r, http.StatusTooManyRequests, false, "Too many requests", nil)
			return
		}

		next.ServeHTTP(w, r)
	})
}
