package syncai

import (
	"path/filepath"
	"strings"
	"syncai/internal/config"
)

func (s *SyncAI) generatePath(agent *config.Agent, kind Kind, stem string) string {
	if agent == nil {
		return ""
	}
	switch kind {
	case KindContext:
		return agent.Context.Path
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
