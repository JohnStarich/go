SHELL := /usr/bin/env bash
LINT_VERSION=1.25.1

MODULES = $(sort $(patsubst %/,%,$(dir $(wildcard */go.mod))))

.PHONY: all
all: lint test

.PHONY: lint-deps
lint-deps:
	@if ! which golangci-lint >/dev/null || [[ "$$(golangci-lint version 2>&1)" != *${LINT_VERSION}* ]]; then \
		curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v${LINT_VERSION}; \
	fi

.PHONY: lint
lint: $(MODULES:=-lint)
.PHONY: %-lint
%-lint: lint-deps
	cd $*; golangci-lint run

.PHONY: lint-fix
lint-fix: $(MODULES:=-lint-fix)
%-lint-fix: lint-deps
	cd $*; golangci-lint run --fix

# 'go mod init ...' below is a hack so 'go tool cover' can generate reports
.PHONY: test
test: $(MODULES:=-test)
	echo 'mode: atomic' > cover.out
	cat cover/* | grep -v '^mode: ' | sed 's@github.com/johnstarich/go@.@' >> cover.out
	set -e -o pipefail; \
		coverage=$$(go tool cover -func "cover.out" | tail -1 | awk '{print $$3}'); \
		printf '##########################\n' >&2; \
		printf '### Coverage is %6s ###\n' "$$coverage" >&2; \
		printf '##########################\n' >&2; \
		echo "$$coverage"
	@if [[ -n "${COVERALLS_TOKEN}" ]]; then \
		set -ex; \
		(cd /tmp; go get github.com/mattn/goveralls); \
		goveralls -coverprofile="cover.out" -service=github; \
	fi

.PHONY: %-test
%-test: test-prep
	cd $*; \
	go test \
		-race \
		-cover -coverprofile "${PWD}/cover/$*.out" \
		./... >&2

.PHONY: test-prep
test-prep:
	rm -rf cover/
	mkdir cover
	# Remove DNS timeouts from CI builds, so DNS tests with bad nameservers fail as expected.
	if [[ "${CI}" == true ]]; then \
		set -e; \
		sudo sed -i'' -e '/options timeout:/d' /etc/resolv.conf; \
		cat /etc/resolv.conf; \
	fi

out:
	mkdir -p out

.PHONY: deploy-docs
deploy-docs: $(MODULES:=-docs)

.PHONY: %-docs
%-docs: docs-prep
	cd $* && \
	../out/gopages \
		-base /go/$* \
		-out "${PWD}/$*" \
		-gh-pages \
		-gh-pages-user "${GIT_USER}" \
		-gh-pages-token "${GIT_TOKEN}"

.PHONY: docs-prep
docs-prep: out
	cd ./gopages; go build -o ../out/gopages .
