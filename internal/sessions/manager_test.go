package sessions

import (
	"context"
	"testing"
	"time"
)

func TestSharedStreamRefCount(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := NewManager(ctx, Config{})
	s1 := "sess-a"
	s2 := "sess-b"
	m.ensureSessionLocked(s1)
	m.ensureSessionLocked(s2)

	uri := "tcp://example.com:28005"

	if err := m.ConnectBrowserURI(s1, uri); err != nil {
		t.Fatal(err)
	}
	if !m.streamActive(uri) {
		t.Fatal("stream should be active after first connect")
	}

	if err := m.ConnectBrowserURI(s2, uri); err != nil {
		t.Fatal(err)
	}
	if m.streamRefCount(uri) != 2 {
		t.Fatalf("ref count: got %d want 2", m.streamRefCount(uri))
	}

	if err := m.Disconnect(s1); err != nil {
		t.Fatal(err)
	}
	if !m.streamActive(uri) {
		t.Fatal("stream should stay up while sess-b remains")
	}
	if m.streamRefCount(uri) != 1 {
		t.Fatalf("ref count after s1 disconnect: got %d want 1", m.streamRefCount(uri))
	}

	if err := m.Disconnect(s2); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !m.streamActive(uri) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("stream should stop after last session disconnects")
}

func TestTwoExplicitSessionsDifferentPorts(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := NewManager(ctx, Config{})
	m.registerSessionLocked("browser-a")
	m.registerSessionLocked("browser-b")

	if err := m.ConnectBrowserURI("browser-a", "tcp://example.com:28005"); err != nil {
		t.Fatal(err)
	}
	if err := m.ConnectBrowserURI("browser-b", "tcp://example.com:28006"); err != nil {
		t.Fatal(err)
	}
	if !m.streamActive("tcp://example.com:28005") || !m.streamActive("tcp://example.com:28006") {
		t.Fatal("both streams should stay active")
	}
	if m.streamRefCount("tcp://example.com:28005") != 1 || m.streamRefCount("tcp://example.com:28006") != 1 {
		t.Fatalf("refcounts: %d %d", m.streamRefCount("tcp://example.com:28005"), m.streamRefCount("tcp://example.com:28006"))
	}
}

func TestDifferentStreams(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := NewManager(ctx, Config{})
	m.ensureSessionLocked("sess-1")
	m.ensureSessionLocked("sess-2")

	if err := m.ConnectBrowserURI("sess-1", "tcp://example.com:28005"); err != nil {
		t.Fatal(err)
	}
	if err := m.ConnectBrowserURI("sess-2", "tcp://example.com:28006"); err != nil {
		t.Fatal(err)
	}
	if !m.streamActive("tcp://example.com:28005") || !m.streamActive("tcp://example.com:28006") {
		t.Fatal("expected both streams active")
	}
}

func (m *Manager) streamActive(key string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	st, ok := m.streams[key]
	return ok && st.cancel != nil
}

func (m *Manager) streamRefCount(key string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	st, ok := m.streams[key]
	if !ok {
		return 0
	}
	return len(st.sessions)
}
