package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config represents application configuration
type Config struct {
	LastStationID string  `json:"last_station_id"` // Last played station ID
	Volume        float64 `json:"volume"`          // Volume 0.0-1.0
	AreaID        string  `json:"area_id"`         // Current area ID
}

// DefaultConfig returns the default configuration
func DefaultConfig() Config {
	return Config{
		LastStationID: "QRR",  // Default station
		Volume:        0.8,    // Default volume 80%
		AreaID:        "JP13", // Default area: Tokyo
	}
}

// getConfigPath returns the configuration file path
func getConfigPath() (string, error) {
	// Get user config directory
	configDir, err := os.UserConfigDir()
	if err != nil {
		// If failed, use current directory
		configDir = "."
	}

	// Create application config directory
	appConfigDir := filepath.Join(configDir, "radiko-tui")
	if err := os.MkdirAll(appConfigDir, 0755); err != nil {
		return "", err
	}

	return filepath.Join(appConfigDir, "config.json"), nil
}

// Load loads the configuration
func Load() (Config, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return DefaultConfig(), err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Config file doesn't exist, return default config
			return DefaultConfig(), nil
		}
		return DefaultConfig(), err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return DefaultConfig(), err
	}

	// Validate volume range
	if cfg.Volume < 0 {
		cfg.Volume = 0
	} else if cfg.Volume > 1 {
		cfg.Volume = 1
	}

	// Validate area ID, use default if empty
	if cfg.AreaID == "" {
		cfg.AreaID = "JP13"
	}

	return cfg, nil
}

// Save saves the configuration
func Save(cfg Config) error {
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

// SaveConfig saves the configuration (station, volume, area)
func SaveConfig(stationID string, volume float64, areaID string) error {
	cfg := Config{
		LastStationID: stationID,
		Volume:        volume,
		AreaID:        areaID,
	}
	return Save(cfg)
}

// SaveLastStation saves the last played station (backwards compatible)
func SaveLastStation(stationID string, volume float64) error {
	// Load existing config first to preserve AreaID
	existing, _ := Load()
	return SaveConfig(stationID, volume, existing.AreaID)
}
