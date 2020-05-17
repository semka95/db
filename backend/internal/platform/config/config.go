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
	Auth struct {
		KeyID          string `yaml:"key_id"`
		PrivateKeyFile string `yaml:"private_key_file"`
		Algorithm      string `yaml:"algorithm"`
	} `yaml:"auth"`
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

	cfg := new(Config)
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(cfg)
	if err != nil {
		return nil, fmt.Errorf("can't decode config file: %w", err)
	}
	return cfg, nil
}
