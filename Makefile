.PHONY: format format-check test build

format:
	go fmt ./...
	ruff format .
	npm exec -- biome format --write .

format-check:
	ruff format --check .
	npm exec -- biome check .

test:
	go test ./...

build:
	go build ./...
