SHELL := /bin/bash

.PHONY: hooks fmt lint lint-fast test test-fast build \
	sast-fast codeql-local license-check \
	test-e2e test-acceptance test-contracts test-ent-consumer-contract test-conformance \
	test-ticket-footer-conformance test-github-summary-golden test-wrkr-compatible-conformance test-serve-hardening test-release-contracts \
	install-smoke release-smoke \
	test-runtime-slo test-scale-profile test-serve-slo test-hardening-acceptance test-v1-acceptance coverage \
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

test-ent-consumer-contract:
	./scripts/test_ent_consumer_contract.sh

test-ticket-footer-conformance:
	./scripts/test_ticket_footer_conformance.sh

test-github-summary-golden:
	./scripts/test_github_summary_golden.sh

test-wrkr-compatible-conformance:
	./scripts/test_wrkr_compatible_conformance.sh

test-serve-hardening:
	./scripts/test_serve_hardening.sh

test-release-contracts:
	./scripts/test_release_contracts.sh

test-conformance:
	./scripts/test_ticket_footer_conformance.sh
	./scripts/test_github_summary_golden.sh
	./scripts/test_wrkr_compatible_conformance.sh
	./scripts/test_serve_hardening.sh
	./scripts/test_release_contracts.sh

install-smoke:
	go build -o ./bin/wrkr ./cmd/wrkr
	./bin/wrkr --json version >/dev/null

release-smoke:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./bin/wrkr-linux-amd64 ./cmd/wrkr
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o ./bin/wrkr-darwin-arm64 ./cmd/wrkr

test-runtime-slo:
	python3 ./scripts/check_command_budgets.py --budgets ./perf/runtime_slo_budgets.json
	python3 ./scripts/check_resource_budgets.py --budgets ./perf/resource_budgets.json

test-scale-profile:
	python3 ./scripts/check_scale_profiles.py --budgets ./perf/scale_profile_budgets.json

test-serve-slo:
	python3 ./scripts/check_serve_perf.py --budgets ./perf/serve_slo_budgets.json

test-hardening-acceptance:
	./scripts/test_chaos_store.sh
	./scripts/test_chaos_runner.sh
	./scripts/test_chaos_serve.sh
	./scripts/test_session_soak.sh 5

test-v1-acceptance:
	$(MAKE) test-contracts
	$(MAKE) test-acceptance
	$(MAKE) test-conformance
	$(MAKE) test-runtime-slo
	$(MAKE) test-scale-profile
	$(MAKE) test-serve-slo

coverage:
	python3 ./scripts/check_coverage_thresholds.py --config ./perf/coverage_thresholds.json

test-adoption:
	./scripts/test_adoption_smoke.sh
	./scripts/test_adapter_parity.sh

test-uat-local:
	./scripts/test_uat_local.sh

docs-site-install:
	cd docs-site && npm ci

docs-site-build:
	cd docs-site && npm run build

docs-site-lint:
	cd docs-site && npm run lint
