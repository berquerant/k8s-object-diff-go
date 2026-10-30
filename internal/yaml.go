package internal

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
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
	b, err := yaml.MarshalContext(
		ctx,
		v,
		yaml.Indent(y.indent),
		yaml.UseLiteralStyleIfMultiline(y.literalStyleIfMultiline),
	)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}
	return b, nil
}

type YamlUnmarshaler[T any] struct {
	r                    io.Reader
	t                    T
	allowDuplicateMapKey bool
}

func NewYamlUnmarshaler[T any](r io.Reader, t T, allowDuplicateMapKey bool) *YamlUnmarshaler[T] {
	return &YamlUnmarshaler[T]{
		r:                    r,
		t:                    t,
		allowDuplicateMapKey: allowDuplicateMapKey,
	}
}

const maxManifestReadLimit = 50 * 1024 * 1024 // 50MB

func (y *YamlUnmarshaler[T]) Unmarshal(ctx context.Context) ([]T, error) {
	limitedReader := io.LimitReader(y.r, maxManifestReadLimit+1)
	b, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("unmarshal read: %w", err)
	}
	if len(b) > maxManifestReadLimit {
		return nil, fmt.Errorf("manifest exceeds maximum allowed size of %d bytes", maxManifestReadLimit)
	}

	var opts []parser.Option
	if y.allowDuplicateMapKey {
		opts = append(opts, parser.AllowDuplicateMapKey())
	}
	fileNode, err := parser.ParseBytes(b, parser.ParseComments, opts...)
	if err != nil {
		return nil, fmt.Errorf("unmarshal parse: %w", err)
	}

	decoderOpts := []yaml.DecodeOption{
		yaml.UseOrderedMap(),
	}
	if y.allowDuplicateMapKey {
		decoderOpts = append(decoderOpts, yaml.AllowDuplicateMapKey())
	}
	result := []T{}
	for i, d := range fileNode.Docs {
		if d.Body == nil {
			slog.Debug("skip to load document due to empty", slog.Int("index", i))
			continue
		}
		switch d.Body.(type) {
		case *ast.CommentNode, *ast.CommentGroupNode:
			slog.Debug("skip to load document due to comment only", slog.Int("index", i))
			continue
		}
		t := new(T)
		decoder := yaml.NewDecoder(strings.NewReader(d.String()), decoderOpts...)
		err := decoder.DecodeFromNode(d.Body, t)
		if err != nil {
			slog.Debug("failed to load document", slog.Int("index", i), slog.Any("err", err))
			continue
		}
		result = append(result, *t)
	}
	return result, nil
}
