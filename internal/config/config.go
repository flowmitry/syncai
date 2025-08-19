package config

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Rules struct {
	Pattern string `json:"pattern"`
}

type Context struct {
	Path string `json:"path"`
}

func (c Context) Index() string {
	return "Context"
}

type Ignore struct {
	Path string `json:"path"`
}

type Agent struct {
	Name    string  `json:"name"`
	Rules   Rules   `json:"rules"`
	Context Context `json:"context"`
	Ignore  Ignore  `json:"ignore"`
}

type Meta struct {
	Interval   int    `json:"interval"`
	WorkingDir string `json:"workdir"`
}

type Config struct {
	Meta   Meta    `json:"config"`
	Agents []Agent `json:"agents"`
}

func (c Config) Interval() time.Duration {
	if c.Meta.Interval == 0 {
		return 5 * time.Second
	}
	return time.Duration(c.Meta.Interval) * time.Second
}

func (c Config) WorkingDir() string {
	return strings.TrimSuffix(c.Meta.WorkingDir, "/")
}

func Load(configPath, basePath string) (Config, error) {
	f, err := os.Open(configPath)
	if err != nil {
		return Config{}, fmt.Errorf("open config: %w", err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	if strings.TrimSpace(basePath) != "" {
		cfg.Meta.WorkingDir = strings.TrimSpace(basePath)
	} else {
		if strings.TrimSpace(cfg.Meta.WorkingDir) == "" {
			cfg.Meta.WorkingDir = strings.TrimSpace(filepath.Dir(configPath))
		}
	}

	if err := validateWorkingDir(cfg.Meta.WorkingDir); err != nil {
		return Config{}, fmt.Errorf("working directory error: %w", err)
	}

	if len(cfg.Agents) == 0 {
		return Config{}, fmt.Errorf("config has no agents defined")
	}

	return cfg, nil
}

func validateWorkingDir(basePath string) error {
	path := strings.TrimSuffix(basePath, "/")
	if info, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("working dir %s does not exist: %w", path, err)
		}
	} else if !info.IsDir() {
		return fmt.Errorf("working dir %s is not a directory", path)
	}
	return nil
}

func (a Agent) Files() []string {
	files := make([]string, 0, 8)

	// Include context if configured
	if p := strings.TrimSpace(a.Context.Path); p != "" {
		files = append(files, p)
	}

	// Include ignore if configured
	if p := strings.TrimSpace(a.Ignore.Path); p != "" {
		files = append(files, p)
	}

	// Include rules only if a non-empty pattern is configured
	if pat := strings.TrimSpace(a.Rules.Pattern); pat != "" {
		matches, err := filepath.Glob(pat)
		if err != nil {
			log.Printf("glob %s: %v", pat, err)
		}
		for _, match := range matches {
			path := strings.TrimSpace(match)
			if path != "" {
				_, err := os.Stat(path)
				if err != nil {
					if os.IsNotExist(err) {
						// File may not exist yet; silently ignore
						continue
					}
					log.Printf("file error %s: %v", path, err)
					continue
				}
				files = append(files, path)
			}
		}
	}

	return files
}
