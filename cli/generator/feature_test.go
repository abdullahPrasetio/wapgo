package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStripBuildIgnore(t *testing.T) {
	in := "//go:build ignore\n\npackage redis\n"
	assert.Equal(t, "package redis\n", stripBuildIgnore(in))

	// No guard — unchanged.
	plain := "package redis\n"
	assert.Equal(t, plain, stripBuildIgnore(plain))

	// Guard with no trailing blank line.
	assert.Equal(t, "package x\n", stripBuildIgnore("//go:build ignore\npackage x\n"))
}

func TestScaffold_SkipsDisabledFeatures(t *testing.T) {
	target := filepath.Join(t.TempDir(), "minimal")

	opts := ScaffoldOptions{
		ProjectName: "minimal",
		Module:      "github.com/me/minimal",
		DB:          "postgres",
		APM:         "none",
		// Redis/Kafka/RabbitMQ all false.
	}
	require.NoError(t, Scaffold(TemplateFS, opts, target))

	// Disabled feature directories must not exist.
	for _, p := range []string{
		"internal/repository/redis",
		"pkg/messaging/kafka",
		"pkg/messaging/rabbitmq",
	} {
		_, err := os.Stat(filepath.Join(target, filepath.FromSlash(p)))
		assert.True(t, os.IsNotExist(err), "expected %s to be absent", p)
	}

	// main.go must not reference the disabled features.
	main := readFile(t, filepath.Join(target, "cmd/api/main.go"))
	assert.NotContains(t, main, "go-redis")
	assert.NotContains(t, main, "messaging/kafka")
	assert.NotContains(t, main, "//go:build ignore")

	// docker-compose includes postgres only.
	compose := readFile(t, filepath.Join(target, "docker-compose.yml"))
	assert.Contains(t, compose, "postgres:")
	assert.NotContains(t, compose, "mysql:")
	assert.NotContains(t, compose, "redis:")
	assert.NotContains(t, compose, "kafka:")
}

func TestScaffold_IncludesEnabledFeatures(t *testing.T) {
	target := filepath.Join(t.TempDir(), "full")

	opts := ScaffoldOptions{
		ProjectName: "full",
		Module:      "github.com/me/full",
		DB:          "mysql",
		APM:         "otel",
		Redis:       true,
		Kafka:       true,
		RabbitMQ:    true,
	}
	require.NoError(t, Scaffold(TemplateFS, opts, target))

	for _, p := range []string{
		"internal/repository/redis/cache.go",
		"pkg/messaging/kafka/producer.go",
		"pkg/messaging/rabbitmq/publisher.go",
	} {
		_, err := os.Stat(filepath.Join(target, filepath.FromSlash(p)))
		assert.NoError(t, err, "expected %s to exist", p)
	}

	main := readFile(t, filepath.Join(target, "cmd/api/main.go"))
	assert.Contains(t, main, "go-redis")
	assert.Contains(t, main, `AddChecker("redis"`)
	assert.Contains(t, main, "messaging/kafka")

	// MySQL selected → port 3306, mysql service only.
	env := readFile(t, filepath.Join(target, ".env.example"))
	assert.Contains(t, env, "DB_PORT=3306")
	assert.Contains(t, env, "OBSERVABILITY_PROVIDER=otel")

	compose := readFile(t, filepath.Join(target, "docker-compose.yml"))
	assert.Contains(t, compose, "mysql:")
	assert.NotContains(t, compose, "postgres:")
	assert.Contains(t, compose, "redis:")
}

func TestAddFeatureFiles(t *testing.T) {
	dest := t.TempDir()

	created, err := AddFeatureFiles(TemplateFS, "internal/repository/redis", "github.com/me/app", dest)
	require.NoError(t, err)
	require.NotEmpty(t, created)

	cachePath := filepath.Join(dest, "internal/repository/redis/cache.go")
	content := readFile(t, cachePath)
	assert.NotContains(t, content, "//go:build ignore")

	// Second run is idempotent — nothing recreated.
	created2, err := AddFeatureFiles(TemplateFS, "internal/repository/redis", "github.com/me/app", dest)
	require.NoError(t, err)
	assert.Empty(t, created2)
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(b)
}
