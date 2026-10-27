package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/hesusruiz/onboardng/internal/configuration"
	"github.com/hesusruiz/onboardng/internal/db"
	"github.com/hesusruiz/onboardng/internal/mail"
	"github.com/hesusruiz/onboardng/views"
)

const domeTestUrl = "https://dome-marketplace-sbx.org/"

var regs = []db.RegistrationRecord{
	{
		RegistrationID:   "12345678",
		VatID:            "vat1",
		CompanyName:      "Company 1",
		Country:          "ES",
		Email:            "john.doe@example.com",
		FirstName:        "John",
		LastName:         "Doe",
		Issued:           true,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
		LRFirstName:      "John",
		LRLastName:       "Doe",
		LREmail:          "legal@example.com",
		LRCountry:        "ES",
		LRIdCard:         "12345678",
		LRCompleted:      true,
		LEARFirstName:    "Jane",
		LEARLastName:     "Doe",
		LEAREmail:        "lear@example.com",
		LEARCountry:      "ES",
		LEARAddress:      "123 Main St",
		LEARIdCard:       "12345678",
		LEARMobileNumber: "1234567890",
		LEARCompleted:    true,
	},
	{
		VatID:       "vat2",
		CompanyName: "Company 2",
		Email:       "[EMAIL_ADDRESS]",
		FirstName:   "Jane",
		LastName:    "Doe",
	},
	{
		VatID:       "vat3",
		CompanyName: "Company 3",
		Email:       "[EMAIL_ADDRESS]",
		FirstName:   "Peter",
		LastName:    "Smith",
	},
}

func main() {
	appViews, err := views.NewAppViews(true)
	if err != nil {
		log.Fatalf("Failed to create views: %v", err)
	}
	adminViews, err := views.NewAdminViews(true)
	if err != nil {
		log.Fatalf("Failed to create views: %v", err)
	}

	mailViews, err := mail.NewMailViews(true)
	if err != nil {
		log.Fatalf("Failed to create mail views: %v", err)
	}

	// Serve static assets if present so styles and images load in the browser
	if _, err := os.Stat("views/assets"); err == nil {
		fileServer := http.FileServer(http.Dir("views/assets"))
		http.Handle("/myassets/", http.StripPrefix("/myassets/", fileServer))
		http.Handle("/views/myassets/", http.StripPrefix("/views/myassets/", fileServer))
	}

	http.HandleFunc("/views/", func(w http.ResponseWriter, r *http.Request) {
		html := `
		<p><a href="/views/dashboard">dashboard</a></p>
		<p><a href="/views/qrcode">qrcode</a></p>
		<p><a href="/mail/first_phase">mail first phase</a></p>
		<p><a href="/views/index">index</a></p>
		<p><a href="/mail/reminder">mail reminder</a></p>
		<p><a href="/mail/second_phase">mail second phase</a></p>
		<p><a href="/mail/check_issuer_error">mail check_issuer_error</a></p>
		<p><a href="/mail/verification">mail verification</a></p>
		<p><a href="/mail/test">mail test</a></p>
		<p><a href="/mail/issuer_error">mail issuer_error</a></p>
		`

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))

	})

	http.HandleFunc("/views/dashboard", func(w http.ResponseWriter, r *http.Request) {

		html, err := adminViews.RenderDashboardView()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	})

	http.HandleFunc("/views/qrcode", func(w http.ResponseWriter, r *http.Request) {
		offerURI := r.URL.Query().Get("credential_offer_uri")
		if offerURI == "" {
			offerURI = "https://google.com"
		}

		html, err := appViews.RenderQRCodeView(offerURI, domeTestUrl)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	})

	http.HandleFunc("/views/index", func(w http.ResponseWriter, r *http.Request) {

		html, err := adminViews.RenderIndexView("admin@alastria.io", regs, 20, 0, 3, 100)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	})

	http.HandleFunc("/mail/first_phase", func(w http.ResponseWriter, r *http.Request) {
		html, err := mailViews.RenderEmailFirstPhaseView(
			configuration.RuntimeEnv("dev"),
			[]string{"onboard@example.com"},
			&regs[0])
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	})

	http.HandleFunc("/mail/second_phase", func(w http.ResponseWriter, r *http.Request) {
		html, err := mailViews.RenderEmailSecondPhaseView(
			configuration.RuntimeEnv("dev"),
			[]string{"onboard@example.com"},
			&regs[0])
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	})

	http.HandleFunc("/mail/check_issuer_error", func(w http.ResponseWriter, r *http.Request) {
		html, err := mailViews.RenderCheckIssuerErrorView(
			configuration.RuntimeEnv("dev"),
			"this is the error",
			"{\"foo\": \"bar\"}")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	})

	http.HandleFunc("/mail/issuer_error", func(w http.ResponseWriter, r *http.Request) {
		html, err := mailViews.RenderIssuerErrorView(
			configuration.RuntimeEnv("dev"),
			"this is the error",
			"{\"foo\": \"bar\"}",
			&regs[0])
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	})

	http.HandleFunc("/mail/reminder", func(w http.ResponseWriter, r *http.Request) {
		html, err := mailViews.RenderReminderEmailView(
			configuration.RuntimeEnv("dev"),
			[]string{"onboard@example.com"},
			&regs[0])
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	})

	http.HandleFunc("/mail/verification", func(w http.ResponseWriter, r *http.Request) {
		html, err := mailViews.RenderVerificationCodeEmailView(
			configuration.RuntimeEnv("dev"),
			[]string{"onboard@example.com"},
			"123456")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	})

	http.HandleFunc("/mail/test", func(w http.ResponseWriter, r *http.Request) {
		html, err := mailViews.RenderTestEmailView(
			configuration.RuntimeEnv("dev"),
			[]string{"onboard@example.com"},
			"123456")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "7777"
	}

	addr := ":" + port
	log.Printf("Starting web server on http://localhost:%s/views/", port)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
