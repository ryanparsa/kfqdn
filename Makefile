BINARY  := kubectl-fqdn
PKG     := github.com/imryanparsa/kfqdn/internal/cli
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS  = -X '$(PKG).Version=$(VERSION)' -X '$(PKG).Commit=$(COMMIT)' -X '$(PKG).Date=$(DATE)'

.PHONY: build install tidy clean

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd

install: build
	@echo ""
	@echo "Binary built: ./$(BINARY)"
	@echo ""
	@echo "To install for current user (no sudo):"
	@echo "  cp $(BINARY) ~/.local/bin/"
	@echo ""
	@echo "To install system-wide (requires sudo):"
	@echo "  sudo cp $(BINARY) /usr/local/bin/"
	@echo ""
	@echo "kubectl discovers plugins from any directory in your PATH."
	@echo "Run 'kubectl plugin list' to verify."

tidy:
	go mod tidy

clean:
	rm -f $(BINARY)