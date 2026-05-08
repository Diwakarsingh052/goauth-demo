package users

import (
	"errors"
	"testing"
)

func TestIsDuplicateEntry(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "contains Duplicate entry",
			err:  errors.New("Error 1062: Duplicate entry 'foo@bar.com' for key 'email'"),
			want: true,
		},
		{
			name: "contains 1062 code",
			err:  errors.New("Error 1062 (23000)"),
			want: true,
		},
		{
			name: "unrelated error",
			err:  errors.New("connection refused"),
			want: false,
		},
		{
			name: "empty error message",
			err:  errors.New(""),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isDuplicateEntry(tt.err)
			if got != tt.want {
				t.Errorf("isDuplicateEntry(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
