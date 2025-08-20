package generator

import (
	"strconv"
	"strings"
)

const yamlSpecialChars = " \":{}[]#&*!|>'%@`"

func isReservedField(field string) bool {
	fl := strings.ToLower(field)
	return fl == "description" || fl == "globs" || fl == "applyto" || fl == "alwaysapply"
}

func quoteIfNeeded(s string) string {
	if s == "" {
		return strconv.Quote(s)
	}
	if strings.HasPrefix(s, "\"") && strings.HasSuffix(s, "\"") {
		return s
	}
	if strings.ContainsAny(s, yamlSpecialChars) {
		return strconv.Quote(s)
	}
	return s
}
