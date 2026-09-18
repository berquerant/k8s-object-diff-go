package config_test

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/berquerant/k8s-object-diff-go/config"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPServer(t *testing.T) {
	const (
		manifestLeft = `apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  namespace: default
data:
  key: value1`
		manifestRight = `apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  namespace: default
data:
  key: value2`
	)

	ctx := context.Background()
	var cfg config.Config
	server := cfg.NewMCPServer()

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}, nil)

	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	sessionErrCh := make(chan error, 1)
	go func() {
		sessionErrCh <- server.Run(ctx, serverTransport)
	}()

	clientSession, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	defer clientSession.Close()

	parseResult := func(res *mcp.CallToolResult) (config.DiffToolResult, error) {
		var toolResult config.DiffToolResult
		if res.StructuredContent != nil {
			b, err := json.Marshal(res.StructuredContent)
			if err != nil {
				return toolResult, err
			}
			if err := json.Unmarshal(b, &toolResult); err != nil {
				return toolResult, err
			}
			return toolResult, nil
		}
		if len(res.Content) > 0 {
			if textContent, ok := res.Content[0].(*mcp.TextContent); ok {
				if err := json.Unmarshal([]byte(textContent.Text), &toolResult); err != nil {
					return toolResult, err
				}
				return toolResult, nil
			}
		}
		return toolResult, nil
	}

	t.Run("list tools", func(t *testing.T) {
		tools, err := clientSession.ListTools(ctx, nil)
		require.NoError(t, err)
		require.NotNil(t, tools)
		require.Len(t, tools.Tools, 1)
		assert.Equal(t, "diff_k8s_manifests", tools.Tools[0].Name)
	})

	for _, tc := range []struct {
		name         string
		args         map[string]any
		wantDiff     bool
		wantContains []string
	}{
		{
			name: "call tool with manifest strings",
			args: map[string]any{
				"left":  manifestLeft,
				"right": manifestRight,
				"out":   "text",
			},
			wantDiff: true,
			wantContains: []string{
				"-  key: value1",
				"+  key: value2",
			},
		},
		{
			name: "call tool identical manifests",
			args: map[string]any{
				"left":  manifestLeft,
				"right": manifestLeft,
			},
			wantDiff: false,
		},
		{
			name: "call tool with file paths",
			args: map[string]any{
				"left":  filepath.Join("..", "tests", "diffs", "left.yml"),
				"right": filepath.Join("..", "tests", "diffs", "right.yml"),
				"out":   "markdown",
			},
			wantDiff: true,
			wantContains: []string{
				"Objdiff Summary",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
				Name:      "diff_k8s_manifests",
				Arguments: tc.args,
			})
			require.NoError(t, err)
			require.NotNil(t, res)
			assert.False(t, res.IsError)

			toolResult, err := parseResult(res)
			require.NoError(t, err)
			assert.Equal(t, tc.wantDiff, toolResult.HasDiff)
			if !tc.wantDiff {
				assert.Empty(t, toolResult.Diff)
			}
			for _, sub := range tc.wantContains {
				assert.Contains(t, toolResult.Diff, sub)
			}
		})
	}
}
