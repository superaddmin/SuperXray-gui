#!/usr/bin/env python3
from __future__ import annotations

import argparse
import re
import sys
import tomllib
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any


TEXT_SUFFIXES = {".md", ".toml", ".yaml", ".yml", ".py"}
REQUIRED_AGENT_FIELDS = {
    "knowledge_inputs",
    "handoff_outputs",
    "collaboration_rules",
    "efficiency_metrics",
}
REQUIRED_CODEX_FILES = [
    ".codex/README.md",
    ".codex/configuration-update.md",
    ".codex/project.toml",
    ".codex/governance.toml",
    ".codex/routing.toml",
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
    ".codex/skills/superxray-project-context/SKILL.md",
    ".codex/skills/superxray-project-context/agents/openai.yaml",
    ".codex/skills/superxray-project-context/references/current-stack.md",
    ".codex/skills/superxray-project-context/scripts/validate_skill_formats.py",
    ".codex/skills/superxray-project-context/scripts/validate_codex_config.py",
    ".codex/skills/superxray-project-context/tests/test_validate_codex_config.py",
]
REQUIRED_SKILLS = [
    ".codex/skills/superxray-project-context",
    ".codex/skills/superxray-ui-first-migration",
    ".codex/skills/superxray-release-cicd",
]
REQUIRED_CONTEXT_INDEXES = [
    ".codex/context/dependency-map.md",
    ".codex/context/business-flow-map.md",
    ".codex/context/codex-config-map.md",
    ".codex/context/conversation-retrospective-map.md",
    ".codex/context/runtime-network-debug-map.md",
]
VALIDATOR_COMMAND = "python .codex/skills/superxray-project-context/scripts/validate_codex_config.py"
WINDOWS_ABSOLUTE_PATH_RE = re.compile(r"(?:^|\s|['\"])[A-Za-z]:[\\/]")
PATH_PREFIXES = (".codex/", "scripts/", "frontend/", ".github/", "docs/", "plans/")


@dataclass(slots=True)
class Issue:
    level: str
    code: str
    path: str
    message: str


@dataclass(slots=True)
class Report:
    errors: list[Issue] = field(default_factory=list)
    warnings: list[Issue] = field(default_factory=list)

    def add_error(self, code: str, path: str | Path, message: str) -> None:
        self.errors.append(Issue("error", code, str(path), message))

    def add_warning(self, code: str, path: str | Path, message: str) -> None:
        self.warnings.append(Issue("warning", code, str(path), message))

    def to_text(self) -> str:
        lines: list[str] = []
        if not self.errors and not self.warnings:
            return "OK: .codex configuration validation passed."
        for issue in [*self.errors, *self.warnings]:
            lines.append(f"{issue.level.upper()} {issue.code} {issue.path}: {issue.message}")
        return "\n".join(lines)


def rel_path(path: Path, root: Path) -> str:
    try:
        return path.relative_to(root).as_posix()
    except ValueError:
        return path.as_posix()


def read_toml(path: Path, root: Path, report: Report) -> dict[str, Any]:
    if not path.exists():
        report.add_error("required_file_missing", rel_path(path, root), "required TOML file is missing")
        return {}
    try:
        return tomllib.loads(path.read_text(encoding="utf-8"))
    except tomllib.TOMLDecodeError as exc:
        report.add_error("toml_parse_error", rel_path(path, root), str(exc))
    except UnicodeDecodeError as exc:
        report.add_error("text_encoding_policy", rel_path(path, root), f"not valid UTF-8: {exc}")
    return {}


def ensure_required_files(root: Path, report: Report) -> None:
    for relative in REQUIRED_CODEX_FILES:
        path = root / relative
        if not path.exists():
            report.add_error("required_file_missing", relative, "required .codex file is missing")
    for skill in REQUIRED_SKILLS:
        skill_dir = root / skill
        if not skill_dir.exists():
            report.add_error("required_skill_missing", skill, "required project skill directory is missing")
            continue
        for relative in ["SKILL.md", "agents/openai.yaml"]:
            path = skill_dir / relative
            if not path.exists():
                report.add_error("required_skill_file_missing", rel_path(path, root), "required skill file is missing")


def check_text_policy(root: Path, report: Report) -> None:
    codex = root / ".codex"
    if not codex.exists():
        report.add_error("required_directory_missing", ".codex", "required .codex directory is missing")
        return
    for path in sorted(codex.rglob("*")):
        if not path.is_file() or path.suffix.lower() not in TEXT_SUFFIXES:
            continue
        relative = rel_path(path, root)
        data = path.read_bytes()
        if data.startswith(b"\xef\xbb\xbf"):
            report.add_error("text_encoding_policy", relative, "must be UTF-8 without BOM")
        if b"\r\n" in data or b"\r" in data:
            report.add_error("text_encoding_policy", relative, "must use LF line endings")
        try:
            data.decode("utf-8")
        except UnicodeDecodeError as exc:
            report.add_error("text_encoding_policy", relative, f"must be valid UTF-8: {exc}")


