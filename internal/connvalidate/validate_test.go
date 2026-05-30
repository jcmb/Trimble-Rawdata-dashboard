package connvalidate_test

import (
	"strings"
	"testing"

	"github.com/gkirk/trimble-rawdata-dashboard/internal/connvalidate"
)

func TestBuildTCPURI(t *testing.T) {
	uri, err := connvalidate.BuildTCPURI("sps855.com", 28005)
	if err != nil || uri != "tcp://sps855.com:28005" {
		t.Fatalf("got %q err=%v", uri, err)
	}
}

func TestValidateBrowserTCPBlocksLoopback(t *testing.T) {
	_, err := connvalidate.ValidateBrowserTCP("tcp://127.0.0.1:28005", false)
	if err == nil || !strings.Contains(err.Error(), "local or private") {
		t.Fatalf("expected block, got %v", err)
	}
}

func TestValidateBrowserTCPAllowsLoopbackWithFlag(t *testing.T) {
	uri, err := connvalidate.ValidateBrowserTCP("127.0.0.1:28005", true)
	if err != nil || uri != "tcp://127.0.0.1:28005" {
		t.Fatalf("got %q err=%v", uri, err)
	}
}

func TestValidateBrowserTCPBlocksPrivate(t *testing.T) {
	_, err := connvalidate.ValidateBrowserTCP("tcp://192.168.1.10:5017", false)
	if err == nil {
		t.Fatal("expected block")
	}
}

func TestValidateBrowserTCPAllowsPublicIP(t *testing.T) {
	uri, err := connvalidate.ValidateBrowserTCP("tcp://8.8.8.8:28005", false)
	if err != nil || uri != "tcp://8.8.8.8:28005" {
		t.Fatalf("got %q err=%v", uri, err)
	}
}

func TestValidateBrowserTCPRejectsSerial(t *testing.T) {
	_, err := connvalidate.ValidateBrowserTCP("serial:///dev/ttyUSB0", false)
	if err == nil {
		t.Fatal("expected reject")
	}
}
