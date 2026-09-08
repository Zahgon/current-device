GO ?= go
GOROOT_DIR := $(shell $(GO) env GOROOT)
DIST := dist
DOCS := docs

# Go 1.24 moved the wasm support files from misc/wasm to lib/wasm.
WASM_LIB_DIR := $(firstword $(wildcard $(GOROOT_DIR)/lib/wasm $(GOROOT_DIR)/misc/wasm))
WASM_EXEC_JS := $(WASM_LIB_DIR)/wasm_exec.js

.DEFAULT_GOAL := check

.PHONY: check
check: fmt-check vet lint test test-wasm build wasm

.PHONY: build
build:
	$(GO) build ./...

.PHONY: test
test:
	$(GO) test -race -covermode=atomic -coverprofile=coverage.out ./...
	$(GO) tool cover -func=coverage.out | tail -n 1

.PHONY: test-wasm
test-wasm:
	PATH="$(WASM_LIB_DIR):$$PATH" GOOS=js GOARCH=wasm $(GO) test ./cmd/wasm/

.PHONY: cover
cover: test
	$(GO) tool cover -html=coverage.out

.PHONY: fmt
fmt:
	gofmt -w .

.PHONY: fmt-check
fmt-check:
	@files="$$(gofmt -l .)"; \
	if [ -n "$$files" ]; then echo "gofmt needed:"; echo "$$files"; exit 1; fi

.PHONY: vet
vet:
	$(GO) vet ./...
	GOOS=js GOARCH=wasm $(GO) vet ./cmd/wasm/

.PHONY: lint
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed; skipping (see .golangci.yml)"; \
	fi

# Produces the browser bundle: the wasm binary, Go's loader glue, and the
# promise-based shim that replaces the upstream IIFE global build.
.PHONY: wasm
wasm:
	@mkdir -p $(DIST)
	GOOS=js GOARCH=wasm $(GO) build -trimpath -ldflags="-s -w" -o $(DIST)/current-device.wasm ./cmd/wasm
	cp $(WASM_EXEC_JS) $(DIST)/wasm_exec.js
	cp web/current-device.js $(DIST)/current-device.js
	@ls -lh $(DIST)

# The demo page loads sibling assets, so mirror the bundle next to it.
.PHONY: docs
docs: wasm
	cp $(DIST)/current-device.wasm $(DIST)/wasm_exec.js $(DIST)/current-device.js $(DOCS)/

.PHONY: serve
serve: docs
	@echo "http://localhost:8080/ (wasm cannot be fetched over file://)"
	@cd $(DOCS) && $(GO) run ../tools/serve

# Regenerates testdata/typescript_oracle.json by executing the original
# TypeScript. Both the harness and the TypeScript sources live outside this
# module, so ORACLE_HARNESS and TS_SOURCE must point at the migration
# verification bundle.
.PHONY: oracle
oracle:
	@test -n "$(ORACLE_HARNESS)" || { \
		echo "ORACLE_HARNESS is unset."; \
		echo "Point it at oracle-harness.ts from the migration verification bundle:"; \
		echo "  make oracle ORACLE_HARNESS=../verification/oracle-harness.ts TS_SOURCE=../source_typescript"; \
		exit 1; }
	@node $(ORACLE_HARNESS) > testdata/typescript_oracle.json.tmp \
		&& mv testdata/typescript_oracle.json.tmp testdata/typescript_oracle.json \
		|| { rm -f testdata/typescript_oracle.json.tmp; exit 1; }

.PHONY: clean
clean:
	rm -rf $(DIST) coverage.out
	rm -f $(DOCS)/current-device.wasm $(DOCS)/wasm_exec.js $(DOCS)/current-device.js
