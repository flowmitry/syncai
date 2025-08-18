package generator

import "strings"

func isReservedField(field string) bool {
	fl := strings.ToLower(field)
	return fl == "description" || fl == "globs" || fl == "applyto" || fl == "alwaysapply"
}
