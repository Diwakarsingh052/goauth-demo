package users

import "golang.org/x/crypto/bcrypt"

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
	user, err := s.GetByID(int(id))
	if err != nil {
		return nil, err
	}
	return user, nil
}

// Authenticate verifies email/password credentials and returns the user.
func (s *UserStore) Authenticate(email, password string) (*User, error) {
	user, err := s.GetByEmail(email)
	if err != nil {
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

// CreateGoogleUser finds an existing Google user or creates a new one.
func (s *UserStore) CreateGoogleUser(email, googleID, fullName string) (*User, error) {
	user, err := s.GetByGoogleID(googleID)
	if err == nil {
		return user, nil
	}

	existing, err := s.GetByEmail(email)
	if err == nil && existing.AuthProvider == "local" {
		return nil, ErrLocalAccount
	}

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
	user, err = s.GetByID(int(id))
	if err != nil {
		return nil, err
	}
	return user, nil
}
