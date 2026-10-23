package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/hesusruiz/onboardng/internal/db"
	"github.com/hesusruiz/onboardng/views"
)

const domeTestUrl = "https://dome-marketplace-sbx.org/"

func main() {
	appViews, err := views.NewAppViews(true)
	if err != nil {
		log.Fatalf("Failed to create views: %v", err)
	}
	adminViews, err := views.NewAdminViews(true)
	if err != nil {
		log.Fatalf("Failed to create views: %v", err)
	}

	// Serve static assets if present so styles and images load in the browser
	if _, err := os.Stat("views/assets"); err == nil {
		fileServer := http.FileServer(http.Dir("views/assets"))
		http.Handle("/myassets/", http.StripPrefix("/myassets/", fileServer))
		http.Handle("/views/myassets/", http.StripPrefix("/views/myassets/", fileServer))
	}

	http.HandleFunc("/views/", func(w http.ResponseWriter, r *http.Request) {
		html := `
	<p><a href="/views/qrcode">qrcode</a></p>
	<p><a href="/views/index">index</a></p>
		`

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
		regs := []db.RegistrationRecord{
			{
				VatID:            "vat1",
				CompanyName:      "Company 1",
				Email:            "[EMAIL_ADDRESS]",
				FirstName:        "John",
				LastName:         "Doe",
				Issued:           true,
				CreatedAt:        time.Now(),
				UpdatedAt:        time.Now(),
				LRFirstName:      "John",
				LRLastName:       "Doe",
				LREmail:          "[EMAIL_ADDRESS]",
				LRCountry:        "ES",
				LRIdCard:         "12345678",
				LRCompleted:      true,
				LEARFirstName:    "Jane",
				LEARLastName:     "Doe",
				LEAREmail:        "[EMAIL_ADDRESS]",
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

		html, err := adminViews.RenderIndexView("admin@alastria.io", regs, 20, 0, 3, 100)
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
