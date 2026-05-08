package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"challenge-go-cyaz/internal/auth"
)

func TestGetUserID(t *testing.T) {
	tests := []struct {
		name     string
		ctxVal   interface{}
		setCtx   bool
		wantID   int
	}{
		{
			name:   "valid user ID in context",
			ctxVal: 42,
			setCtx: true,
			wantID: 42,
		},
		{
			name:   "zero user ID",
			ctxVal: 0,
			setCtx: true,
			wantID: 0,
		},
		{
			name:   "no user ID in context",
			setCtx: false,
			wantID: 0,
		},
		{
			name:   "wrong type in context",
			ctxVal: "not-an-int",
			setCtx: true,
			wantID: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.setCtx {
				ctx := context.WithValue(r.Context(), UserIDKey, tt.ctxVal)
				r = r.WithContext(ctx)
			}

			got := GetUserID(r)
			if got != tt.wantID {
				t.Errorf("GetUserID() = %d, want %d", got, tt.wantID)
			}
		})
	}
}

func TestJWTAuth(t *testing.T) {
	ts := auth.NewTokenService("test-secret")
	validToken, err := ts.GenerateToken(99)
	if err != nil {
		t.Fatalf("setup: GenerateToken failed: %v", err)
	}

	otherTS := auth.NewTokenService("other-secret")
	wrongToken, err := otherTS.GenerateToken(99)
	if err != nil {
		t.Fatalf("setup: GenerateToken (wrong secret) failed: %v", err)
	}

	tests := []struct {
		name           string
		authHeader     string
		wantStatus     int
		wantUserIDSet  bool
		wantUserID     int
	}{
		{
			name:          "valid bearer token",
			authHeader:    "Bearer " + validToken,
			wantStatus:    200,
			wantUserIDSet: true,
			wantUserID:    99,
		},
		{
			name:       "missing authorization header",
			authHeader: "",
			wantStatus: 401,
		},
		{
			name:       "no Bearer prefix",
			authHeader: validToken,
			wantStatus: 401,
		},
		{
			name:       "wrong prefix",
			authHeader: "Token " + validToken,
			wantStatus: 401,
		},
		{
			name:       "invalid token",
			authHeader: "Bearer invalid-token",
			wantStatus: 401,
		},
		{
			name:       "token signed with wrong secret",
			authHeader: "Bearer " + wrongToken,
			wantStatus: 401,
		},
		{
			name:       "Bearer with empty token",
			authHeader: "Bearer ",
			wantStatus: 401,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedUserID int
			var userIDFound bool

			handler := JWTAuth(ts)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedUserID = GetUserID(r)
				userIDFound = true
				w.WriteHeader(http.StatusOK)
			}))

			r := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if tt.authHeader != "" {
				r.Header.Set("Authorization", tt.authHeader)
			}
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}

			if tt.wantUserIDSet {
				if !userIDFound {
					t.Fatal("expected next handler to be called")
				}
				if capturedUserID != tt.wantUserID {
					t.Errorf("userID = %d, want %d", capturedUserID, tt.wantUserID)
				}
			} else if userIDFound {
				t.Error("next handler should not have been called")
			}
		})
	}
}
