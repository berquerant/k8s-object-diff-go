package config

import (
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/berquerant/k8s-object-diff-go/internal"
)

type Config struct {
	Context              int       `name:"context" short:"C" default:"3" usage:"diff context"`
	Separator            string    `name:"separator" short:"d" default:">" usage:"object id separator"`
	Indent               int       `name:"indent" short:"n" default:"2" usage:"yaml indent"`
	Out                  string    `name:"out" short:"o" default:"text" usage:"output format: text,yaml,id,idlist,markdown"`
	Debug                bool      `name:"debug" usage:"enable debug log"`
	Quiet                bool      `name:"quiet" short:"q" usage:"quiet log"`
	Color                bool      `name:"color" short:"c" usage:"colored diff"`
	DiffSuccess          bool      `name:"success" usage:"exit with 0 even if inputs differ"`
	AllowDuplicateKey    bool      `name:"allow-duplicate-key" default:"true" usage:"allow the use of keys with the same name in the same map"`
	DiffCommand          string    `name:"diff-cmd" short:"x" usage:"invoke this to get diff instead of builtin differ"`
	Verbose              bool      `name:"verbose" short:"v" usage:"enable verbose output; annotate diff type and display summary"`
	Labels               []string  `name:"label" short:"L" usage:"use label instead of file name"`
	MarkdownHeadingLevel uint      `name:"markdown-heading" default:"1" usage:"highest heading level in markdown"`
	Stdin                io.Reader `name:"-"`
	IgnoreMatchingLines  []string  `name:"ignore-matching-lines" short:"I" usage:"ignore lines matching regexp (may be specified multiple times)"`
	IgnoreFields         []string  `name:"ignore-field" short:"F" usage:"ignore field by path or yq expression (may be specified multiple times)"`
	IgnoreLabels         []string  `name:"ignore-label" usage:"ignore label by key (may be specified multiple times)"`
	IgnoreAnnotations    []string  `name:"ignore-annotation" usage:"ignore annotation by key (may be specified multiple times)"`
	IgnoreManagedFields  bool      `name:"ignore-managed-fields" usage:"ignore metadata.managedFields"`
	IgnoreStatus         bool      `name:"ignore-status" usage:"ignore status field"`
}

type OutMode string

const (
	OutModeUnknown  OutMode = "unknown"
	OutModeText     OutMode = "text"
	OutModeYaml     OutMode = "yaml"
	OutModeID       OutMode = "id"
	OutModeIDList   OutMode = "idlist"
	OutModeMarkdown OutMode = "markdown"
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
	case string(OutModeMarkdown):
		return OutModeMarkdown
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
