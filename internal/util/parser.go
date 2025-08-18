package util

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"syncai/internal/model"

	yaml "github.com/goccy/go-yaml"
)

// ParseFile reads a Markdown file and returns a Document with populated FileInfo,
// YAML front matter (if present) as DocumentMetadata, and the Markdown body as Content.
//
// Front matter format supported:
//   - YAML delimited by a line with only '---' at the start of the file and a closing
//     line with '---' or '...'.
//   - UTF-8 BOM at the beginning of the file is ignored.
func ParseFile(path string) (model.Document, error) {
	var doc model.Document

	fi, err := os.Stat(path)
	if err != nil {
		return doc, fmt.Errorf("stat %s: %w", path, err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return doc, fmt.Errorf("read %s: %w", path, err)
	}

	// Strip UTF-8 BOM if present
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		data = data[3:]
	}

	metadata := map[string]string{}
	body := data

	// Detect YAML front matter only if the first line is exactly '---'
	if bytes.HasPrefix(data, []byte("---")) {
		reader := bufio.NewReader(bytes.NewReader(data))
		firstLine, _ := reader.ReadString('\n')
		if strings.TrimRight(firstLine, "\r\n") == "---" {
			var yamlBuf bytes.Buffer
			foundEnd := false
			for {
				line, errRead := reader.ReadString('\n')
				trimmed := strings.TrimRight(line, "\r\n")
				if trimmed == "---" || trimmed == "..." {
					// Remaining data is the body
					rest, _ := io.ReadAll(reader)
					body = rest
					foundEnd = true
					break
				}
				yamlBuf.WriteString(line)
				if errRead != nil {
					// EOF without closing delimiter; treat as no front matter
					break
				}
			}

			if foundEnd && yamlBuf.Len() > 0 {
				var m map[string]interface{}
				if err := yaml.Unmarshal(yamlBuf.Bytes(), &m); err == nil {
					for k, v := range m {
						switch vv := v.(type) {
						case string:
							if s, ok := cleanYAMLValue(vv); ok {
								metadata[k] = s
							}
						default:
							if v == nil {
								continue
							}
							if s, ok := cleanYAMLValue(v); ok {
								metadata[k] = s
							}
						}
					}
				}
			} else {
				// If we didn't find the end delimiter, reset body to full data
				body = data
			}
		}
	}

	doc = model.Document{
		FileInfo: model.FileInfo{
			Path:    path,
			ModTime: fi.ModTime(),
		},
		Metadata: model.DocumentMetadata{
			Raw: metadata,
		},
		Content: body,
	}
	return doc, nil
}

func cleanYAMLValue(value interface{}) (string, bool) {
	sv := strings.TrimSpace(fmt.Sprint(value))
	if sv == "" || strings.EqualFold(sv, "<nil>") {
		return "", false
	}
	return sv, true
}
