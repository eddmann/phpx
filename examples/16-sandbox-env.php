<?php
// Passes specific environment variables:
//   API_KEY=secret123 DEBUG=1 phpx run 16-sandbox-env.php --sandbox --allow-env API_KEY,DEBUG

$vars = ['API_KEY', 'DEBUG', 'HOME', 'PATH'];

foreach ($vars as $var) {
    $value = getenv($var);
    $status = $value !== false ? 'set' : 'not set';
    echo "{$var}: {$status}\n";
}
