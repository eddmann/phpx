# phpx Implementation Plan

## Overview

Build a CLI tool in Go that:
1. Runs PHP scripts with inline Composer dependencies (`phpx run`)
2. Executes Composer tools ephemerally (`phpx tool`)
3. Manages cached PHP binaries, dependencies, and tools (`phpx cache`)

This is a **greenfield project** with a complete PRD specification.

---

## Architecture

### Project Structure

```
phpx/
├── cmd/phpx/main.go              # Entry point - calls cli.Execute()
├── internal/
│   ├── cli/                      # Cobra command definitions
│   │   ├── root.go               # Root command, global flags
│   │   ├── run.go                # phpx run <script>
│   │   ├── tool.go               # phpx tool <package>
│   │   ├── cache.go              # phpx cache list|clean|dir
│   │   └── version.go            # phpx version
│   ├── metadata/                 # Script metadata parsing
│   │   └── parser.go             # Parse // phpx TOML blocks
│   ├── index/                    # Remote index management
│   │   └── index.go              # Fetch/cache versions & extensions from static-php.dev
│   ├── php/                      # PHP binary management
│   │   ├── resolver.go           # Version resolution logic
│   │   ├── downloader.go         # Static PHP binary downloads
│   │   └── extensions.go         # Extension tier selection logic
│   ├── composer/                 # Composer integration
│   │   ├── packagist.go          # Packagist API client
│   │   ├── installer.go          # Composer install logic
│   │   └── binary.go             # Binary inference from packages
│   ├── cache/                    # Cache management
│   │   └── cache.go              # Cache paths, cleanup, listing
│   └── exec/                     # Script/tool execution
│       └── runner.go             # Execute PHP with autoload
├── go.mod
├── go.sum
├── Makefile
└── .gitignore
```

### Key Design Decisions

**1. Clean separation of concerns**
- CLI layer (`internal/cli`) handles args/flags only, delegates to core logic
- Core packages (`internal/php`, `internal/composer`, etc.) are CLI-agnostic
- This enables testing business logic without CLI scaffolding

**2. Single-responsibility packages**
- `metadata` - Parse `// phpx` TOML blocks from PHP scripts
- `php` - Resolve, download, and detect PHP binaries
- `composer` - Packagist API, dependency installation
- `cache` - Unified cache management
- `exec` - Execute scripts/tools with proper autoloading

**3. Explicit error handling**
- Sentinel errors for expected conditions (`ErrNotFound`, `ErrNoMatchingVersion`)
- Wrap errors at package boundaries with context
- Exit codes: 0 = success, 1 = phpx error, N = script/tool passthrough

---

## Implementation Phases

### Phase 1: Project Scaffolding

**Files to create:**

1. `go.mod` - Module definition with dependencies
2. `cmd/phpx/main.go` - Entry point
3. `internal/cli/root.go` - Root command with global flags
4. `internal/cli/version.go` - Version command
5. `Makefile` - Build, test, lint targets
6. `.gitignore` - Standard Go ignores

**Dependencies:**
```go
require (
    github.com/spf13/cobra v1.8.0
    github.com/Masterminds/semver/v3 v3.2.1
    github.com/BurntSushi/toml v1.3.2
    github.com/schollz/progressbar/v3 v3.14.1
)
```

Note: PRD specifies `pelletier/go-toml/v2` but `BurntSushi/toml` is more widely used and has similar API. Either works.

### Composer Version (Dynamic)

Fetch compatible version from `https://getcomposer.org/versions`:

```json
{
  "stable": [
    {"path": "/download/2.9.3/composer.phar", "version": "2.9.3", "min-php": 70205},
    {"path": "/download/2.2.26/composer.phar", "version": "2.2.26", "min-php": 50300}
  ]
}
```

**`min-php` encoding:** `MAJOR * 10000 + MINOR * 100 + PATCH` (e.g., `70205` = PHP 7.2.5)

**Resolution logic:**
```go
func SelectComposer(phpVersion string) (path string, version string, error) {
    // 1. Fetch https://getcomposer.org/versions
    // 2. Parse PHP version to int: "8.4.17" → 80417
    // 3. Filter stable[] where min-php <= phpVersionInt
    // 4. Return highest version that satisfies constraint
}
```

