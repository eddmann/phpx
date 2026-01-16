<?php
// phpx
// php = ">=8.1"

// Check loaded extensions
$check = ['curl', 'json', 'mbstring', 'openssl', 'pdo', 'sqlite3'];

echo "Extension status:\n";
foreach ($check as $ext) {
    $status = extension_loaded($ext) ? 'Y' : 'N';
    echo "  [{$status}] {$ext}\n";
}
