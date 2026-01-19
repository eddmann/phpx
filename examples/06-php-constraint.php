#!/usr/bin/env phpx
<?php
// phpx
// php = "^8.2"

// PHP 8.2+ readonly classes
readonly class Config {
    public function __construct(
        public string $name,
        public string $version
    ) {}
}

$config = new Config('phpx', '1.0.0');
echo "App: {$config->name} v{$config->version}\n";
echo "Running on PHP " . PHP_VERSION . "\n";
