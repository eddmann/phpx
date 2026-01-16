<?php
// phpx
// php = ">=8.1"
// extensions = ["intl"]

// Internationalization with the intl extension (bulk tier)
// This triggers download of the larger PHP build

$formatter = new NumberFormatter('en_US', NumberFormatter::CURRENCY);
echo "US Currency: " . $formatter->formatCurrency(1234.56, 'USD') . "\n";

$formatter = new NumberFormatter('de_DE', NumberFormatter::CURRENCY);
echo "DE Currency: " . $formatter->formatCurrency(1234.56, 'EUR') . "\n";

$formatter = new NumberFormatter('ja_JP', NumberFormatter::CURRENCY);
echo "JP Currency: " . $formatter->formatCurrency(1234.56, 'JPY') . "\n";

echo "\n";

// Date formatting
$formatter = new IntlDateFormatter('en_US', IntlDateFormatter::FULL, IntlDateFormatter::SHORT);
echo "US Date: " . $formatter->format(time()) . "\n";

$formatter = new IntlDateFormatter('de_DE', IntlDateFormatter::FULL, IntlDateFormatter::SHORT);
echo "DE Date: " . $formatter->format(time()) . "\n";

// Collation (sorting)
$collator = new Collator('de_DE');
$words = ['Öl', 'Ol', 'Oma'];
$collator->sort($words);
echo "\nGerman sorted: " . implode(', ', $words) . "\n";
