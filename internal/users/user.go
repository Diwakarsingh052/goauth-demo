package users

import (
	"database/sql"
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserNotFound    = errors.New("username entered does not exist")
	ErrDuplicateEmail  = errors.New("email already exists")
	ErrInvalidPassword = errors.New("password is incorrect")
	ErrGoogleAccount   = errors.New("this account uses Google sign-in, please use the Google button")
	ErrLocalAccount    = errors.New("this email is registered with email/password, please use the login form")
)

// User represents a user in the system.

// UserStore provides methods for user CRUD operations.
type UserStore struct {
	db *sql.DB
}

// NewUserStore creates a new UserStore.
func NewUserStore(db *sql.DB) *UserStore {
	return &UserStore{db: db}
}

// Create registers a new user with email and password (local auth).
func (s *UserStore) Create(email, password string) (*User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	result, err := s.db.Exec(
		"INSERT INTO users (email, password_hash, auth_provider) VALUES (?, ?, 'local')",
		email, string(hash),
	)
	if err != nil {
		if isDuplicateEntry(err) {
			return nil, ErrDuplicateEmail
		}
		return nil, err
	}

	id, _ := result.LastInsertId()
	return s.GetByID(int(id))
}

// CreateWithGoogle registers a new user via Google OAuth.
func (s *UserStore) CreateWithGoogle(email, googleID, fullName string) (*User, error) {
	result, err := s.db.Exec(
		"INSERT INTO users (email, google_id, auth_provider, full_name) VALUES (?, ?, 'google', ?)",
		email, googleID, fullName,
	)
	if err != nil {
		if isDuplicateEntry(err) {
			return nil, ErrDuplicateEmail
		}
		return nil, err
	}

	id, _ := result.LastInsertId()
	return s.GetByID(int(id))
}

// Authenticate verifies email/password credentials and returns the user.
func (s *UserStore) Authenticate(email, password string) (*User, error) {
	user, err := s.GetByEmail(email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	if user.AuthProvider == "google" {
		return nil, ErrGoogleAccount
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return nil, ErrInvalidPassword
	}

	return user, nil
}

// FindOrCreateGoogleUser finds an existing Google user or creates a new one.
// Returns the user and a boolean indicating if a new user was created.
func (s *UserStore) FindOrCreateGoogleUser(email, googleID, fullName string) (*User, bool, error) {
	user, err := s.GetByGoogleID(googleID)
	if err == nil {
		return user, false, nil
	}

	// Check if email is already used with local auth
	existing, err := s.GetByEmail(email)
	if err == nil && existing.AuthProvider == "local" {
		return nil, false, ErrLocalAccount
	}

	user, err = s.CreateWithGoogle(email, googleID, fullName)
	if err != nil {
		return nil, false, err
	}
	return user, true, nil
}

// GetByID retrieves a user by their ID.
func (s *UserStore) GetByID(id int) (*User, error) {
	user := &User{}
	err := s.db.QueryRow(
		`SELECT id, email, password_hash, google_id, auth_provider,
		        full_name, telephone, created_at, updated_at
		 FROM users WHERE id = ?`, id,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.GoogleID,
		&user.AuthProvider, &user.FullName, &user.Telephone,
		&user.CreatedAt, &user.UpdatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	return user, err
}

// GetByEmail retrieves a user by their email address.
func (s *UserStore) GetByEmail(email string) (*User, error) {
	user := &User{}
	err := s.db.QueryRow(
		`SELECT id, email, password_hash, google_id, auth_provider,
		        full_name, telephone, created_at, updated_at
		 FROM users WHERE email = ?`, email,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.GoogleID,
		&user.AuthProvider, &user.FullName, &user.Telephone,
		&user.CreatedAt, &user.UpdatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	return user, err
}

// GetByGoogleID retrieves a user by their Google ID.
func (s *UserStore) GetByGoogleID(googleID string) (*User, error) {
	user := &User{}
	err := s.db.QueryRow(
		`SELECT id, email, password_hash, google_id, auth_provider,
		        full_name, telephone, created_at, updated_at
		 FROM users WHERE google_id = ?`, googleID,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.GoogleID,
		&user.AuthProvider, &user.FullName, &user.Telephone,
		&user.CreatedAt, &user.UpdatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	return user, err
}

// UpdateProfile updates the user's profile fields.
// For Google users, email cannot be changed.
func (s *UserStore) UpdateProfile(id int, fullName, telephone, email string) error {
	_, err := s.db.Exec(
		"UPDATE users SET full_name = ?, telephone = ?, email = ?, updated_at = NOW() WHERE id = ?",
		fullName, telephone, email, id,
	)
	if err != nil && isDuplicateEntry(err) {
		return ErrDuplicateEmail
	}
	return err
}

func isDuplicateEntry(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "Duplicate entry") || strings.Contains(err.Error(), "1062"))
}
