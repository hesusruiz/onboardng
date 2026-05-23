package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-webauthn/webauthn/webauthn"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// --- 1. MODELS & INTERFACES ---

func main_old() {
	var err error
	waInstance, err = webauthn.New(&webauthn.Config{
		RPDisplayName: "Admin Panel",
		RPID:          "localhost",
		RPOrigins:     []string{"http://localhost:8080"},
	})
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()

	// --- ROUTES ---

	// SuperAdmin: mTLS + Basic Auth protected
	mux.Handle("/superadmin/invite", superAdminMiddleware(http.HandlerFunc(handleInvitePage)))
	mux.Handle("/api/superadmin/create-token", superAdminMiddleware(http.HandlerFunc(handleCreateToken)))

	// Registration (The "Invite" link landing page)
	mux.HandleFunc("/register", handleRegisterPage)
	mux.HandleFunc("/api/register/begin", handleRegisterBegin)
	mux.HandleFunc("/api/register/finish", handleRegisterFinish)

	// Login
	mux.HandleFunc("/login", handleLoginPage)
	mux.HandleFunc("/api/login/begin", handleLoginBegin)
	mux.HandleFunc("/api/login/finish", handleLoginFinish)

	fmt.Println("Admin server running on localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

// --- 3. MIDDLEWARE ---

func superAdminMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO: Insert your eIDAS mTLS library logic here

		user, pass, ok := r.BasicAuth()
		if !ok || user != "admin" || pass != os.Getenv("SUPERADMIN_PASSWORD") {
			w.Header().Set("WWW-Authenticate", `Basic realm="SuperAdmin"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// --- 4. UI COMPONENTS (Gomponents) ---

func PageLayout(title string, body Node) Node {
	return HTML(
		Lang("en"),
		Head(
			Meta(Charset("utf-8")),
			TitleEl(Text(title)),
			Link(Rel("stylesheet"), Name("href"), Text("https://cdn.jsdelivr.net/npm/@picocss/pico@2/css/pico.min.css")),
			// SimpleWebAuthn Browser Helper
			Script(Src("https://unpkg.com/@simplewebauthn/browser/dist/bundle/index.es5.umd.min.js")),
		),
		Body(
			Main(Class("container"),
				H1(Text(title)),
				body,
			),
		),
	)
}

// --- 5. HANDLERS ---

func handleRegisterPage(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("t")
	if token == "" {
		token = "12345"
	}

	// TODO: Check SQLite if token is valid

	page := PageLayout("Register Admin Passkey", Div(
		P(Text("Welcome. Please register your biometric key to continue.")),
		Button(Type("button"), Text("Register Passkey"), ID("reg-btn"), Attr("onclick", fmt.Sprintf("startRegister('%s')", token))),
		Script(Raw(`
			async function startRegister(token) {
				debugger;
				const res = await fetch('/api/register/begin?t=' + token);
				const options = await res.json();

				let attestation;
				try {
					// Pass the options to the authenticator and wait for a response
					attestation = await SimpleWebAuthnBrowser.startRegistration(options);
				} catch (error) {
					// Some basic error handling
					if (error.name === 'InvalidStateError') {
						alert('Error: Authenticator was probably already registered by user');
					} else {
						alert(error);
					}

					throw error;
				}



				await fetch('/api/register/finish?t=' + token, {
					method: 'POST',
					body: JSON.stringify(attestation)
				});
				alert("Registration successful!");
				window.location.href = "/login";
			}
		`)),
	))
	page.Render(w)
}

// --- MOCK LOGIC FOR OTHERS ---

func handleInvitePage(w http.ResponseWriter, r *http.Request) {
	PageLayout("SuperAdmin - Invite", Div(
		Form(Method("POST"), Action("/api/superadmin/create-token"),
			Label(Text("Email"), Input(Type("email"), Name("email"))),
			Button(Type("submit"), Text("Generate Invite Link")),
		),
	)).Render(w)
}

func handleCreateToken(w http.ResponseWriter, r *http.Request) {
	// TODO: Generate random string, save to SQLite, return URL to browser
	fmt.Fprint(w, "Send this link: http://localhost:8080/register?t=RANDOM_TOKEN")
}

func handleLoginPage(w http.ResponseWriter, r *http.Request) { /* Similar to Register with BeginLogin */
}
func handleLoginBegin(w http.ResponseWriter, r *http.Request)  { /* waInstance.BeginLogin */ }
func handleLoginFinish(w http.ResponseWriter, r *http.Request) { /* waInstance.FinishLogin */ }