def parse_agents(root: Path, report: Report) -> dict[str, dict[str, Any]]:
    agents: dict[str, dict[str, Any]] = {}
    agents_dir = root / ".codex" / "agents"
    if not agents_dir.exists():
        report.add_error("required_directory_missing", ".codex/agents", "agent directory is missing")
        return agents
    for path in sorted(agents_dir.glob("*.toml")):
        data = read_toml(path, root, report)
        name = data.get("name")
        if not isinstance(name, str) or not name:
            report.add_error("agent_name_missing", rel_path(path, root), "agent TOML must contain a non-empty name")
            continue
        agents[name] = data
        missing = sorted(REQUIRED_AGENT_FIELDS - set(data))
        if missing:
            report.add_error("agent_contract_missing", rel_path(path, root), f"missing contract fields: {', '.join(missing)}")
        for field_name in REQUIRED_AGENT_FIELDS:
            value = data.get(field_name)
            if value is not None and (not isinstance(value, list) or not all(isinstance(item, str) and item for item in value)):
                report.add_error("agent_contract_invalid", rel_path(path, root), f"{field_name} must be a non-empty string array")
        for list_field in ["required_context", "knowledge_inputs", "handoff_outputs"]:
            for item in data.get(list_field, []) if isinstance(data.get(list_field), list) else []:
                if not isinstance(item, str) or "*" in item:
                    continue
                candidate = root / item
                if item.startswith(".codex/") and not candidate.exists():
                    report.add_error("agent_context_missing", rel_path(path, root), f"{list_field} references missing file: {item}")
    return agents


def as_list(value: Any) -> list[Any]:
    if isinstance(value, list):
        return value
    return []


def validate_project_references(root: Path, report: Report, agents: dict[str, dict[str, Any]]) -> None:
    project_path = root / ".codex" / "project.toml"
    project = read_toml(project_path, root, report)
    source_of_truth = project.get("source_of_truth", {})
    if not isinstance(source_of_truth, dict):
        report.add_error("project_source_of_truth_invalid", ".codex/project.toml", "source_of_truth must be a table")
        source_of_truth = {}
    required_source_keys = {
        "codex_config": [".codex/context/codex-config-map.md"],
        "dependency_context": [".codex/context/dependency-map.md"],
        "business_flow_context": [".codex/context/business-flow-map.md"],
        "operational_context": [
            ".codex/context/conversation-retrospective-map.md",
            ".codex/context/runtime-network-debug-map.md",
        ],
        "operational_workflows": [
            ".codex/workflows/network-routing-debug-checklist.md",
        ],
    }
    for key, expected_items in required_source_keys.items():
        values = as_list(source_of_truth.get(key))
        for expected in expected_items:
            if expected not in values:
                report.add_error("project_context_index_missing", ".codex/project.toml", f"source_of_truth.{key} must include {expected}")
    stack = project.get("stack", {})
    if not isinstance(stack, dict) or not isinstance(stack.get("ai_config"), dict):
        report.add_error("project_ai_config_missing", ".codex/project.toml", "missing [stack.ai_config]")
    testing = stack.get("testing", {}) if isinstance(stack, dict) else {}
    if not isinstance(testing, dict) or "codex" not in testing:
        report.add_error("project_codex_testing_missing", ".codex/project.toml", "missing [stack.testing].codex validation command")
    skills = project.get("skills", {})
    if not isinstance(skills, dict) or "project_context_validator" not in skills:
        report.add_error("project_validator_skill_missing", ".codex/project.toml", "missing [skills].project_context_validator")
    for alias, agent_name in (project.get("agents", {}) if isinstance(project.get("agents"), dict) else {}).items():
        if isinstance(agent_name, str) and agent_name not in agents:
            report.add_error("project_unknown_agent", ".codex/project.toml", f"[agents].{alias} references unknown agent {agent_name}")
    for key, values in source_of_truth.items():
        for item in as_list(values):
            if isinstance(item, str) and item.startswith(".codex/") and "*" not in item and not (root / item).exists():
                report.add_error("project_reference_missing", ".codex/project.toml", f"source_of_truth.{key} references missing file: {item}")


