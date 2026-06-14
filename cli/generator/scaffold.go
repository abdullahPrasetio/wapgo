package generator

import (
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
	APM         string // "elastic_apm" | "otel" | "none"
	Redis       bool   // include Redis cache layer
	Kafka       bool   // include Kafka producer/consumer
	RabbitMQ    bool   // include RabbitMQ publisher/consumer
	Email       bool   // include SMTP email add-on
	Firebase    bool   // include Firebase FCM add-on
	GoogleAuth  bool   // include Google OAuth2 add-on
}

// templateData is the value passed to every skeleton .tmpl file.
type templateData struct {
	Module     string
	AppName    string
	DB         string
	APM        string
	Redis      bool
	Kafka      bool
	RabbitMQ   bool
	Email      bool
	Firebase   bool
	GoogleAuth bool
}

// skipPath reports whether a skeleton-relative path belongs to a disabled
// feature and should therefore be omitted from the generated project.
func (o ScaffoldOptions) skipPath(rel string) bool {
	rel = filepath.ToSlash(rel)
	switch {
	case !o.Redis && (rel == "internal/repository/redis" || strings.HasPrefix(rel, "internal/repository/redis/")):
		return true
	case !o.Kafka && (rel == "pkg/messaging/kafka" || strings.HasPrefix(rel, "pkg/messaging/kafka/")):
		return true
	case !o.RabbitMQ && (rel == "pkg/messaging/rabbitmq" || strings.HasPrefix(rel, "pkg/messaging/rabbitmq/")):
		return true
	case !o.Email && (rel == "pkg/notification/email" || strings.HasPrefix(rel, "pkg/notification/email/")):
		return true
	case !o.Firebase && (rel == "pkg/notification/firebase" || strings.HasPrefix(rel, "pkg/notification/firebase/")):
		return true
	case !o.GoogleAuth && (rel == "pkg/auth/google" || strings.HasPrefix(rel, "pkg/auth/google/")):
		return true
	case !o.GoogleAuth && rel == "internal/delivery/http/handler/google_auth_handler.go.tmpl":
		return true
	case !o.GoogleAuth && rel == "internal/delivery/http/route/google_auth_route.go.tmpl":
		return true
	}
	return false
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
	if opts.APM == "" {
		opts.APM = "none"
	}
	appName := strings.ReplaceAll(opts.ProjectName, "_", "-")

	data := templateData{
		Module:     opts.Module,
		AppName:    appName,
		DB:         opts.DB,
		APM:        opts.APM,
		Redis:      opts.Redis,
		Kafka:      opts.Kafka,
		RabbitMQ:   opts.RabbitMQ,
		Email:      opts.Email,
		Firebase:   opts.Firebase,
		GoogleAuth: opts.GoogleAuth,
	}

	return fs.WalkDir(fsys, skeletonDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Compute relative path under skeleton root.
		rel, _ := filepath.Rel(skeletonDir, path)

		// Omit files/dirs belonging to disabled features.
		if rel != "." && opts.skipPath(rel) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

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
			// Process as template: [[ .Module ]], [[ .AppName ]], [[ .DB ]],
			// [[ .APM ]], and the [[ if .Redis ]] / [[ if .Kafka ]] / [[ if .RabbitMQ ]] flags.
			tmpl, err := template.New("").
				Delims(templateDelimLeft, templateDelimRight).
				Parse(content)
			if err != nil {
				return fmt.Errorf("parse skeleton template %s: %w", path, err)
			}

			var buf strings.Builder
			if terr := tmpl.Execute(&buf, data); terr != nil {
				return fmt.Errorf("execute skeleton template %s: %w", path, terr)
			}
			rendered := strings.ReplaceAll(buf.String(), skeletonModule, opts.Module)

			if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
				return err
			}
			return os.WriteFile(outPath, []byte(rendered), 0o644)
		}

		// Plain file — replace the module path placeholder and drop the
		// //go:build ignore guard that keeps skeleton files out of the CLI build.
		content = strings.ReplaceAll(content, skeletonModule, opts.Module)
		content = stripBuildIgnore(content)

		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			return err
		}
		return os.WriteFile(outPath, []byte(content), 0o644)
	})
}

// stripBuildIgnore removes a leading `//go:build ignore` constraint (and the
// following blank line) so generated Go files actually compile. Skeleton files
// carry this guard only so they are ignored while building the CLI module.
func stripBuildIgnore(content string) string {
	const guard = "//go:build ignore\n"
	if rest, ok := strings.CutPrefix(content, guard); ok {
		content = strings.TrimPrefix(rest, "\n")
	}
	return content
}

// AddFeatureFiles copies a skeleton subtree (relative to the skeleton root)
// into destRoot, substituting the module path and stripping the build guard.
// Existing files are never overwritten. Returns the created relative paths.
func AddFeatureFiles(fsys fs.FS, subdir, module, destRoot string) ([]string, error) {
	var created []string
	root := skeletonDir + "/" + subdir

	err := fs.WalkDir(fsys, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(skeletonDir, path)
		outPath := filepath.Join(destRoot, filepath.FromSlash(rel))

		if d.IsDir() {
			return os.MkdirAll(outPath, 0o755)
		}

		outPath = strings.TrimSuffix(outPath, ".tmpl")
		if _, statErr := os.Stat(outPath); statErr == nil {
			return nil // never overwrite existing files
		}

		raw, err := fs.ReadFile(fsys, path)
		if err != nil {
			return fmt.Errorf("read skeleton file %s: %w", path, err)
		}
		content := strings.ReplaceAll(string(raw), skeletonModule, module)
		content = stripBuildIgnore(content)

		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(outPath, []byte(content), 0o644); err != nil {
			return err
		}
		relCreated, _ := filepath.Rel(destRoot, outPath)
		created = append(created, filepath.ToSlash(relCreated))
		return nil
	})
	return created, err
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