**Cache structure:**
```
~/.phpx/composer/
├── versions.json           # Cached from getcomposer.org/versions
├── 2.9.3/composer.phar     # Downloaded binaries by version
└── 2.2.26/composer.phar
```

**Cache TTL:** Same as index (24 hours)

Since phpx targets PHP 8.0+, we'll typically always get latest Composer 2.x, but this future-proofs against Composer 3.x requiring PHP 8.2+ etc.

### Extension Handling (Tiered Downloads)

**Two build tiers from static-php.dev:**

| Tier | Extensions | Size | URL |
|------|-----------|------|-----|
| **common** | 41 | ~10MB | `dl.static-php.dev/static-php-cli/common/php-{ver}-cli-{os}-{arch}.tar.gz` |
| **bulk** | 60 | ~25MB | `dl.static-php.dev/static-php-cli/bulk/php-{ver}-cli-{os}-{arch}.tar.gz` |

**Extension lists (fetched dynamically):**

Fetched from `build-extensions.json` endpoints and cached locally.

```go
// Loaded from ~/.phpx/index/common-extensions.json
var CommonExtensions []string  // ~37 extensions

// Loaded from ~/.phpx/index/bulk-extensions.json
var BulkExtensions []string    // ~60 extensions
```

To determine if an extension requires bulk tier:
```go
func RequiresBulk(ext string) bool {
    return contains(BulkExtensions, ext) && !contains(CommonExtensions, ext)
}
```

**Resolution logic:**
1. Parse requested extensions from metadata or `--extensions` flag
2. Check if all extensions are in `CommonExtensions` → use **common** build
3. Else check if all extensions are in common + bulk → use **bulk** build
4. Else → error: `Error: extension 'mongodb' not available in static PHP builds`

**Never use host/system PHP** - always download static binaries.

**Cache structure accounts for tier:**
```
~/.phpx/
├── index/                        # Cached remote index
│   ├── common-versions.json
│   ├── bulk-versions.json
│   ├── common-extensions.json
│   ├── bulk-extensions.json
│   └── fetched_at
├── php/{version}-common/bin/php  # Common tier binaries
├── php/{version}-bulk/bin/php    # Bulk tier binaries
└── ...
```

**How static PHP extensions work:**

Static binaries have extensions **compiled in** - not loaded dynamically as `.so` files. Extensions are baked into the binary and enabled by default.

This means the `extensions` field in `// phpx` metadata is **purely for validation and tier selection**:
1. Parse requested extensions
2. Determine tier (common vs bulk)
3. Validate all extensions available in chosen tier
4. **No runtime action needed** - extensions already present

**Note:** Unlike dynamic PHP, we don't need `-d extension=foo.so` flags.

### Phase 2: Metadata Parsing

**Files to create:**

1. `internal/metadata/parser.go`
2. `internal/metadata/parser_test.go`

**Logic:**
```go
type Metadata struct {
    PHP        string   // e.g., ">=8.2"
    Packages   []string // e.g., ["guzzlehttp/guzzle:^7.0"]
    Extensions []string // e.g., ["redis", "gd"]
}

func Parse(content []byte) (*Metadata, error)
```

**Parsing algorithm:**
1. Scan lines for `// phpx` marker (with optional whitespace)
2. Collect subsequent `//`-prefixed lines
3. Strip `// ` prefix from each line
4. Join into TOML string and parse
5. Stop at first non-comment line

**Edge cases:**
- No metadata block → return empty Metadata
- Invalid TOML → return error
- Missing fields → default to empty/nil
- Multiple `// phpx` blocks → use first only

### Phase 3: PHP Binary Management

**Files to create:**

1. `internal/php/resolver.go` - Version constraint matching
2. `internal/php/downloader.go` - Download from static-php.dev
3. `internal/php/extensions.go` - Extension lists and tier selection
4. `internal/php/resolver_test.go`

**Dynamic index fetching (cached locally):**

On first run (or when cache missing/stale), fetch from static-php.dev:

