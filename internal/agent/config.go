package agent

import (
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		URL      string        `yaml:"url"`
		Interval time.Duration `yaml:"interval"`
	} `yaml:"server"`
}

func LoadConfig(path string) (*Config, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(file, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// configPath возвращает путь к конфигу рядом с исполняемым файлом.
// Это надёжно работает как при запуске службой Windows, так и вручную из любой директории.
func configPath() string {
	exe, err := os.Executable()
	if err != nil {
		return "agent-config.yaml"
	}
	return filepath.Join(filepath.Dir(exe), "agent-config.yaml")
}
