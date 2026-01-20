# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.0.6] - 2026-01-20

### Changed

- Version information now displayed via `--version` flag instead of a separate subcommand

## [0.0.5] - 2026-01-20

### Changed

- Release workflow now extracts version and release notes automatically from CHANGELOG.md

## [0.0.4] - 2026-01-20

### Added

- SOCKS5 proxy port support in sandbox configuration

### Fixed

- Environment inheritance in none sandbox mode

### Changed

- Simplified examples section in README with link to examples directory

## [0.0.3] - 2026-01-19

### Added

- Script flags to root command with improved examples

## [0.0.2] - 2026-01-18

### Added

- Sandboxing and network isolation for script execution
- ASCII logo displayed on the help screen
- Sandbox and security examples

### Fixed

- SSL certificate access and symlink handling for macOS sandbox

## [0.0.1] - 2026-01-16

### Added

- Initial implementation of phpx CLI for running PHP scripts with inline Composer dependencies
- Shebang support for direct script execution
- Automatic PHP version management and downloading
- Script metadata parsing via `// phpx` TOML comment blocks
- Cache management for PHP binaries and dependencies
- Tool execution for running Composer packages without global installation
- Optimized release build with stripped binaries and version info
- GitHub Pages landing page with documentation
- CI/CD workflows for testing and releases

[0.0.6]: https://github.com/eddmann/phpx/compare/v0.0.5...v0.0.6
[0.0.5]: https://github.com/eddmann/phpx/compare/v0.0.4...v0.0.5
[0.0.4]: https://github.com/eddmann/phpx/compare/v0.0.3...v0.0.4
[0.0.3]: https://github.com/eddmann/phpx/compare/v0.0.2...v0.0.3
[0.0.2]: https://github.com/eddmann/phpx/compare/v0.0.1...v0.0.2
[0.0.1]: https://github.com/eddmann/phpx/releases/tag/v0.0.1
