import json
from pathlib import Path

import pytest
from jsonschema import ValidationError

from inkworker import contracts

EXAMPLES = contracts.contracts_dir() / "examples"


def _cases(kind: str) -> list[tuple[str, Path]]:
    out: list[tuple[str, Path]] = []
    for name in contracts.NAMES:
        folder = EXAMPLES / name if kind == "valid" else EXAMPLES / name / "invalid"
        out.extend((name, p) for p in sorted(folder.glob("*.json")))
    return out


@pytest.mark.parametrize(("name", "path"), _cases("valid"), ids=lambda v: str(v) if isinstance(v, Path) else v)
def test_valid_examples_pass(name: str, path: Path) -> None:
    contracts.validate(name, json.loads(path.read_text(encoding="utf-8")))


@pytest.mark.parametrize(("name", "path"), _cases("invalid"), ids=lambda v: str(v) if isinstance(v, Path) else v)
def test_invalid_examples_fail(name: str, path: Path) -> None:
    with pytest.raises(ValidationError):
        contracts.validate(name, json.loads(path.read_text(encoding="utf-8")))


def test_every_contract_has_examples() -> None:
    for name in contracts.NAMES:
        assert list((EXAMPLES / name).glob("*.json")), name
        assert list((EXAMPLES / name / "invalid").glob("*.json")), name


def test_unknown_contract() -> None:
    with pytest.raises(KeyError):
        contracts.schema("nope")
