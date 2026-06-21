package config

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/berquerant/k8s-object-diff-go/internal"
)

type Config struct {
	Context           int
	Separator         string
	Indent            int
	Out               string
	Debug             bool
	Quiet             bool
	Color             bool
	DiffSuccess       bool
	AllowDuplicateKey bool
	DiffCommand       string
	Verbose           bool
	Labels            []string
}

type OutMode string

const (
	OutModeUnknown OutMode = "unknown"
	OutModeText    OutMode = "text"
	OutModeYaml    OutMode = "yaml"
	OutModeID      OutMode = "id"
	OutModeIDList  OutMode = "idlist"
)

func (c *Config) OutMode() OutMode {
	switch c.Out {
	case string(OutModeText):
		return OutModeText
	case string(OutModeYaml):
		return OutModeYaml
	case string(OutModeID):
		return OutModeID
	case string(OutModeIDList):
		return OutModeIDList
	default:
		return OutModeUnknown
	}
}

var ErrNoDiffCommand = errors.New("NoDiffCommand")

func (c *Config) diffCommand() ([]string, error) {
	if len(c.DiffCommand) == 0 {
		return nil, ErrNoDiffCommand
	}
	xs := strings.Split(internal.EscapeCommand(c.DiffCommand), " ")
	if len(xs) == 0 {
		panic("unreachable: diffCommand specified but no command")
	}
	head, err := exec.LookPath(xs[0])
	if err != nil {
		return nil, fmt.Errorf("lookup %s: %w", xs[0], err)
	}
	xs[0] = head
	return xs, nil
}

func (c *Config) newDiffer() (internal.Differ, error) {
	cmd, err := c.diffCommand()
	switch {
	case err == nil:
		return internal.NewProcessDiffer(cmd[0], cmd[1:]), nil
	case errors.Is(err, ErrNoDiffCommand):
		return internal.NewDMPDiffer(), nil
	default:
		return nil, err
	}
}
