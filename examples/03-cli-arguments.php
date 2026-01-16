<?php
echo "Arguments received:\n";
foreach ($argv as $i => $arg) {
    echo "  [{$i}] {$arg}\n";
}
echo "\nTotal: " . ($argc - 1) . " arguments\n";
