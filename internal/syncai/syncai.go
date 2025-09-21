package syncai

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syncai/internal/generator"
	"syncai/internal/model"
	"syncai/internal/util"

	"syncai/internal/config"
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
	if kind == model.KindUnknown || srcAgent == nil {
		return result, nil // nothing to do
	}

	// Only propagate deletions for rules and commands. Context/ignore deletions are not propagated to avoid accidental removals.
	if kind != model.KindRules && kind != model.KindCommands {
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
	if kind == model.KindUnknown || srcAgent == nil {
		return result, nil // unknown file, ignore
	}

	stack := model.DocumentStack{
		Documents:   make([]model.Document, 0),
		ChangedPath: path,
		Properties: model.Properties{
			Kind: kind,
			Stem: stem,
		},
	}
	for i := range s.cfg.Agents {
		dstAgent := &s.cfg.Agents[i]

		var docPath string
		if dstAgent.Name == srcAgent.Name {
			docPath = path
		} else {
			docPath = s.generatePath(dstAgent, kind, stem)
			if docPath == "" {
				continue
			}
		}
		if util.IsFileExists(docPath) {
			doc, err := util.ParseFile(docPath)
			if err != nil {
				return result, fmt.Errorf("parse %s for agent %s: %w", docPath, dstAgent.Name, err)
			}
			stack.Push(doc)
		}
	}

	for i := range s.cfg.Agents {
		dstAgent := &s.cfg.Agents[i]
		if srcAgent.Name == dstAgent.Name {
			continue
		}

		dstPath := s.generatePath(dstAgent, kind, stem)
		if strings.TrimSpace(dstPath) == "" {
			// No target path configured for this agent/kind; skip writing
			continue
		}
		data, err := generate(&stack, dstAgent.Name)
		if err != nil {
			return result, fmt.Errorf("generate stack for agent %s: %w", dstAgent.Name, err)
		}
		if err = util.WriteFile(dstPath, data); err != nil {
			return result, fmt.Errorf("write %s for agent %s: %w", dstPath, dstAgent.Name, err)
		}
		result = append(result, dstPath)
		log.Printf("File %s synced to %s", path, dstPath)
	}

	return result, nil
}

func (s *SyncAI) Identify(path string) (*config.Agent, model.Kind, string) {
	clean := filepath.Clean(path)
	for i := range s.cfg.Agents {
		a := &s.cfg.Agents[i]
		if filepath.Clean(a.Context.Path) == clean {
			return a, model.KindContext, ""
		}
		if filepath.Clean(a.Ignore.Path) == clean {
			return a, model.KindIgnore, ""
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
						return a, model.KindRules, stem
					}
				} else {
					if filename == basePattern {
						stem := filename
						if ext := filepath.Ext(stem); ext != "" {
							stem = strings.TrimSuffix(stem, ext)
						}
						return a, model.KindRules, stem
					}
				}
			}
		}
		if strings.TrimSpace(a.Commands.Pattern) != "" {
			// Match by comparing the directory and base pattern, independent of file existence
			pattern := a.Commands.Pattern
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
						return a, model.KindCommands, stem
					}
				} else {
					if filename == basePattern {
						stem := filename
						if ext := filepath.Ext(stem); ext != "" {
							stem = strings.TrimSuffix(stem, ext)
						}
						return a, model.KindCommands, stem
					}
				}
			}
		}
	}
	return nil, model.KindUnknown, ""
}

func generate(s *model.DocumentStack, agent string) ([]byte, error) {
	// Sort documents by ModTime.
	// The document with ChangedPath is always considered the "newest" and placed last,
	// regardless of its actual modification time. This ensures that the changed document
	// is prioritized for further processing, even if its ModTime is older than others.
	sort.Slice(s.Documents, func(i, j int) bool {
		if s.Documents[i].FileInfo.Path == s.ChangedPath {
			return false
		}
		if s.Documents[j].FileInfo.Path == s.ChangedPath {
			return true
		}
		return s.Documents[i].FileInfo.ModTime.Before(s.Documents[j].FileInfo.ModTime)
	})

	if len(s.Documents) == 0 {
		return []byte{}, fmt.Errorf("no documents in stack")
	}
	newestDoc := s.Documents[len(s.Documents)-1]
	content := newestDoc.Content
	agentName := strings.ToLower(agent)

	if s.Properties.Kind == model.KindRules {
		if gen := generator.GetRulesGenerator(agentName); gen != nil {
			metadata := generator.ExtractRulesMetadata(s)
			content = gen.GenerateRules(metadata, content)
		}
	}

	// Return content of newest file
	return content, nil
}
