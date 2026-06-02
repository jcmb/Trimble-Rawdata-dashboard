package server

import (
	"net"
	"strings"
	"testing"
)

func TestDashboardURL(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	got := DashboardURL(ln.Addr(), "")
	if !strings.HasPrefix(got, "http://127.0.0.1:") || !strings.HasSuffix(got, "/") {
		t.Fatalf("root: got %q", got)
	}

	got = DashboardURL(ln.Addr(), "/trimble-dashboard")
	if !strings.HasSuffix(got, "/trimble-dashboard/") {
		t.Fatalf("base path: got %q", got)
	}
}

func TestLoopbackHost(t *testing.T) {
	if got := loopbackHost("0.0.0.0"); got != "127.0.0.1" {
		t.Fatalf("0.0.0.0: got %q", got)
	}
	if got := loopbackHost(""); got != "127.0.0.1" {
		t.Fatalf("empty: got %q", got)
	}
}
