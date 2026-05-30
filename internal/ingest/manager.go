package ingest

import (
	"context"
	"fmt"
	"sync"

	"github.com/gkirk/trimble-rawdata-dashboard/internal/connvalidate"
	"github.com/gkirk/trimble-rawdata-dashboard/internal/hub"
	"github.com/gkirk/trimble-rawdata-dashboard/internal/prefs"
	"github.com/gkirk/trimble-rawdata-dashboard/internal/store"
	"github.com/gkirk/trimble-rawdata-dashboard/internal/verbose"
)

// ManagerConfig configures optional hosted-mode ingest.
type ManagerConfig struct {
	AllowLocalHosts bool
	Verbose         *verbose.Logger
	VerboseLevel    verbose.Level
}

// Manager starts and stops receiver ingest (fixed at launch or from the web UI).
type Manager struct {
	rootCtx context.Context
	mu      sync.Mutex
	cancel  context.CancelFunc
	mode    string // "", "demo", "port"
	port    string
	fixed   bool
	cfg     ManagerConfig
	st      *store.Store
	h       *hub.Hub
}

// NewManager creates an ingest manager. Call one of the Start* methods or use the HTTP API.
func NewManager(rootCtx context.Context, st *store.Store, h *hub.Hub, cfg ManagerConfig) *Manager {
	return &Manager{
		rootCtx: rootCtx,
		cfg:     cfg,
		st:      st,
		h:       h,
	}
}

// Config describes how clients may connect.
type Config struct {
	Hosted          bool   `json:"hosted"`
	FixedConnection bool   `json:"fixedConnection"`
	AllowLocalHosts bool   `json:"allowLocalHosts"`
	Demo            bool   `json:"demo"`
	Port            string `json:"port,omitempty"`
	LastHost        string `json:"lastHost,omitempty"`
	LastPort        int    `json:"lastPort,omitempty"`
}

// Config returns current connection policy and state.
func (m *Manager) Config() Config {
	m.mu.Lock()
	defer m.mu.Unlock()
	cfg := Config{
		Hosted:          !m.fixed,
		FixedConnection: m.fixed,
		AllowLocalHosts: m.cfg.AllowLocalHosts,
		Demo:            m.mode == "demo",
		Port:            m.port,
	}
	if saved, err := prefs.LoadConnection(); err == nil {
		cfg.LastHost = saved.Host
		cfg.LastPort = saved.Port
	}
	return cfg
}

// StartFixedPort connects at process start (-port). Skips browser local-host checks.
func (m *Manager) StartFixedPort(port string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if port == "" {
		return fmt.Errorf("port is required")
	}
	m.fixed = true
	m.stopLocked()
	m.startPortLocked(port)
	return nil
}

// StartFixedDemo runs demo ingest at process start (-demo).
func (m *Manager) StartFixedDemo() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.fixed = true
	m.stopLocked()
	m.startDemoLocked()
}

// ConnectBrowser validates and connects to a TCP receiver from the web UI.
func (m *Manager) ConnectBrowser(host string, port int) error {
	uri, err := connvalidate.BuildTCPURI(host, port)
	if err != nil {
		return err
	}
	uri, err = connvalidate.ValidateBrowserTCP(uri, m.cfg.AllowLocalHosts)
	if err != nil {
		return err
	}
	return m.connectURI(uri)
}

// ConnectBrowserURI connects using a tcp:// or host:port string from the web UI.
func (m *Manager) ConnectBrowserURI(raw string) error {
	uri, err := connvalidate.ValidateBrowserTCP(raw, m.cfg.AllowLocalHosts)
	if err != nil {
		return err
	}
	return m.connectURI(uri)
}

// StartDemo switches to synthetic data (hosted mode only).
func (m *Manager) StartDemo() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.fixed {
		return fmt.Errorf("demo mode is fixed at server start")
	}
	m.stopLocked()
	m.startDemoLocked()
	return nil
}

// Disconnect stops ingest (hosted mode only).
func (m *Manager) Disconnect() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.fixed {
		return fmt.Errorf("connection is fixed at server start")
	}
	m.stopLocked()
	m.mode = ""
	m.port = ""
	m.st.SetPort("")
	snap := m.st.SetConnected(false)
	snap.LastError = ""
	m.h.PublishSnapshot(snap)
	return nil
}

func (m *Manager) connectURI(uri string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.fixed {
		return fmt.Errorf("connection is fixed at server start")
	}
	m.stopLocked()
	m.startPortLocked(uri)
	return nil
}

func (m *Manager) stopLocked() {
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
}

func (m *Manager) startPortLocked(uri string) {
	ctx, cancel := context.WithCancel(m.rootCtx)
	m.cancel = cancel
	m.mode = "port"
	m.port = uri
	m.st.SetPort(uri)
	opts := Options{Port: uri}
	if m.cfg.Verbose != nil {
		opts.Verbose = m.cfg.Verbose
	} else if m.cfg.VerboseLevel >= verbose.Info {
		opts.Verbose = verbose.New(m.cfg.VerboseLevel)
	}
	go Run(ctx, opts, m.st, m.h)
}

func (m *Manager) startDemoLocked() {
	ctx, cancel := context.WithCancel(m.rootCtx)
	m.cancel = cancel
	m.mode = "demo"
	m.port = ""
	m.st.SetPort("")
	go RunDemo(ctx, m.st, m.h)
}
