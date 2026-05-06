package middleware

import (
	"net/http"

	"github.com/gorilla/sessions"
)

const (
	SessionName = "session"
	TokenKey    = "token"
)

// SessionManager wraps gorilla/sessions for convenience.
type SessionManager struct {
	Store *sessions.CookieStore
}

// NewSessionManager creates a new SessionManager.
func NewSessionManager(secret string) *SessionManager {
	store := sessions.NewCookieStore([]byte(secret))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400, // 24 hours
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	return &SessionManager{Store: store}
}

// GetToken retrieves the JWT token from the session.
func (sm *SessionManager) GetToken(r *http.Request) string {
	session, err := sm.Store.Get(r, SessionName)
	if err != nil {
		return ""
	}
	token, ok := session.Values[TokenKey].(string)
	if !ok {
		return ""
	}
	return token
}

// SetToken stores the JWT token in the session.
func (sm *SessionManager) SetToken(w http.ResponseWriter, r *http.Request, token string) error {
	session, err := sm.Store.Get(r, SessionName)
	if err != nil {
		return err
	}
	session.Values[TokenKey] = token
	return session.Save(r, w)
}

// Clear destroys the session.
func (sm *SessionManager) Clear(w http.ResponseWriter, r *http.Request) error {
	session, err := sm.Store.Get(r, SessionName)
	if err != nil {
		return err
	}
	session.Values = make(map[interface{}]interface{})
	session.Options.MaxAge = -1
	return session.Save(r, w)
}

// RequireAuth is middleware that redirects unauthenticated users to the login page.
// It also sets no-cache headers to prevent viewing protected pages via back button.
func (sm *SessionManager) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		SetNoCacheHeaders(w)

		token := sm.GetToken(r)
		if token == "" {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RedirectIfAuth redirects already-authenticated users away from login/signup pages.
func (sm *SessionManager) RedirectIfAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := sm.GetToken(r)
		if token != "" {
			http.Redirect(w, r, "/profile", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// SetNoCacheHeaders prevents browsers from caching protected pages.
func SetNoCacheHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
}