// Package sessions isolates browser clients and reference-counts shared receiver links.
package sessions

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/gkirk/trimble-rawdata-dashboard/internal/connvalidate"
	"github.com/gkirk/trimble-rawdata-dashboard/internal/hub"
	"github.com/gkirk/trimble-rawdata-dashboard/internal/ingest"
	"github.com/gkirk/trimble-rawdata-dashboard/internal/model"
	"github.com/gkirk/trimble-rawdata-dashboard/internal/prefs"
	"github.com/gkirk/trimble-rawdata-dashboard/internal/store"
	"github.com/gkirk/trimble-rawdata-dashboard/internal/verbose"
)

const (
	sessionCookie  = "dashsid"
	sessionHeader  = "X-Dashboard-Session"
	sessionQuery   = "sid"
	demoStreamKey  = "demo"
)

// Config configures multi-session ingest.
type Config struct {
	AllowLocalHosts bool
	Verbose         *verbose.Logger
	VerboseLevel    verbose.Level
}

// Policy describes how clients may connect (global server policy).
type Policy struct {
	Hosted          bool   `json:"hosted"`
	FixedConnection bool   `json:"fixedConnection"`
	AllowLocalHosts bool   `json:"allowLocalHosts"`
	Demo            bool   `json:"demo"`
	Port            string `json:"port,omitempty"`
	LastHost        string `json:"lastHost,omitempty"`
	LastPort        int    `json:"lastPort,omitempty"`
}

// Manager tracks browser sessions and shared receiver streams (one link per host:port).
type Manager struct {
	rootCtx context.Context
	cfg     Config
	mu      sync.Mutex
	fixed   bool
	fixedKey string
	sessions map[string]*clientSession
	streams  map[string]*sharedStream
	idle     *store.Store
}

type clientSession struct {
	id        string
	streamKey string
}

type sharedStream struct {
	key      string
	uri      string
	demo     bool
	store    *store.Store
	hub      *hub.Hub
	sessions map[string]struct{}
	cancel   context.CancelFunc
}

// NewManager creates a session manager for hosted or fixed-startup mode.
func NewManager(rootCtx context.Context, cfg Config) *Manager {
	return &Manager{
		rootCtx:  rootCtx,
		cfg:      cfg,
		sessions: make(map[string]*clientSession),
		streams:  make(map[string]*sharedStream),
		idle:     store.New(""),
	}
}

// Policy returns connection policy and the fixed stream state if any.
func (m *Manager) Policy() Policy {
	m.mu.Lock()
	defer m.mu.Unlock()
	p := Policy{
		Hosted:          !m.fixed,
		FixedConnection: m.fixed,
		AllowLocalHosts: m.cfg.AllowLocalHosts,
	}
	if m.fixed && m.fixedKey != "" {
		if st := m.streams[m.fixedKey]; st != nil {
			p.Demo = st.demo
			if !st.demo {
				p.Port = st.uri
			}
		}
	}
	if saved, err := prefs.LoadConnection(); err == nil {
		p.LastHost = saved.Host
		p.LastPort = saved.Port
	}
	return p
}

// StartFixedPort connects at process start (-port).
func (m *Manager) StartFixedPort(port string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if port == "" {
		return fmt.Errorf("port is required")
	}
	m.fixed = true
	m.fixedKey = port
	return m.startStreamLocked(port, port, false)
}

// StartFixedDemo runs demo ingest at process start (-demo).
func (m *Manager) StartFixedDemo() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.fixed = true
	m.fixedKey = demoStreamKey
	_ = m.startStreamLocked(demoStreamKey, "", true)
}

// ResolveSessionID returns the browser tab session (header/query preferred over shared cookie).
func (m *Manager) ResolveSessionID(r *http.Request) string {
	if id := strings.TrimSpace(r.Header.Get(sessionHeader)); validSessionID(id) {
		m.registerSessionLocked(id)
		return id
	}
	if id := strings.TrimSpace(r.URL.Query().Get(sessionQuery)); validSessionID(id) {
		m.registerSessionLocked(id)
		return id
	}
	if c, err := r.Cookie(sessionCookie); err == nil && validSessionID(c.Value) {
		m.registerSessionLocked(c.Value)
		return c.Value
	}
	id := newSessionID()
	m.registerSessionLocked(id)
	return id
}

func validSessionID(id string) bool {
	if len(id) < 8 || len(id) > 64 {
		return false
	}
	for _, c := range id {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '-':
		default:
			return false
		}
	}
	return true
}

func (m *Manager) registerSessionLocked(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.sessions[id]; ok {
		return
	}
	m.sessions[id] = &clientSession{id: id}
	if m.fixed && m.fixedKey != "" {
		m.sessions[id].streamKey = m.fixedKey
		if st := m.streams[m.fixedKey]; st != nil {
			st.sessions[id] = struct{}{}
		}
	}
}

// ConnectBrowser attaches the session to a TCP receiver (shared with other sessions on same host:port).
func (m *Manager) ConnectBrowser(sessionID, host string, port int) error {
	uri, err := connvalidate.BuildTCPURI(host, port)
	if err != nil {
		return err
	}
	uri, err = connvalidate.ValidateBrowserTCP(uri, m.cfg.AllowLocalHosts)
	if err != nil {
		return err
	}
	return m.connectURI(sessionID, uri)
}

// ConnectBrowserURI connects using a tcp:// or host:port string.
func (m *Manager) ConnectBrowserURI(sessionID, raw string) error {
	uri, err := connvalidate.ValidateBrowserTCP(raw, m.cfg.AllowLocalHosts)
	if err != nil {
		return err
	}
	return m.connectURI(sessionID, uri)
}

