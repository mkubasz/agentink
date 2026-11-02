# AgentInk — one entry point for the Go hub and the Python worker (ADR 0009).
# `just` lists the recipes; `just gate` is what runs before every commit.

set shell := ["bash", "-euo", "pipefail", "-c"]

go := env("GO", "go")
uv := env("UV", "uv")
version := `git describe --tags --always --dirty 2>/dev/null || echo dev`
ldflags := "-X main.Version=" + version

# List the recipes.
default:
    @just --list

# Build bin/inkd and bin/ink.
build:
    {{go}} build -ldflags "{{ldflags}}" -o bin/inkd ./cmd/inkd
    {{go}} build -ldflags "{{ldflags}}" -o bin/ink ./cmd/ink

# Go tests and the worker's pytest.
test:
    {{go}} test ./... -count=1
    {{uv}} run --project worker pytest -q worker/tests

# golangci-lint, ruff and mypy.
lint:
    golangci-lint run ./...
    golangci-lint run --build-tags e2e ./e2e/...
    {{uv}} run --project worker ruff check worker
    {{uv}} run --project worker mypy worker/inkworker

# Allowed to name the upstream: this justfile, CLAUDE.md, ACKNOWLEDGMENTS.md, docs/.
# No upstream name outside the allowed files (ADR 0003, rule 4).
provenance:
    #!/usr/bin/env bash
    set -euo pipefail
    if grep -rIn -E 'michal-franc|issue-viewer|issue-cli|ai-workflow-engine|dark-factory' \
        --exclude-dir=.git --exclude-dir=.venv --exclude-dir=.mypy_cache --exclude-dir=docs \
        --exclude=justfile --exclude=CLAUDE.md --exclude=ACKNOWLEDGMENTS.md .; then
      echo "provenance: upstream name found outside the allowed files"
      exit 1
    fi
    echo "provenance: clean"

# Before every commit: test, lint, provenance.
gate: test lint provenance

# The whole loop with the real slices so far and stand-ins for the rest (spec 1.7).
e2e: build
    {{go}} test ./e2e/ -count=1 -tags e2e -v -run TestE2E 2>&1 | tail -20

# Sync the worker's Python environment.
worker:
    {{uv}} sync --project worker

# ADR 0008: the schemas are the source of truth; both sides prove their examples.
contracts:
    {{go}} test ./contracts/... -count=1
    {{uv}} run --project worker pytest -q worker/tests/test_contracts.py

# Remove build output.
clean:
    rm -rf bin cover.out
