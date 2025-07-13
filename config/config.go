package config

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/berquerant/k8s-object-diff-go/internal"
)

type Config struct {
	Context           int    `name:"context" short:"C" default:"3" usage:"diff context"`
	Separator         string `name:"separator" short:"d" default:">" usage:"object id separator"`
	Indent            int    `name:"indent" short:"n" default:"2" usage:"yaml indent"`
	Out               string `name:"out" short:"o" default:"text" usage:"output format: text,yaml,id,idlist"`
	Debug             bool   `name:"debug" usage:"enable debug log"`
	Quiet             bool   `name:"quiet" short:"q" usage:"quiet log"`
	Color             bool   `name:"color" short:"c" usage:"colored diff"`
	DiffSuccess       bool   `name:"success" usage:"exit with 0 even if inputs differ"`
	AllowDuplicateKey bool   `name:"allowDuplicateKey" default:"true" usage:"allow the use of keys with the same name in the same map"`
	DiffCommand       string `name:"diffCmd" short:"x" usage:"invoke this to get diff instead of builtin differ"`
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
