<?php
// phpx
// packages = ["symfony/http-client:^7.0"]

// Allows only specific hosts:
//   phpx run 14-sandbox-allow-host.php --allow-host httpbin.org

use Symfony\Component\HttpClient\HttpClient;
use Symfony\Component\HttpClient\Exception\TransportException;

$client = HttpClient::create(['timeout' => 5]);

$urls = [
    'https://httpbin.org/get',
    'https://api.github.com/zen',
];

foreach ($urls as $url) {
    $host = parse_url($url, PHP_URL_HOST);
    try {
        $response = $client->request('GET', $url);
        $status = $response->getStatusCode();
        if ($status === 403) {
            echo "[blocked] {$host}\n";
        } else {
            echo "[allowed] {$host}\n";
        }
    } catch (TransportException $e) {
        echo "[blocked] {$host}\n";
    }
}
