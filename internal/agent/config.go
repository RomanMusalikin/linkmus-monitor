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

// configPath ищет agent-config.yaml в нескольких местах для совместимости
// со старыми установками (бинарник в /usr/local/bin, конфиг в /opt/mon-agent/).
func configPath() string {
	candidates := []string{}

	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "agent-config.yaml"))
	}
	candidates = append(candidates,
		"/opt/mon-agent/agent-config.yaml",
		"/opt/mon-agent/configs/agent-config.yaml",
		"C:\\mon-agent\\agent-config.yaml",
		"agent-config.yaml",
	)

	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return candidates[0]
}
