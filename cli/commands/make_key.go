package commands

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func newMakeKeyCmd() *cobra.Command {
	var (
		length int
		write  bool
	)

	cmd := &cobra.Command{
		Use:   "make:key",
		Short: "Generate a cryptographically-secure random secret key",
		Long: `Generate a random secret key suitable for JWT_SECRET.

Examples:
  wapgo make:key              # print a 32-byte hex key
  wapgo make:key --len 64    # 64-byte key
  wapgo make:key --write      # write key directly into .env (JWT_SECRET=...)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if length < 16 {
				return fmt.Errorf("--len must be at least 16 bytes (got %d)", length)
			}

			b := make([]byte, length)
			if _, err := rand.Read(b); err != nil {
				return fmt.Errorf("generating random bytes: %w", err)
			}
			key := hex.EncodeToString(b)

			if write {
				if err := writeJWTSecret(key); err != nil {
					return err
				}
				fmt.Printf("JWT_SECRET written to .env\n")
				return nil
			}

			fmt.Println(key)
			return nil
		},
	}

	cmd.Flags().IntVar(&length, "len", 32, "Number of random bytes (hex output will be 2× longer)")
	cmd.Flags().BoolVar(&write, "write", false, "Write JWT_SECRET into the local .env file")

	return cmd
}

// writeJWTSecret updates (or appends) JWT_SECRET=<key> in the local .env file.
func writeJWTSecret(key string) error {
	const envFile = ".env"

	raw, err := os.ReadFile(envFile)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading .env: %w", err)
	}

	line := "JWT_SECRET=" + key
	content := string(raw)

	if strings.Contains(content, "JWT_SECRET=") {
		// Replace the existing line in-place.
		lines := strings.Split(content, "\n")
		for i, l := range lines {
			if strings.HasPrefix(l, "JWT_SECRET=") {
				lines[i] = line
				break
			}
		}
		content = strings.Join(lines, "\n")
	} else {
		// Append at the end (add newline if file doesn't end with one).
		if len(content) > 0 && !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		content += line + "\n"
	}

	return os.WriteFile(envFile, []byte(content), 0o600)
}
