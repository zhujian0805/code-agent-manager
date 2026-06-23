package catalogconfig

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config is the shared config.yaml shape used by Chat2AnyLLM catalog repos.
type Config struct {
	Output struct {
		Dir     string   `yaml:"dir"`
		Formats []string `yaml:"formats"`
	} `yaml:"output"`
}

// DataFile derives the generated data file path for a config.yaml catalog.
func DataFile(dataName string, raw []byte) (string, error) {
	var cfg Config
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return "", fmt.Errorf("catalog config: parse: %w", err)
	}
	dir := strings.Trim(strings.TrimSpace(cfg.Output.Dir), "/")
	if dir == "" {
		dir = "dist"
	}
	for _, format := range cfg.Output.Formats {
		if strings.EqualFold(strings.TrimSpace(format), "json") {
			return dir + "/" + dataName + ".json", nil
		}
	}
	return "", fmt.Errorf("catalog config: missing json output format")
}
