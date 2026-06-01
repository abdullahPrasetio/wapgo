package generator

import "embed"

// TemplateFS holds all CLI templates (skeleton + domain) embedded at build time.
//
// all: prefix ensures dotfiles (.env.example.tmpl, .gitignore) are included.
//
//go:embed all:templates
var TemplateFS embed.FS
