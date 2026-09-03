package storage

import "testing"

func TestNormalizeEndpoint(t *testing.T) {
	cases := []struct {
		in     string
		expect string
	}{
		{"http://127.0.0.1:9000", "http://127.0.0.1:9000"},
		{"https://s3.example.com", "https://s3.example.com"},
		{"127.0.0.1:9000", "https://127.0.0.1:9000"},
		{"s3.example.com", "https://s3.example.com"},
		{"", ""},
	}

	for _, c := range cases {
		if got := normalizeEndpoint(c.in); got != c.expect {
			t.Fatalf("normalizeEndpoint(%q) expect %q, got %q", c.in, c.expect, got)
		}
	}
}
