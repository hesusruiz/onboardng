package server

import (
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"sync"

	"golang.org/x/time/rate"

	"github.com/hesusruiz/onboardng/internal/configuration"
	"github.com/hesusruiz/onboardng/views"
	"github.com/hesusruiz/utils/errl"
)

// Server defines the runtime configuration of the Onboarding server
type Server struct {
	// The runtime environment where the server runs (dev, pre, pre2, pro), to enable slight behaviour differences
	Runtime configuration.RuntimeEnv

	// The application URL
	AppURL string

	// The database service provider, can be replaced if a different storage mechanism is needed
	DB DBServiceProvider

	// The Verifiable Credential Issuer service provider
	Issuer IssuanceServiceProvider

	// The Wallet URL
	WalletURL string

	// The mail service provider
	Mail MailServiceProvider

	// A simple but robust rate limit on the emails and verification codes sent, using the source IPs
	EmailRateLimiter  map[string]*RateLimitEntry
	VerificationCodes map[string]*VerificationCodeEntry
	IPLimiters        map[string]*rate.Limiter
	RateLimiterMu     sync.RWMutex
	CodesMu           sync.RWMutex
	IPLimitersMu      sync.Mutex

	// The HTTP web server handler
	Handler http.Handler

	// SuperAdmin credentials, set at server startup via environment variables
	AdminUser     string
	AdminPassword string

	// Optional feature flags implemented
	Features configuration.Features
}

type ServerConfig struct {
	Runtime        configuration.RuntimeEnv
	AppURL         string
	DB             DBServiceProvider
	Issuer         IssuanceServiceProvider
	WalletURL      string
	Mail           MailServiceProvider
	StaticFilesDir string
	AdminUser      string
	AdminPassword  string
	Features       configuration.Features
}

func NewHTTPServer(cfg *ServerConfig) (*Server, error) {

	s := &Server{
		Runtime:           cfg.Runtime,
		AppURL:            cfg.AppURL,
		DB:                cfg.DB,
		Issuer:            cfg.Issuer,
		WalletURL:         cfg.WalletURL,
		Mail:              cfg.Mail,
		EmailRateLimiter:  make(map[string]*RateLimitEntry),
		VerificationCodes: make(map[string]*VerificationCodeEntry),
		IPLimiters:        make(map[string]*rate.Limiter),
		AdminUser:         cfg.AdminUser,
		AdminPassword:     cfg.AdminPassword,
		Features:          cfg.Features,
	}

	mux := http.NewServeMux()

	// Static file serving for the frontend, implemented as static files
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

	mux.Handle("/myassets/", http.StripPrefix("/myassets/", http.FileServer(http.Dir("src/assets"))))

	appViews, err := views.NewAppViews(true)
	if err != nil {
		log.Fatalf("Failed to create views: %v", err)
	}

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
	mux.Handle("/qrold", nonAdminChain(s.QRCodeHandler))
	mux.Handle("/health", s.EnableCORS(s.RateLimitIP(http.HandlerFunc(s.HandleHealth))))

	mux.HandleFunc("/qr", func(w http.ResponseWriter, r *http.Request) {

		// Get the credential_offer_uri from the parameter in the request URL
		credentialOfferURI := r.URL.Query().Get("credential_offer_uri")

		// Return error if credentialOfferURI is not provided
		if credentialOfferURI == "" {
			http.Error(w, "credential_offer_uri is required", http.StatusBadRequest)
			return
		}

		fmt.Printf("credentialOfferURI: %s\n", credentialOfferURI)

		//URLEncode the credentialOfferURI
		credentialOfferURI = url.QueryEscape(credentialOfferURI)

		// Add the prefix for the wallet credential issuance callback URL
		credentialOfferURI = s.WalletURL + "?credential_offer_uri=" + credentialOfferURI

		fmt.Printf("Displaying credentialOfferURI: %s\n", credentialOfferURI)

		html, err := appViews.RenderQRCodeView(credentialOfferURI, "https://dome-marketplace-sbx.org")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	})

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
	// Only if we are not in development
	if !s.Runtime.IsDevelopment() {
		err := s.CheckAll()
		if err != nil {
			err = errl.Errorf("Self-Diagnostics: failed to check all: %v", err)
			slog.Error(err.Error())

			if s.Runtime.IsProduction() {
				return nil, errl.Error(err)
			} else {
				slog.Warn("Self-Diagnostics: failed to check all, but runtime is not production, so we continue")
			}
		}
	} else {
		err = s.CheckMail()
		if err != nil {
			err = errl.Errorf("Self-Diagnostics: failed to check Mail: %v", err)
			slog.Error(err.Error())
		}
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
