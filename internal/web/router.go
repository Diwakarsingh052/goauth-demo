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

	fs := http.FileServer(http.Dir("static"))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))

	public := r.PathPrefix("").Subrouter()
	public.Use(sessions.RedirectIfAuth)
	public.HandleFunc("/login", h.LoginPage).Methods("GET")
	public.HandleFunc("/login", h.LoginSubmit).Methods("POST")
	public.HandleFunc("/signup", h.SignupPage).Methods("GET")
	public.HandleFunc("/signup", h.SignupSubmit).Methods("POST")
	public.HandleFunc("/auth/google", h.GoogleLogin).Methods("GET")
	public.HandleFunc("/auth/google/callback", h.GoogleCallback).Methods("GET")
	public.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	})

	protected := r.PathPrefix("").Subrouter()
	protected.Use(sessions.RequireAuth)
	protected.HandleFunc("/profile", h.ViewProfile).Methods("GET")
	protected.HandleFunc("/profile/edit", h.EditProfile).Methods("GET")
	protected.HandleFunc("/profile/edit", h.EditProfileSubmit).Methods("POST")
	protected.HandleFunc("/logout", h.Logout).Methods("POST")

	return r
}

// loadTemplates parses each page template together with the base layout.
func loadTemplates() map[string]*template.Template {
	templates := make(map[string]*template.Template)
	base := filepath.Join("templates", "base.html")
	pages := []string{"login", "signup", "profile_edit", "profile_view"}

	for _, page := range pages {
		file := filepath.Join("templates", page+".html")
		tmpl, err := template.ParseFiles(base, file)
		if err != nil {
			log.Fatalf("failed to parse template %s: %v", page, err)
		}
		templates[page] = tmpl
	}

	return templates
}
