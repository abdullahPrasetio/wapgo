package generator

import (
	"strings"
	"unicode"
)

// Names holds every naming variant derived from a single input identifier.
// Callers set Module and AppName before passing to templates.
type Names struct {
	Original string // original input as given by the user
	Snake    string // lower_snake_case
	Pascal   string // PascalCase
	Camel    string // camelCase
	Table    string // plural snake (naive)
	Module   string // go module path  (set by caller)
	AppName  string // APP_NAME default (set by caller)
	DB       string // "postgres" | "mysql" (set by caller)
}

// NewNames derives naming variants from an arbitrary input string.
// Input may be snake_case, PascalCase, camelCase, or kebab-case.
func NewNames(input string) Names {
	snake := toSnake(input)
	pascal := toPascal(snake)
	return Names{
		Original: input,
		Snake:    snake,
		Pascal:   pascal,
		Camel:    toCamel(pascal),
		Table:    toTable(snake),
	}
}

// toSnake converts any casing to lower_snake_case.
func toSnake(s string) string {
	// Replace kebab with underscore first.
	s = strings.ReplaceAll(strings.TrimSpace(s), "-", "_")

	var b strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 && !unicode.IsUpper(rune(s[i-1])) {
				b.WriteByte('_')
			}
			b.WriteRune(unicode.ToLower(r))
		} else {
			b.WriteRune(r)
		}
	}
	return strings.ToLower(b.String())
}

// toPascal converts lower_snake_case to PascalCase.
func toPascal(snake string) string {
	parts := strings.Split(snake, "_")
	var b strings.Builder
	for _, p := range parts {
		if len(p) == 0 {
			continue
		}
		b.WriteRune(unicode.ToUpper(rune(p[0])))
		b.WriteString(p[1:])
	}
	return b.String()
}

// toCamel converts PascalCase to camelCase.
func toCamel(pascal string) string {
	if len(pascal) == 0 {
		return pascal
	}
	r := []rune(pascal)
	r[0] = unicode.ToLower(r[0])
	return string(r)
}

// toTable produces a naive plural from snake_case (covers the common cases).
func toTable(snake string) string {
	if strings.HasSuffix(snake, "y") && len(snake) > 1 {
		prev := snake[len(snake)-2]
		vowels := "aeiou"
		if !strings.ContainsRune(vowels, rune(prev)) {
			return snake[:len(snake)-1] + "ies"
		}
	}
	if strings.HasSuffix(snake, "s") || strings.HasSuffix(snake, "x") ||
		strings.HasSuffix(snake, "z") || strings.HasSuffix(snake, "sh") ||
		strings.HasSuffix(snake, "ch") {
		return snake + "es"
	}
	return snake + "s"
}
