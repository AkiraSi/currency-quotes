package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.yaml.in/yaml/v3"
)

const (
	configPath     = "configs/worker/rate"
	profileEnv     = "PROFILE"
	defaultProfile = "dev"
)

var (
	errHostRequired                = errors.New("host is required")
	errPortInvalid                 = errors.New("port must be between 1 and 65535")
	errUpdateIntervalRequired      = errors.New("updateInterval must be greater than zero")
	errUpdateQueueCapacityRequired = errors.New("updateQueueCapacity must be greater than zero")
)

type Config struct {
	Host                string        `yaml:"host"`
	Port                int           `yaml:"port"`
	UpdateInterval      time.Duration `yaml:"updateInterval"`
	UpdateQueueCapacity int           `yaml:"updateQueueCapacity"`
}

func NewConfig() (*Config, error) {
	profile := os.Getenv(profileEnv)
	if profile == "" {
		profile = defaultProfile
	}

	return load(filepath.Join(configPath, profile+".yaml"))
}

func (c *Config) Print() error {
	_, err := fmt.Fprintf(
		os.Stdout,
		"rate worker config: host=%s port=%d updateInterval=%s updateQueueCapacity=%d\n",
		c.Host,
		c.Port,
		c.UpdateInterval,
		c.UpdateQueueCapacity,
	)

	return err
}

func load(path string) (*Config, error) {
	configFile, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open rate worker config %q: %w", path, err)
	}

	defer func() {
		_ = configFile.Close()
	}()

	cfg := &Config{}
	decoder := yaml.NewDecoder(configFile)
	decoder.KnownFields(true)

	if err := decoder.Decode(cfg); err != nil {
		return nil, err
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) validate() error {
	if c.Host == "" {
		return errHostRequired
	}
	if c.Port <= 0 {
		return errPortInvalid
	}
	if c.UpdateInterval <= 0 {
		return errUpdateIntervalRequired
	}
	if c.UpdateQueueCapacity <= 0 {
		return errUpdateQueueCapacityRequired
	}

	return nil
}