def validate_governance(root: Path, report: Report) -> None:
    governance_path = root / ".codex" / "governance.toml"
    governance = read_toml(governance_path, root, report)
    version = governance.get("governance", {}).get("version") if isinstance(governance.get("governance"), dict) else None
    if version != 3:
        report.add_error("governance_version_outdated", ".codex/governance.toml", "governance.version must be 3")
    context_budget = governance.get("context_budget", {})
    if isinstance(context_budget, dict):
        context_reads = [
            *as_list(context_budget.get("first_read")),
            *as_list(context_budget.get("bootstrap_read")),
            *as_list(context_budget.get("extended_read")),
            *as_list(context_budget.get("on_demand_read")),
        ]
    else:
        context_reads = []
    for expected in REQUIRED_CONTEXT_INDEXES:
        if expected not in context_reads:
            report.add_error("governance_first_read_missing", ".codex/governance.toml", f"context_budget.first_read must include {expected}")
    codex_validation = governance.get("codex_validation")
    if not isinstance(codex_validation, dict):
        report.add_error("governance_validation_missing", ".codex/governance.toml", "missing [codex_validation]")
    else:
        script = codex_validation.get("script")
        if script != ".codex/skills/superxray-project-context/scripts/validate_codex_config.py":
            report.add_error("governance_validation_invalid", ".codex/governance.toml", "codex_validation.script must point to validate_codex_config.py")
    policy = governance.get("codex_directory_policy", {})
    allowed = as_list(policy.get("allowed")) if isinstance(policy, dict) else []
    if ".codex/configuration-update.md" not in allowed:
        report.add_error("governance_configuration_update_not_allowed", ".codex/governance.toml", "allowed list must include .codex/configuration-update.md")
    validate_sensitive_globs_are_ignored(root, report, governance)


def validate_agent_context_budget(root: Path, report: Report, agents: dict[str, dict[str, Any]]) -> None:
    governance = read_toml(root / ".codex" / "governance.toml", root, report)
    context_budget = governance.get("context_budget", {}) if isinstance(governance.get("context_budget"), dict) else {}
    max_context = context_budget.get("max_context_files_per_turn", 10)
    if not isinstance(max_context, int) or max_context <= 0:
        return
    for agent_name, data in sorted(agents.items()):
        total = len(as_list(data.get("required_context"))) + len(as_list(data.get("knowledge_inputs")))
        if total > max_context:
            report.add_warning(
                "agent_context_budget_exceeded",
                agent_name,
                f"declares {total} required_context + knowledge_inputs entries; budget is {max_context}",
            )


def normalized_ignore_patterns(root: Path) -> set[str]:
    gitignore = root / ".gitignore"
    if not gitignore.exists():
        return set()
    patterns: set[str] = set()
    for raw in gitignore.read_text(encoding="utf-8").splitlines():
        line = raw.strip()
        if not line or line.startswith("#") or line.startswith("!"):
            continue
        variants = {line}
        stripped = line.lstrip("/")
        variants.add(stripped)
        variants.add(stripped.rstrip("/"))
        if stripped.endswith("/"):
            variants.add(stripped.rstrip("/") + "/**")
        if stripped.endswith("/**"):
            variants.add(stripped.removesuffix("/**"))
            variants.add(stripped.removesuffix("/**") + "/")
        patterns.update(variants)
    return patterns


def validate_sensitive_globs_are_ignored(root: Path, report: Report, governance: dict[str, Any]) -> None:
    security = governance.get("security", {})
    if not isinstance(security, dict):
        return
    sensitive_globs = [item for item in as_list(security.get("sensitive_artifact_globs")) if isinstance(item, str)]
    if not sensitive_globs:
        return
    ignored = normalized_ignore_patterns(root)
    for pattern in sensitive_globs:
        normalized = pattern.lstrip("/")
        accepted = {
            normalized,
            normalized.rstrip("/"),
            normalized.rstrip("/") + "/",
            normalized.rstrip("/") + "/**",
        }
        if not accepted.intersection(ignored):
            report.add_warning(
                "sensitive_glob_not_ignored",
                ".codex/governance.toml",
                f"security.sensitive_artifact_globs includes {pattern!r}, but .gitignore does not ignore it",
            )


