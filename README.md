# phpx

![phpx](docs/heading.png)

Run PHP scripts with inline dependencies. Execute Composer tools without global installation.

phpx brings patterns established in other ecosystems to PHP - ephemeral tool execution like [npx](https://docs.npmjs.com/cli/commands/npx) and [uvx](https://docs.astral.sh/uv/guides/tools/), inline script dependencies like [PEP 723](https://peps.python.org/pep-0723/).

## Features

- **Inline dependencies** - declare packages in a `// phpx` comment block, they're installed automatically
- **Ephemeral tools** - run PHPStan, Psalm, PHP-CS-Fixer without polluting your global environment
- **Automatic PHP management** - downloads pre-built static PHP binaries matching your version constraints
- **Smart caching** - PHP binaries, dependencies, and tools are cached for fast subsequent runs
- **Sandboxing & isolation** - run scripts in isolated environments with controlled filesystem, network, and resource limits

## Installation

### Homebrew (Recommended)

```bash
brew install eddmann/tap/phpx
```

### Quick Install

```bash
curl -fsSL https://raw.githubusercontent.com/eddmann/phpx/main/install.sh | sh
```

### Download Binary

```bash
# macOS (Apple Silicon)
curl -L https://github.com/eddmann/phpx/releases/latest/download/phpx-macos-arm64 -o phpx
chmod +x phpx && sudo mv phpx /usr/local/bin/

# macOS (Intel)
curl -L https://github.com/eddmann/phpx/releases/latest/download/phpx-macos-x64 -o phpx
chmod +x phpx && sudo mv phpx /usr/local/bin/

# Linux (x64)
curl -L https://github.com/eddmann/phpx/releases/latest/download/phpx-linux-x64 -o phpx
chmod +x phpx && sudo mv phpx /usr/local/bin/
```

### From Source

```bash
git clone https://github.com/eddmann/phpx
cd phpx
make build-release VERSION=0.1.0
make install  # Installs to ~/.local/bin
```

## Quick Start

**1. Run a simple script**

```bash
phpx script.php
```

**2. Add inline dependencies**

```php
<?php
// phpx
// packages = ["nesbot/carbon:^3.0"]

echo \Carbon\Carbon::parse('next friday')->diffForHumans();
```

**3. Run it**

```bash
phpx script.php
# 6 days from now
```

**4. Run a tool without installing it**

```bash
phpx tool phpstan -- analyze src/
```

**5. Pipe PHP from stdin**

```bash
echo '<?php echo PHP_VERSION;' | phpx run -
```

## Script Metadata

Declare dependencies in a `// phpx` comment block at the top of your script:

```php
<?php
// phpx
// php = ">=8.2"
// packages = ["guzzlehttp/guzzle:^7.0", "monolog/monolog:^3.0"]
// extensions = ["intl"]

// Your code here...
```

| Field        | Type     | Description                                   |
| ------------ | -------- | --------------------------------------------- |
| `php`        | string   | PHP version constraint (semver)               |
| `packages`   | string[] | Composer packages as `vendor/name:constraint` |
| `extensions` | string[] | Required PHP extensions                       |

## Shebang Support

Make PHP scripts directly executable:

```php
#!/usr/bin/env phpx
<?php
// phpx
// packages = ["nesbot/carbon:^3.0"]

echo \Carbon\Carbon::now()->format('Y-m-d');
```

```bash
chmod +x script.php
./script.php
```

## Command Reference

### phpx run

Run a PHP script with inline dependencies.

```bash
phpx run <script.php> [-- args...]
phpx <script.php> [-- args...]  # Shorthand
```

| Flag           | Short | Description                               |
| -------------- | ----- | ----------------------------------------- |
| `--php`        |       | PHP version constraint (overrides script) |
| `--packages`   |       | Comma-separated packages to add           |
| `--extensions` |       | Comma-separated PHP extensions            |
| `--sandbox`    |       | Enable sandboxing (restricts filesystem)  |
| `--offline`    |       | Block all network access                  |
| `--allow-host` |       | Allow network to specific hosts           |
| `--allow-read` |       | Additional readable paths                 |
| `--allow-write`|       | Additional writable paths                 |
| `--allow-env`  |       | Environment variables to pass             |
| `--memory`     |       | Memory limit in MB (default: 128)         |
| `--timeout`    |       | Execution timeout in seconds (default: 30)|
| `--cpu`        |       | CPU time limit in seconds (default: 30)   |
| `--verbose`    | `-v`  | Show detailed output                      |
| `--quiet`      | `-q`  | Suppress phpx output                      |

### phpx tool

Run a Composer package's binary without global installation.

```bash
phpx tool <package[@version]> [-- args...]
```

| Flag           | Short | Description                                |
| -------------- | ----- | ------------------------------------------ |
| `--php`        |       | PHP version constraint                     |
| `--extensions` |       | Comma-separated PHP extensions             |
| `--from`       |       | Explicit package name when binary differs  |
| `--sandbox`    |       | Enable sandboxing (restricts filesystem)   |
| `--offline`    |       | Block all network access                   |
| `--allow-host` |       | Allow network to specific hosts            |
| `--allow-read` |       | Additional readable paths                  |
| `--allow-write`|       | Additional writable paths                  |
| `--allow-env`  |       | Environment variables to pass              |
| `--memory`     |       | Memory limit in MB (default: 256)          |
| `--timeout`    |       | Execution timeout in seconds (default: 300)|
| `--cpu`        |       | CPU time limit in seconds (default: 300)   |
| `--verbose`    | `-v`  | Show detailed output                       |
| `--quiet`      | `-q`  | Suppress phpx output                       |

**Version specifiers:**

- `phpstan` - latest stable
- `phpstan@1.10.0` - exact version
- `phpstan:^1.10` - version constraint

**Built-in aliases:**

| Alias          | Package                   |
| -------------- | ------------------------- |
| `phpstan`      | phpstan/phpstan           |
| `psalm`        | vimeo/psalm               |
| `php-cs-fixer` | friendsofphp/php-cs-fixer |
| `pint`         | laravel/pint              |
| `phpunit`      | phpunit/phpunit           |
| `pest`         | pestphp/pest              |
| `rector`       | rector/rector             |
| `phpcs`        | squizlabs/php_codesniffer |
| `laravel`      | laravel/installer         |
| `psysh`        | psy/psysh                 |

### phpx cache

Manage the phpx cache.

```bash
phpx cache list              # Show cached items
phpx cache clean             # Remove tool cache (default)
phpx cache clean --php       # Remove PHP binaries
phpx cache clean --deps      # Remove dependencies
phpx cache clean --index     # Remove version index
phpx cache clean --all       # Remove everything
phpx cache dir               # Print cache path
phpx cache refresh           # Force re-fetch of version index
```

### phpx version

Print version information.

## Examples

The `examples/` directory contains progressive examples:

| Example                    | Description                       |
| -------------------------- | --------------------------------- |
| `01-hello-world.php`       | Simplest script, no dependencies  |
| `02-php-version.php`       | Display PHP environment info      |
| `03-cli-arguments.php`     | Handle command-line arguments     |
| `04-single-package.php`    | One dependency (Carbon)           |
| `05-multiple-packages.php` | Multiple dependencies             |
| `06-php-constraint.php`    | Require specific PHP version      |
| `07-common-extensions.php` | Check common extensions           |
| `08-bulk-extensions.php`   | Use intl extension                |
| `09-http-client.php`       | HTTP requests with Symfony        |
| `10-json-processing.php`   | JSON processing from stdin        |
| `11-cli-app.php`           | Full CLI app with Symfony Console |

```bash
# Run examples
phpx examples/04-single-package.php
phpx examples/11-cli-app.php -- greet --name=World
echo '{"test": 123}' | phpx examples/10-json-processing.php -- -
```

## How It Works

```
Script → Parse metadata → Resolve PHP version → Download PHP (if needed)
                                              → Install dependencies (if needed)
                                              → Execute script
```

**PHP binaries** are downloaded from [static-php-cli](https://github.com/crazywhalecc/static-php-cli) - pre-built static PHP binaries with common extensions included. Two tiers are available:

- **Common** - smaller download with standard extensions (curl, gd, redis, mysql, postgres, sqlite, xml, json, mbstring)
- **Bulk** - larger download with additional extensions (imagick, intl, swoole, opcache, apcu, readline, xsl, event)

The tier is selected automatically based on required extensions.

**Dependencies** are installed via Composer into content-addressed cache directories at `~/.phpx/deps/{hash}/`.

**Tools** are installed once and cached at `~/.phpx/tools/{package}-{version}/`.

## Security & Sandboxing

phpx supports running scripts and tools in isolated environments with controlled resource limits.

### Sandbox Modes

**Filesystem sandboxing** (`--sandbox`):
- macOS: Uses `sandbox-exec` profiles
- Linux: Uses `bubblewrap` or `nsjail` (if available)
- Restricts filesystem access to only what the script needs

**Network isolation** (`--offline`, `--allow-host`):

```bash
# Completely block network access
phpx run script.php --offline

# Allow only specific hosts
phpx run script.php --allow-host api.example.com,cdn.example.com
```

### Resource Limits

```bash
# Run with 64MB memory, 10 second timeout, 5 second CPU limit
phpx run script.php --sandbox --memory 64 --timeout 10 --cpu 5
```

### Filesystem Access

```bash
# Allow reading from additional paths
phpx run script.php --sandbox --allow-read /path/to/data

# Allow writing to additional paths
phpx run script.php --sandbox --allow-write /path/to/output
```

### Environment Variables

By default, sandbox mode filters environment variables to avoid leaking secrets. Use `--allow-env` to pass specific variables:

```bash
phpx run script.php --sandbox --allow-env API_KEY,DEBUG
```

## Cache Structure

```
~/.phpx/
├── php/{version}-{tier}/bin/php        # PHP binaries
├── deps/{hash}/vendor/                 # Script dependencies
├── tools/{pkg}-{ver}/vendor/bin/       # Tool installations
├── composer/{version}/composer.phar    # Composer binaries
└── index/                              # Version/extension index
```

## Development

```bash
git clone https://github.com/eddmann/phpx
cd phpx
make test                           # Run tests
make lint                           # Run linters
make build                          # Build binary (dev, with debug symbols)
make build-release VERSION=x.x.x    # Build binary (release, optimized)
make install                        # Install to ~/.local/bin
```

## Credits

- [static-php-cli](https://github.com/crazywhalecc/static-php-cli) - Pre-built static PHP binaries
- [Composer](https://getcomposer.org/) - PHP dependency management

## License

MIT License - see [LICENSE](LICENSE) for details.
