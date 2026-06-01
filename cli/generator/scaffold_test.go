package generator

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildTestFS creates an in-memory FS that mimics the expected skeleton layout.
func buildTestFS() fstest.MapFS {
	return fstest.MapFS{
		// .tmpl file — processed by template engine
		"templates/skeleton/go.mod.tmpl": {Data: []byte("module [[.Module]]\n\ngo 1.25.0\n")},
		// .tmpl file that uses AppName
		"templates/skeleton/.env.example.tmpl": {Data: []byte("APP_NAME=[[.AppName]]\nDB_DRIVER=[[.DB]]\n")},
		// plain file — only module replacement
		"templates/skeleton/pkg/logger/logger.go": {Data: []byte("package logger\n// module: github.com/abdullahPrasetio/wapgo\n")},
		// domain template
		"templates/domain/entity.go.tmpl": {Data: []byte("package entity\ntype [[.Pascal]] struct{}\n")},
	}
}

func TestScaffold_CreatesFiles(t *testing.T) {
	fsys := buildTestFS()
	dir := t.TempDir()
	target := filepath.Join(dir, "myapp")

	opts := ScaffoldOptions{
		ProjectName: "myapp",
		Module:      "github.com/me/myapp",
		DB:          "postgres",
	}
	require.NoError(t, Scaffold(fsys, opts, target))

	// go.mod should be created from .tmpl (extension stripped)
	data, err := os.ReadFile(filepath.Join(target, "go.mod"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "module github.com/me/myapp")

	// .env.example should contain substituted AppName
	data, err = os.ReadFile(filepath.Join(target, ".env.example"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "APP_NAME=myapp")
	assert.Contains(t, string(data), "DB_DRIVER=postgres")

	// plain Go file should have module path replaced
	data, err = os.ReadFile(filepath.Join(target, "pkg", "logger", "logger.go"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "github.com/me/myapp")
	assert.NotContains(t, string(data), skeletonModule)
}

func TestScaffold_ErrorIfTargetExists(t *testing.T) {
	fsys := buildTestFS()
	dir := t.TempDir()
	target := filepath.Join(dir, "existing")
	require.NoError(t, os.Mkdir(target, 0o755))

	err := Scaffold(fsys, ScaffoldOptions{Module: "m", ProjectName: "existing"}, target)
	assert.ErrorContains(t, err, "already exists")
}

func TestScaffold_DefaultDB(t *testing.T) {
	fsys := buildTestFS()
	dir := t.TempDir()
	target := filepath.Join(dir, "proj")

	opts := ScaffoldOptions{
		ProjectName: "proj",
		Module:      "github.com/me/proj",
		// DB not set — should default to postgres
	}
	require.NoError(t, Scaffold(fsys, opts, target))

	data, err := os.ReadFile(filepath.Join(target, ".env.example"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "DB_DRIVER=postgres")
}

func TestScaffold_AppNameDerivedFromProjectName(t *testing.T) {
	fsys := buildTestFS()
	dir := t.TempDir()
	target := filepath.Join(dir, "my_project")

	opts := ScaffoldOptions{
		ProjectName: "my_project",
		Module:      "github.com/me/my-project",
		DB:          "mysql",
	}
	require.NoError(t, Scaffold(fsys, opts, target))

	data, err := os.ReadFile(filepath.Join(target, ".env.example"))
	require.NoError(t, err)
	// underscores should become hyphens in APP_NAME
	assert.Contains(t, string(data), "APP_NAME=my-project")
}

func TestDomainTemplateContent_Exists(t *testing.T) {
	fsys := buildTestFS()
	content, err := DomainTemplateContent(fsys, "entity.go.tmpl")
	require.NoError(t, err)
	assert.Contains(t, content, "[[.Pascal]]")
}

func TestDomainTemplateContent_Missing(t *testing.T) {
	fsys := buildTestFS()
	_, err := DomainTemplateContent(fsys, "nonexistent.go.tmpl")
	assert.Error(t, err)
}
