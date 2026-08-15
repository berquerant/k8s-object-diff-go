# AGENTS.md

This document provides a comprehensive overview of the `k8s-object-diff-go` project structure, its purpose, functionality, package architecture, and development conventions for AI agents and developers.

---

## 1. Project Purpose & Overview

**`k8s-object-diff-go`** (CLI binary name: **`objdiff`**) is a specialized diff tool designed for Kubernetes YAML manifest files.

Standard text diff tools compare files line-by-line, which often yields messy or confusing results when comparing multi-document Kubernetes YAML files where resources might be ordered differently or separated arbitrarily.

`objdiff` solves this problem by:
1. Parsing Kubernetes resource manifests into individual objects.
2. Identifying each resource by its unique **Object ID**: `<apiVersion><sep><kind><sep><namespace><sep><name>` (default separator: `>`).
3. Matching resources between left and right inputs regardless of document ordering.
4. Categorizing changes into **`add`**, **`change`**, or **`destroy`**.
5. Rendering unified diffs in various structured formats (Text, YAML, JSON/ID lists, Markdown summaries).

---

## 2. Key Features

- **Order-Independent Comparison**: Matches resources by Object ID (`apiVersion>kind>namespace>name`), ignoring their physical position in the YAML file.
- **Multiple Output Formats (`-o / --out`)**:
  - `text`: Standard unified diff format with optional color and summary annotations.
  - `yaml`: Array of structured diff objects containing ID, diff string, diff type (`add`, `change`, `destroy`), and object content.
  - `id`: Diff formatted at the Object ID level.
  - `idlist`: Lists all Object IDs found in the inputs.
  - `markdown`: Rich Markdown output with collapsible `<details>` blocks for diffs and summary tables, suitable for PR comments or CI reports.
- **Custom Differ Integration (`-x / --diff-cmd` or `DIFFCMD`)**: Uses a built-in diff engine based on `sergi/go-diff` by default, but allows executing external diff tools (e.g. `diff`).
- **Flexible Input Handling**: Supports stdin (`-`), custom Object ID separators (`-d`), custom context line count (`-C`), label overrides (`-L`), and tolerates duplicate map keys (`--allow-duplicate-key`).
- **Standard Exit Codes**: Exits with `0` when inputs are identical, `1` when diffs exist (unless `--success` is specified), and `2` on error.

---

## 3. Package Architecture & Directory Structure

```
.
├── cmd/
│   └── objdiff/            # CLI binary main entry point
├── config/                 # Command-line configuration, runner, and output formatting modes
├── internal/               # Core domain logic, object parsing, diff calculations, and marshaling
├── version/                # Version string definition
├── tests/                  # Test data and fixtures for diff comparison
├── bin/                    # Helper shell scripts (build, golden testing, licensing)
├── Makefile                # Target automation for build, lint, test, etc.
└── go.mod                  # Go module definition and dependencies
```

### Key Packages & Responsibilities

#### 1. `cmd/objdiff`
- [main.go](cmd/objdiff/main.go): Parses CLI flags using `spf13/pflag`, builds the `config.Config` struct, handles stdin resolution, executes `config.Run`, and manages process exit codes.

#### 2. `config`
- [config.go](config/config.go): Defines `Config` struct, output modes (`OutModeText`, `OutModeYaml`, etc.), and instantiates built-in or external differ engines.
- [run.go](config/run.go): Orchestrates the end-to-end execution flow — loading objects from left/right sources into map structures, calculating pairs, and initializing diff printing.
- [mode.go](config/mode.go): Implements `diffPrinter`, formatting diff results for each supported output mode (text, YAML, markdown, ID summaries).

#### 3. `internal`
- [object.go](internal/object.go): Defines `Header` (`APIVersion`, `Kind`, `Namespace`, `Name`), `Object`, and the logic to generate an Object ID string.
- [load.go](internal/load.go): Scans multi-document YAML inputs into raw Go maps/structures and converts them into typed Kubernetes `Object` instances.
- [yaml.go](internal/yaml.go): Handles YAML encoding, decoding, and normalization using `goccy/go-yaml`.
- [pair.go](internal/pair.go): Manages `ObjectPair` and `ObjectPairMap`, linking left and right objects by ID and identifying diff types (`add`, `change`, `destroy`).
- [diff.go](internal/diff.go): Defines the `Differ` interface, `ObjectDiffBuilder`, and unified diff line parsing/rendering.
- [diffmatchpatch.go](internal/diffmatchpatch.go): Implements `DMPDiffer` wrapping `github.com/sergi/go-diff/diffmatchpatch`.
- [command.go](internal/command.go): Provides command string escaping utilities for executing external diff tools safely via `al.essio.dev/pkg/shellescape`.
- [color.go](internal/color.go), [map.go](internal/map.go), [string.go](internal/string.go): Auxiliary utilities for ANSI terminal color styling, map processing, and string handling.

#### 4. `version`
- [version.go](version/version.go): Stores the application version string.

---

## 4. Dependencies

Primary dependencies declared in `go.mod`:
- **`github.com/goccy/go-yaml`**: High-performance YAML parser/encoder supporting flexible options (e.g. duplicate map key tolerance).
- **`github.com/sergi/go-diff`**: Go implementation of diff-match-patch for textual line/character diff calculations.
- **`github.com/spf13/pflag`**: POSIX-compliant command-line flag parsing.
- **`al.essio.dev/pkg/shellescape`**: Safe shell argument escaping when invoking external diff commands.
- **`github.com/stretchr/testify`**: Testing framework assertions (`assert`, `require`).

---

## 5. Development & Testing Workflow

### Prerequisites
- Go `1.26+`

### Build Commands (`Makefile`)

| Command | Action |
| :--- | :--- |
| `make` / `make dist/objdiff` | Builds binary to `dist/objdiff` via `./bin/build.sh` |
| `make test` | Runs unit tests with `-cover -race` across all packages |
| `make lint` | Runs `vet`, `check-licenses`, and `golangci-lint` |
| `make golden` | Updates golden test files using `./bin/golden.sh` |
| `make vuln` | Checks vulnerabilities using `govulncheck` |
| `make bench` | Runs benchmark tests in `config/` and reports stats |

### Running Tests
To run unit and integration tests:
```bash
make test
```

To update golden test files after intentional changes to output format:
```bash
make golden
```

---

## 6. Guidelines for AI Agents

When working on this codebase:
1. **Maintain Clean Package Boundaries**: Keep YAML parsing logic within `internal/yaml.go`, diff generation inside `internal/diff.go`, and output rendering within `config/mode.go`.
2. **Preserve Exit Status Contract**: Exit status `0` means identical inputs; `1` means diffs found; `2` means error occurred (unless `--success` is toggled).
3. **Verify Lint & Tests**: Always run `make test` and `make lint` after making changes. Ensure zero lint warnings or test regressions.
4. **Golden Files**: If output formatting is updated, check if test fixtures in `tests/` need updating via `make golden`.
