package model

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type Properties struct {
	Kind Kind
	Stem string
}

type DocumentStack struct {
	Documents   []Document
	Properties  Properties
	ChangedPath string
}

func (s *DocumentStack) Push(d Document) {
	s.Documents = append(s.Documents, d)
}

func (s *DocumentStack) Generate(agent string) ([]byte, error) {
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
		return nil, fmt.Errorf("no documents in stack")
	}
	newestDoc := s.Documents[len(s.Documents)-1]
	content := newestDoc.Content

	if s.Properties.Kind == KindRules {
		// Create metadata map and combine all metadata fields
		metadata := RulesMetadata{
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

		agentName := strings.ToLower(agent)
		if agentName == AgentCursor {
			// Build Cursor front matter with safe YAML quoting and preserve extras
			var sb strings.Builder
			sb.WriteString("---\n")
			sb.WriteString("description: ")
			sb.WriteString(strconv.Quote(metadata.Description))
			sb.WriteString("\n")
			sb.WriteString(fmt.Sprintf("alwaysApply: %t\n", metadata.IsAlwaysApply()))
			sb.WriteString("globs: ")
			sb.WriteString(strconv.Quote(metadata.Globs))
			sb.WriteString("\n")
			// Extra fields
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
					sb.WriteString(strconv.Quote(metadata.ExtraFields[k]))
					sb.WriteString("\n")
				}
			}
			sb.WriteString("---\n")
			content = append([]byte(sb.String()), content...)
		} else if agentName == AgentCopilot {
			// Build Copilot front matter, quoting string values and preserving extras
			var sb strings.Builder
			sb.WriteString("---\n")
			// Copilot uses applyTo, but also include description for preservation
			sb.WriteString("description: ")
			sb.WriteString(strconv.Quote(metadata.Description))
			sb.WriteString("\n")
			sb.WriteString("applyTo: ")
			sb.WriteString(strconv.Quote(metadata.Globs))
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
					sb.WriteString(strconv.Quote(metadata.ExtraFields[k]))
					sb.WriteString("\n")
				}
			}
			sb.WriteString("---\n")
			content = append([]byte(sb.String()), content...)
		} else {
			// Unknown agent, return content of newest file without metadata
		}
	} else {
		// For other kinds, just return content of newest file without metadata
	}

	// Return content of newest file
	return content, nil
}

func isReservedField(field string) bool {
	fl := strings.ToLower(field)
	return fl == "description" || fl == "globs" || fl == "applyto" || fl == "alwaysapply"
}
