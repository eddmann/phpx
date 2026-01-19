#!/usr/bin/env phpx
<?php
// Example:
//   phpx 03-cli-arguments.php arg1 arg2 arg3

echo "Arguments received:\n";
foreach ($argv as $i => $arg) {
    echo "  [{$i}] {$arg}\n";
}
echo "\nTotal: " . ($argc - 1) . " arguments\n";
