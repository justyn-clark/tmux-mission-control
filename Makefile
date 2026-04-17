.PHONY: bootstrap format format-check test build

RUFF := ./scripts/ruffw

bootstrap:
	npm install
	$(RUFF) --version

format:
	go fmt ./...
	$(RUFF) format .
	npm exec -- biome format --write .

format-check:
	$(RUFF) format --check .
	npm exec -- biome check .

test:
	go test ./...

build:
	go build ./...
