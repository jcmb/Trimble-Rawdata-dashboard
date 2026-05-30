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
	"time"

	"github.com/gkirk/trimble-rawdata-dashboard/internal/hub"
	"github.com/gkirk/trimble-rawdata-dashboard/internal/store"
)

//go:embed web/*
var webFS embed.FS

// Server serves the dashboard UI and SSE API.
type Server struct {
	Addr  string
	Store *store.Store
	Hub   *hub.Hub
}

// Run serves until ctx is cancelled (e.g. Ctrl-C), then shuts down gracefully.
func (s *Server) Run(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/snapshot", s.handleSnapshot)
	mux.HandleFunc("/api/events", s.handleEvents)
	mux.Handle("/", s.handleStatic())

	httpSrv := &http.Server{Addr: s.Addr, Handler: mux}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("dashboard listening", "addr", s.Addr)
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

func (s *Server) handleSnapshot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(s.Store.Snapshot())
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := s.Hub.Subscribe()
	defer s.Hub.Unsubscribe(ch)

	init, _ := json.Marshal(map[string]any{
		"type":     "snapshot",
		"snapshot": s.Store.Snapshot(),
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

func (s *Server) handleStatic() http.Handler {
	sub, err := fs.Sub(webFS, "web")
	if err != nil {
		panic(err)
	}
	return http.FileServer(http.FS(sub))
}
