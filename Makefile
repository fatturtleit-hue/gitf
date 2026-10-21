.PHONY: dev backend frontend build build-compress run test test-cover lint lint-frontend lint-all clean

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -s -w -X main.version=$(VERSION)

backend:
	go run ./cmd/server

frontend:
	cd web && npm run dev

build:
	cd web && npm ci && npm run build
	rm -rf internal/web/dist
	mkdir -p internal/web/dist
	cp -r web/dist/web/browser/* internal/web/dist/
	CGO_ENABLED=0 go build -trimpath -ldflags="$(LDFLAGS)" -o bin/gitf ./cmd/server
	@ls -lh bin/gitf

build-compress: build
	upx --best --lzma bin/gitf
	@ls -lh bin/gitf

run: build
	./bin/gitf

test:
	go test -race -count=1 ./...

test-cover:
	go test -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -func=coverage.out | grep total
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

lint:
	@golangci-lint run ./... || (echo "golangci-lint failed, falling back to go vet" && go vet ./...)

lint-frontend:
	cd web && npx eslint src/ --max-warnings=0

lint-all: lint lint-frontend

clean:
	rm -rf bin internal/web/dist web/dist coverage.out coverage.html

