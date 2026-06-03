package generator_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abdullahPrasetio/wapgo/cli/generator"
)

const sampleEntity = `package entity

import (
	"time"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Product struct {
	ID          uuid.UUID      ` + "`gorm:\"primaryKey\"`" + `
	Name        string
	Price       float64
	Stock       int
	IsActive    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt
}
`

func TestParseEntityFields(t *testing.T) {
	dir := t.TempDir()
	entityDir := filepath.Join(dir, "internal", "domain", "entity")
	require.NoError(t, os.MkdirAll(entityDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(entityDir, "product.go"), []byte(sampleEntity), 0o644))

	// ParseEntityFields reads relative to CWD — chdir to temp dir
	orig, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(orig) })

	fields := generator.ParseEntityFields("product")
	require.NotNil(t, fields)

	names := make(map[string]string)
	for _, f := range fields {
		names[f.Name] = f.SQLType
	}

	// Base fields (ID, CreatedAt, UpdatedAt, DeletedAt) must be skipped
	assert.NotContains(t, names, "ID")
	assert.NotContains(t, names, "CreatedAt")
	assert.NotContains(t, names, "DeletedAt")

	// Domain fields must be present with correct SQL types
	assert.Equal(t, "VARCHAR(255) NOT NULL", names["Name"])
	assert.Equal(t, "DECIMAL(18,4) NOT NULL DEFAULT 0", names["Price"])
	assert.Equal(t, "INT NOT NULL DEFAULT 0", names["Stock"])
	assert.Equal(t, "TINYINT(1) NOT NULL DEFAULT 0", names["IsActive"])
}

func TestParseEntityFields_MissingFile(t *testing.T) {
	orig, _ := os.Getwd()
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(orig) })

	fields := generator.ParseEntityFields("nonexistent")
	assert.Nil(t, fields)
}

func TestRenderMigrationColumns_NoFields(t *testing.T) {
	out := generator.RenderMigrationColumns(nil)
	assert.Contains(t, out, "TODO")
}

func TestRenderMigrationColumns_WithFields(t *testing.T) {
	fields := []generator.EntityField{
		{Name: "Name", SQLName: "name", SQLType: "VARCHAR(255) NOT NULL"},
		{Name: "Price", SQLName: "price", SQLType: "DECIMAL(18,4) NOT NULL DEFAULT 0"},
	}
	out := generator.RenderMigrationColumns(fields)
	assert.Contains(t, out, "name")
	assert.Contains(t, out, "VARCHAR(255)")
	assert.Contains(t, out, "price")
	assert.Contains(t, out, "DECIMAL")
}
