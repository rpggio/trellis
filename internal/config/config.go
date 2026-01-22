package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Config defines server configuration.
type Config struct {
	Transport TransportConfig `yaml:"transport"`
	Server    ServerConfig    `yaml:"server"`
	DB        DBConfig        `yaml:"db"`
	Log       LogConfig       `yaml:"log"`
	Auth      AuthConfig      `yaml:"auth"`
}

type TransportConfig struct {
	Mode string `yaml:"mode"` // "stdio" or "http"
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type DBConfig struct {
	Path string `yaml:"path"`
}

type LogConfig struct {
	Level string `yaml:"level"`
}

type AuthConfig struct {
	Enabled bool `yaml:"enabled"`
}

// Load reads configuration from an optional YAML file and environment variables.
func Load() (Config, error) {
	// Determine default DB path: same directory as binary
	defaultDBPath := "trellis.db"
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		defaultDBPath = filepath.Join(exeDir, "trellis.db")
	}

	cfg := Config{
		Transport: TransportConfig{
			Mode: "stdio", // default to stdio for local development
		},
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: 8080,
		},
		DB: DBConfig{
			Path: defaultDBPath,
		},
		Log: LogConfig{
			Level: "info",
		},
		Auth: AuthConfig{
			Enabled: true,
		},
	}

	if path := os.Getenv("TRELLIS_CONFIG_PATH"); path != "" {
		if err := loadFromFile(path, &cfg); err != nil {
			return Config{}, err
		}
	}

	if mode := os.Getenv("TRELLIS_TRANSPORT"); mode != "" {
		cfg.Transport.Mode = mode
	}
	if host := os.Getenv("TRELLIS_SERVER_HOST"); host != "" {
		cfg.Server.Host = host
	}
	if portStr := os.Getenv("TRELLIS_SERVER_PORT"); portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return Config{}, fmt.Errorf("invalid TRELLIS_SERVER_PORT: %w", err)
		}
		cfg.Server.Port = port
	}
	if dbPath := os.Getenv("TRELLIS_DB_PATH"); dbPath != "" {
		cfg.DB.Path = dbPath
	}
	if level := os.Getenv("TRELLIS_LOG_LEVEL"); level != "" {
		cfg.Log.Level = level
	}
	if enabled := os.Getenv("TRELLIS_AUTH_ENABLED"); enabled != "" {
		value, err := strconv.ParseBool(enabled)
		if err != nil {
			return Config{}, fmt.Errorf("invalid TRELLIS_AUTH_ENABLED: %w", err)
		}
		cfg.Auth.Enabled = value
	}

	return cfg, nil
}

func loadFromFile(path string, cfg *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("parse config file: %w", err)
	}
	return nil
}
