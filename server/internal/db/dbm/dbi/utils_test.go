package dbi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQuoteEscape(t *testing.T) {
	kases := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"abc", "abc"},
		{"it's", "it''s"},
		{"a'b'c", "a''b''c"},
		{"''", "''''"},
		{`a"b`, `a"b`}, // 双引号不处理
		{"'; DROP TABLE users; --", "''; DROP TABLE users; --"},
	}

	for _, k := range kases {
		assert.Equal(t, k.expected, QuoteEscape(k.input))
	}
}
