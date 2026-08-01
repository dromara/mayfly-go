package terminal

import (
	"testing"
)

func TestDisplayWidth(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"ascii", "hello", 5},
		{"chinese", "你好", 4},
		{"mixed", "hello你好", 9},
		{"empty", "", 0},
		{"japanese", "こんにちは", 10},
		{"korean", "안녕하세요", 10},
		{"numbers", "12345", 5},
		{"special", "!@#$%", 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := displayWidth(tt.input); got != tt.want {
				t.Errorf("displayWidth(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestStatusText(t *testing.T) {
	online := StatusText(true)
	offline := StatusText(false)

	if online == "" {
		t.Error("StatusText(true) should not be empty")
	}
	if offline == "" {
		t.Error("StatusText(false) should not be empty")
	}
	if online == offline {
		t.Error("StatusText(true) and StatusText(false) should be different")
	}
}

func TestColorize(t *testing.T) {
	result := Colorize("test", Red)
	if result == "" {
		t.Error("Colorize should not return empty string")
	}
	if result == "test" {
		t.Error("Colorize should add color codes")
	}
}

func TestBoldText(t *testing.T) {
	result := BoldText("test")
	if result == "" {
		t.Error("BoldText should not return empty string")
	}
	if result == "test" {
		t.Error("BoldText should add bold codes")
	}
}
