package internal

import (
	"strings"
)

// ObjectHeader is a seed of the Object ID.
type ObjectHeader struct {
	APIVersion string     `yaml:"apiVersion"`
	Kind       string     `yaml:"kind"`
	Metadata   ObjectMeta `yaml:"metadata"`
}

type ObjectMeta struct {
	Namespace string `yaml:"namespace"`
	Name      string `yaml:"name"`
}

func escapeSep(s, sep string) string {
	if sep == "" || !strings.Contains(s, sep) {
		return s
	}
	return strings.ReplaceAll(s, sep, `\`+sep)
}

func (s ObjectHeader) IntoID(sep string) string {
	return strings.Join([]string{
		escapeSep(s.APIVersion, sep),
		escapeSep(s.Kind, sep),
		escapeSep(s.Metadata.Namespace, sep),
		escapeSep(s.Metadata.Name, sep),
	}, sep)
}

type Object struct {
	Header ObjectHeader
	Body   string
}
