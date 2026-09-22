.PHONY: build test race vet fmt-check scripts-check vendor-check check clean

GO ?= go

build:
	$(GO) build ./...

test:
	$(GO) test ./...

race:
	$(GO) test -race ./model/gosystemone ./internal/httpinput ./webui

vet:
	$(GO) vet ./...

fmt-check:
	@test -z "$$(gofmt -l $$(find . -type f -name '*.go' -not -path './.git/*' -not -path './vendor/*'))" || \
		{ echo 'gofmt required for:'; gofmt -l $$(find . -type f -name '*.go' -not -path './.git/*' -not -path './vendor/*'); exit 1; }

scripts-check:
	bash -n scripts/*.sh

vendor-check:
	GOPROXY=off GOSUMDB=off $(GO) test -mod=vendor ./...

check: fmt-check scripts-check vendor-check vet build
	@git diff --check -- . ':(exclude)vendor/**'

clean:
	rm -rf bin dist coverage.out
