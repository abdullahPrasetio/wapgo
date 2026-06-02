package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunMakeMigration_CreatesUpAndDown(t *testing.T) {
	dir := t.TempDir()
	orig, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { os.Chdir(orig) })

	require.NoError(t, runMakeMigration("create_orders"))

	entries, err := os.ReadDir(filepath.Join(dir, "migrations"))
	require.NoError(t, err)
	require.Len(t, entries, 2)

	names := []string{entries[0].Name(), entries[1].Name()}
	var upFile, downFile string
	for _, n := range names {
		if strings.HasSuffix(n, ".up.sql") {
			upFile = n
		}
		if strings.HasSuffix(n, ".down.sql") {
			downFile = n
		}
	}
	assert.NotEmpty(t, upFile, "expected .up.sql file")
	assert.NotEmpty(t, downFile, "expected .down.sql file")

	// Both files should share the same timestamp_name prefix.
	upBase := strings.TrimSuffix(upFile, ".up.sql")
	downBase := strings.TrimSuffix(downFile, ".down.sql")
	assert.Equal(t, upBase, downBase)

	// Prefix: 14-digit timestamp + "_create_orders".
	assert.True(t, strings.HasSuffix(upBase, "_create_orders"), "expected snake_case name suffix")
	assert.Len(t, strings.SplitN(upBase, "_", 2)[0], 14, "expected 14-digit timestamp")
}

func TestRunMakeMigration_ContentContainsTableName(t *testing.T) {
	dir := t.TempDir()
	orig, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { os.Chdir(orig) })

	require.NoError(t, runMakeMigration("product"))

	entries, err := os.ReadDir(filepath.Join(dir, "migrations"))
	require.NoError(t, err)

	for _, e := range entries {
		data, err := os.ReadFile(filepath.Join(dir, "migrations", e.Name()))
		require.NoError(t, err)
		assert.Contains(t, string(data), "products", "table name should appear in migration file")
	}
}

func TestRunMakeMigration_PascalCaseInput(t *testing.T) {
	dir := t.TempDir()
	orig, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { os.Chdir(orig) })

	require.NoError(t, runMakeMigration("OrderItem"))

	entries, err := os.ReadDir(filepath.Join(dir, "migrations"))
	require.NoError(t, err)
	require.Len(t, entries, 2)

	for _, e := range entries {
		assert.Contains(t, e.Name(), "order_item", "should convert PascalCase to snake_case")
	}
}

func TestRunMakeMigration_ErrorOnDuplicate(t *testing.T) {
	dir := t.TempDir()
	orig, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { os.Chdir(orig) })

	// Pre-create the up file to simulate a duplicate.
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "migrations"), 0o755))

	// We can't easily predict the exact timestamp, so test writeMigrationFile directly.
	path := filepath.Join(dir, "migrations", "existing.up.sql")
	require.NoError(t, os.WriteFile(path, []byte("exists"), 0o644))

	err = writeMigrationFile(path, "new content")
	assert.ErrorContains(t, err, "file already exists")
}

func TestNewMakeMigrationCmd_Registration(t *testing.T) {
	cmd := newMakeMigrationCmd()
	assert.Equal(t, "make:migration <name>", cmd.Use)
	assert.NoError(t, cmd.Args(cmd, []string{"x"}), "exactly one arg should be accepted")
	assert.Error(t, cmd.Args(cmd, []string{}), "zero args should be rejected")
}
