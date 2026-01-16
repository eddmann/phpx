<?php
// phpx
// packages = ["nesbot/carbon:^3.0", "symfony/var-dumper:^7.0"]

use Carbon\Carbon;
use Symfony\Component\VarDumper\VarDumper;

$data = [
    'timestamp' => Carbon::now()->toIso8601String(),
    'timezone' => Carbon::now()->timezoneName,
    'week_number' => Carbon::now()->weekOfYear,
];

VarDumper::dump($data);
