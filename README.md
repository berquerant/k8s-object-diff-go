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

```markdown
# Objdiff Summary

Left file <-> Right file

| **add** | **change** | **destroy** |
| :---: | :---: | :---: |
| x | y | z |
## Diff type Object ID

<details><summary>View Diff</summary>
Unified diff
</details>
```

or

```markdown
# Objdiff Summary

Left file <-> Right file

No changes.
```

## Exit status

0 if inputs are the same.
1 if inputs differ.
Otherwise 2.

## Override differ

```shell
objdiff -x diff left.yml right.yml
```
invokes
```shell
diff --unified=3 --color=never --label left.yml --label right.yml LEFT_FILE RIGHT_FILE
```

```shell
OBJDIFF_DIFF_CMD='diff' objdiff -c -C 5 left.yml right.yml
```
invokes
```shell
diff --unified=5 --color=always --label left.yml --label right.yml LEFT_FILE RIGHT_FILE
```

## Configuration & Precedence

Configuration values are resolved in the following order of precedence (highest to lowest):
1. Command-line flags (e.g. --diff-cmd)
2. Environment variables prefixed with OBJDIFF_ (e.g. OBJDIFF_DIFF_CMD, OBJDIFF_CONTEXT)
3. Default values defined for each flag

Environment variables are derived from flag names in uppercase with hyphens replaced by underscores.
e.g. --ignore-matching-lines -> OBJDIFF_IGNORE_MATCHING_LINES

## Flags

```
      --allow-duplicate-key                 allow the use of keys with the same name in the same map (default true)
  -c, --color                               colored diff
  -C, --context int                         diff context (default 3)
      --debug                               enable debug log
  -x, --diff-cmd string                     invoke this to get diff instead of builtin differ
      --ignore-annotation stringArray       ignore annotation by key (may be separated by ';' or specified multiple times)
  -F, --ignore-field stringArray            ignore field by path or yq expression (may be separated by ';' or specified multiple times)
      --ignore-label stringArray            ignore label by key (may be separated by ';' or specified multiple times)
      --ignore-managed-fields               ignore metadata.managedFields
  -I, --ignore-matching-lines stringArray   ignore lines matching regexp (may be separated by ';' or specified multiple times)
      --ignore-status                       ignore status field
  -n, --indent int                          yaml indent (default 2)
  -L, --label stringArray                   use label instead of file name (may be separated by ';' or specified multiple times)
      --markdown-heading uint               highest heading level in markdown (default 1)
  -o, --out string                          output format: text,yaml,id,idlist,markdown (default "text")
  -q, --quiet                               quiet log
  -d, --separator string                    object id separator (default ">")
      --success                             exit with 0 even if inputs differ
  -v, --verbose                             enable verbose output; annotate diff type and display summary
      --version                             print objdiff version
```

## Example

For [left.yml](./tests/diffs/left.yml) and [right.yml](./tests/diffs/right.yml), executing

``` shell
objdiff left.yml right.yml
```

yields the [result](./tests/diffs/out.txt).

## Installation

``` shell
go install github.com/berquerant/k8s-object-diff-go/cmd/objdiff@latest
```
