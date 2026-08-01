package config

import (
	"testing"
)

func TestValidateServerURL(t *testing.T) {
	tests := []struct {
		name    string
		server  string
		wantErr bool
	}{
		{"valid http", "http://localhost:8888", false},
		{"valid https", "https://example.com", false},
		{"valid with path", "http://localhost:8888/api", false},
		{"missing scheme", "localhost:8888", true},
		{"ftp scheme", "ftp://localhost:8888", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateServerURL(tt.server)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateServerURL(%q) error = %v, wantErr %v", tt.server, err, tt.wantErr)
			}
		})
	}
}
