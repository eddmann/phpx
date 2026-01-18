<?php
// phpx
// packages = ["symfony/http-client:^7.0"]

// Blocks network access:
//   phpx run 13-sandbox-offline.php --offline

use Symfony\Component\HttpClient\HttpClient;
use Symfony\Component\HttpClient\Exception\TransportException;

$client = HttpClient::create(['timeout' => 5]);

try {
    $response = $client->request('GET', 'https://httpbin.org/get');
    $data = $response->toArray();
    echo "Network allowed - origin: {$data['origin']}\n";
} catch (TransportException $e) {
    echo "Network blocked (expected with --offline)\n";
}
