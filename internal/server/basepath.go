package server

import "strings"

// NormalizeBasePath returns a URL prefix without trailing slash (empty = site root).
func NormalizeBasePath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" || p == "/" {
		return ""
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return strings.TrimSuffix(p, "/")
}

func cookiePath(basePath string) string {
	if basePath == "" {
		return "/"
	}
	return basePath + "/"
}
