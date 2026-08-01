package shared

import (
	"bufio"
	"strings"
	"testing"
)

func TestReadLine(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		want   string
		wantOk bool
	}{
		{"normal input", "hello\n", "hello", true},
		{"with spaces", "  hello world  \n", "hello world", true},
		{"empty line", "\n", "", true},
		{"no newline", "hello", "hello", true},
		{"empty input", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bufio.NewReader(strings.NewReader(tt.input))
			got, ok := ReadLine(reader)
			if got != tt.want {
				t.Errorf("ReadLine() = %q, want %q", got, tt.want)
			}
			if ok != tt.wantOk {
				t.Errorf("ReadLine() ok = %v, want %v", ok, tt.wantOk)
			}
		})
	}
}

func TestFormatAddress(t *testing.T) {
	tests := []struct {
		name string
		host interface{}
		port interface{}
		want string
	}{
		{"normal", "localhost", 8080, "localhost:8080"},
		{"nil host", nil, 3306, ":3306"},
		{"ip address", "192.168.1.1", 22, "192.168.1.1:22"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatAddress(tt.host, tt.port)
			if got != tt.want {
				t.Errorf("FormatAddress() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatId(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  string
	}{
		{"uint64", uint64(123), "123"},
		{"float64", float64(456), "456"},
		{"int", 789, "789"},
		{"string", "100", "100"},
		{"zero", 0, "0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatId(tt.input)
			if got != tt.want {
				t.Errorf("FormatId() = %q, want %q", got, tt.want)
			}
		})
	}
}
