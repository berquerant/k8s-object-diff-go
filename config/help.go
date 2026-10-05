package config

import (
	_ "embed"
	"strings"
	"sync"
	"text/template"
)

//go:embed help.tmpl
var helpTemplateSource string

var (
	helpTmplOnce sync.Once
	helpTmpl     *template.Template
)

func getHelpTemplate() *template.Template {
	helpTmplOnce.Do(func() {
		helpTmpl = template.Must(template.New("help").Parse(helpTemplateSource))
	})
	return helpTmpl
}

// HelpMarkdown returns the markdown help string.
func HelpMarkdown() string {
	var sb strings.Builder
	_ = getHelpTemplate().Execute(&sb, nil)
	return sb.String()
}

// Usage returns the CLI usage string.
func Usage() string {
	return HelpMarkdown()
}
