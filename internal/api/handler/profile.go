package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"challenge-go-cyaz/internal/api/middleware"
	"challenge-go-cyaz/internal/users"
)

// ProfileHandler handles profile-related API requests.
type ProfileHandler struct {
	usr *users.UserStore
}

// NewProfileHandler creates a new ProfileHandler.
func NewProfileHandler(users *users.UserStore) *ProfileHandler {
	return &ProfileHandler{usr: users}
}

type updateProfileRequest struct {
	FullName  string `json:"full_name"`
	Telephone string `json:"telephone"`
	Email     string `json:"email"`
}

// GetProfile handles GET /api/profile
func (h *ProfileHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)

	user, err := h.usr.GetByID(userID)
	if err != nil {
		if errors.Is(err, users.ErrUserNotFound) {
			jsonError(w, "user not found", http.StatusNotFound)
			return
		}
		jsonError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	jsonResponse(w, map[string]*users.User{"user": user}, http.StatusOK)
}

func validatePhone(phone string) (string, error) {
	if phone == "" {
		return "", nil
	}

	if len(phone) != 10 {
		return "", errors.New("phone number must be exactly 10 digits")
	}
	for i := 0; i < len(phone); i++ {
		if phone[i] < '0' || phone[i] > '9' {
			return "", errors.New("phone number must be exactly 10 digits")
		}
	}

	return phone, nil
}

// UpdateProfile handles PUT /api/profile
func (h *ProfileHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)

	var req updateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	telephone, err := validatePhone(req.Telephone)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Fetch current user to enforce Google email restriction
	currentUser, err := h.usr.GetByID(userID)
	if err != nil {
		jsonError(w, "user not found", http.StatusNotFound)
		return
	}

	email := req.Email
	if currentUser.AuthProvider == "google" {
		email = currentUser.Email // Google users cannot change their email
	}

	if err := h.usr.UpdateProfile(userID, req.FullName, telephone, email); err != nil {
		if errors.Is(err, users.ErrDuplicateEmail) {
			jsonError(w, "email already in use by another account", http.StatusConflict)
			return
		}
		jsonError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	user, err := h.usr.GetByID(userID)
	if err != nil {
		jsonError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	jsonResponse(w, map[string]*users.User{"user": user}, http.StatusOK)
}
