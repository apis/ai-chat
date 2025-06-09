package web

import "embed"

var (
	//go:embed templates
	TemplateFS embed.FS

	//go:embed frontend/dist
	EmbedFs embed.FS
)
