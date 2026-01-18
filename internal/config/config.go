package config

import (
	"fmt"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Config defines server configuration.
type Config struct {
	Server ServerConfig `yaml:"server"`
	DB     DBConfig     `yaml:"db"`
	Log    LogConfig    `yaml:"log"`
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

// Load reads configuration from an optional YAML file and environment variables.
func Load() (Config, error) {
	cfg := Config{
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: 8080,
		},
		DB: DBConfig{
			Path: "threds.db",
		},
		Log: LogConfig{
			Level: "info",
		},
	}

	if path := os.Getenv("THREDS_CONFIG_PATH"); path != "" {
		if err := loadFromFile(path, &cfg); err != nil {
			return Config{}, err
		}
	}

	if host := os.Getenv("THREDS_SERVER_HOST"); host != "" {
		cfg.Server.Host = host
	}
	if portStr := os.Getenv("THREDS_SERVER_PORT"); portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return Config{}, fmt.Errorf("invalid THREDS_SERVER_PORT: %w", err)
		}
		cfg.Server.Port = port
	}
	if dbPath := os.Getenv("THREDS_DB_PATH"); dbPath != "" {
		cfg.DB.Path = dbPath
	}
	if level := os.Getenv("THREDS_LOG_LEVEL"); level != "" {
		cfg.Log.Level = level
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
