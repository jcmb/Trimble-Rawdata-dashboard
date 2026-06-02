// Package browseropen opens the dashboard in the system default browser when a GUI is available.
package browseropen

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// HasGUI reports whether the process likely runs in a desktop session.
func HasGUI() bool {
	if os.Getenv("NO_BROWSER") != "" || os.Getenv("CI") != "" {
		return false
	}
	switch runtime.GOOS {
	case "darwin", "windows":
		return true
	default:
		return os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != ""
	}
}

// Open launches the default browser for url (non-blocking).
func Open(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open browser: %w", err)
	}
	return nil
}
