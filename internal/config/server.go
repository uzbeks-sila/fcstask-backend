package config

import (
	//"os"
	"time"
	//"gopkg.in/yaml.v3"
)

type Config struct {
	Server ServerConfig `yaml:"server"`
}

type ServerConfig struct {
	Host            string        `yaml:"host"`
	Port            int           `yaml:"port"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
}

func Load(path string) (*Config, error) {
	/*
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}

		var cfg Config
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, err
		}
	*/
	return &Config{
		Server: ServerConfig{
			Host:            "localhost",
			Port:            8080,
			ShutdownTimeout: 5 * time.Second,
		},
	}, nil
}
