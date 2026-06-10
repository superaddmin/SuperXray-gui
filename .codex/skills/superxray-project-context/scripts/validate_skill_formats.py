#!/usr/bin/env python3
from __future__ import annotations

import os
import subprocess
import sys
from pathlib import Path


def candidate_validators() -> list[Path]:
    candidates: list[Path] = []
    env_path = os.environ.get("CODEX_SKILL_VALIDATOR")
    if env_path:
        candidates.append(Path(env_path))
    candidates.append(Path.home() / ".codex" / "skills" / ".system" / "skill-creator" / "scripts" / "quick_validate.py")
    return candidates


def main(argv: list[str] | None = None) -> int:
    args = list(sys.argv[1:] if argv is None else argv)
    if not args:
        args = [
            ".codex/skills/superxray-project-context",
            ".codex/skills/superxray-ui-first-migration",
            ".codex/skills/superxray-release-cicd",
        ]

    validator = next((path for path in candidate_validators() if path.exists()), None)
    if validator is None:
        print("SKIP: global skill quick_validate.py not found; structural .codex validation should still be run.")
        return 0

    for skill_path in args:
        result = subprocess.run([sys.executable, str(validator), skill_path], text=True)
        if result.returncode != 0:
            return result.returncode
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
