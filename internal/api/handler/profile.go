package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"challange-go-cyaz/internal/api/middleware"
	"challange-go-cyaz/internal/users"
)

// ProfileHandler handles profile-related API requests.
type ProfileHandler struct {
	Users *users.UserStore
}

// NewProfileHandler creates a new ProfileHandler.
func NewProfileHandler(users *users.UserStore) *ProfileHandler {
	return &ProfileHandler{Users: users}
}

type updateProfileRequest struct {
	FullName  string `json:"full_name"`
	Telephone string `json:"telephone"`
	Email     string `json:"email"`
}

// GetProfile handles GET /api/profile
func (h *ProfileHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)

	user, err := h.Users.GetByID(userID)
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

// validateUSPhone validates and formats a US phone number.
// It accepts any format (digits, parentheses, dashes, spaces) and returns
// a formatted (XXX) XXX-XXXX string. An empty input is allowed (field is optional).
func validateUSPhone(phone string) (string, error) {
	if phone == "" {
		return "", nil
	}

	var digits []byte
	for i := 0; i < len(phone); i++ {
		if phone[i] >= '0' && phone[i] <= '9' {
			digits = append(digits, phone[i])
		}
	}

	// Strip leading country code 1
	if len(digits) == 11 && digits[0] == '1' {
		digits = digits[1:]
	}

	if len(digits) != 10 {
		jsonErr := "telephone must be a valid 10-digit US phone number"
		return "", errors.New(jsonErr)
	}

	// NANP: area code cannot start with 0 or 1
	if digits[0] == '0' || digits[0] == '1' {
		return "", errors.New("invalid US area code")
	}

	return "(" + string(digits[0:3]) + ") " + string(digits[3:6]) + "-" + string(digits[6:10]), nil
}

// UpdateProfile handles PUT /api/profile
func (h *ProfileHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)

	var req updateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	telephone, err := validateUSPhone(req.Telephone)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Fetch current user to enforce Google email restriction
	currentUser, err := h.Users.GetByID(userID)
	if err != nil {
		jsonError(w, "user not found", http.StatusNotFound)
		return
	}

	email := req.Email
	if currentUser.AuthProvider == "google" {
		email = currentUser.Email // Google users cannot change their email
	}

	if err := h.Users.UpdateProfile(userID, req.FullName, telephone, email); err != nil {
		if errors.Is(err, users.ErrDuplicateEmail) {
			jsonError(w, "email already in use by another account", http.StatusConflict)
			return
		}
		jsonError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	user, err := h.Users.GetByID(userID)
	if err != nil {
		jsonError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	jsonResponse(w, map[string]*users.User{"user": user}, http.StatusOK)
}
