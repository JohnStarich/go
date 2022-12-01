SHELL := /usr/bin/env bash
LINT_VERSION=1.50.1

MODULES = $(sort $(patsubst %/,%,$(dir $(wildcard */go.mod))))
GOLANGCI_FLAGS =
ifeq (${CI},true)
	GOLANGCI_FLAGS = --out-format github-actions
endif

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
	cd $*; golangci-lint run ${GOLANGCI_FLAGS}

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

.PHONY: test-publish-coverage
test-publish-coverage:
	if [[ -n "$$GITHUB_TOKEN" ]]; then \
		set -ex; \
		set -o pipefail; \
		if [[ ! -f cover.out ]]; then \
			echo 'Missing coverage profile cover.out' >&2; \
			exit 1; \
		fi; \
		go install github.com/mattn/goveralls@v0.0.11; \
		COVERALLS_TOKEN=$$GITHUB_TOKEN goveralls -coverprofile="cover.out" -service=github; \
		(cd covet; go install ./cmd/covet); \
		[[ "$$GITHUB_REF" =~ [0-9]+ ]] && ISSUE_NUMBER=$${BASH_REMATCH[0]}; \
		git diff origin/master | \
			covet \
				-diff-file - \
				-cover-go ./cover.out \
				-show-diff-coverage \
				-gh-token "$$GITHUB_TOKEN" \
				-gh-issue "github.com/$${GITHUB_REPOSITORY}/pull/$${ISSUE_NUMBER}" \
				; \
	fi

.PHONY: %-test
%-test: test-prep
	WD="$$PWD"; \
	cd $*; \
	go test \
		-race \
		-cover -coverprofile "$$WD/cover/$*.out" \
		./... >&2

.PHONY: test-prep
test-prep:
	rm -rf cover/
	mkdir cover
	# Remove DNS timeouts from CI builds, so DNS tests with bad nameservers fail as expected.
	@if [[ "${CI}" == true ]]; then \
		set -ex; \
		SED_ARGS=(-e '/options timeout:/d'); \
		if [[ "$$(uname)" == Linux ]]; then \
			sudo sed -i'' "$${SED_ARGS[@]}" /etc/resolv.conf; \
			cat /etc/resolv.conf; \
		elif [[ "$$(uname)" == Darwin ]]; then \
			sudo cp /etc/resolv.conf /etc/resolv.conf.new; \
			sudo sed -i '' "$${SED_ARGS[@]}" /etc/resolv.conf.new; \
			sudo mv /etc/resolv.conf.new /etc/resolv.conf; \
			cat /etc/resolv.conf; \
		fi; \
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
