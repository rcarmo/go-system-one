SHELL := /usr/bin/env bash

GO ?= go
GOOS ?= $(shell $(GO) env GOOS)
GOARCH ?= $(shell $(GO) env GOARCH)
VERSION ?= $(shell git describe --tags --always --dirty)
BUILD_DIR ?= bin
DIST_DIR ?= dist
PREFIX ?= /usr/local
DESTDIR ?=
BINARY ?= $(BUILD_DIR)/go-system-one
LISTEN ?= 127.0.0.1:8080
BACKEND ?= nvidia
ARTIFACT_DIR ?= $(or $(GO_SYSTEM_ONE_ARTIFACT_DIR),$(or $(XDG_CACHE_HOME),$(HOME)/.cache)/go-system-one/v1)
MODEL ?= $(ARTIFACT_DIR)/model/gemma-4-12b-it-UD-Q4_K_XL.gguf
TOKENIZER_DIR ?= $(ARTIFACT_DIR)/tokenizer
UPSTREAM_COMMIT ?= $(shell . scripts/upstream.env && printf '%s' "$$UPSTREAM_COMMIT")

.PHONY: help prerequisites setup vendor build install uninstall run test race coverage vet fmt-check scripts-check \
	vendor-check check cross-build artifacts-info artifacts-download artifacts-verify \
	artifacts-clean hardware-check package update clean distclean

help:
	@printf '%s\n' \
	  'Go System One lifecycle targets' \
	  '' \
	  '  make prerequisites      Check required and optional tools' \
	  '  make setup              Regenerate vendor and build the service' \
	  '  make build              Build bin/go-system-one' \
	  '  make install            Install under PREFIX (default /usr/local)' \
	  '  make uninstall          Remove the installed binary' \
	  '  make run                Verify external artifacts and serve on LISTEN' \
	  '  make test               Run the offline test suite' \
	  '  make coverage           Write coverage.out and print package coverage' \
	  '  make race               Run race tests across first-party packages' \
	  '  make check              Run formatting, policy, test, vet and build gates' \
	  '  make cross-build        Compile Linux ARM64 and RISC-V binaries' \
	  '  make hardware-check     Run both pinned released-model NVIDIA gates' \
	  '  make package            Build code-only release archives for three targets' \
	  '  make update             Import UPSTREAM_COMMIT through the one-way manifest' \
	  '' \
	  'External artifacts (never stored in the repository)' \
	  '  make artifacts-info' \
	  '  make artifacts-download ACCEPT_GEMMA_LICENSE=1 [HF_TOKEN=...]' \
	  '  make artifacts-verify' \
	  '  make artifacts-clean CONFIRM_ARTIFACT_DELETE=1' \
	  '' \
	  'Overrides: ARTIFACT_DIR, MODEL, TOKENIZER_DIR, BACKEND, LISTEN, VERSION'

prerequisites:
	@command -v $(GO) >/dev/null || { echo 'Go is required' >&2; exit 1; }
	@command -v git >/dev/null || { echo 'git is required' >&2; exit 1; }
	@command -v curl >/dev/null || echo 'warning: curl is required only for artifact downloads' >&2
	@command -v sha256sum >/dev/null || command -v shasum >/dev/null || { echo 'sha256sum or shasum is required' >&2; exit 1; }
	@printf 'go=%s\n' "$$($(GO) version)"
	@printf 'git=%s\n' "$$(git --version)"

setup: prerequisites vendor build

vendor:
	./scripts/vendor.sh

build:
	@mkdir -p $(BUILD_DIR)
	$(GO) build -mod=vendor -trimpath -o $(BINARY) ./cmd/go-system-one

install: build
	install -D -m 0755 $(BINARY) "$(DESTDIR)$(PREFIX)/bin/go-system-one"

uninstall:
	rm -f "$(DESTDIR)$(PREFIX)/bin/go-system-one"

run: artifacts-verify build
	$(BINARY) -model "$(MODEL)" -tokenizer-dir "$(TOKENIZER_DIR)" -backend "$(BACKEND)" -listen "$(LISTEN)"

test:
	GOPROXY=off GOSUMDB=off $(GO) test -mod=vendor ./...

race:
	$(GO) test -race ./model ./model/gosystemone ./backends/nvidia/runtime ./backends/simd/... ./loader/... ./runtime/... ./tensor ./webui ./internal/...

coverage:
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -func=coverage.out

vet:
	$(GO) vet ./...

fmt-check:
	@test -z "$$(gofmt -l $$(find . -type f -name '*.go' -not -path './.git/*' -not -path './vendor/*'))" || \
		{ echo 'gofmt required for:'; gofmt -l $$(find . -type f -name '*.go' -not -path './.git/*' -not -path './vendor/*'); exit 1; }

