package generator

import (
	"sort"
	"strings"
	"syncai/internal/model"
)

type CopilotRulesGenerator struct{}

func (g CopilotRulesGenerator) GenerateRules(metadata model.RulesMetadata, content []byte) []byte {
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString("description: ")
	sb.WriteString(quoteIfNeeded(metadata.Description))
	sb.WriteString("\n")
	sb.WriteString("applyTo: ")
	if metadata.Globs == "" {
		sb.WriteString(quoteIfNeeded("**"))
	} else {
		sb.WriteString(quoteIfNeeded(metadata.Globs))
	}
	sb.WriteString("\n")
	if len(metadata.ExtraFields) > 0 {
		keys := make([]string, 0, len(metadata.ExtraFields))
		for k := range metadata.ExtraFields {
			if isReservedField(k) {
				continue
			}
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			sb.WriteString(k)
			sb.WriteString(": ")
			sb.WriteString(quoteIfNeeded(metadata.ExtraFields[k]))
			sb.WriteString("\n")
		}
	}
	sb.WriteString("---\n")
	return append([]byte(sb.String()), content...)
}
