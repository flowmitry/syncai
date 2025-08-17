package util

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"syncai/internal/model"

	gyaml "github.com/goccy/go-yaml"
)

func ParseYAML(path string) (model.Document, error) {
	var out model.Document
	if path == "" {
		return out, fmt.Errorf("yaml: empty path")
	}

	// Read file bytes using existing util helper
	b, err := ReadFile(path)
	if err != nil {
		return out, fmt.Errorf("yaml: read %s: %w", path, err)
	}

	// Extract YAML front matter if present
	meta, body := extractFrontMatter(b)

	// Prepare content
	var content []byte
	if meta != nil { // front matter present
		if len(bytes.TrimSpace(body)) > 0 {
			if err := gyaml.Unmarshal(body, &content); err != nil {
				return out, fmt.Errorf("yaml: unmarshal body %s: %w", path, err)
			}
		}
	} else {
		// No front matter: treat entire file as YAML
		if err := gyaml.Unmarshal(b, &content); err != nil {
			return out, fmt.Errorf("yaml: unmarshal %s: %w", path, err)
		}
	}

	// Collect file info
	fi, err := os.Stat(path)
	if err != nil {
		return out, fmt.Errorf("yaml: stat %s: %w", path, err)
	}
	hash, err := FileHash(path)
	if err != nil {
		return out, fmt.Errorf("yaml: hash %s: %w", path, err)
	}

	out = model.Document{
		FileInfo: model.FileInfo{Path: path, Size: fi.Size(), ModTime: fi.ModTime(), Hash: hash},
		Metadata: meta,
		Content:  content,
	}
	return out, nil
}

// extractFrontMatter extracts YAML front matter from the start of the file content.
// Returns:
//   - map[string]string with converted scalar values (or nil if no front matter)
//   - the body bytes after the closing '---' line (or the original bytes if none)
func extractFrontMatter(b []byte) (map[string]string, []byte) {
	// Read first line
	i := bytes.IndexByte(b, '\n')
	var firstLine []byte
	var rest []byte
	if i == -1 {
		firstLine = b
		rest = nil
	} else {
		firstLine = b[:i]
		rest = b[i+1:]
	}
	first := strings.TrimSpace(strings.TrimSuffix(string(firstLine), "\r"))
	if first != "---" {
		return nil, b
	}

	// Scan lines for closing '---'
	pos := 0
	end := -1
	// Iterate over lines in 'rest'
	for {
		if pos >= len(rest) {
			break
		}
		j := bytes.IndexByte(rest[pos:], '\n')
		var line []byte
		var next int
		if j == -1 {
			line = rest[pos:]
			next = len(rest)
		} else {
			line = rest[pos : pos+j]
			next = pos + j + 1
		}
		ln := strings.TrimSpace(strings.TrimSuffix(string(line), "\r"))
		if ln == "---" {
			end = pos
			pos = next
			break
		}
		pos = next
	}
	if end == -1 {
		// No closing delimiter; treat as no front matter
		return nil, b
	}

	fm := rest[:end]
	body := rest[pos:]

	// Unmarshal front matter YAML
	raw := map[string]any{}
	if err := gyaml.Unmarshal(fm, &raw); err != nil {
		// On error, ignore as front matter and return whole content
		return nil, b
	}

	meta := make(map[string]string, len(raw))
	for k, v := range raw {
		// Only string keys are expected; convert values to string
		if k == "" {
			continue
		}
		meta[k] = scalarToString(v)
	}

	return meta, body
}

func scalarToString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case fmt.Stringer:
		return t.String()
	default:
		return fmt.Sprint(v)
	}
}
