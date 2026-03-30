.PHONY: build install test vet clean release-local

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -s -w -X main.version=$(VERSION)

build:
	go build -ldflags "$(LDFLAGS)" -o gitea-robot ./cmd/gitea-robot/

install: build
	cp gitea-robot ~/bin/gitea-robot
	@echo "Installed gitea-robot to ~/bin/gitea-robot"

test:
	go test -race -count=1 ./...

vet:
	go vet ./...

clean:
	rm -f gitea-robot
	rm -rf dist/

release-local:
	mkdir -p dist
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/gitea-robot-darwin-arm64 ./cmd/gitea-robot/
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/gitea-robot-darwin-amd64 ./cmd/gitea-robot/
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/gitea-robot-linux-amd64 ./cmd/gitea-robot/
	GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/gitea-robot-linux-arm64 ./cmd/gitea-robot/
	@echo "Built binaries in dist/"

help:
	@echo "gitea-robot Makefile"
	@echo ""
	@echo "  build          Build for current platform"
	@echo "  install        Build and install to ~/bin/"
	@echo "  test           Run go test -race"
	@echo "  vet            Run go vet"
	@echo "  clean          Remove build artifacts"
	@echo "  release-local  Cross-compile for common platforms"
	@echo "  help           Show this help"
