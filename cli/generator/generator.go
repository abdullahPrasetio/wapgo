package generator

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

// templateDelimLeft / Right avoid collisions with Go struct-literal braces.
const templateDelimLeft = "[["
const templateDelimRight = "]]"

// Render parses tmplContent as a text/template (delimiters [[ ]]) and writes
// the executed result to outPath.  Returns an error if outPath already exists.
func Render(tmplContent, outPath string, data any) error {
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(outPath), err)
	}

	if _, err := os.Stat(outPath); err == nil {
		return fmt.Errorf("file already exists: %s", outPath)
	}

	tmpl, err := template.New("").
		Delims(templateDelimLeft, templateDelimRight).
		Parse(tmplContent)
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	f, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("create file %s: %w", outPath, err)
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	if err := tmpl.Execute(w, data); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}
	return w.Flush()
}

// ReadModulePath reads the module path from the go.mod file in the current
// directory (walking up at most 5 levels if not found immediately).
func ReadModulePath() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for i := 0; i < 5; i++ {
		path := filepath.Join(dir, "go.mod")
		data, err := os.ReadFile(path)
		if err == nil {
			return parseModuleLine(string(data))
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("go.mod not found (run wapgo make:* from inside a wapgo project)")
}

// parseModuleLine extracts the module path from go.mod content.
func parseModuleLine(content string) (string, error) {
	for _, line := range splitLines(content) {
		line = trimSpace(line)
		if len(line) > 7 && line[:7] == "module " {
			return trimSpace(line[7:]), nil
		}
	}
	return "", fmt.Errorf("module declaration not found in go.mod")
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i, c := range s {
		if c == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func trimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}
