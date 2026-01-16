<?php
// phpx
// php = ">=8.2"
// packages = ["symfony/console:^7.0"]

use Symfony\Component\Console\Application;
use Symfony\Component\Console\Command\Command;
use Symfony\Component\Console\Input\InputInterface;
use Symfony\Component\Console\Input\InputOption;
use Symfony\Component\Console\Output\OutputInterface;

$app = new Application('phpx-example', '1.0.0');

$app->add(new class extends Command {
    protected function configure(): void
    {
        $this
            ->setName('greet')
            ->setDescription('Greet someone')
            ->addOption('name', null, InputOption::VALUE_REQUIRED, 'Who to greet', 'World')
            ->addOption('shout', null, InputOption::VALUE_NONE, 'Shout the greeting');
    }

    protected function execute(InputInterface $input, OutputInterface $output): int
    {
        $name = $input->getOption('name');
        $message = "Hello, {$name}!";

        if ($input->getOption('shout')) {
            $message = strtoupper($message);
        }

        $output->writeln($message);
        return Command::SUCCESS;
    }
});

$app->setDefaultCommand('greet', true);
$app->run();
