package users

import (
	"database/sql"
	"errors"
	"strings"
	"time"
)

var (
	ErrUserNotFound    = errors.New("username entered does not exist")
	ErrDuplicateEmail  = errors.New("email already exists")
	ErrInvalidPassword = errors.New("password is incorrect")
	ErrGoogleAccount   = errors.New("this account uses Google sign-in, please use the Google button")
	ErrLocalAccount    = errors.New("this email is registered with email/password, please use the login form")
)

// User represents a user in the system.
type User struct {
	ID           int       `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	GoogleID     string    `json:"google_id,omitempty"`
	AuthProvider string    `json:"auth_provider"`
	FullName     string    `json:"full_name"`
	Telephone    string    `json:"telephone"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// UserStore provides methods for user operations.
type UserStore struct {
	db *sql.DB
}

// NewUserStore creates a new UserStore.
func NewUserStore(db *sql.DB) *UserStore {
	return &UserStore{db: db}
}

func isDuplicateEntry(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "Duplicate entry") || strings.Contains(err.Error(), "1062"))
}
