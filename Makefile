.PHONY: bootstrap format format-check test build

BIOME := ./node_modules/.bin/biome

bootstrap:
	npm install

format:
	go fmt ./...
	$(BIOME) format --write .

format-check:
	$(BIOME) check .

test:
	go test ./...

build:
	go build ./...
