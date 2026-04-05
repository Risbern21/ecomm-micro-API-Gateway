package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		expected string
	}{
		{
			name:     "returns hashed password",
			password: "hello_john_doe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := HashPassword(tt.password)
			require.NoError(t, err, "should not be any hashing errors")

			assert.NotEmpty(t, actual, "check if hash was computed")
		})
	}
}

func TestCheckPassword(t *testing.T) {
	tests := []struct {
		name           string
		password       string
		hashedPassword func(password string) string
		validate       func(string, string)
	}{
		{
			name:     "invalid check returns error",
			password: "hello_john_doe",
			hashedPassword: func(string) string {
				return "$2a$10$CLcTsbLw0Zjk2Ja.1UYC0OGwMfU4nha47tdJ3hnLq4dMhu7wT7GFw"
			},
			validate: func(p string, h string) {
				err := CheckPassword(p, h)
				assert.Error(t, err, "error must be nil when password and hash are equal")
			},
		},
		{
			name:     "valid check returns no error",
			password: "hello_john_doe",
			hashedPassword: func(p string) string {
				hash, err := HashPassword(p)
				require.NoError(t, err, "must return no error")

				return hash
			},
			validate: func(p string, h string) {
				err := CheckPassword(p, h)
				assert.NoError(t, err, "error must be nil when password and hash are equal")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.validate(tt.password, tt.hashedPassword(tt.password))
		})
	}
}
