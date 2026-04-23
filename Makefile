BINARY := kubectl-fqdn

.PHONY: build install tidy

build:
	go build -o $(BINARY) ./cmd

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