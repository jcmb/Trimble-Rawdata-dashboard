package server

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gkirk/trimble-rawdata-dashboard/internal/connvalidate"
	"github.com/gkirk/trimble-rawdata-dashboard/internal/prefs"
	"github.com/gkirk/trimble-rawdata-dashboard/internal/sessions"
	"github.com/gkirk/trimble-rawdata-dashboard/internal/version"
)

//go:embed web/*
var webFS embed.FS

// Server serves the dashboard UI and SSE API.
type Server struct {
	Addr     string
	BasePath string
	Sessions *sessions.Manager
	ShowDev  bool
}

type configResponse struct {
	sessions.Policy
	ShowDev  bool   `json:"showDev"`
	BasePath string `json:"basePath,omitempty"`
}

// Run serves until ctx is cancelled (e.g. Ctrl-C), then shuts down gracefully.
func (s *Server) Run(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/config", s.handleConfig)
	mux.HandleFunc("/api/connect", s.handleConnect)
	mux.HandleFunc("/api/disconnect", s.handleDisconnect)
	mux.HandleFunc("/api/demo", s.handleDemo)
	mux.HandleFunc("/api/snapshot", s.handleSnapshot)
	mux.HandleFunc("/api/events", s.handleEvents)
	mux.HandleFunc("/api/version", s.handleVersion)
	mux.HandleFunc("/", s.handleWeb)

	httpSrv := &http.Server{Addr: s.Addr, Handler: s.mountHandler(mux)}

	errCh := make(chan error, 1)
	go func() {
		if s.BasePath != "" {
			slog.Info("dashboard listening", "addr", s.Addr, "basePath", s.BasePath, "version", version.String())
		} else {
			slog.Info("dashboard listening", "addr", s.Addr, "version", version.String())
		}
		err := httpSrv.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := httpSrv.Shutdown(shutdownCtx); err != nil {
			return err
		}
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(configResponse{
		Policy:   s.Sessions.Policy(),
		ShowDev:  s.ShowDev,
		BasePath: s.BasePath,
	})
}

func (s *Server) mountHandler(inner http.Handler) http.Handler {
	if s.BasePath == "" {
		return inner
	}
	outer := http.NewServeMux()
	outer.Handle(s.BasePath+"/", http.StripPrefix(s.BasePath, inner))
	outer.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, s.BasePath+"/", http.StatusFound)
	})
	return outer
}

func (s *Server) sessionCookiePath() string {
	return cookiePath(s.BasePath)
}

type connectRequest struct {
	Host string `json:"host"`
	Port int    `json:"port"`
	URI  string `json:"uri"`
}

func (s *Server) handleConnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	sid := s.Sessions.SessionID(w, r, s.sessionCookiePath())
	var req connectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	var err error
	var saveHost string
	var savePort int
	switch {
	case req.Host != "" || req.Port != 0:
		saveHost = strings.TrimSpace(req.Host)
		savePort = req.Port
		err = s.Sessions.ConnectBrowser(sid, req.Host, req.Port)
	case req.URI != "":
		if norm, normErr := connvalidate.NormalizeTCPURI(req.URI); normErr == nil {
			if u, parseErr := url.Parse(norm); parseErr == nil {
				saveHost = u.Hostname()
				if p := u.Port(); p != "" {
					savePort, _ = strconv.Atoi(p)
				}
			}
		}
		err = s.Sessions.ConnectBrowserURI(sid, req.URI)
	default:
		writeJSONError(w, http.StatusBadRequest, "provide host and port, or uri")
		return
	}
	if saveHost != "" && savePort > 0 {
		if saveErr := prefs.SaveConnection(saveHost, savePort); saveErr != nil {
			slog.Warn("save connection prefs", "err", saveErr)
		}
	}
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "connecting"})
}

func (s *Server) handleDisconnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	sid := s.Sessions.SessionID(w, r, s.sessionCookiePath())
	if err := s.Sessions.Disconnect(sid); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":   "disconnected",
		"snapshot": s.Sessions.Snapshot(sid),
	})
}

func (s *Server) handleDemo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	sid := s.Sessions.SessionID(w, r, s.sessionCookiePath())
	if err := s.Sessions.StartDemo(sid); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "demo"})
}

func writeJSONError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"number": version.Number,
		"build":  version.Build,
		"full":   version.String(),
	})
}

func (s *Server) handleSnapshot(w http.ResponseWriter, r *http.Request) {
	sid := s.Sessions.SessionID(w, r, s.sessionCookiePath())
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(s.Sessions.Snapshot(sid))
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	sid := s.Sessions.SessionID(w, r, s.sessionCookiePath())
	ch := s.Sessions.Subscribe(sid)
	defer s.Sessions.Unsubscribe(sid, ch)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	init, _ := json.Marshal(map[string]any{
		"type":     "snapshot",
		"snapshot": s.Sessions.Snapshot(sid),
	})
	fmt.Fprintf(w, "data: %s\n\n", init)
	flusher.Flush()

	tick := time.NewTicker(15 * time.Second)
	defer tick.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case data, ok := <-ch:
			if !ok {
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		case <-tick.C:
			fmt.Fprintf(w, ": keepalive\n\n")
			flusher.Flush()
		}
	}
}

func (s *Server) handleWeb(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		path = "index.html"
	}

	if path == "index.html" {
		s.serveIndex(w, r)
		return
	}

	sub, err := fs.Sub(webFS, "web")
	if err != nil {
		http.Error(w, "assets unavailable", http.StatusInternalServerError)
		return
	}

	data, err := fs.ReadFile(sub, path)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	ct := "application/octet-stream"
	switch {
	case strings.HasSuffix(path, ".css"):
		ct = "text/css; charset=utf-8"
	case strings.HasSuffix(path, ".js"):
		ct = "application/javascript; charset=utf-8"
	case strings.HasSuffix(path, ".html"):
		ct = "text/html; charset=utf-8"
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Cache-Control", "no-cache, must-revalidate")
	w.Write(data)
}

func (s *Server) serveIndex(w http.ResponseWriter, r *http.Request) {
	sub, err := fs.Sub(webFS, "web")
	if err != nil {
		http.Error(w, "index unavailable", http.StatusInternalServerError)
		return
	}
	data, err := fs.ReadFile(sub, "index.html")
	if err != nil {
		http.Error(w, "index unavailable", http.StatusInternalServerError)
		return
	}
	tag := version.AssetTag()
	html := string(data)
	pathPrefix := "/"
	if s.BasePath != "" {
		pathPrefix = s.BasePath + "/"
	}
	html = strings.ReplaceAll(html, "__VERSION__", tag)
	html = strings.ReplaceAll(html, "__VERSION_NUMBER__", version.Number)
	html = strings.ReplaceAll(html, "__VERSION_FULL__", version.String())
	html = strings.ReplaceAll(html, "__BASE_PATH__", s.BasePath)
	html = strings.ReplaceAll(html, "__PATH_PREFIX__", pathPrefix)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, must-revalidate")
	_, _ = w.Write([]byte(html))
}
