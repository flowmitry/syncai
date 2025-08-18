package generator

import (
	"strings"
	"syncai/internal/model"
)

type RulesGenerator interface {
	GenerateRules(metadata model.RulesMetadata, content []byte) []byte
}

func GetRulesGenerator(agentName string) RulesGenerator {
	switch strings.ToLower(agentName) {
	case model.AgentCursor:
		return CursorRulesGenerator{}
	case model.AgentCopilot:
		return CopilotRulesGenerator{}
	default:
		return OtherRulesGenerator{}
	}
}

func ExtractRulesMetadata(s *model.DocumentStack) model.RulesMetadata {
	metadata := model.RulesMetadata{
		ExtraFields: make(map[string]string),
	}
	for _, d := range s.Documents {
		for k, v := range d.Metadata.Raw {
			keyName := strings.ToLower(k)
			switch keyName {
			case "description":
				if strings.TrimSpace(v) != "" {
					metadata.Description = v
				}
			case "globs":
				if strings.TrimSpace(v) != "" {
					metadata.Globs = v
				}
			case "applyto":
				// Copilot uses `applyTo` for globs
				if strings.TrimSpace(v) != "" {
					metadata.Globs = v
				}
			case "alwaysapply":
				// Only set to always-apply when the value is truthy
				vv := strings.ToLower(strings.TrimSpace(v))
				if vv == "true" || vv == "1" || vv == "yes" || vv == "on" {
					metadata.Globs = "**"
				}
			default:
				metadata.ExtraFields[k] = v
			}
		}
	}
	return metadata
}
