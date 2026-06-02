package server

import (
	"net"
	"net/url"
	"strings"
)

// DashboardURL builds the browser URL for a listening TCP address and optional base path.
func DashboardURL(listenAddr net.Addr, basePath string) string {
	host, port, err := net.SplitHostPort(listenAddr.String())
	if err != nil {
		return "http://127.0.0.1" + basePath + "/"
	}
	host = loopbackHost(host)
	u := url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort(host, port),
		Path:   "/",
	}
	if basePath != "" {
		u.Path = basePath + "/"
	}
	return u.String()
}

func loopbackHost(host string) string {
	switch host {
	case "", "0.0.0.0", "::", "[::]":
		return "127.0.0.1"
	case "::1":
		return "[::1]"
	default:
		return host
	}
}

// NormalizeListenAddr returns host:port suitable for net.Listen.
func NormalizeListenAddr(addr string) string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return ":8080"
	}
	if strings.HasPrefix(addr, ":") {
		return addr
	}
	if !strings.Contains(addr, ":") {
		return ":" + addr
	}
	return addr
}
