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
	apiVersion, err := getRequiredString(obj, "apiVersion")
	if err != nil {
		return nil, err
	}
	kind, err := getRequiredString(obj, "kind")
	if err != nil {
		return nil, err
	}

	metaMap, err := extractMetadataMap(obj)
	if err != nil {
		return nil, err
	}

	name, err := getRequiredString(metaMap, "name")
	if err != nil {
		return nil, err
	}
	namespace, _ := getRequiredString(metaMap, "namespace")

	h := ObjectHeader{
		APIVersion: apiVersion,
		Kind:       kind,
		Metadata: ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
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

func getRequiredString(m map[string]any, key string) (string, error) {
	x, ok := m[key]
	if !ok {
		return "", fmt.Errorf("%s is missing: %w", key, ErrLoadObject)
	}
	v, ok := x.(string)
	if !ok {
		return "", fmt.Errorf("%s is not a string: %w", key, ErrLoadObject)
	}
	return v, nil
}

func extractMetadataMap(obj map[string]any) (map[string]any, error) {
	meta, ok := obj["metadata"]
	if !ok {
		return nil, fmt.Errorf("metadata is missing: %w", ErrLoadObject)
	}
	if v, ok := meta.(map[string]any); ok && len(v) > 0 {
		return v, nil
	}
	if metaMapSlice, ok := meta.(yaml.MapSlice); ok {
		converted := make(map[string]any, len(metaMapSlice))
		for _, item := range metaMapSlice {
			if k, ok := item.Key.(string); ok {
				converted[k] = item.Value
			}
		}
		if len(converted) > 0 {
			return converted, nil
		}
	}
	return nil, fmt.Errorf("metadata is invalid: %#v: %w", meta, ErrLoadObject)
}