def validate_routing(root: Path, report: Report, agents: dict[str, dict[str, Any]]) -> None:
    routing_path = root / ".codex" / "routing.toml"
    routing = read_toml(routing_path, root, report)
    routes = as_list(routing.get("routes"))
    if not routes:
        report.add_error("routing_empty", ".codex/routing.toml", "routing.toml must define at least one [[routes]] entry")
    for index, route in enumerate(routes):
        if not isinstance(route, dict):
            report.add_error("routing_invalid", ".codex/routing.toml", f"route #{index} must be a table")
            continue
        name = route.get("name", f"#{index}")
        globs = as_list(route.get("globs"))
        if not globs or not all(isinstance(item, str) and item for item in globs):
            report.add_error("route_globs_missing", ".codex/routing.toml", f"route {name} must define a non-empty globs string array")
        primary = route.get("primary")
        if isinstance(primary, str) and primary not in agents:
            report.add_error("route_unknown_agent", ".codex/routing.toml", f"route {name} primary references unknown agent {primary}")
        for reviewer in as_list(route.get("reviewers")):
            if isinstance(reviewer, str) and reviewer not in agents:
                report.add_error("route_unknown_agent", ".codex/routing.toml", f"route {name} reviewer references unknown agent {reviewer}")
        for command in as_list(route.get("verification")):
            validate_command_portability(root, report, ".codex/routing.toml", command, f"route {name}")
    governance_routes = [r for r in routes if isinstance(r, dict) and r.get("name") == "codex-governance"]
    if governance_routes:
        route = governance_routes[0]
        globs = as_list(route.get("globs"))
        verification = as_list(route.get("verification"))
        if ".codex/configuration-update.md" not in globs:
            report.add_error("routing_configuration_update_missing", ".codex/routing.toml", "codex-governance globs must include .codex/configuration-update.md")
        if VALIDATOR_COMMAND not in verification:
            report.add_error("routing_validator_missing", ".codex/routing.toml", "codex-governance verification must include validate_codex_config.py")
    project_skill_routes = [r for r in routes if isinstance(r, dict) and r.get("name") == "project-skills"]
    if project_skill_routes and VALIDATOR_COMMAND not in as_list(project_skill_routes[0].get("verification")):
        report.add_error("routing_validator_missing", ".codex/routing.toml", "project-skills verification must include validate_codex_config.py")


def validate_command_portability(root: Path, report: Report, path: str, command: Any, label: str) -> None:
    if not isinstance(command, str):
        return
    if WINDOWS_ABSOLUTE_PATH_RE.search(command):
        report.add_warning("non_portable_absolute_path", path, f"{label} verification command contains a machine-specific absolute path: {command}")
    for raw_token in re.split(r"\s+", command):
        token = raw_token.replace("\\", "/").strip("`'\";,)")
        if not token.startswith(PATH_PREFIXES):
            continue
        if "*" in token or token.endswith("/..."):
            continue
        token = token.rstrip(":")
        if not (root / token).exists():
            report.add_warning("verification_path_missing", path, f"{label} verification command references missing path: {token}")


FRONTMATTER_RE = re.compile(r"^---\n(?P<body>.*?)\n---\n", re.DOTALL)


def validate_skills(root: Path, report: Report) -> None:
    skills_dir = root / ".codex" / "skills"
    if not skills_dir.exists():
        report.add_error("required_directory_missing", ".codex/skills", "skills directory is missing")
        return
    for skill_dir in sorted(path for path in skills_dir.iterdir() if path.is_dir()):
        skill_md = skill_dir / "SKILL.md"
        openai_yaml = skill_dir / "agents" / "openai.yaml"
        if not skill_md.exists():
            report.add_error("skill_frontmatter_missing", rel_path(skill_md, root), "SKILL.md is missing")
            continue
        text = skill_md.read_text(encoding="utf-8")
        match = FRONTMATTER_RE.match(text)
        if not match:
            report.add_error("skill_frontmatter_missing", rel_path(skill_md, root), "SKILL.md must start with YAML frontmatter")
        else:
            frontmatter = match.group("body")
            if not re.search(r"(?m)^name:\s*\S+", frontmatter):
                report.add_error("skill_frontmatter_invalid", rel_path(skill_md, root), "frontmatter missing name")
            if not re.search(r"(?m)^description:\s*.+", frontmatter):
                report.add_error("skill_frontmatter_invalid", rel_path(skill_md, root), "frontmatter missing description")
        if not openai_yaml.exists():
            report.add_error("skill_openai_yaml_missing", rel_path(openai_yaml, root), "agents/openai.yaml is missing")
            continue
        yaml_text = openai_yaml.read_text(encoding="utf-8")
        for key in ["display_name", "short_description", "default_prompt"]:
            if not re.search(rf"(?m)^\s*{re.escape(key)}:\s*.+", yaml_text):
                report.add_error("skill_openai_yaml_invalid", rel_path(openai_yaml, root), f"missing {key}")


def validate(root: str | Path = ".") -> Report:
    root_path = Path(root).resolve()
    report = Report()
    ensure_required_files(root_path, report)
    check_text_policy(root_path, report)
    agents = parse_agents(root_path, report)
    validate_project_references(root_path, report, agents)
    validate_agent_context_budget(root_path, report, agents)
    validate_governance(root_path, report)
    validate_routing(root_path, report, agents)
    validate_skills(root_path, report)
    return report


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Validate SuperXray-gui project-local .codex configuration.")
    parser.add_argument("root", nargs="?", default=".", help="repository root, default: current directory")
    args = parser.parse_args(argv)
    report = validate(args.root)
    print(report.to_text())
    return 1 if report.errors else 0


if __name__ == "__main__":
    raise SystemExit(main())
