<?php
// Runs in sandbox with resource limits:
//   phpx run 12-sandbox-basic.php --sandbox --memory 64 --timeout 10 --cpu 5

echo "PHP Version: " . PHP_VERSION . "\n";
echo "Memory limit: " . ini_get('memory_limit') . "\n";
echo "Memory usage: " . round(memory_get_usage() / 1024 / 1024, 2) . " MB\n";
echo "Script path: " . __FILE__ . "\n";
