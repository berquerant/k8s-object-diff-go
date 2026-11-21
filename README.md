# k8s-object-diff-go

objdiff - k8s object diff by object id

## Usage

```shell
objdiff [flags] LEFT_FILE RIGHT_FILE
```

Either LEFT_FILE or RIGHT_FILE can be set to "-". Here, "-" represents stdin.

## Object ID

A unique ID for a k8s object.
e.g.

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: nginx
  namespace: default
```

then id is 'v1>Pod>default>nginx'.

## Output format

### idlist

All object IDs.

### id

ID diff.

### text

Unified diff.

### yaml

Array of

```yaml
id: "Object ID"
diff: "Unified diff"
left: "Left object (optional)"
right: "Right object (optional)"
type: "Diff type (add or change or destroy)"
```

### markdown

````markdown
# Objdiff Summary: Left file <-> Right file

| **add** | **change** | **destroy** |
| :---: | :---: | :---: |
| x | y | z |

<details>
<summary><b>Diff type</b> <code>Object ID</code></summary>

``` diff
Unified diff
```

</details>
````

or

```markdown
# Objdiff Summary: Left file <-> Right file

No changes.
```

### template

Formats diff output using a Go template string or file (`@file`, `@@` to escape leading `@`) specified via `--template` (`-T`).
`--template` is required for template mode and cannot be used with other modes.
Hermetic Sprig template functions (https://masterminds.github.io/sprig/) are available (`env`, `expandenv`, and non-deterministic functions such as `now`/`uuidv4` are disabled for safety and reproducibility).

#### Template Variables

Root context (`.`):
- `.Left`: Left input file name or label (`string`).
- `.Right`: Right input file name or label (`string`).
- `.HasDiff`: `true` if differences were found (`bool`).
- `.Stats`: Diff count statistics (`DiffStats`).
  - `.Stats.Add`: Count of added objects (`int`).
  - `.Stats.Change`: Count of changed objects (`int`).
  - `.Stats.Destroy`: Count of destroyed objects (`int`).
- `.Diffs`: List of diff items (`[]*TemplateDiffItem`).
  - `.ID`: Object ID string (e.g. `apps/v1>Deployment>default>nginx`).
  - `.Diff`: Unified diff string.
  - `.DiffType`: Diff type (`"add"`, `"change"`, `"destroy"`).
  - `.Left`: Left Kubernetes object before changes (or `nil` if added).
    - `.Header.APIVersion`: `apiVersion` (`string`).
    - `.Header.Kind`: `kind` (`string`).
    - `.Header.Metadata.Namespace`: `metadata.namespace` (`string`).
    - `.Header.Metadata.Name`: `metadata.name` (`string`).
    - `.Body`: Manifest YAML string (`string`).
  - `.Right`: Right Kubernetes object after changes (or `nil` if destroyed). Same fields as `.Left`.

Example (reproducing the default markdown output):

````gotemplate
# Objdiff Summary: {{ .Left }} <-> {{ .Right }}

{{ if not .HasDiff -}}
No changes.
{{ else -}}
| **add** | **change** | **destroy** |
| :---: | :---: | :---: |
| {{ .Stats.Add }} | {{ .Stats.Change }} | {{ .Stats.Destroy }} |
{{ range .Diffs }}
<details>
<summary><b>{{ .DiffType }}</b> <code>{{ .ID }}</code></summary>

``` diff
{{ .Diff }}```

</details>
{{ end -}}
{{ end }}
````

## Exit status

0 if inputs are the same.
1 if inputs differ (0 if `--success` is specified).
Otherwise 2.

## Filtering

objdiff supports filtering lines, fields, labels, and annotations:
- Line filter (`-i`, `--ignore-matching-lines`): ignores lines matching regular expressions.
- Field filter (`-f`, `--ignore-field`): ignores fields by path (e.g. `status`, `metadata.managedFields`) or single yq deletion expressions (e.g. `del(.status)`); only safe path deletion expressions are allowed, with yq env/file/system operations disabled.
- Label filter (`-l`, `--ignore-label`): ignores labels across metadata and nested pod templates by key.
- Annotation filter (`-a`, `--ignore-annotation`): ignores annotations across metadata and nested pod templates by key.
- Shortcut flags:
  - `-m`, `--ignore-managed-fields`: shortcut for `-f metadata.managedFields`.
  - `-s`, `--ignore-status`: shortcut for `-f status`.

## Override differ

```shell
objdiff -x diff left.yml right.yml
```
invokes (per object; for `-o id`, only the file name/label is passed to `--label`)
```shell
diff --unified=3 --color=never --label 'left.yml <objectID>' --label 'right.yml <objectID>' LEFT_FILE RIGHT_FILE
```

```shell
OBJDIFF_DIFF_CMD='diff' objdiff -c -C 5 left.yml right.yml
```
invokes
```shell
diff --unified=5 --color=always --label 'left.yml <objectID>' --label 'right.yml <objectID>' LEFT_FILE RIGHT_FILE
```

Note: `--diff-cmd` / `OBJDIFF_DIFF_CMD` is split by whitespace; shell quoting is not supported.

## Configuration & Precedence

Configuration values are resolved in the following order of precedence (highest to lowest):
1. Command-line flags (e.g. --diff-cmd)
2. Environment variables prefixed with OBJDIFF_ (e.g. OBJDIFF_DIFF_CMD, OBJDIFF_CONTEXT)
3. Default values defined for each flag

Environment variables are derived from flag names in uppercase with hyphens replaced by underscores.
e.g. --ignore-matching-lines -> OBJDIFF_IGNORE_MATCHING_LINES

## Flags

```
      --allow-duplicate-key[=true|false]     allow the use of keys with the same name in the same map (default true)
  -c, --color[=true|false]                   colored diff
  -C, --context int                          diff context (default 3)
      --debug[=true|false]                   enable debug log
  -x, --diff-cmd string                      invoke this to get diff instead of builtin differ
  -a, --ignore-annotation stringArray        ignore annotation by key (may be separated by ';' or specified multiple times)
  -f, --ignore-field stringArray             ignore field by path or yq expression (may be separated by ';' or specified multiple times)
  -l, --ignore-label stringArray             ignore label by key (may be separated by ';' or specified multiple times)
  -m, --ignore-managed-fields[=true|false]   ignore metadata.managedFields
  -i, --ignore-matching-lines stringArray    ignore lines matching regexp (may be separated by ';' or specified multiple times)
  -s, --ignore-status[=true|false]           ignore status field
  -n, --indent int                           yaml indent (default 2)
  -L, --label stringArray                    use label instead of file name (may be separated by ';' or specified multiple times)
      --markdown-heading uint                highest heading level in markdown (default 1)
      --markdown-title string                title in markdown summary; template variables: LEFT_FILE, RIGHT_FILE (default "Objdiff Summary: {{ .LEFT_FILE }} <-> {{ .RIGHT_FILE }}")
      --mcp[=true|false]                     start MCP server over stdio
  -o, --out string                           output format: text,yaml,id,idlist,markdown,template (default "text")
  -q, --quiet[=true|false]                   quiet log
  -d, --separator string                     object id separator (default ">")
      --success[=true|false]                 exit with 0 even if inputs differ
  -T, --template string                      template string or file (prefix with '@') for template output mode
  -v, --verbose[=true|false]                 enable verbose output; annotate diff type and display summary
      --version[=true|false]                 print objdiff version
