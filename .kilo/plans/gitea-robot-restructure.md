# gitea-robot Full Idiomatic Restructure Plan

Addresses all code review findings (4 Critical, 6 High, 13 Medium, 7 Low, 3 Nitpick) and adds unit + integration tests.

---

## Phase 1: Project Restructure

### 1.1 New Package Layout

```
.
├── cmd/
│   └── gitea-robot/
│       ├── main.go              # Entry point, wire deps, call cobra
│       ├── root.go              # Root command, version, help
│       ├── triage.go            # triage command
│       ├── ready.go             # ready command
│       ├── graph.go             # graph command
│       ├── add_dep.go           # add-dep command
│       ├── issues.go            # list/create/view/close/edit issue commands
│       ├── pulls.go             # list/create/merge/view PR commands
│       ├── labels.go            # list/create label commands
│       ├── repos.go             # list/create/fork repo commands
│       ├── releases.go          # create-release command
│       └── mcp.go               # mcp-server command
├── internal/
│   ├── config/
│   │   └── config.go            # Config struct (URL, token, timeout)
│   ├── client/
│   │   ├── client.go            # HTTP client interface + implementation
│   │   └── client_test.go       # Unit tests for client
│   ├── gitea/
│   │   ├── types.go             # Structured API response types
│   │   ├── issues.go            # Issue CRUD operations
│   │   ├── pulls.go             # PR operations
│   │   ├── labels.go            # Label operations
│   │   ├── repos.go             # Repo operations
│   │   ├── releases.go          # Release operations
│   │   ├── robot.go             # Robot API (triage, ready, graph, add-dep)
│   │   ├── deps.go              # Dependency operations
│   │   └── gitea_test.go        # httptest integration tests
│   ├── triage/
│   │   ├── format.go            # Markdown/text formatting for triage output
│   │   └── format_test.go       # Unit tests for formatting
│   ├── mcp/
│   │   ├── types.go             # MCPRequest, MCPResponse, MCPError
│   │   ├── server.go            # MCP server (stdin/stdout)
│   │   ├── registry.go          # Tool registry (register tools once, not duplicated)
│   │   ├── handlers.go          # Tool call handlers (thin, delegate to gitea package)
│   │   ├── registry_test.go     # Unit tests for tool registry
│   │   ├── handlers_test.go     # Unit tests for handlers (with mock client)
│   │   └── server_test.go       # Integration tests (pipe stdin/stdout)
│   └── cmdutil/
│       ├── cmdutil.go           # Shared flag parsing, owner/repo validation, body reading
│       └── cmdutil_test.go      # Unit tests for cmdutil
├── main.go                      # (REMOVED — moved to cmd/gitea-robot/)
├── cli.go                       # (REMOVED — replaced by cobra commands)
├── helpers.go                   # (REMOVED — replaced by internal/client/)
├── mcp.go                       # (REMOVED — replaced by internal/mcp/)
├── main_test.go                 # (REMOVED — replaced by distributed tests)
├── go.mod
├── Makefile
└── ...
```

### 1.2 Dependency Addition

Add **cobra** for CLI (the only external dependency):
```bash
go get github.com/spf13/cobra@latest
```

---

## Phase 2: Core Infrastructure

### 2.1 `internal/config/config.go`

- `Config` struct: `BaseURL string`, `Token string`, `Timeout time.Duration`, `MaxResponseBytes int64`
- `LoadFromEnv() (*Config, error)` — reads `GITEA_URL`, `GITEA_TOKEN`; warns on stderr if URL scheme is not HTTPS
- No package-level globals, no init-time side effects

### 2.2 `internal/client/client.go`

- `Client` interface:
  ```go
  type Client interface {
      Get(ctx context.Context, url string) ([]byte, error)
      Post(ctx context.Context, url string, body []byte) ([]byte, error)
      Patch(ctx context.Context, url string, body []byte) ([]byte, error)
      Delete(ctx context.Context, url string) ([]byte, error)
  }
  ```
- `NewHTTPClient(cfg *config.Config) Client` — creates `http.Client` with 30s timeout
- All methods accept `context.Context` for cancellation propagation
- Response body limited via `io.LimitReader(resp.Body, cfg.MaxResponseBytes)` (default 10MB)
- Request errors return `error`, never call `os.Exit`

### 2.3 `internal/gitea/types.go`

