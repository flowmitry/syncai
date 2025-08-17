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

type Guidelines struct {
	Path string `json:"path"`
}

func (g Guidelines) Index() string {
	return "Guidelines"
}

type Ignore struct {
	Path string `json:"path"`
}

type Agent struct {
	Name       string     `json:"name"`
	Rules      Rules      `json:"rules"`
	Guidelines Guidelines `json:"guidelines"`
	Ignore     Ignore     `json:"ignore"`
}

type Meta struct {
	Interval int `json:"interval"`
}

type Config struct {
	Meta   Meta    `json:"config"`
	Agents []Agent `json:"agents"`
}

func Load(path string) (Config, error) {
	f, err := os.Open(path)
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

	if len(cfg.Agents) == 0 {
		return Config{}, fmt.Errorf("config has no agents defined")
	}

	return cfg, nil
}

func (a Agent) Files() []string {
	files := make([]string, 0, 8)

	// Include guidelines if configured
	if p := strings.TrimSpace(a.Guidelines.Path); p != "" {
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
				files = append(files, match)
			}
		}
	}

	return files
}

func (c Config) Interval() time.Duration {
	if c.Meta.Interval == 0 {
		return 5 * time.Second
	}
	return time.Duration(c.Meta.Interval) * time.Second
}