```

## Example

For [left.yml](./tests/diffs/left.yml) and [right.yml](./tests/diffs/right.yml), executing

``` shell
objdiff left.yml right.yml
```

yields the [result](./tests/diffs/out.txt).

## Installation

Requires Go 1.27+ (or download a prebuilt binary from [GitHub Releases](https://github.com/berquerant/k8s-object-diff-go/releases)).

``` shell
go install github.com/berquerant/k8s-object-diff-go/cmd/objdiff@latest
```

## MCP Server Configuration

You can register `objdiff` as a Model Context Protocol (MCP) server for AI agents using `--mcp`.

### MCP Settings JSON

```json
{
  "mcpServers": {
    "objdiff": {
      "command": "objdiff",
      "args": ["--mcp"]
    }
  }
}
```

### Exposed Tool: `diff_k8s_manifests`

Compares two Kubernetes manifests by Object ID and returns `{ "diff": string, "hasDiff": bool }`.

Key parameters:
- `left`, `right` (required): Inline Kubernetes manifest YAML strings (filesystem paths are not opened).
- `out`: Output format (`text`, `yaml`, `id`, `idlist`, `markdown`, `template`).
- `context`, `indent`, `separator`, `verbose`, `markdownHeadingLevel` (1-6), `markdownTitle`, `template`.
- `ignoreMatchingLines`, `ignoreFields`, `ignoreLabels`, `ignoreAnnotations`, `ignoreManagedFields`, `ignoreStatus`.

Security constraints in MCP mode:
- `left` and `right` accept inline YAML content only (max 50MB per input).
- Template file references (`@file`) are disabled; pass inline templates only.
- `yq` filters run with environment, file, and system operations disabled and only allow safe `del(...)` path expressions.
