# phpx

![phpx](docs/heading.png)

Run PHP scripts with inline dependencies. Execute Composer tools without global installation.

phpx brings patterns established in other ecosystems to PHP - ephemeral tool execution like [npx](https://docs.npmjs.com/cli/commands/npx) and [uvx](https://docs.astral.sh/uv/guides/tools/), inline script dependencies like [PEP 723](https://peps.python.org/pep-0723/).

## Features

- **Inline dependencies** - declare packages in a `// phpx` comment block, they're installed automatically
- **Ephemeral tools** - run PHPStan, Psalm, PHP-CS-Fixer without polluting your global environment
- **Automatic PHP management** - downloads pre-built static PHP binaries matching your version constraints
- **Smart caching** - PHP binaries, dependencies, and tools are cached for fast subsequent runs

## Installation

### From Source

```bash
git clone https://github.com/eddmann/phpx
cd phpx
make build
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

echo Carbon\Carbon::now()->diffForHumans();
```

**3. Run it**

```bash
phpx script.php
# 2 minutes ago
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
| `--verbose`    | `-v`  | Show detailed output                      |
| `--quiet`      | `-q`  | Suppress phpx output                      |

### phpx tool

Run a Composer package's binary without global installation.

```bash
phpx tool <package[@version]> [-- args...]
```

| Flag           | Short | Description                               |
| -------------- | ----- | ----------------------------------------- |
| `--php`        |       | PHP version constraint                    |
| `--extensions` |       | Comma-separated PHP extensions            |
| `--from`       |       | Explicit package name when binary differs |
| `--verbose`    | `-v`  | Show detailed output                      |
| `--quiet`      | `-q`  | Suppress phpx output                      |

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

**PHP binaries** are downloaded from [static-php-cli](https://github.com/crazywhalecc/static-php-cli) - pre-built static PHP binaries with common extensions included.

**Dependencies** are installed via Composer into content-addressed cache directories at `~/.phpx/deps/{hash}/`.

**Tools** are installed once and cached at `~/.phpx/tools/{package}-{version}/`.

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
make test       # Run tests
make lint       # Run linters
make build      # Build binary
make install    # Install to ~/.local/bin
```

## Credits

- [static-php-cli](https://github.com/crazywhalecc/static-php-cli) - Pre-built static PHP binaries
- [Composer](https://getcomposer.org/) - PHP dependency management

## License

MIT License - see [LICENSE](LICENSE) for details.
