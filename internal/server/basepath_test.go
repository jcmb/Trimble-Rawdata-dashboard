package server

import "testing"

func TestNormalizeBasePath(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"", ""},
		{"/", ""},
		{"/dashboard", "/dashboard"},
		{"dashboard/", "/dashboard"},
		{"/dashboard/", "/dashboard"},
	}
	for _, tc := range tests {
		if got := NormalizeBasePath(tc.in); got != tc.want {
			t.Fatalf("NormalizeBasePath(%q): got %q want %q", tc.in, got, tc.want)
		}
	}
}

func TestCookiePath(t *testing.T) {
	if got := cookiePath(""); got != "/" {
		t.Fatalf("root: got %q", got)
	}
	if got := cookiePath("/dash"); got != "/dash/" {
		t.Fatalf("prefixed: got %q", got)
	}
}
