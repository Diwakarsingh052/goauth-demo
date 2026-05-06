package handler

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"html/template"
	"net/http"

	"golang.org/x/oauth2"

	"challange-go-cyaz/internal/web/client"
	"challange-go-cyaz/internal/web/middleware"
)

// AuthWebHandler handles web authentication pages and form submissions.
type AuthWebHandler struct {
	Templates map[string]*template.Template
	APIClient *client.APIClient
	Sessions  *middleware.SessionManager
	OAuth     *oauth2.Config
}

// PageData is the data structure passed to templates.
type PageData struct {
	Title    string
	Error    string
	Success  string
	User     *client.UserData
	IsGoogle bool
}

// NewAuthWebHandler creates a new AuthWebHandler.
func NewAuthWebHandler(
	templates map[string]*template.Template,
	apiClient *client.APIClient,
	sessions *middleware.SessionManager,
	oauthCfg *oauth2.Config,
) *AuthWebHandler {
	return &AuthWebHandler{
		Templates: templates,
		APIClient: apiClient,
		Sessions:  sessions,
		OAuth:     oauthCfg,
	}
}

// LoginPage renders the login page (2-A).
func (h *AuthWebHandler) LoginPage(w http.ResponseWriter, r *http.Request) {
	data := PageData{Title: "Login"}
	if msg := r.URL.Query().Get("error"); msg != "" {
		data.Error = msg
	}
	h.Templates["login"].ExecuteTemplate(w, "base", data)
}

// LoginSubmit processes the login form submission.
func (h *AuthWebHandler) LoginSubmit(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	password := r.FormValue("password")

	resp, err := h.APIClient.Login(email, password)
	if err != nil {
		data := PageData{Title: "Login", Error: err.Error()}
		h.Templates["login"].ExecuteTemplate(w, "base", data)
		return
	}

	h.Sessions.SetToken(w, r, resp.Token)
	http.Redirect(w, r, "/profile", http.StatusSeeOther)
}

// SignupPage renders the signup page (2-B).
func (h *AuthWebHandler) SignupPage(w http.ResponseWriter, r *http.Request) {
	data := PageData{Title: "Sign Up"}
	if msg := r.URL.Query().Get("error"); msg != "" {
		data.Error = msg
	}
	h.Templates["signup"].ExecuteTemplate(w, "base", data)
}

// SignupSubmit processes the signup form submission.
func (h *AuthWebHandler) SignupSubmit(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	password := r.FormValue("password")

	resp, err := h.APIClient.Signup(email, password)
	if err != nil {
		data := PageData{Title: "Sign Up", Error: err.Error()}
		h.Templates["signup"].ExecuteTemplate(w, "base", data)
		return
	}

	h.Sessions.SetToken(w, r, resp.Token)
	http.Redirect(w, r, "/profile/edit", http.StatusSeeOther)
}

