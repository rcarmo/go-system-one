.PHONY: build test race vet fmt-check check clean

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
	@test -z "$$(gofmt -l $$(find . -type f -name '*.go' -not -path './.git/*'))" || \
		{ echo 'gofmt required for:'; gofmt -l $$(find . -type f -name '*.go' -not -path './.git/*'); exit 1; }

check: fmt-check test vet build
	@git diff --check

clean:
	rm -rf bin dist coverage.out
