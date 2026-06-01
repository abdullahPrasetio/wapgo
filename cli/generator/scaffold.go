package generator

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

const (
	// skeletonModule is the placeholder module path embedded in skeleton files.
	skeletonModule = "github.com/abdullahPrasetio/wapgo"
	skeletonDir    = "templates/skeleton"
	domainDir      = "templates/domain"
)

// ScaffoldOptions holds input for wapgo new.
type ScaffoldOptions struct {
	ProjectName string // e.g. "my-service" (used as dir name and APP_NAME default)
	Module      string // e.g. "github.com/me/my-service"
	DB          string // "postgres" | "mysql"
}

// Scaffold copies the embedded skeleton to targetDir, substituting module path.
// targetDir must not already exist.
func Scaffold(fsys fs.FS, opts ScaffoldOptions, targetDir string) error {
	if _, err := os.Stat(targetDir); err == nil {
		return fmt.Errorf("directory already exists: %s", targetDir)
	}

	if opts.DB == "" {
		opts.DB = "postgres"
	}
	appName := strings.ReplaceAll(opts.ProjectName, "_", "-")

	return fs.WalkDir(fsys, skeletonDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Compute relative path under skeleton root.
		rel, _ := filepath.Rel(skeletonDir, path)
		outPath := filepath.Join(targetDir, filepath.FromSlash(rel))

		if d.IsDir() {
			return os.MkdirAll(outPath, 0o755)
		}

		raw, err := fs.ReadFile(fsys, path)
		if err != nil {
			return fmt.Errorf("read skeleton file %s: %w", path, err)
		}

		// Determine actual output filename (strip .tmpl suffix if present).
		outBase := filepath.Base(outPath)
		if strings.HasSuffix(outBase, ".tmpl") {
			outPath = strings.TrimSuffix(outPath, ".tmpl")
		}

		content := string(raw)

		if strings.HasSuffix(path, ".tmpl") {
			// Process as template: [[ .Module ]], [[ .AppName ]], [[ .DB ]].
			data := struct {
				Module  string
				AppName string
				DB      string
			}{Module: opts.Module, AppName: appName, DB: opts.DB}

			tmpl, err := template.New("").
				Delims(templateDelimLeft, templateDelimRight).
				Parse(content)
			if err != nil {
				return fmt.Errorf("parse skeleton template %s: %w", path, err)
			}

			if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
				return err
			}
			f, err := os.Create(outPath)
			if err != nil {
				return err
			}
			w := bufio.NewWriter(f)
			if terr := tmpl.Execute(w, data); terr != nil {
				f.Close()
				return fmt.Errorf("execute skeleton template %s: %w", path, terr)
			}
			if ferr := w.Flush(); ferr != nil {
				f.Close()
				return ferr
			}
			return f.Close()
		}

		// Plain file — just replace the module path placeholder.
		content = strings.ReplaceAll(content, skeletonModule, opts.Module)

		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			return err
		}
		return os.WriteFile(outPath, []byte(content), 0o644)
	})
}

// DomainTemplateContent reads a named domain template from the embedded FS.
// name is the bare filename without directory prefix (e.g. "entity.go.tmpl").
func DomainTemplateContent(fsys fs.FS, name string) (string, error) {
	path := domainDir + "/" + name
	raw, err := fs.ReadFile(fsys, path)
	if err != nil {
		return "", fmt.Errorf("read domain template %s: %w", name, err)
	}
	return string(raw), nil
}