Replace all `map[string]any` with typed structs:
```go
type TriageResult struct { QuickRef QuickRef; Recommendations []Recommendation }
type QuickRef struct { Total, Open, Blocked, Ready float64 }
type Recommendation struct { Index float64; Title string; PageRank float64 }
type Issue struct { Number float64; Title string; HTMLURL string; State string }
type PullRequest struct { Number float64; Title string; HTMLURL string; State string }
type Label struct { ID float64; Name string; Color string }
type Repo struct { FullName string; HTMLURL string }
type Release struct { TagName string; HTMLURL string }
```

---

## Phase 3: Gitea Operations Layer

All functions follow the same pattern: accept `context.Context`, `client.Client`, return typed results + error, URL-encode path params.

### 3.1 `internal/gitea/robot.go`
- `GetTriage(ctx, c, baseURL, owner, repo) (*TriageResult, error)`
- `GetReady(ctx, c, baseURL, owner, repo) (json.RawMessage, error)`
- `GetGraph(ctx, c, baseURL, owner, repo) (json.RawMessage, error)`
- `AddDependency(ctx, c, baseURL, owner, repo, issue, dependsOn int64, depType string) error`

### 3.2 `internal/gitea/issues.go`
- `ListIssues(ctx, c, baseURL, owner, repo, opts) ([]Issue, error)`
- `GetIssue(ctx, c, baseURL, owner, repo, index) (*Issue, error)`
- `CreateIssue(ctx, c, baseURL, owner, repo, opts) (*Issue, error)`
- `UpdateIssue(ctx, c, baseURL, owner, repo, index, opts) (*Issue, error)`
- `CloseIssue(ctx, c, baseURL, owner, repo, index) (*Issue, error)`
- `AddComment(ctx, c, baseURL, owner, repo, index, body) error`
- `ResolveLabels(ctx, c, baseURL, owner, repo, names) ([]int64, error)`

### 3.3 `internal/gitea/pulls.go`, `labels.go`, `repos.go`, `releases.go`
Same pattern — typed structs, error returns, context propagation.

---

## Phase 4: CLI Layer (Cobra)

### 4.1 `cmd/gitea-robot/main.go`
```go
func main() {
    cfg, err := config.LoadFromEnv()
    // ... os.Exit(1) only here
    c := giteaclient.NewHTTPClient(cfg)
    rootCmd := NewRootCmd(cfg, c)
    if err := rootCmd.Execute(); err != nil { os.Exit(1) }
}
```

### 4.2 All 20+ commands as cobra commands
Each uses `RunE` (returns error, never calls `os.Exit`). Shared `cmdutil.RequireOwnerRepo()` validates common flags.

Key fixes:
- `ready` command: add `--format json|markdown` flag (fixes migration doc reference)
- `add-dep` command: restore `--relates-to` flag (was in original, removed in current)
- Add `--token` flag as alternative to `GITEA_TOKEN` env var

---

## Phase 5: MCP Server Restructure

### 5.1 `internal/mcp/registry.go` — Tool Registry (DRY fix)
```go
type Tool struct {
    Name, Description string
    InputSchema map[string]any
    Handler func(ctx context.Context, args json.RawMessage) (any, error)
}
type Registry struct { tools map[string]Tool }
func NewRegistry(c client.Client, cfg *config.Config) *Registry  // registers all 20 tools
func (r *Registry) List() []Tool
func (r *Registry) Call(ctx context.Context, name string, args json.RawMessage) (any, error)
```

### 5.2 `internal/mcp/server.go`
- `RunServer(ctx, registry, reader, writer) error` — no `os.Exit`, accepts deps
- Context cancellation support for clean shutdown

### 5.3 `internal/mcp/handlers.go`
Thin handlers that delegate to `internal/gitea/` functions via `client.Client`.

---

## Phase 6: Bug & Security Fixes (applied within new structure)

### Critical
| # | Finding | Fix |
|---|---------|-----|
| 1 | Unguarded type assertion panic in `printTriageMarkdown` | Two-value assertions in `triage/format.go` |
| 2 | `http.NewRequest` error ignored | Check error in `client/client.go` |
| 3 | No HTTP client timeout | `http.Client{Timeout: 30s}` in `client/client.go` |
| 4 | `json.Unmarshal` error ignored in `triageCmd` | Check error in `gitea/robot.go` |

### High
| # | Finding | Fix |
|---|---------|-----|
| 5 | URL injection via `--owner`/`--repo` | `url.PathEscape`/`url.QueryEscape` in all `gitea/*.go` |
| 6 | `os.Exit` in helpers makes code untestable | Return `error` from all functions; only `main()` exits |
| 7 | No `context.Context` support | All HTTP calls accept `ctx` |
| 8 | Massive DRY violation in CLI commands | Cobra command factory + `cmdutil` helpers |
| 9 | Massive DRY violation in MCP handlers | Tool registry pattern |

