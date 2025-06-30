package internal

import (
	"context"
	"io"
	"log/slog"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/parser"
)

type Marshaler interface {
	Marshal(ctx context.Context, v any) ([]byte, error)
}

var _ Marshaler = &YamlMarshaler{}

type Unmarshaler[T any] interface {
	// Unmarshal reads all documents and unmarshal them as a list of T.
	Unmarshal(ctx context.Context) ([]T, error)
}

var _ Unmarshaler[string] = &YamlUnmarshaler[string]{}

type YamlMarshaler struct {
	indent                  int
	literalStyleIfMultiline bool
}

func NewYamlMarshaler(indent int, literalStyleIfMultiline bool) *YamlMarshaler {
	return &YamlMarshaler{
		indent:                  indent,
		literalStyleIfMultiline: literalStyleIfMultiline,
	}
}

func (y *YamlMarshaler) Marshal(ctx context.Context, v any) ([]byte, error) {
	return yaml.MarshalContext(
		ctx,
		v,
		yaml.Indent(y.indent),
		yaml.UseLiteralStyleIfMultiline(y.literalStyleIfMultiline),
	)
}

type YamlUnmarshaler[T any] struct {
	r io.Reader
	t T
}

func NewYamlUnmarshaler[T any](r io.Reader, t T) *YamlUnmarshaler[T] {
	return &YamlUnmarshaler[T]{
		r: r,
		t: t,
	}
}

func (y *YamlUnmarshaler[T]) Unmarshal(ctx context.Context) ([]T, error) {
	b, err := io.ReadAll(y.r)
	if err != nil {
		return nil, err
	}
	fileNode, err := parser.ParseBytes(b, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	result := []T{}
	for i, d := range fileNode.Docs {
		if d.Body == nil {
			slog.Debug("skip load document due to empty", slog.Int("index", i))
			continue
		}
		t := new(T)
		decoder := yaml.NewDecoder(strings.NewReader(d.String()), yaml.UseOrderedMap())
		err := decoder.DecodeFromNode(d.Body, t)
		if err != nil {
			slog.Debug(
				"failed to load document",
				slog.Int("index", i),
				slog.Any("err", err),
			)
			continue
		}
		result = append(result, *t)
	}
	return result, nil
}
