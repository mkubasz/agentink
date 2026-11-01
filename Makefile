# Ink Hub — one entry point for the Go hub and the Python worker.
.PHONY: build test lint provenance e2e worker contracts clean

GO      ?= go
UV      ?= uv
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X main.Version=$(VERSION)

build:
	$(GO) build -ldflags "$(LDFLAGS)" -o bin/inkd ./cmd/inkd
	$(GO) build -ldflags "$(LDFLAGS)" -o bin/ink ./cmd/ink

test:
	$(GO) test ./... -count=1
	@test -f worker/pyproject.toml && $(UV) run --project worker pytest -q worker/tests || true

lint:
	golangci-lint run ./...
	@test -f worker/pyproject.toml && $(UV) run --project worker ruff check worker && $(UV) run --project worker mypy worker/inkworker || true

# ADR 0003, rule 4: no upstream name anywhere but the two allowed files.
# Allowed to name them: this Makefile, CLAUDE.md (the rule itself), ACKNOWLEDGMENTS.md,
# docs/architecture/inventory.md (the concept map) and the ADRs and specs that record the decision.
provenance:
	@! grep -rIn -E 'michal-franc|issue-viewer|issue-cli|ai-workflow-engine|dark-factory' \
	    --exclude-dir=.git --exclude-dir=.venv --exclude-dir=docs \
	    --exclude=Makefile --exclude=CLAUDE.md --exclude=ACKNOWLEDGMENTS.md . \
	  || { echo "provenance: upstream name found outside the allowed files"; exit 1; }
	@echo "provenance: clean"

e2e:
	@echo "e2e: Phase 1.7 adds the skeleton"; exit 1

worker:
	$(UV) sync --project worker

# ADR 0008: the schemas are the source of truth; both sides prove their examples.
contracts:
	$(GO) test ./contracts/... -count=1
	$(UV) run --project worker pytest -q worker/tests/test_contracts.py

clean:
	rm -rf bin cover.out