```go
// Index endpoints
const (
    CommonListURL    = "https://dl.static-php.dev/static-php-cli/common/?format=json"
    BulkListURL      = "https://dl.static-php.dev/static-php-cli/bulk/?format=json"
    CommonExtURL     = "https://dl.static-php.dev/static-php-cli/common/build-extensions.json"
    BulkExtURL       = "https://dl.static-php.dev/static-php-cli/bulk/build-extensions.json"
    ComposerVersions = "https://getcomposer.org/versions"
)
```

**Index cache structure:**
```
~/.phpx/index/
├── common-versions.json    # Parsed PHP versions for common tier
├── bulk-versions.json      # Parsed PHP versions for bulk tier
├── common-extensions.json  # Extensions in common builds
├── bulk-extensions.json    # Extensions in bulk builds
├── composer-versions.json  # Composer versions from getcomposer.org/versions
└── fetched_at              # Timestamp file
```

**Cache TTL:** 24 hours (re-fetch if older)

**Parsing versions from file listing:**
```go
// Parse "php-8.4.17-cli-linux-x86_64.tar.gz" → "8.4.17"
// Filter for current OS/arch, CLI builds only
// Deduplicate versions (same version appears for multiple platforms)
var versionRegex = regexp.MustCompile(`php-(\d+\.\d+\.\d+)-cli-`)
```

**Fallback:** If fetch fails and no cache exists, error with helpful message:
```
Error: failed to fetch PHP index from static-php.dev: {details}
Run with network access to initialize the index cache.
```

**Version resolution:**
1. Parse constraint (e.g., `>=8.2`, `^8.3`, `8.4.10`)
2. Filter `AvailableVersions` against constraint using semver
3. If no match → error: `Error: no PHP version satisfies '{constraint}'`
4. Return highest matching version

**Binary resolution order:**
1. Determine required tier based on extensions (common or bulk)
2. Check cache: `~/.phpx/php/{version}-{tier}/bin/php`
3. If not cached → download from static-php.dev
4. **Never use system PHP** - always use static binaries for consistency

**Download URL patterns:**
```
Common: https://dl.static-php.dev/static-php-cli/common/php-{version}-cli-{os}-{arch}.tar.gz
Bulk:   https://dl.static-php.dev/static-php-cli/bulk/php-{version}-cli-{os}-{arch}.tar.gz
```

**Platform detection:**
- OS: `runtime.GOOS` → "linux" or "macos" (darwin maps to macos)
- Arch: `runtime.GOARCH` → "x86_64" (amd64) or "aarch64" (arm64)

**No system PHP fallback** - phpx always uses its own static binaries for:
- Consistent behavior across environments
- Known extension availability
- No surprises from system PHP configuration

### Phase 4: Composer Integration

**Files to create:**

1. `internal/composer/packagist.go` - API client
2. `internal/composer/installer.go` - Run composer install
3. `internal/composer/binary.go` - Infer binary from package
4. `internal/composer/packagist_test.go`

**Packagist API:**
```
GET https://repo.packagist.org/p2/{vendor}/{package}.json
```

Response structure:
```go
type PackagistResponse struct {
    Packages map[string][]PackageVersion `json:"packages"`
}

type PackageVersion struct {
    Version           string            `json:"version"`
    VersionNormalized string            `json:"version_normalized"`
    Require           map[string]string `json:"require"`
    Bin               []string          `json:"bin"`
    Type              string            `json:"type"`
}
```

**Version resolution:**
- No version specified → highest stable (exclude `dev-*`, prerelease like `-alpha`, `-beta`, `-RC`)
- Version specified → parse constraint, filter, return highest match

**Binary inference:**
1. Use `--from` if provided
2. Get `bin` array from package info
3. Single binary → use it
4. Multiple → match against package short name (e.g., `phpstan` from `phpstan/phpstan`)
5. Default to first binary

**Composer download:**
- Download `composer.phar` to `~/.phpx/cache/composer/composer.phar`
- Use official download URL: `https://getcomposer.org/download/latest-stable/composer.phar`

### Phase 5: Cache Management

**Files to create:**

1. `internal/cache/cache.go`
2. `internal/cache/cache_test.go`

