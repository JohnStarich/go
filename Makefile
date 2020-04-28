SHELL := /usr/bin/env bash
LINT_VERSION=1.25.1

MODULES = $(sort $(patsubst %/,%,$(dir $(wildcard */go.mod))))

# Make a template to run $CMD for every module's prefix, e.g. datasize-lint, datasize-test
define MODULE_TEMPL
.PHONY: $(1)-%
$(1)-%:
	cd $(1); \
	$${CMD}
endef
$(foreach module,$(MODULES),$(eval $(call MODULE_TEMPL,$(module))))

.PHONY: all
all: lint test

.PHONY: lint-deps
lint-deps:
	@if ! which golangci-lint >/dev/null || [[ "$$(golangci-lint version 2>&1)" != *${LINT_VERSION}* ]]; then \
		curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v${LINT_VERSION}; \
	fi

.PHONY: lint
lint: CMD := golangci-lint run
lint: lint-deps
lint: $(MODULES:=-lint)

.PHONY: lint-fix
lint-fix: CMD := golangci-lint run --fix
lint-fix: lint-deps
lint-fix: $(MODULES:=-lint-fix)

.PHONY: test
test: COVER := $(shell mktemp)
test: CMD := \
	set -e; \
	go test ./... -race -cover -coverprofile "${COVER}" >&2; \
	coverage=$$(go tool cover -func "${COVER}" | tail -1 | awk '{print $$3}'); \
	printf '##########################\n' >&2; \
	printf '### Coverage is %6s ###\n' "$$coverage" >&2; \
	printf '##########################\n' >&2; \
	echo "$$coverage";
test: $(MODULES:=-test)
	if [[ -n "$$COVERALLS_TOKEN" ]]; then \
		go get github.com/mattn/goveralls; \
		goveralls -coverprofile="${COVER}" -service=travis-ci -repotoken "$$COVERALLS_TOKEN"; \
	fi
