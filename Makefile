.PHONY: build test lint can-release clean install

build: ## Build phpx binary
	go build -o bin/phpx ./cmd/phpx

test: ## Run tests
	go test ./...

lint: ## Run linters
	golangci-lint run

can-release: test lint ## CI gate - all checks

clean: ## Remove build artifacts
	rm -rf bin/

install: build ## Install to ~/bin
	cp bin/phpx ~/bin/

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
