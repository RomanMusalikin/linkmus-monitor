// internal/agent/config.go
package agent

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config отражает структуру нашего agent-config.yaml
type Config struct {
	Server struct {
		URL      string        `yaml:"url"`
		Interval time.Duration `yaml:"interval"`
	} `yaml:"server"`
}

// LoadConfig читает файл по указанному пути
func LoadConfig(path string) (*Config, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	err = yaml.Unmarshal(file, &cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
