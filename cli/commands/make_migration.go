package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/abdullahPrasetio/wapgo/cli/generator"
	"github.com/spf13/cobra"
)

const migrationUpSQL = `-- Migration up: {{table}}
-- Compatible with MySQL 8+ and PostgreSQL 14+.
-- Postgres: replace VARCHAR(36) → UUID, TIMESTAMP(3) → TIMESTAMPTZ.

CREATE TABLE IF NOT EXISTS {{table}} (
    id         VARCHAR(36)  NOT NULL,
    created_at TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    deleted_at TIMESTAMP(3) NULL,
{{columns}}
    PRIMARY KEY (id),
    INDEX idx_{{table}}_deleted_at (deleted_at)
);
`

const migrationDownSQL = `-- Migration down: {{table}}

DROP TABLE IF EXISTS {{table}};
`

func newMakeMigrationCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "make:migration <name>",
		Short: "Generate a timestamped SQL migration file pair (up + down)",
		Long: `Generate a timestamped up/down SQL migration pair in migrations/.

The file names follow the golang-migrate convention:
  {timestamp}_{name}.up.sql
  {timestamp}_{name}.down.sql

Example:
  wapgo make:migration create_orders`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMakeMigration(args[0])
		},
	}
}

func runMakeMigration(name string) error {
	n := generator.NewNames(name)
	ts := time.Now().UTC().Format("20060102150405")
	prefix := ts + "_" + n.Snake

	fields := generator.ParseEntityFields(n.Snake)
	columns := generator.RenderMigrationColumns(fields)
	if len(fields) > 0 {
		fmt.Printf("  detected %d field(s) from entity/%s.go\n", len(fields), n.Snake)
	}

	r := strings.NewReplacer("{{table}}", n.Table, "{{columns}}", columns)
	upPath := filepath.Join("migrations", prefix+".up.sql")
	downPath := filepath.Join("migrations", prefix+".down.sql")

	if err := writeMigrationFile(upPath, r.Replace(migrationUpSQL)); err != nil {
		return err
	}
	fmt.Printf("  created  %s\n", upPath)

	if err := writeMigrationFile(downPath, r.Replace(migrationDownSQL)); err != nil {
		return err
	}
	fmt.Printf("  created  %s\n", downPath)

	return nil
}

func writeMigrationFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(path), err)
	}
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("file already exists: %s", path)
	}
	return os.WriteFile(path, []byte(content), 0o644)
}
