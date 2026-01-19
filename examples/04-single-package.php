#!/usr/bin/env phpx
<?php
// phpx
// packages = ["nesbot/carbon:^3.0"]

use Carbon\Carbon;

$now = Carbon::now();
echo "Current time: " . $now->format('Y-m-d H:i:s') . "\n";
echo "Day of week: " . $now->dayName . "\n";
echo "Days until weekend: " . $now->diffInDays($now->copy()->endOfWeek()) . "\n";
