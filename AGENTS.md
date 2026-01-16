# AGENTS.md

## Project Overview

Go CLI tool (`phpx`) that runs PHP scripts with inline dependencies and executes Composer tools without global installation. Built with Cobra for CLI commands.

## Setup

```bash
# Install dependencies (Go modules + golangci-lint)
make deps

# Build binary
make build
```

## Common Commands

| Task | Command |
|------|---------|
| Build (dev) | `make build` |
| Build (release) | `make build-release VERSION=x.x.x` |
| Test | `make test` |
| Lint | `make lint` |
| CI gate | `make can-release` |
| Clean | `make clean` |

**Build flags:**
- `build` - Development build with debug symbols
- `build-release` - Optimized build with `-s -w` (stripped), `-trimpath`, `CGO_ENABLED=0`, and version/commit/time injected via ldflags

## Code Conventions

**Structure:**
```
cmd/phpx/main.go     # Entry point - just calls cli.Execute()
internal/
  cli/               # Cobra commands (root, run, tool, cache, version)
  cache/             # Cache directory management (~/.phpx/)
  composer/          # Packagist API, dependency installation
  exec/              # PHP script/tool execution
  index/             # Version index fetching
  metadata/          # Parse // phpx comment blocks
  php/               # PHP binary resolution and download
```

**Go Style:**
- Packages: lowercase, single word, no underscores
- Receivers: single letter (`func (c *Cache) Path()`)
- Errors: `ErrNotFound` (sentinel), `ValidationError` (type)
- No Get prefix: `user.Name()` not `user.GetName()`
- Acronyms stay caps: `userID`, `httpClient`
- Return early with guard clauses
- Explicit ignores: `_ = writer.Write(data)`

**Dependencies:**
- `github.com/spf13/cobra` - CLI framework
- `github.com/BurntSushi/toml` - TOML parsing
- `github.com/Masterminds/semver/v3` - Version constraints
- `github.com/schollz/progressbar/v3` - Progress visualization

## Tests & CI

**Running tests:**
```bash
make test      # Runs go test ./...
make lint      # Runs golangci-lint
```

**Test conventions:**
- Table-driven tests with `t.Run()`
- Arrange-Act-Assert with blank line separation
- Test files: `*_test.go` co-located with source
- Real collaborators, fakes only for external boundaries
- No mockgen - write simple fakes by hand

**CI checks (GitHub Actions):**
- `lint` job: golangci-lint on ubuntu-latest
- `test` job: `make test` on ubuntu-latest
- Both run on push to main and PRs

## PR & Workflow Rules

**Commits:** Conventional commits format
```
<type>(<scope>): <description>

feat(tool): add version resolution
fix(composer): handle missing packages
chore: update dependencies
docs: improve README examples
```
Types: `feat`, `fix`, `refactor`, `chore`, `docs`, `test`, `style`, `perf`

**Branches:** `feat/feature-name`, `fix/bug-description`

**Release process:**
- Manual workflow dispatch with version input
- Builds: macOS ARM64, macOS x64, Linux x64
- Updates Homebrew tap automatically

## Security & Gotchas

**Never commit:**
- IDE settings (`.idea/`, `.vscode/`)
- Build artifacts (`bin/`, `dist/`)
- Test coverage (`coverage.out`)

**External APIs (no auth required):**
- Packagist: `https://packagist.org/packages/{pkg}.json`
- PHP binaries: `https://dl.static-php.dev/static-php-cli/`
- Composer versions: `https://getcomposer.org/versions`

**Cache location:** `~/.phpx/` (PHP binaries, deps, tools, index)

**Gotchas:**
- Go 1.24 required (check go.mod)
- `cmd/phpx` and `internal/cli` have no tests
- Index cache TTL is 24 hours
- All external URLs use HTTPS
