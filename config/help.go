package config

import (
	"fmt"
	"strings"
)

// MarkdownDoc represents a markdown document builder.
type MarkdownDoc struct {
	sections []string
}

func NewMarkdownDoc() *MarkdownDoc {
	return &MarkdownDoc{}
}

func (d *MarkdownDoc) AddHeading(level int, title string) *MarkdownDoc {
	prefix := strings.Repeat("#", level)
	d.sections = append(d.sections, fmt.Sprintf("%s %s", prefix, title))
	return d
}

func (d *MarkdownDoc) AddParagraph(text string) *MarkdownDoc {
	d.sections = append(d.sections, text)
	return d
}

func (d *MarkdownDoc) AddCodeBlock(lang, code string) *MarkdownDoc {
	d.sections = append(d.sections, fmt.Sprintf("```%s\n%s\n```", lang, code))
	return d
}

func (d *MarkdownDoc) AddRaw(content string) *MarkdownDoc {
	d.sections = append(d.sections, content)
	return d
}

func (d *MarkdownDoc) String() string {
	return strings.Join(d.sections, "\n\n")
}

// Help represents the help documentation structure for objdiff.
type Help struct{}

func NewHelp() *Help {
	return &Help{}
}

func (h *Help) Title() string {
	return "objdiff - k8s object diff by object id"
}

func (h *Help) UsageHeading() string {
	return "## Usage"
}

func (h *Help) UsageContent() string {
	doc := NewMarkdownDoc()
	doc.AddCodeBlock("shell", "objdiff [flags] LEFT_FILE RIGHT_FILE")
	doc.AddParagraph(`Either LEFT_FILE or RIGHT_FILE can be set to "-". Here, "-" represents stdin.`)
	return doc.String()
}

func (h *Help) UsageSection() string {
	return fmt.Sprintf("%s\n\n%s", h.UsageHeading(), h.UsageContent())
}

func (h *Help) ObjectIDHeading() string {
	return "## Object ID"
}

func (h *Help) ObjectIDContent() string {
	doc := NewMarkdownDoc()
	doc.AddParagraph("A unique ID for a k8s object.\ne.g.")
	doc.AddCodeBlock("yaml", `apiVersion: v1
kind: Pod
metadata:
  name: nginx
  namespace: default`)
	doc.AddParagraph("then id is 'v1>Pod>default>nginx'.")
	return doc.String()
}

func (h *Help) ObjectIDSection() string {
	return fmt.Sprintf("%s\n\n%s", h.ObjectIDHeading(), h.ObjectIDContent())
}

func (h *Help) OutputFormatHeading() string {
	return "## Output format"
}

func (h *Help) OutputFormatContent() string {
	return `### idlist

All object IDs.

### id

ID diff.

### text

Unified diff.

### yaml

Array of

` + "```yaml\n" + `id: "Object ID"
diff: "Unified diff"
left: "Left object (optional)"
right: "Right object (optional)"
type: "Diff type (add or change or destroy)"
` + "```\n" + `
### markdown

` + "```markdown\n" + `# Objdiff Summary

Left file <-> Right file

| **add** | **change** | **destroy** |
| :---: | :---: | :---: |
| x | y | z |
## Diff type Object ID

<details><summary>View Diff</summary>
Unified diff
</details>
` + "```\n" + `
or

` + "```markdown\n" + `# Objdiff Summary

Left file <-> Right file

No changes.
` + "```"
}

func (h *Help) OutputFormatSection() string {
	return fmt.Sprintf("%s\n\n%s", h.OutputFormatHeading(), h.OutputFormatContent())
}

func (h *Help) ExitStatusHeading() string {
	return "## Exit status"
}

func (h *Help) ExitStatusContent() string {
	return `0 if inputs are the same.
1 if inputs differ.
Otherwise 2.`
}

func (h *Help) ExitStatusSection() string {
	return fmt.Sprintf("%s\n\n%s", h.ExitStatusHeading(), h.ExitStatusContent())
}

func (h *Help) OverrideDifferHeading() string {
	return "## Override differ"
}

func (h *Help) OverrideDifferContent() string {
	return "```shell\n" + `objdiff -x diff left.yml right.yml
` + "```\n" + `invokes
` + "```shell\n" + `diff --unified=3 --color=never --label left.yml --label right.yml LEFT_FILE RIGHT_FILE
` + "```\n\n" + "```shell\n" + `OBJDIFF_DIFF_CMD='diff' objdiff -c -C 5 left.yml right.yml
` + "```\n" + `invokes
` + "```shell\n" + `diff --unified=5 --color=always --label left.yml --label right.yml LEFT_FILE RIGHT_FILE
` + "```"
}

func (h *Help) OverrideDifferSection() string {
	return fmt.Sprintf("%s\n\n%s", h.OverrideDifferHeading(), h.OverrideDifferContent())
}

func (h *Help) ConfigurationHeading() string {
	return "## Configuration & Precedence"
}

func (h *Help) ConfigurationContent() string {
	return `Configuration values are resolved in the following order of precedence (highest to lowest):
1. Command-line flags (e.g. --diff-cmd)
2. Environment variables prefixed with OBJDIFF_ (e.g. OBJDIFF_DIFF_CMD, OBJDIFF_CONTEXT)
3. Default values defined for each flag

Environment variables are derived from flag names in uppercase with hyphens replaced by underscores.
e.g. --ignore-matching-lines -> OBJDIFF_IGNORE_MATCHING_LINES`
}

func (h *Help) ConfigurationSection() string {
	return fmt.Sprintf("%s\n\n%s", h.ConfigurationHeading(), h.ConfigurationContent())
}

func (h *Help) FlagsHeading() string {
	return "## Flags"
}

// HelpMarkdown generates the markdown representation of help without flag contents.
func (h *Help) HelpMarkdown() string {
	doc := NewMarkdownDoc()
	doc.AddParagraph(h.Title())
	doc.AddRaw(h.UsageSection())
	doc.AddRaw(h.ObjectIDSection())
	doc.AddRaw(h.OutputFormatSection())
	doc.AddRaw(h.ExitStatusSection())
	doc.AddRaw(h.OverrideDifferSection())
	doc.AddRaw(h.ConfigurationSection())
	doc.AddRaw(h.FlagsHeading())
	return doc.String() + "\n"
}

// Usage generates the usage text used in CLI help (excluding dynamic flag options).
func (h *Help) Usage() string {
	return h.HelpMarkdown()
}

// HelpMarkdown returns the markdown help string.
func HelpMarkdown() string {
	return NewHelp().HelpMarkdown()
}

// Usage returns the CLI usage string.
func Usage() string {
	return NewHelp().Usage()
}
