package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"challenge-go-cyaz/internal/api/middleware"
	"challenge-go-cyaz/internal/users"
)

// AuthHandler handles authentication-related API requests.
type AuthHandler struct {
	u         *users.UserStore
	jwtSecret string
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(users *users.UserStore, jwtSecret string) *AuthHandler {
	return &AuthHandler{u: users, jwtSecret: jwtSecret}
}

type signupRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type googleAuthRequest struct {
	GoogleID string `json:"google_id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
}

type authResponse struct {
	Token string      `json:"token"`
	User  *users.User `json:"user"`

}

// Signup handles POST /api/auth/signup
func (h *AuthHandler) Signup(w http.ResponseWriter, r *http.Request) {
	var req signupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" {
		jsonError(w, "email and password are required", http.StatusBadRequest)
		return
	}

	if len(req.Password) < 6 {
		jsonError(w, "password must be at least 6 characters", http.StatusBadRequest)
		return
	}

	user, err := h.u.Create(req.Email, req.Password)
	if err != nil {
		if errors.Is(err, users.ErrDuplicateEmail) {
			jsonError(w, "email already exists", http.StatusConflict)
			return
		}
		jsonError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	token, err := middleware.GenerateToken(user.ID, h.jwtSecret)
	if err != nil {
		jsonError(w, "failed to generate token", http.StatusInternalServerError)
		return
	}

	jsonResponse(w, authResponse{Token: token, User: user}, http.StatusCreated)
}

// Login handles POST /api/auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" {
		jsonError(w, "email and password are required", http.StatusBadRequest)
		return
	}

	user, err := h.u.Authenticate(req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, users.ErrUserNotFound):
			jsonError(w, "username entered does not exist", http.StatusUnauthorized)
		case errors.Is(err, users.ErrInvalidPassword):
			jsonError(w, "password is incorrect", http.StatusUnauthorized)
		case errors.Is(err, users.ErrGoogleAccount):
			jsonError(w, err.Error(), http.StatusBadRequest)
		default:
			jsonError(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	token, err := middleware.GenerateToken(user.ID, h.jwtSecret)
	if err != nil {
		jsonError(w, "failed to generate token", http.StatusInternalServerError)
		return
	}

	jsonResponse(w, authResponse{Token: token, User: user}, http.StatusOK)
}

// GoogleSignup handles POST /api/auth/google/signup
// Creates a new Google account or returns an existing one.
func (h *AuthHandler) GoogleSignup(w http.ResponseWriter, r *http.Request) {
	var req googleAuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.GoogleID == "" || req.Email == "" {
		jsonError(w, "google_id and email are required", http.StatusBadRequest)
		return
	}

	user, err := h.u.FindOrCreateGoogleUser(req.Email, req.GoogleID, req.Name)
	if err != nil {
		if errors.Is(err, users.ErrLocalAccount) {
			jsonError(w, err.Error(), http.StatusConflict)
			return
		}
		jsonError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	token, err := middleware.GenerateToken(user.ID, h.jwtSecret)
	if err != nil {
		jsonError(w, "failed to generate token", http.StatusInternalServerError)
		return
	}

	jsonResponse(w, authResponse{Token: token, User: user}, http.StatusOK)
}

// GoogleLogin handles POST /api/auth/google/login
// Only finds existing Google accounts, does not create new ones.
func (h *AuthHandler) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	var req googleAuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.GoogleID == "" || req.Email == "" {
		jsonError(w, "google_id and email are required", http.StatusBadRequest)
		return
	}

	user, err := h.u.GetByGoogleID(req.GoogleID)
	if err != nil {
		if errors.Is(err, users.ErrUserNotFound) {
			existing, emailErr := h.u.GetByEmail(req.Email)
			if emailErr == nil && existing.AuthProvider == "local" {
				jsonError(w, users.ErrLocalAccount.Error(), http.StatusConflict)
				return
			}
			jsonError(w, "no account found with this Google account, please sign up first", http.StatusNotFound)
			return
		}
		jsonError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	token, err := middleware.GenerateToken(user.ID, h.jwtSecret)
	if err != nil {
		jsonError(w, "failed to generate token", http.StatusInternalServerError)
		return
	}

	jsonResponse(w, authResponse{Token: token, User: user}, http.StatusOK)
}

func jsonResponse(w http.ResponseWriter, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
