package prefs_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gkirk/trimble-rawdata-dashboard/internal/prefs"
)

func TestSaveLoadConnection(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TRIMBLE_DASHBOARD_CONFIG_DIR", dir)

	if err := prefs.SaveConnection("example.com", 28005); err != nil {
		t.Fatal(err)
	}
	got, err := prefs.LoadConnection()
	if err != nil {
		t.Fatal(err)
	}
	if got.Host != "example.com" || got.Port != 28005 {
		t.Fatalf("got %+v", got)
	}

	path := filepath.Join(dir, "connection.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}