### Medium
| # | Finding | Fix |
|---|---------|-----|
| 10 | Plaintext HTTP default | Warn on stderr if URL not HTTPS (`config.go`) |
| 11 | Package-level env var reads | `config.LoadFromEnv()` called in `main()` |
| 12 | No structured API response types | Typed structs in `gitea/types.go` |
| 13 | Error-path `io.ReadAll` discarded | Check error in `client/client.go` |
| 14 | No response body size limit | `io.LimitReader` in `client/client.go` |
| 15 | Global mutable state | Config passed as dependency, not global |
| 16 | String/[]byte round-trip | Keep as `[]byte` throughout |
| 17 | Migration doc refs non-existent `--format` on ready | Add `--format` flag to ready command |

### Low
| # | Finding | Fix |
|---|---------|-----|
| 18 | JSON injection risk in dep_type | Use `json.Marshal` for body construction |
| 19 | No Windows build in Gitea release | Add `windows/amd64` to `.gitea/workflows/release.yml` |
| 20 | No TOCTOU race note in agent-coordination.md | Add note to docs |
| 21 | Token visible in `/proc/*/environ` | Document risk; add `--token` flag alternative |
| 22 | Inconsistent job names across CI | Standardize CI workflow job names |

### Nitpick
| # | Finding | Fix |
|---|---------|-----|
| 23 | Redundant tag check in Gitea release | Remove `if: startsWith(...)` |
| 24 | Copyright year 2026 | Verify intentional |

---

## Phase 7: Tests

### 7.1 `internal/client/client_test.go`
- Test Get/Post/Patch/Delete success, non-OK status, timeout, large response truncation, URL path escaping

### 7.2 `internal/gitea/gitea_test.go` (httptest mock server)
- Test every gitea operation: triage, ready, graph, add-dep, list/create/close/edit issues, comment, resolve labels, list/create/merge PRs, list/create labels, list/create/fork repos, create release

### 7.3 `internal/triage/format_test.go`
- Test markdown output with normal, missing, empty, and malformed data

### 7.4 `internal/mcp/registry_test.go`
- Test all 20 tools registered, unknown tool error, invalid JSON error, missing required args validation

### 7.5 `internal/mcp/handlers_test.go` (mock client)
- Test handler success and API error paths for key tools (triage, ready, add-dep, create-issue, create-pull)

### 7.6 `internal/mcp/server_test.go` (pipe integration)
- Test initialize, tools/list, ping, invalid JSON, unknown method, context cancellation

### 7.7 `internal/cmdutil/cmdutil_test.go`
- Test RequireOwnerRepo, ReadBody (file/stdin/flag), SplitLabelNames

---

## Phase 8: CI/CD Fixes

### Makefile
- `test`: `go test -race -count=1 ./...` (was only `go vet`)
- Add separate `vet` target
- Update `build` and `release-local` for `cmd/gitea-robot` path

### CI Workflows
- Add `go test -race ./...` step to both `.github/workflows/ci.yml` and `.gitea/workflows/ci.yml`
- Add consistent job names
- `.gitea/workflows/release.yml`: add `set -e`, add checksums, add `windows/amd64`, remove redundant tag check

---

## Phase 9: Documentation Fixes

- `skill/scripts/gitea-setup-labels.sh`: fix usage comment, add error checking to `curl` calls
- `skill/references/migration.md`: fix `--format` reference on ready command
- `skill/references/agent-coordination.md`: add TOCTOU race note
- `skill/SKILL.md`: add `--token` flag docs, add env var security note

---

## Phase 10: Cleanup & Verification

1. Delete old files: `main.go`, `cli.go`, `helpers.go`, `mcp.go`, `main_test.go`
2. `go mod tidy`
3. Add copyright headers to all new `.go` files
4. Remove trailing whitespace from all source files
5. Verify: `go vet ./...`, `go test -race -count=1 ./...`, `go build ./cmd/gitea-robot/`

---

## Implementation Order

1. **Phase 1+2** — Restructure layout + config + client interface (foundation)
2. **Phase 3** — Gitea operations layer with typed structs
3. **Phase 6** — Bug/security fixes (within new structure)
4. **Phase 4** — Cobra CLI commands
5. **Phase 5** — MCP server with tool registry
6. **Phase 7** — Unit + integration tests
7. **Phase 8** — CI/CD workflow updates
8. **Phase 9** — Documentation fixes
9. **Phase 10** — Cleanup and final verification
