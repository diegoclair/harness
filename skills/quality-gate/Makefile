# Makefile — quality-gate skill
# ------------------------------
#   make install-local   build → install to ~/.claude/skills/quality-gate/ → symlink to ~/.local/bin/
#   make uninstall-local remove what install-local created
#   make build           compile to bin/quality-gate
#   make build-all       cross-compile the 5 release platforms
#   make test            run the Go tests
#   make self            run the gate on its own source

.PHONY: build build-all test self install-local uninstall-local clean clean-dist

BIN_DIR      := bin
DIST_DIR     := cli/dist
SKILL_NAME   := quality-gate
TAG_PREFIX   := gate-v
VERSION      ?= $(shell git describe --tags --match "$(TAG_PREFIX)*" --always --dirty 2>/dev/null || echo dev)
LDFLAGS      := -s -w -X main.version=$(VERSION)
INSTALL_DIR  := $(HOME)/.claude/skills/$(SKILL_NAME)
USER_BIN_DIR := $(HOME)/.local/bin
SYMLINK      := $(USER_BIN_DIR)/$(SKILL_NAME)

build:
	@mkdir -p $(BIN_DIR)
	cd cli && go build -ldflags "$(LDFLAGS)" -o ../$(BIN_DIR)/$(SKILL_NAME) .
	@echo "✓ Built: $(BIN_DIR)/$(SKILL_NAME) ($(VERSION))"

build-all: clean-dist
	@mkdir -p $(DIST_DIR)
	cd cli && GOOS=darwin  GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(SKILL_NAME)-darwin-amd64      .
	cd cli && GOOS=darwin  GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/$(SKILL_NAME)-darwin-arm64      .
	cd cli && GOOS=linux   GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(SKILL_NAME)-linux-amd64       .
	cd cli && GOOS=linux   GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/$(SKILL_NAME)-linux-arm64       .
	cd cli && GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(SKILL_NAME)-windows-amd64.exe .
	@echo "✓ Cross-built $(VERSION) for 5 platforms in $(DIST_DIR)/"

test:
	cd cli && go test ./...

# Eating the dog food: the gate has to survive its own rules.
self: build
	cd cli && ../$(BIN_DIR)/$(SKILL_NAME) check --all

install-local: build
	@mkdir -p $(INSTALL_DIR)/bin
	@cp $(BIN_DIR)/$(SKILL_NAME) $(INSTALL_DIR)/bin/$(SKILL_NAME)
	@chmod +x $(INSTALL_DIR)/bin/$(SKILL_NAME)
	@cp SKILL.md $(INSTALL_DIR)/SKILL.md
	@rm -rf $(INSTALL_DIR)/reference
	@cp -r reference $(INSTALL_DIR)/reference
	@mkdir -p $(USER_BIN_DIR)
	@rm -f $(SYMLINK)
	@ln -s $(INSTALL_DIR)/bin/$(SKILL_NAME) $(SYMLINK)
	@case ":$$PATH:" in \
	  *":$(USER_BIN_DIR):"*) echo "✓ $(USER_BIN_DIR) already on PATH." ;; \
	  *) echo "⚠ Add to your shell profile: export PATH=\"$$HOME/.local/bin:\$$PATH\"" ;; \
	esac
	@$(INSTALL_DIR)/bin/$(SKILL_NAME) --version
	@echo "✓ $(SKILL_NAME) installed. Try: quality-gate rules"

uninstall-local:
	@rm -rf $(INSTALL_DIR)
	@rm -f $(SYMLINK)
	@echo "✓ Uninstalled."

clean: clean-dist
	rm -rf $(BIN_DIR)

clean-dist:
	rm -rf $(DIST_DIR)
