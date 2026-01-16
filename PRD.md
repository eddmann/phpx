# phpx MVP - Build Specification

Build a CLI tool called `phpx` that runs PHP scripts with inline Composer dependencies and executes Composer tools ephemerally.

---

## Prior Art

phpx brings patterns established in other ecosystems to PHP:

| Concept | Inspiration | phpx Equivalent |
|---------|-------------|-----------------|
| Ephemeral tool execution | npx (Node.js), uvx (Python) | `phpx tool` |
| Inline script dependencies | PEP 723, uv inline scripts | `// phpx` metadata block |

**References:**
- [npx](https://docs.npmjs.com/cli/commands/npx) - npm package runner
- [uvx](https://docs.astral.sh/uv/guides/tools/) - uv tool runner
- [PEP 723](https://peps.python.org/pep-0723/) - Inline script metadata specification
- [uv inline scripts](https://docs.astral.sh/uv/guides/scripts/#declaring-script-dependencies) - uv's PEP 723 implementation

---

## Commands

### `phpx run <script.php> [-- args...]`

Run a PHP script, installing any declared dependencies first.

**Flags:**
| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--php` | string | "" | PHP version constraint (overrides script) |
| `--packages` | string | "" | Comma-separated packages to add |
| `--extensions` | string | "" | Comma-separated PHP extensions |
| `-v, --verbose` | bool | false | Show detailed output |
| `-q, --quiet` | bool | false | Suppress phpx output |

**Stdin support:** `echo '<?php echo "hi";' | phpx run -`

**Default behavior:** `phpx script.php` is shorthand for `phpx run script.php`

### `phpx tool <package[@version]> [-- args...]`

Run a Composer package's binary without global installation.

**Flags:** Same as `run`, plus:
| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--from` | string | "" | Explicit package name when binary differs |

**Version specifiers:**
- `phpstan` - latest stable
- `phpstan@1.10.0` - exact version
- `phpstan:^1.10` - constraint

**Aliases:**
```
phpstan      → phpstan/phpstan
psalm        → vimeo/psalm
php-cs-fixer → friendsofphp/php-cs-fixer
pint         → laravel/pint
phpunit      → phpunit/phpunit
pest         → pestphp/pest
rector       → rector/rector
phpcs        → squizlabs/php_codesniffer
laravel      → laravel/installer
psysh        → psy/psysh
```

### `phpx cache list|clean|dir`

- `list` - show cached PHP builds, deps, tools
- `clean` - remove tool cache (default)
- `clean --php` - remove PHP builds
- `clean --deps` - remove dependencies
- `clean --all` - remove everything
- `dir` - print cache path

### `phpx version`

Print version string.

---

## Script Metadata Format

Scripts declare dependencies in a `// phpx` TOML comment block:

```php
<?php
// phpx
// php = ">=8.2"
// packages = ["guzzlehttp/guzzle:^7.0", "monolog/monolog:^3.0"]
// extensions = ["redis", "gd"]

// Script code here...
```

**Parsing rules:**
1. Find line matching `// phpx` (with optional whitespace)
2. Read subsequent lines starting with `//`
3. Strip `// ` prefix from each line
4. Parse accumulated text as TOML
5. Stop at first non-comment line

**Fields:**
| Field | Type | Description |
|-------|------|-------------|
| `php` | string | Version constraint (semver) |
| `packages` | string[] | Composer packages as `vendor/name:constraint` |
| `extensions` | string[] | Additional PHP extensions |

---

## PHP Binary Management

### Resolution Order

1. Check cache for matching version + extensions
2. Download pre-built static binary
3. Fall back to system PHP if satisfies constraint

### Download URL

```
https://dl.static-php.dev/static-php-cli/common/php-{version}-cli-{os}-{arch}.tar.gz
```

- `{os}`: `linux` or `macos`
- `{arch}`: `x86_64` or `aarch64`

### Available Versions

```
8.4.x: 8.4.13, 8.4.12, 8.4.11, 8.4.10, 8.4.8, 8.4.6, 8.4.5, 8.4.4, 8.4.1
8.3.x: 8.3.17, 8.3.16, 8.3.15, 8.3.14, 8.3.13, 8.3.12, 8.3.11, 8.3.10
8.2.x: 8.2.27, 8.2.26, 8.2.25, 8.2.24, 8.2.23
8.1.x: 8.1.31, 8.1.30, 8.1.29
```

### Version Resolution

Given constraint like `>=8.2` or `^8.3`:
1. Parse using semver library
2. Filter available versions against constraint
3. Return highest matching version

### Base Extensions (always included)

```
bcmath, calendar, ctype, filter, mbstring, pcre, phar, tokenizer,
openssl, sodium, zlib, zip, curl, pdo, pdo_sqlite, sqlite3,
dom, xml, simplexml, xmlreader, xmlwriter, fileinfo, iconv, posix
```

### System PHP Detection

Check in order: `php` (PATH), `/usr/bin/php`, `/usr/local/bin/php`, `/opt/homebrew/bin/php`

Parse version from `php -v` output: `PHP (\d+\.\d+\.\d+)`

---

## Packagist API

### Endpoint

```
GET https://repo.packagist.org/p2/{vendor}/{package}.json
```

### Response

```json
{
  "packages": {
    "vendor/package": [
      {
        "version": "1.2.3",
        "version_normalized": "1.2.3.0",
        "require": {"php": ">=8.1"},
        "bin": ["bin/tool"],
        "type": "library"
      }
    ]
  }
}
```

### Version Resolution

1. If no version specified: return highest stable (exclude `dev-*`, prerelease)
2. Parse constraint, filter versions, return highest match

### Binary Inference

1. Use `--from` if provided
2. Get `bin` array from package info
3. If single binary, use it
4. If multiple, match against package short name
5. Default to first binary

---

## Dependency Management

### Installation

Generate `composer.json`:
```json
{
  "require": {
    "vendor/package": "^1.0"
  },
  "config": {
    "allow-plugins": false,
    "optimize-autoloader": true
  }
}
```

Run:
```bash
php composer.phar install --no-dev --no-interaction --no-scripts --prefer-dist --optimize-autoloader
```

### Caching

**Cache key:** SHA-256 of sorted, lowercase package list

**Structure:**
```
~/.phpx/deps/{hash}/
├── composer.json
├── composer.lock
└── vendor/autoload.php
```

**Cache hit:** Check if `vendor/autoload.php` exists

---

## Execution Flow

### Script (`phpx run`)

```
1. If "-", read stdin to temp file
2. Parse script for // phpx metadata
3. Merge --packages with metadata packages
4. Resolve PHP version (--php > metadata > default)
5. Get PHP binary (cache > download > system)
6. If packages exist:
   a. Hash package list
   b. Check ~/.phpx/deps/{hash}/
   c. If miss: composer install
7. Execute:
   php -d auto_prepend_file={autoload} script.php args...
8. Return script's exit code
```

### Tool (`phpx tool`)

```
1. Parse package@version argument
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

---

## Cache Structure

```
~/.phpx/
├── php/{version}-{hash}/bin/php    # PHP binaries
├── deps/{hash}/vendor/             # Script dependencies
├── tools/{pkg}-{ver}/vendor/bin/   # Tool installations
└── cache/composer/composer.phar    # Composer binary
```

---

## Example Usage

```bash
# Run script with inline deps
$ cat script.php
<?php
// phpx
// packages = ["nesbot/carbon:^3.0"]
echo Carbon\Carbon::now();

$ phpx script.php
2025-01-15 12:00:00

# Run tool
$ phpx tool phpstan -- analyze src/
 [OK] No errors

# Stdin
$ echo '<?php echo PHP_VERSION;' | phpx run -
8.4.13

# Override PHP version
$ phpx run --php="^8.2" script.php

# Add runtime package
$ phpx run --packages=monolog/monolog:^3.0 script.php
```

---

## Tech Stack

- Language: Go 1.22+
- CLI: github.com/spf13/cobra
- Semver: github.com/Masterminds/semver/v3
- TOML: github.com/pelletier/go-toml/v2
- Progress: github.com/schollz/progressbar/v3

---

## Error Cases

```
Error: script not found: {path}
Error: package not found: {name}
Error: no PHP version satisfies '{constraint}'
Error: failed to download PHP: {details}
Error: failed to install dependencies: {details}
Error: binary not found in package: {package}
```

Exit codes: 0 = success, 1 = phpx error, N = script/tool exit code (passthrough)
