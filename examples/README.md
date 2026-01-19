# phpx Examples

Progressive examples demonstrating all phpx features.

## Quick Start

```bash
cd examples
phpx 01-hello-world.php
```

## Examples

| # | File | Description |
|---|------|-------------|
| 01 | `01-hello-world.php` | Simplest script - no dependencies |
| 02 | `02-php-version.php` | Display PHP version info |
| 03 | `03-cli-arguments.php` | Pass arguments to scripts |
| 04 | `04-single-package.php` | Single dependency (Carbon) |
| 05 | `05-multiple-packages.php` | Multiple dependencies |
| 06 | `06-php-constraint.php` | Require PHP ^8.2 |
| 07 | `07-common-extensions.php` | Extensions from common tier |
| 08 | `08-bulk-extensions.php` | Extensions from bulk tier (intl) |
| 09 | `09-http-client.php` | HTTP requests with Symfony |
| 10 | `10-json-processing.php` | Process JSON from stdin/file |
| 11 | `11-cli-app.php` | Full CLI app with Symfony Console |
| 12 | `12-sandbox-basic.php` | Basic sandboxing with resource limits |
| 13 | `13-sandbox-offline.php` | Network isolation (offline mode) |
| 14 | `14-sandbox-allow-host.php` | Host-based network filtering |
| 15 | `15-sandbox-filesystem.php` | Filesystem path permissions |
| 16 | `16-sandbox-env.php` | Environment variable filtering |

## PHP Build Tiers

phpx uses two PHP build tiers:

- **Common tier** - Smaller download with common extensions (curl, gd, redis, mysql, postgres, sqlite, xml, json, mbstring, etc.)
- **Bulk tier** - Larger download with additional extensions (imagick, intl, swoole, opcache, apcu, readline, xsl, event)

The tier is selected automatically based on required extensions. Use `-v` to see which tier was selected.

## Running Examples

```bash
# Basic scripts (no dependencies)
phpx 01-hello-world.php
phpx 02-php-version.php
phpx 03-cli-arguments.php -- arg1 arg2 --flag

# Scripts with dependencies
phpx 04-single-package.php
phpx 05-multiple-packages.php

# PHP version constraints
phpx 06-php-constraint.php

# Extensions
phpx 07-common-extensions.php
phpx 08-bulk-extensions.php

# Real-world examples
phpx 09-http-client.php
echo '{"name":"test","count":42}' | phpx 10-json-processing.php -- -
phpx 11-cli-app.php -- --name=Developer
phpx 11-cli-app.php -- --name=Developer --shout

# Security/sandbox examples
phpx 12-sandbox-basic.php --sandbox --memory 64 --timeout 10 --cpu 5
phpx 13-sandbox-offline.php --offline
phpx 14-sandbox-allow-host.php --allow-host httpbin.org
echo "test input" > /tmp/phpx-input.txt && phpx 15-sandbox-filesystem.php --sandbox --allow-read /tmp --allow-write /tmp
API_KEY=secret DEBUG=1 phpx 16-sandbox-env.php --sandbox --allow-env API_KEY,DEBUG
```

## Security & Sandboxing

phpx supports running scripts in isolated environments with controlled resource limits.

### Sandbox Flags

| Flag | Description |
|------|-------------|
| `--sandbox` | Enable filesystem sandboxing |
| `--offline` | Block all network access |
| `--allow-host` | Allow network to specific hosts (comma-separated) |
| `--allow-read` | Additional readable paths (comma-separated) |
| `--allow-write` | Additional writable paths (comma-separated) |
| `--allow-env` | Environment variables to pass through (comma-separated) |
| `--memory` | Memory limit in MB (default: 128 for scripts, 256 for tools) |
| `--timeout` | Execution timeout in seconds (default: 30 for scripts, 300 for tools) |
| `--cpu` | CPU time limit in seconds (default: 30 for scripts, 300 for tools) |

### Platform Support

- **macOS**: Uses `sandbox-exec` profiles
- **Linux**: Uses `bubblewrap` (bwrap) or `nsjail` if available

## Script Metadata

Declare dependencies in a `// phpx` comment block:

```php
<?php
// phpx
// php = ">=8.2"
// packages = ["vendor/package:^1.0", "another/package:^2.0"]
// extensions = ["redis", "gd"]

// Your code here...
```

| Field | Type | Description |
|-------|------|-------------|
| `php` | string | PHP version constraint (semver) |
| `packages` | string[] | Composer packages as `vendor/name:constraint` |
| `extensions` | string[] | Required PHP extensions |

## Running Tools

phpx can run Composer tools without global installation:

```bash
# Run latest version
phpx tool phpstan -- analyze src/
phpx tool psalm -- --init
phpx tool php-cs-fixer -- fix src/

# Run specific version
phpx tool phpstan@1.10.0 -- --version
phpx tool phpcs:^3.9 -- --version

# Interactive REPL
phpx tool psysh
```

### Built-in Aliases

| Alias | Package |
|-------|---------|
| `phpstan` | phpstan/phpstan |
| `psalm` | vimeo/psalm |
| `php-cs-fixer` | friendsofphp/php-cs-fixer |
| `pint` | laravel/pint |
| `phpunit` | phpunit/phpunit |
| `pest` | pestphp/pest |
| `rector` | rector/rector |
| `phpcs` | squizlabs/php_codesniffer |
| `laravel` | laravel/installer |
| `psysh` | psy/psysh |

## CLI Flags

### For scripts

| Flag | Description |
|------|-------------|
| `--php` | Override PHP version constraint |
| `--packages` | Add packages (comma-separated) |
| `--extensions` | Add extensions (comma-separated) |
| `-v, --verbose` | Show detailed output |
| `-q, --quiet` | Suppress phpx output |

### For `phpx tool`

Same as above, plus:

| Flag | Description |
|------|-------------|
| `--from` | Explicit package when binary differs |

### Examples

```bash
# Override PHP version
phpx --php="^8.2" script.php

# Add runtime packages
phpx --packages=monolog/monolog:^3.0 script.php

# Verbose mode (see what phpx is doing)
phpx -v script.php

# Quiet mode (only script output)
phpx -q script.php
```

## Cache Management

```bash
# Show cached items
phpx cache list

# Print cache directory
phpx cache dir

# Remove tool cache (default)
phpx cache clean

# Remove specific caches
phpx cache clean --php      # Remove PHP binaries
phpx cache clean --deps     # Remove dependencies
phpx cache clean --index    # Remove version index
phpx cache clean --all      # Remove everything

# Force re-fetch of version index
phpx cache refresh
```

## Stdin Support

Read PHP from stdin:

```bash
echo '<?php echo PHP_VERSION . "\n";' | phpx -
```
