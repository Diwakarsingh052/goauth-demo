package web

import (
	"challenge-go-cyaz/internal/web/client"
	"challenge-go-cyaz/internal/web/handler"
	"challenge-go-cyaz/internal/web/middleware"
	"html/template"
	"log"
	"net/http"
	"path/filepath"

	"github.com/gorilla/mux"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"challenge-go-cyaz/internal/config"
)

// NewRouter creates the web frontend router with all routes and middleware.
func NewRouter(cfg *config.Config) http.Handler {
	sessions := middleware.NewSessionManager(cfg.SessionSecret)
	apiClient := client.New(cfg.APIBaseURL)

	// Google OAuth2 configuration
	oauthCfg := &oauth2.Config{
		ClientID:     cfg.GoogleClientID,
		ClientSecret: cfg.GoogleClientSecret,
		RedirectURL:  cfg.GoogleRedirectURL,
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     google.Endpoint,
	}

	templates := loadTemplates()
	h := handler.NewHandler(templates, apiClient, sessions, oauthCfg)

	r := mux.NewRouter()

	// Static files
	fs := http.FileServer(http.Dir("static"))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))

	// Public routes (redirect to profile if already logged in)
	public := r.PathPrefix("").Subrouter()
	public.Use(sessions.RedirectIfAuth)
	public.HandleFunc("/login", h.LoginPage).Methods("GET")
	public.HandleFunc("/signup", h.SignupPage).Methods("GET")

	// Auth form submissions (no redirect-if-auth, allow POST always)
	r.HandleFunc("/login", h.LoginSubmit).Methods("POST")
	r.HandleFunc("/signup", h.SignupSubmit).Methods("POST")

	// Google OAuth
	r.HandleFunc("/auth/google", h.GoogleLogin).Methods("GET")
	r.HandleFunc("/auth/google/callback", h.GoogleCallback).Methods("GET")

	// Protected routes
	protected := r.PathPrefix("").Subrouter()
	protected.Use(sessions.RequireAuth)
	protected.HandleFunc("/profile", h.ViewProfile).Methods("GET")
	protected.HandleFunc("/profile/edit", h.EditProfile).Methods("GET")
	protected.HandleFunc("/profile/edit", h.EditProfileSubmit).Methods("POST")

	// Logout (protected so only authenticated users can log out)
	protected.HandleFunc("/logout", h.Logout).Methods("POST")

	// Root redirect
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	})

	return r
}

// loadTemplates parses each page template together with the base layout.
func loadTemplates() map[string]*template.Template {
	// create an empty map to store parsed templates by name
	templates := make(map[string]*template.Template)
	// path to the shared base layout that wraps every page
	base := filepath.Join("templates", "base.html")
	// list of all page template names (each has a matching .html file)
	pages := []string{"login", "signup", "profile_edit", "profile_view"}

	for _, page := range pages {
		// build the full file path, e.g. "templates/login.html"
		file := filepath.Join("templates", page+".html")
		// parse base.html + the page file together so {{template "content" .}} works
		tmpl, err := template.ParseFiles(base, file)
		if err != nil {
			log.Fatalf("failed to parse template %s: %v", page, err)
		}
		// store the parsed template in the map keyed by page name
		templates[page] = tmpl
	}

	return templates
}
