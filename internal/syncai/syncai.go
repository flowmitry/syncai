package syncai

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"syncai/internal/config"
	"syncai/internal/util"
)

type Kind string

const (
	KindUnknown    Kind = ""
	KindRules      Kind = "rules"
	KindGuidelines Kind = "guidelines"
	KindIgnore     Kind = "ignore"
)

type SyncAI struct {
	cfg config.Config
}

func New(cfg config.Config) *SyncAI {
	return &SyncAI{cfg: cfg}
}

// Delete propagates deletion of a watched file to corresponding destinations across other agents.
func (s *SyncAI) Delete(path string) error {
	srcAgent, kind, stem := s.Identify(path)
	if kind == KindUnknown || srcAgent == nil {
		return nil // nothing to do
	}
	for i := range s.cfg.Agents {
		dstAgent := &s.cfg.Agents[i]
		if srcAgent.Name == dstAgent.Name {
			continue
		}

		dstPath := s.GeneratePath(dstAgent, kind, stem)
		if dstPath == "" {
			continue
		}
		if err := os.Remove(dstPath); err != nil {
			if os.IsNotExist(err) {
				// Already gone at the destination; nothing to do
				continue
			}
			return err
		}
	}
	return nil
}

// Sync propagates creation/update of a watched file across other agents.
func (s *SyncAI) Sync(path string) ([]string, error) {
	result := make([]string, 0)
	srcAgent, kind, stem := s.Identify(path)
	if kind == KindUnknown || srcAgent == nil {
		return result, nil // unknown file, ignore
	}

	// Read source content once
	f, err := os.Open(path)
	if err != nil {
		return result, fmt.Errorf("open source: %w", err)
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return result, fmt.Errorf("read source: %w", err)
	}

	for i := range s.cfg.Agents {
		dstAgent := &s.cfg.Agents[i]
		if dstAgent.Name == srcAgent.Name {
			continue
		}

		dstPath := s.GeneratePath(dstAgent, kind, stem)
		if dstPath == "" {
			continue
		}
		if err := util.EnsureDir(filepath.Dir(dstPath)); err != nil {
			return result, err
		}
		if err := util.WriteIfChanged(dstPath, data); err != nil {
			return result, fmt.Errorf("write %s: %w", dstPath, err)
		}
		result = append(result, dstPath)
		log.Printf("File %s synced to %s", path, dstPath)
	}

	return result, nil
}

func (s *SyncAI) Identify(path string) (*config.Agent, Kind, string) {
	clean := filepath.Clean(path)
	for i := range s.cfg.Agents {
		a := &s.cfg.Agents[i]
		if filepath.Clean(a.Guidelines.Path) == clean {
			return a, KindGuidelines, ""
		}
		if filepath.Clean(a.Ignore.Path) == clean {
			return a, KindIgnore, ""
		}
		if strings.TrimSpace(a.Rules.Pattern) != "" {
			// Match by comparing the directory and base pattern, independent of file existence
			pattern := a.Rules.Pattern
			patDir := filepath.Clean(filepath.Dir(pattern))
			fileDir := filepath.Clean(filepath.Dir(clean))
			if patDir == fileDir {
				basePattern := filepath.Base(pattern)
				filename := filepath.Base(clean)
				// Attempt to extract stem depending on wildcard presence
				if strings.Contains(basePattern, "*") {
					parts := strings.Split(basePattern, "*")
					prefix := parts[0]
					suffix := ""
					if len(parts) > 1 {
						suffix = parts[len(parts)-1]
					}
					if strings.HasPrefix(filename, prefix) && strings.HasSuffix(filename, suffix) {
						stem := strings.TrimPrefix(filename, prefix)
						stem = strings.TrimSuffix(stem, suffix)
						return a, KindRules, stem
					}
				} else {
					if filename == basePattern {
						stem := filename
						if ext := filepath.Ext(stem); ext != "" {
							stem = strings.TrimSuffix(stem, ext)
						}
						return a, KindRules, stem
					}
				}
			}
		}
	}
	return nil, KindUnknown, ""
}

func (s *SyncAI) GeneratePath(agent *config.Agent, kind Kind, stem string) string {
	if agent == nil {
		return ""
	}
	switch kind {
	case KindGuidelines:
		return agent.Guidelines.Path
	case KindIgnore:
		return agent.Ignore.Path
	case KindRules:
		pattern := strings.TrimSpace(agent.Rules.Pattern)
		if pattern == "" {
			return ""
		}
		dir := filepath.Dir(pattern)
		base := filepath.Base(pattern)
		var filename string
		if strings.Contains(base, "*") {
			filename = strings.ReplaceAll(base, "*", stem)
		} else {
			// No wildcard in base, just use stem with the same extension as the pattern base
			ext := filepath.Ext(base)
			if ext == "" {
				filename = stem
			} else {
				filename = stem + ext
			}
		}
		return filepath.Join(dir, filename)
	default:
		return ""
	}
}
