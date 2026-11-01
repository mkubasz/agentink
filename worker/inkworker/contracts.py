"""The shared JSON Schema contracts (ADR 0008), read from the repository's `contracts/`."""

from __future__ import annotations

import json
from functools import cache
from pathlib import Path
from typing import Any

from jsonschema import Draft202012Validator

NAMES = ("event", "question", "page", "zones", "ocr", "answer")


def contracts_dir() -> Path:
    """Return the `contracts/` directory of the monorepo this package lives in."""
    return Path(__file__).resolve().parents[2] / "contracts"


@cache
def schema(name: str) -> dict[str, Any]:
    """Load one contract by name."""
    if name not in NAMES:
        raise KeyError(f"unknown contract {name!r}")

    path = contracts_dir() / f"{name}.schema.json"
    with path.open(encoding="utf-8") as fh:
        loaded: dict[str, Any] = json.load(fh)

    return loaded


@cache
def validator(name: str) -> Draft202012Validator:
    """Return a compiled validator for one contract."""
    sch = schema(name)
    Draft202012Validator.check_schema(sch)

    return Draft202012Validator(sch)


def validate(name: str, doc: Any) -> None:
    """Raise jsonschema.ValidationError when doc does not match the named contract."""
    validator(name).validate(doc)
