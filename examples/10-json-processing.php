#!/usr/bin/env phpx
<?php
// phpx
// packages = ["symfony/var-dumper:^7.0"]

// Examples:
//   echo '{"key": "value"}' | phpx 10-json-processing.php
//   phpx 10-json-processing.php data.json

use Symfony\Component\VarDumper\VarDumper;

$input = $argv[1] ?? '-';

if ($input === '-') {
    $json = file_get_contents('php://stdin');
} else {
    if (!file_exists($input)) {
        fwrite(STDERR, "Error: File not found: {$input}\n");
        exit(1);
    }
    $json = file_get_contents($input);
}

$data = json_decode($json, true);

if (json_last_error() !== JSON_ERROR_NONE) {
    fwrite(STDERR, "Error: Invalid JSON: " . json_last_error_msg() . "\n");
    exit(1);
}

echo "Parsed JSON:\n";
VarDumper::dump($data);
