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

func (s ObjectHeader) IntoID(sep string) string {
	return strings.Join([]string{
		s.APIVersion,
		s.Kind,
		s.Metadata.Namespace,
		s.Metadata.Name,
	}, sep)
}

type Object struct {
	Header ObjectHeader
	Body   string
}
