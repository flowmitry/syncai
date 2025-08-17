package syncai

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"syncai/internal/util"

	"syncai/internal/config"
)

type Kind string

const (
	KindUnknown Kind = ""
	KindRules   Kind = "rules"
	KindContext Kind = "context"
	KindIgnore  Kind = "ignore"
)

type SyncAI struct {
	cfg config.Config
}

func New(cfg config.Config) *SyncAI {
	return &SyncAI{cfg: cfg}
}

// Delete propagates deletion of a watched file to corresponding destinations across other agents.
func (s *SyncAI) Delete(path string) ([]string, error) {
	result := make([]string, 0)
	srcAgent, kind, stem := s.Identify(path)
	result = append(result, path)
	if kind == KindUnknown || srcAgent == nil {
		return result, nil // nothing to do
	}

	// Only propagate deletions for rules. Context/ignore deletions are not propagated to avoid accidental removals.
	if kind != KindRules {
		return result, nil
	}

	for i := range s.cfg.Agents {
		dstAgent := &s.cfg.Agents[i]
		if srcAgent.Name == dstAgent.Name {
			continue
		}

		dstPath := s.generatePath(dstAgent, kind, stem)
		if dstPath == "" {
			continue
		}
		if err := os.Remove(dstPath); err != nil {
			if os.IsNotExist(err) {
				// Already gone at the destination; nothing to do
				result = append(result, dstPath)
				continue
			}
			return result, err
		} else {
			result = append(result, dstPath)
		}
	}
	return result, nil
}

// Sync propagates creation/update of a watched file across other agents.
func (s *SyncAI) Sync(path string) ([]string, error) {
	result := make([]string, 0)
	srcAgent, kind, stem := s.Identify(path)
	if kind == KindUnknown || srcAgent == nil {
		return result, nil // unknown file, ignore
	}

	for i := range s.cfg.Agents {
		dstAgent := &s.cfg.Agents[i]
		if dstAgent.Name == srcAgent.Name {
			continue
		}

		dstPath := s.generatePath(dstAgent, kind, stem)
		if dstPath == "" {
			continue
		}

		switch kind {
		case KindRules:
			err := s.syncRule(srcAgent.Name, path, dstAgent.Name, dstPath)
			if err != nil {
				return result, fmt.Errorf("sync rule: %w", err)
			}
			result = append(result, dstPath)
		case KindContext:
			err := s.syncContext(srcAgent.Name, path, dstAgent.Name, dstPath)
			if err != nil {
				return result, fmt.Errorf("sync context: %w", err)
			}
			result = append(result, dstPath)
		case KindIgnore:
			err := s.syncIgnore(srcAgent.Name, path, dstAgent.Name, dstPath)
			if err != nil {
				return result, fmt.Errorf("sync ignore: %w", err)
			}
			result = append(result, dstPath)
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
		if filepath.Clean(a.Context.Path) == clean {
			return a, KindContext, ""
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

func (s *SyncAI) syncRule(srcAgent, srcPath, dstAgent, dstPath string) error {
	content, err := util.ReadFile(srcPath)
	if err != nil {
		return err
	}
	if err := util.EnsureDir(filepath.Dir(dstPath)); err != nil {
		return err
	}
	if err := util.WriteIfChanged(dstPath, content); err != nil {
		return fmt.Errorf("write %s: %w", dstPath, err)
	}

	return nil
}

func (s *SyncAI) syncContext(srcAgent, srcPath, dstAgent, dstPath string) error {
	content, err := util.ReadFile(srcPath)
	if err != nil {
		return err
	}
	if err := util.EnsureDir(filepath.Dir(dstPath)); err != nil {
		return err
	}
	if err := util.WriteIfChanged(dstPath, content); err != nil {
		return fmt.Errorf("write %s: %w", dstPath, err)
	}

	return nil
}

func (s *SyncAI) syncIgnore(srcAgent, srcPath, dstAgent, dstPath string) error {
	content, err := util.ReadFile(srcPath)
	if err != nil {
		return err
	}
	if err := util.EnsureDir(filepath.Dir(dstPath)); err != nil {
		return err
	}
	if err := util.WriteIfChanged(dstPath, content); err != nil {
		return fmt.Errorf("write %s: %w", dstPath, err)
	}

	return nil
}
