package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestNewTokenService(t *testing.T) {
	ts := NewTokenService("my-secret")
	if ts == nil {
		t.Fatal("expected non-nil TokenService")
	}
	if string(ts.secret) != "my-secret" {
		t.Fatalf("expected secret %q, got %q", "my-secret", string(ts.secret))
	}
}

func TestGenerateToken(t *testing.T) {
	tests := []struct {
		name    string
		secret  string
		userID  int
		wantErr bool
	}{
		{
			name:   "valid user ID",
			secret: "test-secret",
			userID: 1,
		},
		{
			name:   "zero user ID",
			secret: "test-secret",
			userID: 0,
		},
		{
			name:   "large user ID",
			secret: "test-secret",
			userID: 999999,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := NewTokenService(tt.secret)
			token, err := ts.GenerateToken(tt.userID)

			if tt.wantErr {
				if err == nil {
					t.Fatal("GenerateToken() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("GenerateToken() unexpected error: %v", err)
			}
			if token == "" {
				t.Fatal("expected non-empty token")
			}
		})
	}
}

func TestValidateToken(t *testing.T) {
	secret := "test-secret"
	ts := NewTokenService(secret)

	validToken, err := ts.GenerateToken(42)
	if err != nil {
		t.Fatalf("setup: GenerateToken failed: %v", err)
	}

	// Build an expired token for the expired-token test case.
	expiredClaims := Claims{
		UserID: 42,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	expiredToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("setup: failed to create expired token: %v", err)
	}

	// Token signed with a different secret.
	wrongSecretTS := NewTokenService("wrong-secret")
	wrongSecretToken, err := wrongSecretTS.GenerateToken(42)
	if err != nil {
		t.Fatalf("setup: GenerateToken with wrong secret failed: %v", err)
	}

	tests := []struct {
		name       string
		token      string
		wantUserID int
		wantErr    bool
	}{
		{
			name:       "valid token",
			token:      validToken,
			wantUserID: 42,
		},
		{
			name:    "expired token",
			token:   expiredToken,
			wantErr: true,
		},
		{
			name:    "wrong secret",
			token:   wrongSecretToken,
			wantErr: true,
		},
		{
			name:    "malformed token",
			token:   "not-a-jwt-token",
			wantErr: true,
		},
		{
			name:    "empty token",
			token:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := ts.ValidateToken(tt.token)

			if tt.wantErr {
				if err == nil {
					t.Fatal("ValidateToken() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("ValidateToken() unexpected error: %v", err)
			}
			if claims.UserID != tt.wantUserID {
				t.Errorf("UserID = %d, want %d", claims.UserID, tt.wantUserID)
			}
		})
	}
}
