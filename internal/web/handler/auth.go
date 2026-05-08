package handler

import (
	"small-app/internal/web/client"
	"small-app/internal/web/middleware"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"html/template"
	"log"
	"net/http"

	"golang.org/x/oauth2"
)

// Handler serves all web pages and form submissions.
type Handler struct {
	templates map[string]*template.Template
	api       *client.APIClient
	sessions  *middleware.SessionManager
	oauth     *oauth2.Config
}

// PageData holds the data passed to every HTML template.
type PageData struct {
	Title    string
	Error    string
	User     *client.UserData
	IsGoogle bool
}

// NewHandler creates a Handler with all dependencies.
func NewHandler(
	templates map[string]*template.Template,
	apiClient *client.APIClient,
	sessions *middleware.SessionManager,
	oauthCfg *oauth2.Config,
) *Handler {

	return &Handler{
		templates: templates,
		api:       apiClient,
		sessions:  sessions,
		oauth:     oauthCfg,
	}
}

// render executes a named template with the given data.
func (h *Handler) render(w http.ResponseWriter, page string, data PageData) {
	h.templates[page].ExecuteTemplate(w, "base", data)
}

// LoginPage renders the login form.
func (h *Handler) LoginPage(w http.ResponseWriter, r *http.Request) {
	data := PageData{Title: "Login"}
	if msg := r.URL.Query().Get("error"); msg != "" {
		data.Error = msg
	}
	h.render(w, "login", data)
}

// LoginSubmit authenticates the user with email and password.
func (h *Handler) LoginSubmit(w http.ResponseWriter, r *http.Request) {
	resp, err := h.api.Login(r.FormValue("email"), r.FormValue("password"))
	if err != nil {
		h.render(w, "login", PageData{Title: "Login", Error: err.Error()})
		return
	}

	if err := h.sessions.SetToken(w, r, resp.Token); err != nil {
		log.Printf("failed to save session after login: %v", err)
		h.render(w, "login", PageData{Title: "Login", Error: "failed to create session"})
		return
	}
	http.Redirect(w, r, "/profile", http.StatusSeeOther)
}

// SignupPage renders the signup form.
func (h *Handler) SignupPage(w http.ResponseWriter, r *http.Request) {
	data := PageData{Title: "Sign Up"}
	if msg := r.URL.Query().Get("error"); msg != "" {
		data.Error = msg
	}
	h.render(w, "signup", data)
}

// SignupSubmit creates a new user with email and password.
func (h *Handler) SignupSubmit(w http.ResponseWriter, r *http.Request) {
	resp, err := h.api.Signup(r.FormValue("email"), r.FormValue("password"))
	if err != nil {
		h.render(w, "signup", PageData{Title: "Sign Up", Error: err.Error()})
		return
	}

	if err := h.sessions.SetToken(w, r, resp.Token); err != nil {
		log.Printf("failed to save session after signup: %v", err)
		h.render(w, "signup", PageData{Title: "Sign Up", Error: "failed to create session"})
		return
	}
	http.Redirect(w, r, "/profile/edit", http.StatusSeeOther)
}

// GoogleLogin starts the Google OAuth flow by redirecting to Google's consent screen.
func (h *Handler) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	state := generateOAuthState()

	session, _ := h.sessions.Store.Get(r, middleware.SessionName)

	mode := r.URL.Query().Get("mode")
	session.Values["oauth_state"] = state
	session.Values["oauth_mode"] = mode
	if err := session.Save(r, w); err != nil {
		log.Printf("failed to save OAuth state to session: %v", err)
		tmpl, title := "login", "Login"
		if mode == "signup" {
			tmpl, title = "signup", "Sign Up"
		}
		h.render(w, tmpl, PageData{Title: title, Error: "Session error"})
		return
	}

	url := h.oauth.AuthCodeURL(state)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// GoogleCallback handles the redirect from Google after user grants consent.
func (h *Handler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	session, _ := h.sessions.Store.Get(r, middleware.SessionName)

	savedState, ok := session.Values["oauth_state"].(string)
	mode, _ := session.Values["oauth_mode"].(string)
	delete(session.Values, "oauth_state")
	delete(session.Values, "oauth_mode")
	if err := session.Save(r, w); err != nil {
		log.Printf("failed to clear OAuth state from session: %v", err)
	}

	errorTemplate, errorTitle := "login", "Login"
	if mode == "signup" {
		errorTemplate, errorTitle = "signup", "Sign Up"
	}

	renderError := func(msg string) {
		h.render(w, errorTemplate, PageData{Title: errorTitle, Error: msg})
	}

	if !ok || savedState == "" || r.URL.Query().Get("state") != savedState {
		renderError("Invalid OAuth state")
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		renderError("OAuth authorization failed")
		return
	}

	token, err := h.oauth.Exchange(r.Context(), code)
	if err != nil {
		renderError("Failed to exchange token")
		return
	}

	httpClient := h.oauth.Client(r.Context(), token)
	resp, err := httpClient.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		renderError("Failed to get user info")
		return
	}
	defer resp.Body.Close()

	var googleUser struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&googleUser); err != nil {
		renderError("Failed to parse user info")
		return
	}

	var authResp *client.APIResponse
	if mode == "signup" {
		authResp, err = h.api.GoogleSignup(googleUser.ID, googleUser.Email, googleUser.Name)
	} else {
		authResp, err = h.api.GoogleLogin(googleUser.ID, googleUser.Email)
	}
	if err != nil {
		renderError(err.Error())
		return
	}

	if err := h.sessions.SetToken(w, r, authResp.Token); err != nil {
		log.Printf("failed to save session after Google auth: %v", err)
		renderError("Failed to create session")
		return
	}

	if mode == "signup" {
		http.Redirect(w, r, "/profile/edit", http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/profile", http.StatusSeeOther)
	}
}

// Logout clears the session and redirects to the login page.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if err := h.sessions.Clear(w, r); err != nil {
		log.Printf("failed to clear session on logout: %v", err)
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// generateOAuthState returns a cryptographically random base64url string for CSRF protection.
func generateOAuthState() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}
