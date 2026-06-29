BIN     := kha
PKG     := ./cmd/kha
OUTDIR  := dist

.PHONY: all clean install

all: $(OUTDIR)/$(BIN)-darwin-arm64 $(OUTDIR)/$(BIN)-darwin-amd64 $(OUTDIR)/$(BIN)-linux-amd64 $(OUTDIR)/$(BIN)-windows-amd64.exe

$(OUTDIR)/$(BIN)-darwin-arm64:
	GOOS=darwin  GOARCH=arm64  go build -o $@ $(PKG)

$(OUTDIR)/$(BIN)-darwin-amd64:
	GOOS=darwin  GOARCH=amd64  go build -o $@ $(PKG)

$(OUTDIR)/$(BIN)-linux-amd64:
	GOOS=linux   GOARCH=amd64  go build -o $@ $(PKG)

$(OUTDIR)/$(BIN)-windows-amd64.exe:
	GOOS=windows GOARCH=amd64  go build -o $@ $(PKG)

# Install the binary for the current platform into ~/.kha/ as both the
# platform-specific name and the canonical ~/.kha/kha (no suffix).
# Skills always call ~/.kha/kha — no runtime platform detection needed.
install:
	@mkdir -p ~/.kha
	@OS=$$(uname -s); ARCH=$$(uname -m); \
	if [ "$$OS" = "Darwin" ] && [ "$$ARCH" = "arm64" ]; then \
		GOOS=darwin GOARCH=arm64 go build -o $(OUTDIR)/$(BIN)-darwin-arm64 $(PKG); \
		cp $(OUTDIR)/$(BIN)-darwin-arm64 ~/.kha/$(BIN)-darwin-arm64; \
		cp $(OUTDIR)/$(BIN)-darwin-arm64 ~/.kha/$(BIN); \
		echo "Installed ~/.kha/$(BIN) (darwin-arm64)"; \
	elif [ "$$OS" = "Darwin" ]; then \
		GOOS=darwin GOARCH=amd64 go build -o $(OUTDIR)/$(BIN)-darwin-amd64 $(PKG); \
		cp $(OUTDIR)/$(BIN)-darwin-amd64 ~/.kha/$(BIN)-darwin-amd64; \
		cp $(OUTDIR)/$(BIN)-darwin-amd64 ~/.kha/$(BIN); \
		echo "Installed ~/.kha/$(BIN) (darwin-amd64)"; \
	elif [ "$$OS" = "Linux" ]; then \
		GOOS=linux GOARCH=amd64 go build -o $(OUTDIR)/$(BIN)-linux-amd64 $(PKG); \
		cp $(OUTDIR)/$(BIN)-linux-amd64 ~/.kha/$(BIN)-linux-amd64; \
		cp $(OUTDIR)/$(BIN)-linux-amd64 ~/.kha/$(BIN); \
		echo "Installed ~/.kha/$(BIN) (linux-amd64)"; \
	else \
		echo "Windows: copy dist/$(BIN)-windows-amd64.exe to %APPDATA%\\kha\\kha.exe manually"; \
	fi

clean:
	rm -rf $(OUTDIR)
