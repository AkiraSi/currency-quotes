package config

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"go.yaml.in/yaml/v3"

	"currency-quotes/common"
)

const (
	configPath     = "configs/router"
	profileEnv     = "PROFILE"
	defaultProfile = "dev"
)

var (
	errRateWorkerHostRequired = errors.New("rateWorkerHost is required")
	errRateWorkerPortInvalid  = errors.New("rateWorkerPort must be between 1 and 65535")
	errRequestTimeoutInvalid  = errors.New("requestTimeout must be greater than zero")
)

type Config struct {
	Host           string        `yaml:"host"`
	Port           int           `yaml:"port"`
	RateWorkerHost string        `yaml:"rateWorkerHost"`
	RateWorkerPort int           `yaml:"rateWorkerPort"`
	RequestTimeout time.Duration `yaml:"requestTimeout"`
}

func NewConfig() (*Config, error) {
	profile := os.Getenv(profileEnv)
	if profile == "" {
		profile = defaultProfile
	}

	return load(filepath.Join(configPath, profile+".yaml"))
}

func (c *Config) RouterAddress() string {
	return net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
}

func (c *Config) RateWorkerAddress() string {
	return net.JoinHostPort(c.RateWorkerHost, strconv.Itoa(c.RateWorkerPort))
}

func load(path string) (*Config, error) {
	configFile, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open router config %q: %w", path, err)
	}

	defer func() {
		_ = configFile.Close()
	}()

	cfg := &Config{}
	decoder := yaml.NewDecoder(configFile)
	decoder.KnownFields(true)

	if err := decoder.Decode(cfg); err != nil {
		return nil, fmt.Errorf("decode router config %q: %w", path, err)
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) validate() error {
	if c.Host == "" {
		return common.ErrHostRequired
	}
	if c.Port < common.MinPort || c.Port > common.MaxPort {
		return common.ErrPortInvalid
	}
	if c.RateWorkerHost == "" {
		return errRateWorkerHostRequired
	}
	if c.RateWorkerPort < common.MinPort || c.RateWorkerPort > common.MaxPort {
		return errRateWorkerPortInvalid
	}
	if c.RequestTimeout <= 0 {
		return errRequestTimeoutInvalid
	}

	return nil
}
