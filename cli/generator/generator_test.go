package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRender_Basic(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "out.go")

	err := Render("package [[.Package]]\n\nconst Name = \"[[.Name]]\"", out, map[string]string{
		"Package": "example",
		"Name":    "hello",
	})
	require.NoError(t, err)

	data, err := os.ReadFile(out)
	require.NoError(t, err)
	assert.Contains(t, string(data), "package example")
	assert.Contains(t, string(data), `const Name = "hello"`)
}

func TestRender_CreatesDirectories(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "sub", "dir", "file.go")

	err := Render("package x", out, nil)
	require.NoError(t, err)

	_, err = os.Stat(out)
	assert.NoError(t, err)
}

func TestRender_ErrorOnExistingFile(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "existing.go")
	require.NoError(t, os.WriteFile(out, []byte("exists"), 0o644))

	err := Render("package x", out, nil)
	assert.ErrorContains(t, err, "file already exists")
}

func TestRender_InvalidTemplate(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "bad.go")

	err := Render("[[.Missing", out, nil)
	assert.Error(t, err)
}

func TestReadModulePath_FromModFile(t *testing.T) {
	dir := t.TempDir()
	gomod := "module github.com/test/myapp\n\ngo 1.22\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(gomod), 0o644))

	orig, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { os.Chdir(orig) })

	mod, err := ReadModulePath()
	require.NoError(t, err)
	assert.Equal(t, "github.com/test/myapp", mod)
}

func TestReadModulePath_NotFound(t *testing.T) {
	dir := t.TempDir()
	orig, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { os.Chdir(orig) })

	_, err = ReadModulePath()
	assert.Error(t, err)
}

func TestParseModuleLine_Valid(t *testing.T) {
	mod, err := parseModuleLine("module github.com/me/svc\n\ngo 1.22\n")
	require.NoError(t, err)
	assert.Equal(t, "github.com/me/svc", mod)
}

func TestParseModuleLine_MissingDeclaration(t *testing.T) {
	_, err := parseModuleLine("go 1.22\n")
	assert.Error(t, err)
}

func TestTrimSpace(t *testing.T) {
	assert.Equal(t, "hello", trimSpace("  hello  "))
	assert.Equal(t, "hello", trimSpace("\thello\t"))
	assert.Equal(t, "", trimSpace("   "))
}

func TestSplitLines(t *testing.T) {
	lines := splitLines("a\nb\nc")
	assert.Equal(t, []string{"a", "b", "c"}, lines)
}

func TestSplitLines_EmptyString(t *testing.T) {
	lines := splitLines("")
	assert.Equal(t, []string(nil), lines)
}
