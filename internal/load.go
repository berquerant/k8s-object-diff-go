package internal

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/goccy/go-yaml"
)

func LoadObjects(ctx context.Context, r io.Reader, marshaler Marshaler, allowDuplicteMapKey bool) ([]*Object, error) {
	m := NewYamlUnmarshaler(r, map[string]any{}, allowDuplicteMapKey)
	xs, err := m.Unmarshal(ctx)
	if err != nil {
		return nil, fmt.Errorf("load objects: %w", err)
	}

	result := make([]*Object, len(xs))
	for i, x := range xs {
		v, err := LoadObjectFromMap(ctx, marshaler, x)
		if err != nil {
			return nil, fmt.Errorf("load obejcts: index %d: %w", i, err)
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
			return "", fmt.Errorf("%s is missing: %w", key, ErrLoadObject)
		}
		v, ok := x.(string)
		if !ok {
			return "", fmt.Errorf("%s is not a string: %w", key, ErrLoadObject)
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
		return nil, fmt.Errorf("metadata is missing: %w", ErrLoadObject)
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
		return nil, fmt.Errorf("metadata is invalid: %#v: %w", meta, ErrLoadObject)
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
			return "", fmt.Errorf("%s is missing: %w", key, ErrLoadObject)
		}
		v, ok := x.(string)
		if !ok {
			return "", fmt.Errorf("%s is not a string: %w", key, ErrLoadObject)
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
		return nil, fmt.Errorf("failed to marshal: %w", errors.Join(err, ErrLoadObject))
	}

	return &Object{
		Header: h,
		Body:   string(b),
	}, nil
}
