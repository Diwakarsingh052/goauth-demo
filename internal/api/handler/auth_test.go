package handler

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestJsonResponse(t *testing.T) {
	tests := []struct {
		name       string
		data       interface{}
		status     int
		wantStatus int
	}{
		{
			name:       "200 with map",
			data:       map[string]string{"message": "ok"},
			status:     200,
			wantStatus: 200,
		},
		{
			name:       "201 created",
			data:       map[string]int{"id": 1},
			status:     201,
			wantStatus: 201,
		},
		{
			name:       "empty body",
			data:       map[string]string{},
			status:     200,
			wantStatus: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			jsonResponse(w, tt.data, tt.status)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if ct := w.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type = %q, want %q", ct, "application/json")
			}

			var decoded map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&decoded); err != nil {
				t.Fatalf("failed to decode response body: %v", err)
			}
		})
	}
}

func TestJsonError(t *testing.T) {
	tests := []struct {
		name       string
		message    string
		status     int
		wantStatus int
	}{
		{
			name:       "bad request",
			message:    "invalid input",
			status:     400,
			wantStatus: 400,
		},
		{
			name:       "unauthorized",
			message:    "not logged in",
			status:     401,
			wantStatus: 401,
		},
		{
			name:       "internal server error",
			message:    "something went wrong",
			status:     500,
			wantStatus: 500,
		},
		{
			name:       "not found",
			message:    "resource not found",
			status:     404,
			wantStatus: 404,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			jsonError(w, tt.message, tt.status)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if ct := w.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type = %q, want %q", ct, "application/json")
			}

			var body map[string]string
			if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}
			if body["error"] != tt.message {
				t.Errorf("error = %q, want %q", body["error"], tt.message)
			}
		})
	}
}
