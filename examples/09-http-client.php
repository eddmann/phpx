#!/usr/bin/env phpx
<?php
// phpx
// packages = ["symfony/http-client:^7.0"]

use Symfony\Component\HttpClient\HttpClient;

$client = HttpClient::create();

echo "Fetching random user from API...\n\n";

$response = $client->request('GET', 'https://randomuser.me/api/');
$data = $response->toArray();

$user = $data['results'][0];
echo "Name: {$user['name']['first']} {$user['name']['last']}\n";
echo "Email: {$user['email']}\n";
echo "Country: {$user['location']['country']}\n";
