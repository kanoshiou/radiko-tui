package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config 应用配置
type Config struct {
	LastStationID string  `json:"last_station_id"` // 上次播放的电台ID
	Volume        float64 `json:"volume"`          // 音量 0.0-1.0
	AreaID        string  `json:"area_id"`         // 当前地区ID
}

// DefaultConfig 默认配置
func DefaultConfig() Config {
	return Config{
		LastStationID: "QRR",  // 默认电台
		Volume:        0.8,    // 默认音量 80%
		AreaID:        "JP13", // 默认地区：東京
	}
}

// getConfigPath 获取配置文件路径
func getConfigPath() (string, error) {
	// 获取用户配置目录
	configDir, err := os.UserConfigDir()
	if err != nil {
		// 如果获取失败，使用当前目录
		configDir = "."
	}

	// 创建应用配置目录
	appConfigDir := filepath.Join(configDir, "radikojp")
	if err := os.MkdirAll(appConfigDir, 0755); err != nil {
		return "", err
	}

	return filepath.Join(appConfigDir, "config.json"), nil
}

// Load 加载配置
func Load() (Config, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return DefaultConfig(), err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// 配置文件不存在，返回默认配置
			return DefaultConfig(), nil
		}
		return DefaultConfig(), err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return DefaultConfig(), err
	}

	// 验证音量范围
	if cfg.Volume < 0 {
		cfg.Volume = 0
	} else if cfg.Volume > 1 {
		cfg.Volume = 1
	}

	// 验证地区 ID，如果为空则使用默认值
	if cfg.AreaID == "" {
		cfg.AreaID = "JP13"
	}

	return cfg, nil
}

// Save 保存配置
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

// SaveConfig 保存配置（电台、音量、地区）
func SaveConfig(stationID string, volume float64, areaID string) error {
	cfg := Config{
		LastStationID: stationID,
		Volume:        volume,
		AreaID:        areaID,
	}
	return Save(cfg)
}

// SaveLastStation 保存上次播放的电台（兼容旧版调用）
func SaveLastStation(stationID string, volume float64) error {
	// 先加载现有配置以保留 AreaID
	existing, _ := Load()
	return SaveConfig(stationID, volume, existing.AreaID)
}
