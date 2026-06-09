from __future__ import annotations

import importlib.util
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path


SCRIPT_PATH = Path(__file__).resolve().parents[1] / "scripts" / "validate_codex_config.py"


def load_validator():
    spec = importlib.util.spec_from_file_location("validate_codex_config", SCRIPT_PATH)
    if spec is None or spec.loader is None:
        raise RuntimeError(f"cannot load {SCRIPT_PATH}")
    module = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = module
    spec.loader.exec_module(module)
    return module


class CodexConfigValidatorTests(unittest.TestCase):
    def setUp(self) -> None:
        self.tmp = tempfile.TemporaryDirectory()
        self.root = Path(self.tmp.name)
        self.codex = self.root / ".codex"
        self.write_minimal_tree()

    def tearDown(self) -> None:
        self.tmp.cleanup()

    def write(self, relative: str, text: str, *, bom: bool = False) -> None:
        path = self.root / relative
        path.parent.mkdir(parents=True, exist_ok=True)
        data = text.replace("\r\n", "\n").replace("\r", "\n").encode("utf-8")
        if bom:
            data = b"\xef\xbb\xbf" + data
        path.write_bytes(data)

    def write_minimal_tree(self) -> None:
        self.write(
            ".codex/project.toml",
            """
[project]
name = "demo"

[source_of_truth]
codex_config = [".codex/context/codex-config-map.md"]
dependency_context = [".codex/context/dependency-map.md"]
business_flow_context = [".codex/context/business-flow-map.md"]
operational_context = [".codex/context/conversation-retrospective-map.md", ".codex/context/runtime-network-debug-map.md"]
operational_workflows = [".codex/workflows/network-routing-debug-checklist.md"]

[stack.testing]
codex = ["python .codex/skills/superxray-project-context/scripts/validate_codex_config.py"]

[stack.ai_config]
agent_contract_fields = ["knowledge_inputs", "handoff_outputs", "collaboration_rules", "efficiency_metrics"]

[agents]
program_manager = "demo-agent"

[skills]
project_context = ".codex/skills/superxray-project-context/SKILL.md"
project_context_validator = ".codex/skills/superxray-project-context/scripts/validate_codex_config.py"
""",
        )
        self.write(
            ".codex/governance.toml",
            """
[governance]
version = 3

[context_budget]
first_read = [
  ".codex/project.toml",
  ".codex/governance.toml",
  ".codex/routing.toml",
  ".codex/context/project-map.md",
  ".codex/context/dependency-map.md",
  ".codex/context/business-flow-map.md",
  ".codex/context/codex-config-map.md",
  ".codex/context/conversation-retrospective-map.md",
  ".codex/context/runtime-network-debug-map.md",
]

[codex_validation]
script = ".codex/skills/superxray-project-context/scripts/validate_codex_config.py"

[codex_directory_policy]
allowed = [".codex/configuration-update.md"]
""",
        )
        self.write(
            ".codex/routing.toml",
            """
[[routes]]
name = "codex-governance"
globs = [".codex/configuration-update.md"]
primary = "demo-agent"
reviewers = ["demo-agent"]
verification = ["python .codex/skills/superxray-project-context/scripts/validate_codex_config.py"]
""",
        )
        self.write(
            ".codex/agents/demo-agent.toml",
            """
name = "demo-agent"
description = "demo"
required_context = [".codex/context/project-map.md"]
knowledge_inputs = [".codex/context/dependency-map.md"]
handoff_outputs = [".codex/context/handoff-template.md", ".codex/context/project-map.md"]
collaboration_rules = ["route first"]
efficiency_metrics = ["first_route_accuracy"]
""",
        )
        for relative in [
            ".codex/README.md",
            ".codex/configuration-update.md",
            ".codex/agents/README.md",
            ".codex/context/project-map.md",
            ".codex/context/handoff-template.md",
            ".codex/context/dependency-map.md",
            ".codex/context/business-flow-map.md",
            ".codex/context/codex-config-map.md",
            ".codex/context/conversation-retrospective-map.md",
            ".codex/context/runtime-network-debug-map.md",
            ".codex/prompts/shared-system-prompt.md",
            ".codex/workflows/multi-agent-workflow.md",
            ".codex/workflows/verification-matrix.md",
            ".codex/workflows/config-validation-and-efficiency.md",
            ".codex/workflows/network-routing-debug-checklist.md",
            ".codex/skills/superxray-project-context/references/current-stack.md",
        ]:
            self.write(relative, "# ok\n")
        self.write(
            ".codex/skills/superxray-project-context/SKILL.md",
            """---
name: superxray-project-context
description: Use when validating project context.
---

# Skill
""",
        )
        self.write(
            ".codex/skills/superxray-project-context/agents/openai.yaml",
            """
display_name: Project Context
short_description: Validate project context.
default_prompt: Validate the project context.
""",
        )
        self.write(".codex/skills/superxray-project-context/scripts/validate_codex_config.py", "# placeholder\n")
        self.write(".codex/skills/superxray-project-context/tests/test_validate_codex_config.py", "# placeholder\n")
        for skill_name in ["superxray-ui-first-migration", "superxray-release-cicd"]:
            self.write(
                f".codex/skills/{skill_name}/SKILL.md",
                f"""---
name: {skill_name}
description: Use when validating {skill_name}.
---

# Skill
""",
            )
            self.write(
                f".codex/skills/{skill_name}/agents/openai.yaml",
                f"""
display_name: {skill_name}
short_description: Validate {skill_name}.
default_prompt: Validate {skill_name}.
""",
            )

    def test_valid_minimal_tree_has_no_errors(self) -> None:
        validator = load_validator()
        report = validator.validate(self.root)
        self.assertEqual([], [issue.code for issue in report.errors], report.to_text())

    def test_route_unknown_agent_is_reported(self) -> None:
        self.write(
            ".codex/routing.toml",
            """
[[routes]]
name = "bad"
primary = "missing-agent"
reviewers = ["demo-agent"]
verification = ["python .codex/skills/superxray-project-context/scripts/validate_codex_config.py"]
""",
        )
        validator = load_validator()
        report = validator.validate(self.root)
        self.assertIn("route_unknown_agent", [issue.code for issue in report.errors])

    def test_utf8_bom_is_rejected(self) -> None:
        self.write(".codex/context/project-map.md", "# ok\n", bom=True)
        validator = load_validator()
        report = validator.validate(self.root)
        self.assertIn("text_encoding_policy", [issue.code for issue in report.errors])

    def test_nested_openai_interface_yaml_is_accepted(self) -> None:
        self.write(
            ".codex/skills/superxray-project-context/agents/openai.yaml",
            """
interface:
  display_name: "Project Context"
  short_description: "Validate project context."
  default_prompt: "Use $superxray-project-context to validate the project context."
""",
        )
        validator = load_validator()
        report = validator.validate(self.root)
        self.assertNotIn("skill_openai_yaml_invalid", [issue.code for issue in report.errors], report.to_text())


if __name__ == "__main__":
    if not SCRIPT_PATH.exists():
        result = subprocess.run(
            [sys.executable, str(SCRIPT_PATH)],
            cwd=Path(__file__).resolve().parents[4],
            text=True,
            capture_output=True,
        )
        print(result.stdout)
        print(result.stderr)
    unittest.main()
