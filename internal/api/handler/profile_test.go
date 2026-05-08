package handler

import (
	"testing"
)

func TestValidatePhone(t *testing.T) {
	tests := []struct {
		name    string
		phone   string
		want    string
		wantErr bool
	}{
		{
			name:  "valid 10-digit number",
			phone: "1234567890",
			want:  "1234567890",
		},
		{
			name:  "empty string is allowed",
			phone: "",
			want:  "",
		},
		{
			name:    "too short",
			phone:   "12345",
			wantErr: true,
		},
		{
			name:    "too long",
			phone:   "12345678901",
			wantErr: true,
		},
		{
			name:    "contains letters",
			phone:   "12345abcde",
			wantErr: true,
		},
		{
			name:    "contains special characters",
			phone:   "123-456-78",
			wantErr: true,
		},
		{
			name:    "contains spaces",
			phone:   "123 456 78",
			wantErr: true,
		},

		{
			name:    "single character",
			phone:   "5",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validatePhone(tt.phone)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("validatePhone(%q) expected error, got nil", tt.phone)
				}
				return
			}

			if err != nil {
				t.Fatalf("validatePhone(%q) unexpected error: %v", tt.phone, err)
			}
			if got != tt.want {
				t.Errorf("validatePhone(%q) = %q, want %q", tt.phone, got, tt.want)
			}
		})
	}
}
