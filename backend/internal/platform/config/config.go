package config

import (
	"fmt"
	"os"

	"bitbucket.org/dbproject_ivt/db/backend/internal/platform/database"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

// Config stores app configuration
type Config struct {
	Server struct {
		Address string `yaml:"address"`
		Timeout int    `yaml:"timeout"`
	} `yaml:"server"`
	database.MongoConfig `yaml:"mongo"`
}

// AppConfig reads config from file and creates config struct
func AppConfig(cfgPath string, logger *zap.Logger) (*Config, error) {
	f, err := os.Open(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("can't open config file: %w", err)
	}
	defer func() {
		err := f.Close()
		if err != nil {
			logger.Error("can't close config file: %w", zap.Error(err))
		}
	}()

	var cfg *Config
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(cfg)
	if err != nil {
		return nil, fmt.Errorf("can't decode config file: %w", err)
	}

	return cfg, nil
}
