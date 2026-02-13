SHELL := /bin/bash

.PHONY: hooks fmt lint lint-fast test test-fast build \
	sast-fast codeql-local license-check \
	test-e2e test-acceptance test-contracts test-conformance \
	test-runtime-slo test-hardening-acceptance test-v1-acceptance \
	test-adoption test-uat-local docs-site-install docs-site-build docs-site-lint

hooks:
	git config core.hooksPath .githooks
	chmod +x .githooks/pre-push
	@echo "[wrkr] git hooks installed"

fmt:
	@files=$$(find . -type f -name '*.go' -not -path './.git/*'); \
	if [ -n "$$files" ]; then gofmt -w $$files; fi

lint-fast:
	go vet ./...

lint: lint-fast
	@echo "[wrkr] lint complete"

test-fast:
	go test ./...
	cd sdk/python && uv run --python 3.13 --extra dev pytest -q

test: test-fast
	@echo "[wrkr] test complete"

build:
	go build -o ./bin/wrkr ./cmd/wrkr

sast-fast:
	./scripts/sast_fast.sh

codeql-local:
	./scripts/run_codeql_local.sh

license-check:
	./scripts/license_check.sh

test-e2e:
	go test ./internal/integration/... -count=1

test-acceptance:
	go test ./core/accept/... ./core/report ./cmd/wrkr -run 'TestAccept|TestReport|TestBuildAndWriteGitHubSummary|TestRunWritesAcceptanceResult|TestRunFailureCode' -count=1

test-contracts:
	./scripts/test_contracts.sh

test-conformance:
	@echo "[wrkr] test-conformance placeholder (Epic >0)"

test-runtime-slo:
	@echo "[wrkr] test-runtime-slo placeholder (Epic >0)"

test-hardening-acceptance:
	@echo "[wrkr] test-hardening-acceptance placeholder (Epic >0)"

test-v1-acceptance:
	@echo "[wrkr] test-v1-acceptance placeholder (Epic >0)"

test-adoption:
	@echo "[wrkr] test-adoption placeholder (Epic >0)"

test-uat-local:
	@echo "[wrkr] test-uat-local placeholder (Epic >0)"

docs-site-install:
	@echo "[wrkr] docs-site-install placeholder (Epic >0)"

docs-site-build:
	@echo "[wrkr] docs-site-build placeholder (Epic >0)"

docs-site-lint:
	@echo "[wrkr] docs-site-lint placeholder (Epic >0)"
