package middleware

import (
	"net/http"

	"github.com/gorilla/sessions"
)

const (
	SessionName = "session"
	TokenKey    = "token"
)

// SessionManager wraps gorilla/sessions for cookie-based session storage.
type SessionManager struct {
	Store *sessions.CookieStore
}

// NewSessionManager creates a SessionManager with the given secret key.
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

// GetToken returns the JWT token from the session cookie, or "" if absent.
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

// SetToken saves the JWT token into the session cookie.
func (sm *SessionManager) SetToken(w http.ResponseWriter, r *http.Request, token string) error {
	session, _ := sm.Store.Get(r, SessionName)

	session.Values[TokenKey] = token
	return session.Save(r, w)
}

// Clear destroys the session by expiring the cookie.
func (sm *SessionManager) Clear(w http.ResponseWriter, r *http.Request) error {
	session, _ := sm.Store.Get(r, SessionName)

	session.Options.MaxAge = -1
	return session.Save(r, w)
}

// RequireAuth redirects unauthenticated users to login and sets no-cache headers.
func (sm *SessionManager) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")

		token := sm.GetToken(r)
		if token == "" {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RedirectIfAuth redirects logged-in users to their profile.
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