**Cache structure:**
```
~/.phpx/
├── php/{version}/bin/php           # PHP binaries
├── deps/{hash}/                    # Script dependencies
│   ├── composer.json
│   ├── composer.lock
│   └── vendor/autoload.php
├── tools/{pkg}-{ver}/              # Tool installations
│   └── vendor/bin/{binary}
└── cache/composer/composer.phar    # Composer binary
```

**Cache key for deps:**
SHA-256 of sorted, lowercase package list (joined by newlines)

**Cache hit detection:**
- PHP: Check if `~/.phpx/php/{version}/bin/php` exists and is executable
- Deps: Check if `~/.phpx/deps/{hash}/vendor/autoload.php` exists
- Tools: Check if `~/.phpx/tools/{pkg}-{ver}/vendor/bin/{binary}` exists

### Phase 6: Run Command

**Files to create/modify:**

1. `internal/cli/run.go`
2. `internal/exec/runner.go`

**Flags:**
- `--php` - PHP version constraint override
- `--packages` - Additional packages (comma-separated)
- `--extensions` - PHP extensions
- `-v, --verbose` - Show detailed output
- `-q, --quiet` - Suppress phpx output

**Execution flow:**
```
1. If "-", read stdin to temp file
2. Parse script for // phpx metadata
3. Merge --packages with metadata packages
4. Resolve PHP version (--php > metadata > default 8.4)
5. Get PHP binary (cache > download > system)
6. If packages exist:
   a. Hash package list
   b. Check ~/.phpx/deps/{hash}/
   c. If miss: composer install
7. Execute:
   php -d auto_prepend_file={autoload} script.php args...
8. Return script's exit code
```

**Default behavior:** `phpx script.php` → `phpx run script.php`

### Phase 7: Tool Command

**Files to create/modify:**

1. `internal/cli/tool.go`

**Additional flags:**
- `--from` - Explicit package name when binary differs

**Aliases (hardcoded):**
```go
var Aliases = map[string]string{
    "phpstan":      "phpstan/phpstan",
    "psalm":        "vimeo/psalm",
    "php-cs-fixer": "friendsofphp/php-cs-fixer",
    "pint":         "laravel/pint",
    "phpunit":      "phpunit/phpunit",
    "pest":         "pestphp/pest",
    "rector":       "rector/rector",
    "phpcs":        "squizlabs/php_codesniffer",
    "laravel":      "laravel/installer",
    "psysh":        "psy/psysh",
}
```

**Version specifiers:**
- `phpstan` - latest stable
- `phpstan@1.10.0` - exact version
- `phpstan:^1.10` - constraint

**Execution flow:**
```
1. Parse package@version or package:constraint argument
2. Resolve alias (phpstan → phpstan/phpstan)
3. Fetch package info from Packagist
4. Resolve version (specified or latest stable)
5. Infer binary name
6. Get PHP binary
7. Check ~/.phpx/tools/{package}-{version}/
8. If miss: composer require, install
9. Execute:
   php vendor/bin/{binary} args...
10. Return tool's exit code
```

### Phase 8: Cache Command

**Files to modify:**

1. `internal/cli/cache.go`

**Subcommands:**
- `phpx cache list` - Show cached PHP builds, deps, tools, index age
- `phpx cache clean` - Remove tool cache (default)
- `phpx cache clean --php` - Remove PHP builds
- `phpx cache clean --deps` - Remove dependencies
- `phpx cache clean --index` - Remove index cache (forces re-fetch)
- `phpx cache clean --all` - Remove everything
- `phpx cache dir` - Print cache path
- `phpx cache refresh` - Force re-fetch of version/extension index

---

## Error Messages

Following PRD specification:
```
Error: script not found: {path}
Error: package not found: {name}
Error: no PHP version satisfies '{constraint}'
Error: failed to download PHP: {details}
Error: failed to install dependencies: {details}
Error: binary not found in package: {package}
```

---

## Testing Strategy

**Unit tests:**
- `metadata/parser_test.go` - TOML parsing edge cases
- `php/resolver_test.go` - Version constraint matching
- `composer/packagist_test.go` - Version resolution, binary inference
- `cache/cache_test.go` - Cache key generation, path handling

