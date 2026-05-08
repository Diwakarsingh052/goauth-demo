package users

import (
	"database/sql"
	"errors"
)

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