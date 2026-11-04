.PHONY: build generate design test vet e2e up down

build:
	mkdir -p bin
	go build -o bin/gobpmbank ./cmd/gobpmbank
	go build -o bin/mock-core-banking ./cmd/mock-core-banking

generate:
	go generate .

# Пересоздаёт исходную раскладку BPMN/DMN; ручные изменения моделера будут заменены.
design:
	python3 scripts/design.py

test:
	go test ./...

vet:
	go vet ./...

e2e:
	python3 scripts/e2e.py

up:
	docker compose up -d --build

down:
	docker compose down