// GoogleLogin initiates the Google OAuth 2.0 Authorization Code flow.
// This is STEP 1 of the OAuth flow:
//
//   Browser                  This Server              Google
//   ───────                  ───────────              ──────
//   1. GET /auth/google  ──> GoogleLogin
//                            generates state,
//                            saves it in cookie
//                        <── 302 redirect to ──────> Google consent screen
//   2. User logs into Google, grants permission
//      Google redirects back with ?code=...&state=...
//   3. GET /auth/google/callback ──> GoogleCallback (see below)
//
// The oauth2.Config (h.OAuth) is set up in router.go with:
//   - ClientID / ClientSecret: from Google Cloud Console credentials
//   - RedirectURL: must exactly match what's registered in Google Console
//   - Scopes: "openid", "email", "profile" — what data we request from Google
//   - Endpoint: google.Endpoint — Google's authorize + token URLs
func (h *AuthWebHandler) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	// Generate a cryptographically random "state" string (32 bytes, base64url-encoded).
	// This is a CSRF protection token: we save it in our session cookie now, and when
	// Google redirects back, we verify the returned state matches. This prevents an
	// attacker from tricking a user into completing an OAuth flow they didn't start.
	state := generateOAuthState()

	// Save the state in the session cookie so GoogleCallback can verify it later.
	// Also save the mode ("login" or "signup") so we know whether to create a new
	// account or only look up an existing one after Google returns.
	session, _ := h.Sessions.Store.Get(r, middleware.SessionName)
	session.Values["oauth_state"] = state
	session.Values["oauth_mode"] = r.URL.Query().Get("mode")
	session.Save(r, w)

	// AuthCodeURL builds the full Google authorization URL:
	//   https://accounts.google.com/o/oauth2/auth?
	//     client_id=<our client ID>
	//     &redirect_uri=<our callback URL>
	//     &response_type=code          ← we want an authorization code back
	//     &scope=openid+email+profile  ← what user data we're requesting
	//     &state=<our random state>    ← CSRF token
	//     &access_type=offline         ← also request a refresh token
	//
	// The browser is then redirected to this URL. Google shows the consent screen.
	// After the user approves, Google redirects to our RedirectURL (the callback)
	// with ?code=<authorization_code>&state=<our_state> in the query string.
	url := h.OAuth.AuthCodeURL(state, oauth2.AccessTypeOffline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// GoogleCallback handles the redirect back from Google after the user grants consent.
// This is STEP 2 of the OAuth flow. Google has redirected the user's browser to:
//
//	/auth/google/callback?code=<authorization_code>&state=<state>
//
// This handler performs the following sequence:
//  1. Verifies the state parameter matches what we saved (CSRF protection)
//  2. Exchanges the short-lived authorization code for an access token (server-to-server)
//  3. Uses the access token to call Google's userinfo API to get the user's profile
//  4. Passes that profile data to our own API to find or create the user
//  5. Stores the JWT from our API in the session cookie
//  6. Redirects to the profile page
func (h *AuthWebHandler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	// Retrieve the state and mode we saved in GoogleLogin, then delete them
	// from the session — they are single-use values.
	session, _ := h.Sessions.Store.Get(r, middleware.SessionName)
	savedState, _ := session.Values["oauth_state"].(string)
	mode, _ := session.Values["oauth_mode"].(string)
	delete(session.Values, "oauth_state")
	delete(session.Values, "oauth_mode")
	session.Save(r, w)

	// Redirect errors to signup page when in signup mode, login page otherwise.
	errorPage := "/login"
	if mode == "signup" {
		errorPage = "/signup"
	}

	// ── CSRF CHECK ──
	// Compare the "state" Google sent back with the one we saved in the session.
	// If they don't match, someone may have forged this request (CSRF attack).
	if r.URL.Query().Get("state") != savedState {
		http.Redirect(w, r, errorPage+"?error=Invalid+OAuth+state", http.StatusSeeOther)
		return
	}

	// The "code" is a short-lived authorization code that Google gave us.
	// It can only be used once and expires in a few minutes.
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Redirect(w, r, errorPage+"?error=OAuth+authorization+failed", http.StatusSeeOther)
		return
	}

	// ── TOKEN EXCHANGE (server-to-server, not visible to the browser) ──
	// Exchange the authorization code for an access token by making a POST request
	// to Google's token endpoint (https://oauth2.googleapis.com/token).
	// This sends our client_id, client_secret, the code, and the redirect_uri.
	// Google verifies all of these and returns an access token (+ optional refresh token).
	// The access token is what lets us call Google APIs on behalf of this user.
	token, err := h.OAuth.Exchange(context.Background(), code)
	if err != nil {
		http.Redirect(w, r, errorPage+"?error=Failed+to+exchange+token", http.StatusSeeOther)
		return
	}

	// ── FETCH USER INFO FROM GOOGLE ──
	// Create an HTTP client that automatically attaches the access token to requests
	// as "Authorization: Bearer <access_token>". Then call Google's userinfo endpoint
	// to get the user's Google ID, email address, and display name.
	httpClient := h.OAuth.Client(context.Background(), token)
	resp, err := httpClient.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		http.Redirect(w, r, errorPage+"?error=Failed+to+get+user+info", http.StatusSeeOther)
		return
	}
	defer resp.Body.Close()

	// Parse the JSON response from Google. We only need id, email, and name.
	// Example response: {"id":"1234567890","email":"user@gmail.com","name":"John Doe"}
	var googleUser struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&googleUser); err != nil {
		http.Redirect(w, r, errorPage+"?error=Failed+to+parse+user+info", http.StatusSeeOther)
		return
	}

	// ── HAND OFF TO OUR API ──
	// Send the Google user's info to our own REST API. The API will either find an
	// existing user (login mode) or create a new one (signup mode), then return a
	// JWT token that we store in the session cookie for subsequent requests.
	authResp, err := h.APIClient.GoogleAuth(googleUser.ID, googleUser.Email, googleUser.Name, mode)
	if err != nil {
		http.Redirect(w, r, errorPage+"?error="+err.Error(), http.StatusSeeOther)
		return
	}

	// Store the JWT from our API in the session cookie. From this point on the user
	// is authenticated — the web server sends this JWT to the API on every request.
	h.Sessions.SetToken(w, r, authResp.Token)

	// New Google users go to profile edit (2-C) to enter profile info;
	// existing Google users go straight to main profile (2-D).
	if authResp.IsNew {
		http.Redirect(w, r, "/profile/edit", http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/profile", http.StatusSeeOther)
	}
}

// Logout clears the session and redirects to login.
func (h *AuthWebHandler) Logout(w http.ResponseWriter, r *http.Request) {
	h.Sessions.Clear(w, r)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// generateOAuthState creates a random string used as the OAuth "state" parameter.
// 32 bytes of crypto/rand gives 256 bits of entropy — impossible to guess.
// base64url encoding makes it URL-safe. This value is saved in the session before
// redirecting to Google and verified when Google redirects back, preventing CSRF.
func generateOAuthState() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}
