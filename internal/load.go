package internal

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/goccy/go-yaml"
)

func LoadObjects(ctx context.Context, r io.Reader, marshaler Marshaler) ([]*Object, error) {
	m := NewYamlUnmarshaler(r, map[string]any{})
	xs, err := m.Unmarshal(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*Object, len(xs))
	for i, x := range xs {
		v, err := LoadObjectFromMap(ctx, marshaler, x)
		if err != nil {
			return nil, fmt.Errorf("%w: index %d", err, i)
		}
		result[i] = v
	}

	return result, nil
}

var ErrLoadObject = errors.New("LoadObject")

func LoadObjectFromMap(ctx context.Context, marshaler Marshaler, obj map[string]any) (*Object, error) {
	getString := func(key string) (string, error) {
		x, ok := obj[key]
		if !ok {
			return "", fmt.Errorf("%w: %s is missing", ErrLoadObject, key)
		}
		v, ok := x.(string)
		if !ok {
			return "", fmt.Errorf("%w: %s is not a string", ErrLoadObject, key)
		}
		return v, nil
	}

	var h ObjectHeader
	{
		x, err := getString("apiVersion")
		if err != nil {
			return nil, err
		}
		h.APIVersion = x
	}
	{
		x, err := getString("kind")
		if err != nil {
			return nil, err
		}
		h.Kind = x
	}

	meta, ok := obj["metadata"]
	if !ok {
		return nil, fmt.Errorf("%w: metadata is missing", ErrLoadObject)
	}

	var (
		metav  map[string]any
		metav2 map[any]any
	)
	if v, ok := meta.(map[string]any); ok {
		metav = v
	}
	if metaMapSlice, ok := meta.(yaml.MapSlice); ok {
		metav2 = metaMapSlice.ToMap()
	}
	if len(metav) == 0 && len(metav2) == 0 {
		return nil, fmt.Errorf("%w: metadata is invalid: %#v", ErrLoadObject, meta)
	}

	getString2 := func(key string) (string, error) {
		var (
			x  any
			ok bool
		)
		if x, ok = metav[key]; !ok {
			x, ok = metav2[key]
		}
		if !ok {
			return "", fmt.Errorf("%w: %s is missing", ErrLoadObject, key)
		}
		v, ok := x.(string)
		if !ok {
			return "", fmt.Errorf("%w: %s is not a string", ErrLoadObject, key)
		}
		return v, nil
	}
	{
		x, err := getString2("namespace")
		if err != nil {
			x = ""
		}
		h.Metadata.Namespace = x
	}
	{
		x, err := getString2("name")
		if err != nil {
			return nil, err
		}
		h.Metadata.Name = x
	}

	b, err := marshaler.Marshal(ctx, obj)
	if err != nil {
		return nil, errors.Join(ErrLoadObject, fmt.Errorf("%w: failed to marshal", err))
	}

	return &Object{
		Header: h,
		Body:   string(b),
	}, nil
}
