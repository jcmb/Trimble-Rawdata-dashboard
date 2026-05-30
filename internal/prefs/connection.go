package prefs

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// Connection holds the last browser connect form values.
type Connection struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

func configPath() (string, error) {
	if dir := os.Getenv("TRIMBLE_DASHBOARD_CONFIG_DIR"); dir != "" {
		return filepath.Join(dir, "connection.json"), nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "trimble-rawdata-dashboard", "connection.json"), nil
}

// LoadConnection reads the saved host/port from the previous run.
func LoadConnection() (Connection, error) {
	path, err := configPath()
	if err != nil {
		return Connection{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Connection{}, nil
		}
		return Connection{}, err
	}
	var c Connection
	if err := json.Unmarshal(data, &c); err != nil {
		return Connection{}, err
	}
	return c, nil
}

// SaveConnection persists host/port for the next server run.
func SaveConnection(host string, port int) error {
	if host == "" || port <= 0 {
		return nil
	}
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(Connection{Host: host, Port: port})
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