// StartDemo attaches the session to shared demo data.
func (m *Manager) StartDemo(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.fixed {
		return fmt.Errorf("demo mode is fixed at server start")
	}
	sess := m.ensureSessionLocked(sessionID)
	m.detachSessionLocked(sess)
	m.attachSessionLocked(sess, demoStreamKey, "", true)
	return nil
}

// Disconnect releases this session's use of its receiver stream (link stays up while others remain).
func (m *Manager) Disconnect(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.fixed {
		return fmt.Errorf("connection is fixed at server start")
	}
	sess := m.ensureSessionLocked(sessionID)
	m.detachSessionLocked(sess)
	m.idle.SetConnected(false)
	m.idle.SetError("")
	return nil
}

func (m *Manager) connectURI(sessionID, uri string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.fixed {
		return fmt.Errorf("connection is fixed at server start")
	}
	sess := m.ensureSessionLocked(sessionID)
	if sess.streamKey == uri {
		return nil
	}
	m.detachSessionLocked(sess)
	if err := m.attachSessionLocked(sess, uri, uri, false); err != nil {
		return err
	}
	return nil
}

// Snapshot returns the current view for a browser session.
func (m *Manager) Snapshot(sessionID string) model.Snapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	if st := m.streamForSessionLocked(sessionID); st != nil {
		return st.store.Snapshot()
	}
	return m.idle.Snapshot()
}

// Subscribe returns an SSE channel for the session's current stream.
func (m *Manager) Subscribe(sessionID string) chan []byte {
	m.mu.Lock()
	st := m.streamForSessionLocked(sessionID)
	m.mu.Unlock()
	if st != nil {
		return st.hub.Subscribe()
	}
	return m.idleHub().Subscribe()
}

// Unsubscribe removes an SSE subscriber.
func (m *Manager) Unsubscribe(sessionID string, ch chan []byte) {
	m.mu.Lock()
	st := m.streamForSessionLocked(sessionID)
	m.mu.Unlock()
	if st != nil {
		st.hub.Unsubscribe(ch)
		return
	}
	m.idleHub().Unsubscribe(ch)
}

func (m *Manager) idleHub() *hub.Hub {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.streams["_idle"] == nil {
		m.streams["_idle"] = &sharedStream{
			key:      "_idle",
			store:    m.idle,
			hub:      hub.New(),
			sessions: make(map[string]struct{}),
		}
	}
	return m.streams["_idle"].hub
}

func (m *Manager) ensureSessionLocked(id string) *clientSession {
	if sess := m.sessions[id]; sess != nil {
		return sess
	}
	sess := &clientSession{id: id}
	m.sessions[id] = sess
	if m.fixed && m.fixedKey != "" {
		sess.streamKey = m.fixedKey
		if st := m.streams[m.fixedKey]; st != nil {
			st.sessions[id] = struct{}{}
		}
	}
	return sess
}

func (m *Manager) sessionLocked(id string) *clientSession {
	return m.sessions[id]
}

func (m *Manager) streamForSessionLocked(sessionID string) *sharedStream {
	sess := m.sessions[sessionID]
	if sess == nil || sess.streamKey == "" {
		return nil
	}
	return m.streams[sess.streamKey]
}

func (m *Manager) attachSessionLocked(sess *clientSession, key, uri string, demo bool) error {
	st, ok := m.streams[key]
	if !ok {
		if err := m.startStreamLocked(key, uri, demo); err != nil {
			return err
		}
		st = m.streams[key]
	}
	st.sessions[sess.id] = struct{}{}
	sess.streamKey = key
	return nil
}

func (m *Manager) detachSessionLocked(sess *clientSession) {
	if sess.streamKey == "" {
		return
	}
	if m.fixed {
		// Fixed startup link is not reference-counted from browser disconnects.
		if sess.streamKey == m.fixedKey {
			delete(m.streams[sess.streamKey].sessions, sess.id)
			sess.streamKey = ""
		}
		return
	}
	st, ok := m.streams[sess.streamKey]
	if !ok {
		sess.streamKey = ""
		return
	}
	delete(st.sessions, sess.id)
	sess.streamKey = ""
	if len(st.sessions) == 0 {
		m.stopStreamLocked(st)
	}
}

func (m *Manager) startStreamLocked(key, uri string, demo bool) error {
	if _, exists := m.streams[key]; exists {
		return nil
	}
	ctx, cancel := context.WithCancel(m.rootCtx)
	st := &sharedStream{
		key:      key,
		uri:      uri,
		demo:     demo,
		store:    store.New(uri),
		hub:      hub.New(),
		sessions: make(map[string]struct{}),
		cancel:   cancel,
	}
	m.streams[key] = st
	if demo {
		go ingest.RunDemo(ctx, st.store, st.hub)
	} else {
		opts := ingest.Options{Port: uri}
		if m.cfg.Verbose != nil {
			opts.Verbose = m.cfg.Verbose
		} else if m.cfg.VerboseLevel >= verbose.Info {
			opts.Verbose = verbose.New(m.cfg.VerboseLevel)
		}
		go ingest.Run(ctx, opts, st.store, st.hub)
	}
	return nil
}

func (m *Manager) stopStreamLocked(st *sharedStream) {
	if st.cancel != nil {
		st.cancel()
		st.cancel = nil
	}
	disconnected := st.store.SetConnected(false)
	disconnected.LastError = ""
	st.hub.PublishSnapshot(disconnected)
	delete(m.streams, st.key)
}

func newSessionID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
