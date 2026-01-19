#!/usr/bin/env phpx
<?php
// Example:
//   echo "hello" > /tmp/phpx-test.txt
//   phpx 15-sandbox-filesystem.php --sandbox --allow-read /tmp --allow-write /tmp

$inputFile = '/tmp/phpx-test.txt';
$outputFile = '/tmp/phpx-output.txt';

// Read
if (file_exists($inputFile)) {
    $content = @file_get_contents($inputFile);
    if ($content !== false) {
        echo "Read: " . trim($content) . "\n";
    } else {
        echo "Read blocked\n";
    }
} else {
    echo "Create input: echo \"hello\" > {$inputFile}\n";
}

// Write
$result = @file_put_contents($outputFile, "written at " . date('H:i:s') . "\n");
echo $result !== false ? "Write: OK\n" : "Write blocked\n";