scripts-check:
	bash -n scripts/*.sh
	./scripts/check-local-imports.sh
	./scripts/check-no-gguf-artifacts.sh
	./scripts/artifacts_test.sh

vendor-check:
	@test ! -d vendor/github.com/rcarmo/go-pherence
	@! grep -q 'github.com/rcarmo/go-pherence' go.mod go.sum vendor/modules.txt
	GOPROXY=off GOSUMDB=off $(GO) test -mod=vendor ./...

check: fmt-check scripts-check vendor-check vet build
	@git diff --check -- . ':(exclude)vendor/**'

cross-build:
	@mkdir -p $(BUILD_DIR)/cross
	GOPROXY=off GOSUMDB=off CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GO) build -mod=vendor -trimpath -o $(BUILD_DIR)/cross/go-system-one-linux-arm64 ./cmd/go-system-one
	GOPROXY=off GOSUMDB=off CGO_ENABLED=0 GOOS=linux GOARCH=riscv64 $(GO) build -mod=vendor -trimpath -o $(BUILD_DIR)/cross/go-system-one-linux-riscv64 ./cmd/go-system-one

artifacts-info:
	GO_SYSTEM_ONE_ARTIFACT_DIR="$(ARTIFACT_DIR)" ./scripts/artifacts.sh info

artifacts-download:
	GO_SYSTEM_ONE_ARTIFACT_DIR="$(ARTIFACT_DIR)" ACCEPT_GEMMA_LICENSE="$(ACCEPT_GEMMA_LICENSE)" ./scripts/artifacts.sh download

artifacts-verify:
	GO_SYSTEM_ONE_ARTIFACT_DIR="$(ARTIFACT_DIR)" GO_SYSTEM_ONE_MODEL="$(MODEL)" GO_SYSTEM_ONE_TOKENIZER_DIR="$(TOKENIZER_DIR)" ./scripts/artifacts.sh verify

artifacts-clean:
	GO_SYSTEM_ONE_ARTIFACT_DIR="$(ARTIFACT_DIR)" CONFIRM_ARTIFACT_DELETE="$(CONFIRM_ARTIFACT_DELETE)" ./scripts/artifacts.sh clean

hardware-check: artifacts-verify
	GO_SYSTEM_ONE_MODEL="$(MODEL)" GO_SYSTEM_ONE_TOKENIZER_DIR="$(TOKENIZER_DIR)" \
		$(GO) test -mod=vendor ./model/gosystemone \
		-run '^(TestPinnedGemma4Artifacts|TestGoSystemOneNVIDIAReleasedModelMatchesPinnedLlamaCppDecision|TestGoSystemOneNVIDIAMultiFieldReleasedModelMatchesPinnedLlamaCpp)$$' \
		-count=1 -v

package: check
	@rm -rf $(DIST_DIR)
	@mkdir -p $(DIST_DIR)
	@set -euo pipefail; \
	for target in linux/amd64 linux/arm64 linux/riscv64; do \
		os=$${target%/*}; arch=$${target#*/}; name=go-system-one-$(VERSION)-$$os-$$arch; \
		mkdir -p "$(DIST_DIR)/$$name"; \
		GOPROXY=off GOSUMDB=off CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch $(GO) build -mod=vendor -trimpath \
			-ldflags '-s -w' -o "$(DIST_DIR)/$$name/go-system-one" ./cmd/go-system-one; \
		cp LICENSE README.md "$(DIST_DIR)/$$name/"; \
		cp -R docs "$(DIST_DIR)/$$name/docs"; \
		tar -C $(DIST_DIR) -czf "$(DIST_DIR)/$$name.tar.gz" "$$name"; \
		rm -rf "$(DIST_DIR)/$$name"; \
	done
	@./scripts/check-no-gguf-artifacts.sh
	@./scripts/check-release-archives.sh "$(DIST_DIR)"
	@cd $(DIST_DIR) && sha256sum *.tar.gz > SHA256SUMS
	@find $(DIST_DIR) -type f -maxdepth 1 -print

update:
	./scripts/update-upstream.sh "$(UPSTREAM_COMMIT)"

clean:
	rm -rf $(BUILD_DIR) $(DIST_DIR) coverage.out

distclean: clean
	GO_SYSTEM_ONE_ARTIFACT_DIR="$(ARTIFACT_DIR)" CONFIRM_ARTIFACT_DELETE="$(CONFIRM_ARTIFACT_DELETE)" ./scripts/artifacts.sh clean