**Integration tests:**
- Full execution flow with real scripts (can be skipped in CI if slow)

**Test doubles:**
- Fake HTTP client for Packagist API tests
- In-memory filesystem for cache tests (or temp directories)

---

## Risks and Open Questions

**Resolved by PRD:**
- Static PHP download URL format ✓
- Packagist API format ✓
- Cache structure ✓
- Tool aliases ✓

**Resolved during analysis:**

1. **Composer version** ✓ - Always use latest stable Composer 2.x from `getcomposer.org/download/latest-stable/composer.phar`

2. **Extension handling** ✓ - Tiered approach: use "common" by default (41 ext), upgrade to "bulk" (60 ext) if needed, error if extension unavailable in either

3. **PHP download source** ✓ - Use static-php.dev with tier selection (common vs bulk)

4. **System PHP** ✓ - Never use host PHP, always download static binaries for consistency

**Implementation considerations:**

1. **PHP version default** - When no constraint specified, default to latest stable (8.4.x).

2. **TOML library** - PRD says `pelletier/go-toml/v2`, but `BurntSushi/toml` is standard. Will use BurntSushi/toml unless there's a specific reason for pelletier.

---

## Output Modes

**Quiet (`-q`):** Script/tool output only. No phpx output, no progress bars.
```
$ phpx run -q script.php
Hello from script!
```

**Normal (default):** Script/tool output only. Progress bar for downloads (since they can take time).
```
$ phpx run script.php
Downloading PHP 8.4.17... [====================] 100%
Hello from script!
```

**Verbose (`-v`):** Full visibility into phpx operations.
```
$ phpx run -v script.php
[phpx] Parsing script metadata...
[phpx] Found: php=">=8.3", packages=["guzzlehttp/guzzle:^7.0"], extensions=["redis"]
[phpx] Checking index cache... expired, refreshing
[phpx] Fetching https://dl.static-php.dev/static-php-cli/common/?format=json
[phpx] Fetching https://dl.static-php.dev/static-php-cli/bulk/?format=json
[phpx] Extension 'redis' available in common tier
[phpx] Resolving PHP version for constraint '>=8.3'
[phpx] Matched: 8.4.17 (common tier)
[phpx] PHP binary: ~/.phpx/php/8.4.17-common/bin/php (cached)
[phpx] Resolving Composer version for PHP 8.4.17
[phpx] Using Composer 2.9.3 (cached)
[phpx] Dependencies hash: a1b2c3d4e5f6...
[phpx] Cache miss: ~/.phpx/deps/a1b2c3d4e5f6/
[phpx] Running: composer install --no-dev --no-interaction...
[phpx] Dependencies installed
[phpx] Executing: php -d auto_prepend_file=~/.phpx/deps/a1b2c3d4e5f6/vendor/autoload.php script.php
Hello from script!
[phpx] Exit code: 0
```

**Error output:** Always shown (to stderr), regardless of mode.
```
$ phpx run script.php
Error: extension 'mongodb' not available in static PHP builds
```

---

## File Creation Order

1. `go.mod`, `.gitignore`, `Makefile`
2. `cmd/phpx/main.go`
3. `internal/cli/root.go`, `internal/cli/version.go`
4. `internal/metadata/parser.go` + tests
5. `internal/cache/cache.go` + tests
6. `internal/index/index.go` + tests (fetch/cache versions & extensions)
7. `internal/php/extensions.go`, `resolver.go`, `downloader.go` + tests
8. `internal/composer/packagist.go`, `binary.go`, `installer.go` + tests
9. `internal/exec/runner.go`
10. `internal/cli/run.go`
11. `internal/cli/tool.go`
12. `internal/cli/cache.go`

---

## Makefile Targets

```makefile
.PHONY: build test lint can-release clean install

build:                    ## Build phpx binary
	go build -o bin/phpx ./cmd/phpx

test:                     ## Run tests
	go test ./...

lint:                     ## Run linters
	golangci-lint run

can-release: test lint    ## CI gate

clean:                    ## Remove build artifacts
	rm -rf bin/

install: build            ## Install to ~/bin
	cp bin/phpx ~/bin/
```
