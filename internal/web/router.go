package web

import (
	"html/template"
	"log"
	"net/http"
	"path/filepath"

	"github.com/gorilla/mux"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"challenge-go-cyaz/internal/config"
	"challenge-go-cyaz/internal/web/client"
	"challenge-go-cyaz/internal/web/handler"
	"challenge-go-cyaz/internal/web/middleware"
)

// NewRouter creates and configures the web frontend router.
func NewRouter(cfg *config.Config) http.Handler {
	sessions := middleware.NewSessionManager(cfg.SessionSecret)
	apiClient := client.New(cfg.APIBaseURL)

	// OAuth 2.0 configuration for Google sign-in.
	// ClientID and ClientSecret come from the Google Cloud Console:
	//   https://console.cloud.google.com/apis/credentials
	// RedirectURL must exactly match what's configured in the Google Console.
	// Scopes define what user data we request:
	//   "openid"  — confirms the user's identity (required for OpenID Connect)
	//   "email"   — access to the user's email address
	//   "profile" — access to the user's name and profile picture
	// Endpoint is Google's authorization and token URLs, provided by the library.
	oauthCfg := &oauth2.Config{
		ClientID:     cfg.GoogleClientID,
		ClientSecret: cfg.GoogleClientSecret,
		RedirectURL:  cfg.GoogleRedirectURL,
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     google.Endpoint,
	}

	templates := loadTemplates()

	authHandler := handler.NewAuthWebHandler(templates, apiClient, sessions, oauthCfg)
	profileHandler := handler.NewProfileWebHandler(templates, apiClient, sessions)

	r := mux.NewRouter()

	// Static files
	fs := http.FileServer(http.Dir("static"))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))

	// Public routes (redirect to profile if already logged in)
	public := r.PathPrefix("").Subrouter()
	public.Use(sessions.RedirectIfAuth)
	public.HandleFunc("/login", authHandler.LoginPage).Methods("GET")
	public.HandleFunc("/signup", authHandler.SignupPage).Methods("GET")

	// Auth form submissions (no redirect-if-auth, allow POST always)
	r.HandleFunc("/login", authHandler.LoginSubmit).Methods("POST")
	r.HandleFunc("/signup", authHandler.SignupSubmit).Methods("POST")

	// Google OAuth
	r.HandleFunc("/auth/google", authHandler.GoogleLogin).Methods("GET")
	r.HandleFunc("/auth/google/callback", authHandler.GoogleCallback).Methods("GET")

	// Protected routes
	protected := r.PathPrefix("").Subrouter()
	protected.Use(sessions.RequireAuth)
	protected.HandleFunc("/profile", profileHandler.ViewProfile).Methods("GET")
	protected.HandleFunc("/profile/edit", profileHandler.EditProfile).Methods("GET")
	protected.HandleFunc("/profile/edit", profileHandler.EditProfileSubmit).Methods("POST")

	// Logout (protected so only authenticated users can log out)
	protected.HandleFunc("/logout", authHandler.Logout).Methods("POST")

	// Root redirect
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	})

	return r
}

func loadTemplates() map[string]*template.Template {
	templates := make(map[string]*template.Template)
	base := filepath.Join("templates", "base.html")
	pages := []string{"login", "signup", "profile_edit", "profile_view"}

	for _, page := range pages {
		file := filepath.Join("templates", page+".html")
		tmpl, err := template.ParseFiles(base, file)
		if err != nil {
			log.Fatalf("Error parsing template %s: %v", page, err)
		}
		templates[page] = tmpl
	}

	return templates
}