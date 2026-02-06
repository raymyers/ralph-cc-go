#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.12"
# dependencies = [
#   "openhands-sdk==1.11.1",
#   "openhands-tools==1.11.1",
#   "rich>=13.7.0"
# ]
# ///
"""
Hans - Compiler hardening with OpenHands SDK.

4 parallel agents coordinated by an orchestrator for compiler debugging.
Uses OpenHands SDK delegation for in-process parallel execution.

Usage:
    export LLM_API_KEY=your_api_key
    uv run hans.py [--task TASK_DESCRIPTION] [--phase PHASE]

Examples:
    uv run hans.py --phase stabilize
    uv run hans.py --task "Fix these seeds: 12345, 67890, 11111, 22222"

Lint/Typecheck:
    uvx ruff check hans.py
    uvx ruff format hans.py
    uvx --with openhands-sdk --with openhands-tools --with rich ty check hans.py
"""

import argparse
import os
import subprocess
import sys
from pathlib import Path

from pydantic import SecretStr
from rich.console import Console
from rich.panel import Panel

from openhands.sdk import (
    LLM,
    Agent,
    AgentContext,
    Conversation,
    Tool,
    get_logger,
)
from openhands.sdk.context import Skill
from openhands.sdk.tool import register_tool
from openhands.tools.delegate import (
    DelegateTool,
    DelegationVisualizer,
    register_agent,
)
from openhands.tools.file_editor import FileEditorTool
from openhands.tools.terminal import TerminalTool

logger = get_logger(__name__)
console = Console()


# =============================================================================
# Git & Shell Helpers
# =============================================================================


def sh(cmd: str) -> tuple[int, str]:
    """Run shell command. Returns (returncode, stdout+stderr)."""
    r = subprocess.run(cmd, shell=True, capture_output=True, text=True, timeout=120)
    return r.returncode, r.stdout + r.stderr


def is_dirty() -> bool:
    """True if uncommitted changes exist."""
    return bool(sh("git status --porcelain 2>/dev/null")[1].strip())


def git_reset() -> None:
    """Discard all local changes."""
    sh("git reset --hard HEAD && git clean -fd")
    console.print("[green]✓ Reset[/green]")


def git_pull() -> None:
    """Pull latest."""
    code, out = sh("git pull --rebase origin main")
    console.print(
        "[green]✓ Pulled[/green]" if code == 0 else "[yellow]⚠ Pull failed[/yellow]"
    )


def git_commit_and_push(msg: str) -> bool:
    """Stage, commit, push."""
    sh("git add -A")
    code, out = sh(f'git commit -m "{msg}"')
    if code != 0:
        return "nothing to commit" in out
    console.print("[green]✓ Committed[/green]")
    code, _ = sh("git push origin main")
    console.print(
        "[green]✓ Pushed[/green]" if code == 0 else "[yellow]⚠ Push failed[/yellow]"
    )
    return True


def run_tests() -> bool:
    """Run make test."""
    console.print("[dim]Running make test...[/dim]")
    code, out = sh("make test")
    if code == 0:
        console.print("[green]✓ Tests passed[/green]")
        return True
    if "No rule" in out or "not found" in out:
        return True  # No tests configured
    console.print("[red]✗ Tests failed[/red]")
    return False


def generate_commit_message(llm: "LLM") -> str:
    """Use a small agent to generate a commit message from git diff."""
    _, diff = sh("git diff --cached --stat && echo '---' && git diff --cached")
    if len(diff) > 8000:
        diff = diff[:8000] + "\n... (truncated)"

    # Quick single-turn LLM call for commit message
    from openhands.sdk import Agent, Conversation, Tool
    from openhands.tools.terminal import TerminalTool

    agent = Agent(llm=llm, tools=[Tool(name=TerminalTool.name)])
    conv = Conversation(agent=agent, workspace=os.getcwd())
    conv.send_message(  # type: ignore[attr-defined]
        f"""Generate a git commit message for these changes. Format:
<type>(<scope>): <subject>

<body>

Where type is fix/feat/refactor, scope is the component, subject is ~50 chars.
Body should be 2-3 bullet points of what was fixed.

Diff:
```
{diff}
```

Reply with ONLY the commit message, nothing else."""
    )
    conv.run()  # type: ignore[attr-defined]

    # Extract final response
    from openhands.sdk.conversation.response_utils import get_agent_final_response

    msg = get_agent_final_response(conv.state.events) or "fix: HANS automated fixes"  # type: ignore[attr-defined]
    return msg.strip()


# =============================================================================
# Agent Factory Functions
# =============================================================================

COMPILER_DEBUG_SKILL = Skill(
    name="compiler_debug",
    content="""You are debugging the ralph-cc compiler. Key commands:

BUILD & TEST:
- `make build` (~1s) - build compiler
- `make test` (~2s) - smoke test
- `make check` (~30s) - full test suite

IR DUMPS (find divergence stage):
- `./bin/ralph-cc --dparse  test.c`  - parse tree
- `./bin/ralph-cc --dclight test.c`  - Clight IR
- `./bin/ralph-cc --drtl    test.c`  - RTL IR
- `./bin/ralph-cc --dltl    test.c`  - LTL IR
- `./bin/ralph-cc --dmach   test.c`  - machine code
- `./bin/ralph-cc --dasm    test.c`  - assembly

CLASSIFICATION:
- Panic: Go stack trace in ralph-cc
- Runtime crash: exit code > 128
- Wrong value: output differs from gcc
- Bad assembly: invalid asm generated

GCC IS ALWAYS RIGHT. If gcc and ralph-cc disagree, ralph-cc has the bug.

After fixing, ALWAYS run `make test` to verify.""",
    trigger=None,
)


def create_seed_fixer_agent(llm: LLM) -> Agent:
    """Agent specialized in fixing csmith seed failures."""
    skills = [
        COMPILER_DEBUG_SKILL,
        Skill(
            name="seed_fixer",
            content="""You fix csmith-generated test failures.

WORKFLOW:
1. Get the test file from `csmith-reports/crash_<seed>.c` or `mismatch_<seed>.c`
2. If missing, regenerate: check `scripts/csmith-fuzz.sh` for flags, run with `--seed <seed>`
3. Compile with both gcc and ralph-cc, compare outputs
4. Use IR dumps to find where ralph-cc diverges
5. Make minimal fix - one bug, one fix
6. Add seed to regression.sh
7. Verify with `make test`

Return a summary: what was wrong, what you fixed, file(s) changed.""",
            trigger=None,
        ),
    ]
    return Agent(
        llm=llm,
        tools=[Tool(name=TerminalTool.name), Tool(name=FileEditorTool.name)],
        agent_context=AgentContext(skills=skills),
    )


def create_feature_hardener_agent(llm: LLM) -> Agent:
    """Agent specialized in hardening compiler features."""
    skills = [
        COMPILER_DEBUG_SKILL,
        Skill(
            name="feature_hardener",
            content="""You harden specific compiler features.

WORKFLOW:
1. Write a minimal C program exercising the target feature
2. Test with gcc vs ralph-cc
3. If they differ, diagnose with IR dumps
4. Fix ralph-cc to match gcc behavior
5. Add test case to e2e_runtime.yaml
6. Verify with `make test`

Focus on edge cases and corner cases for the feature.
Return: feature tested, issues found, fixes applied.""",
            trigger=None,
        ),
    ]
    return Agent(
        llm=llm,
        tools=[Tool(name=TerminalTool.name), Tool(name=FileEditorTool.name)],
        agent_context=AgentContext(skills=skills),
    )


def create_program_porter_agent(llm: LLM) -> Agent:
    """Agent specialized in porting real programs."""
    skills = [
        COMPILER_DEBUG_SKILL,
        Skill(
            name="program_porter",
            content="""You port real-world C programs to compile with ralph-cc.

WORKFLOW:
1. Download the target program (see HARDEN_PLAN.md for URLs)
2. Preprocess with `gcc -E -w -P`
3. Attempt compilation with ralph-cc
4. Fix any compiler bugs encountered
5. Compare runtime output with gcc-compiled version
6. Document any limitations

Return: program name, compilation status, bugs fixed, remaining issues.""",
            trigger=None,
        ),
    ]
    return Agent(
        llm=llm,
        tools=[Tool(name=TerminalTool.name), Tool(name=FileEditorTool.name)],
        agent_context=AgentContext(skills=skills),
    )


def create_diagnostician_agent(llm: LLM) -> Agent:
    """Agent specialized in deep diagnosis of tricky bugs."""
    skills = [
        COMPILER_DEBUG_SKILL,
        Skill(
            name="diagnostician",
            content="""You diagnose complex compiler bugs that others got stuck on.

WORKFLOW:
1. Read the progress file for prior attempts
2. Reproduce the issue
3. Use ALL IR dump stages to trace the bug
4. Check `plan/05-fix-research-ralph/COMMON_CAUSES.md` for patterns
5. If it's a known pattern, apply the fix
6. If novel, document your findings thoroughly

You're the expert called in when others are stuck.
Return: root cause analysis, fix applied or detailed findings for human review.""",
            trigger=None,
        ),
    ]
    return Agent(
        llm=llm,
        tools=[Tool(name=TerminalTool.name), Tool(name=FileEditorTool.name)],
        agent_context=AgentContext(skills=skills),
    )


# =============================================================================
# Task Discovery
# =============================================================================


def discover_failing_seeds(limit: int = 4) -> list[str]:
    """Run regression.sh and find failing seeds."""
    regression_script = Path("plan/06-regression-ralph/scripts/regression.sh")
    if not regression_script.exists():
        logger.warning("regression.sh not found")
        return []

    try:
        result = subprocess.run(
            ["bash", str(regression_script)],
            capture_output=True,
            text=True,
            timeout=300,
        )
        seeds = []
        for line in result.stdout.splitlines() + result.stderr.splitlines():
            if "FAIL" in line or "fail" in line:
                parts = line.split()
                for part in parts:
                    if part.isdigit() or part.startswith("seed_"):
                        seeds.append(part.replace("seed_", ""))
                        if len(seeds) >= limit:
                            return seeds
        return seeds
    except Exception as e:
        logger.error(f"Failed to run regression: {e}")
        return []


def read_phase() -> str:
    """Read current phase from phase.txt."""
    phase_file = Path("plan/08-parallel-sdk-triage/phase.txt")
    if phase_file.exists():
        return phase_file.read_text().strip()
    return "STABILIZE"


def discover_tasks_for_phase(phase: str) -> dict[str, str]:
    """Discover available tasks based on current phase."""
    tasks = {}

    if phase in ("STABILIZE", "EXPAND"):
        seeds = discover_failing_seeds(limit=4)
        for i, seed in enumerate(seeds):
            agent_id = f"fixer{i + 1}"
            tasks[agent_id] = (
                f"Fix the compiler bug causing seed {seed} to fail. Check csmith-reports/ for the test file."
            )

    elif phase == "HARDEN":
        plan_file = Path("plan/08-parallel-sdk-triage/HARDEN_PLAN.md")
        if plan_file.exists():
            content = plan_file.read_text()
            unchecked = []
            for line in content.splitlines():
                if "- [ ]" in line:
                    feature = line.replace("- [ ]", "").strip()
                    unchecked.append(feature)
            for i, feature in enumerate(unchecked[:4]):
                agent_id = f"hardener{i + 1}"
                tasks[agent_id] = f"Harden the compiler feature: {feature}"

    elif phase == "REAL_PROGRAMS":
        programs = ["jsmn", "miniz", "sqlite", "lua"]
        for i, prog in enumerate(programs[:4]):
            agent_id = f"porter{i + 1}"
            tasks[agent_id] = f"Port {prog} to compile with ralph-cc"

    return tasks


# =============================================================================
# Main Orchestrator
# =============================================================================


def create_orchestrator_prompt(phase: str, tasks: dict[str, str]) -> str:
    """Create the orchestrator's task message."""
    task_list = "\n".join(f"  - {aid}: {task}" for aid, task in tasks.items())
    agent_types = []
    for aid in tasks:
        if "fixer" in aid:
            agent_types.append("seed_fixer")
        elif "hardener" in aid:
            agent_types.append("feature_hardener")
        elif "porter" in aid:
            agent_types.append("program_porter")
        else:
            agent_types.append("diagnostician")

    agent_ids = list(tasks.keys())
    types_str = ", ".join(agent_types)
    ids_str = ", ".join(agent_ids)

    return f"""You are coordinating compiler debugging for ralph-cc.
Current phase: {phase}

STEP 1: Spawn {len(tasks)} sub-agents with these IDs and types:
  IDs: {ids_str}
  Types: {types_str}

STEP 2: Delegate these tasks in parallel:
{task_list}

STEP 3: Wait for all results, then:
  - Summarize what each agent fixed
  - List any agents that got stuck
  - Run `make test` to verify all fixes work together
  - If tests fail, identify which fix broke things

STEP 4: Report final status - how many bugs fixed, any remaining issues.
"""


def main():
    parser = argparse.ArgumentParser(
        description="Hans - Compiler hardening with OpenHands SDK"
    )
    parser.add_argument(
        "--task",
        type=str,
        help="Custom task description (overrides auto-discovery)",
    )
    parser.add_argument(
        "--phase",
        type=str,
        choices=["STABILIZE", "EXPAND", "HARDEN", "REAL_PROGRAMS", "SQLITE"],
        help="Force a specific phase (default: read from phase.txt)",
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Show what would be done without running agents",
    )
    parser.add_argument(
        "--agents",
        type=int,
        default=4,
        choices=[1, 2, 3, 4, 5],
        help="Number of parallel agents (default: 4)",
    )
    parser.add_argument(
        "--no-commit",
        action="store_true",
        help="Skip git commit/push at the end",
    )
    parser.add_argument(
        "--no-pull",
        action="store_true",
        help="Skip git pull at the start",
    )
    dirty_group = parser.add_mutually_exclusive_group()
    dirty_group.add_argument(
        "--reset",
        action="store_true",
        help="Discard any uncommitted changes before starting",
    )
    dirty_group.add_argument(
        "--keep",
        action="store_true",
        help="Keep uncommitted changes and continue (risky)",
    )
    args = parser.parse_args()

    # Discover phase and tasks early (for dry-run)
    phase = args.phase or read_phase()

    if args.task:
        # Custom task - create agents based on --agents count
        tasks = {}
        for i in range(args.agents):
            tasks[f"agent{i + 1}"] = args.task
        prompt = f"""You are coordinating compiler debugging.

Spawn {args.agents} sub-agents ({", ".join(tasks.keys())}) of type 'seed_fixer'.
Delegate this task to all of them (they should each work on different aspects):
{args.task}

Collect results and report what was fixed."""
    else:
        tasks = discover_tasks_for_phase(phase)
        if not tasks:
            console.print(
                "[yellow]⚠️  No tasks discovered. Creating sample tasks.[/yellow]"
            )
            tasks = {
                "fixer1": "Run regression.sh and fix the first failing test",
                "fixer2": "Run regression.sh and fix the second failing test",
                "fixer3": "Check csmith-reports/ for any crash files and fix one",
                "fixer4": "Check csmith-reports/ for any mismatch files and fix one",
            }
        # Limit to requested number of agents
        tasks = dict(list(tasks.items())[: args.agents])
        prompt = create_orchestrator_prompt(phase, tasks)

    # Handle dry-run before requiring API key
    if args.dry_run:
        console.print(Panel(f"[bold cyan]Phase: {phase}[/bold cyan]", title="HANS"))
        console.print(f"[green]📋 Tasks: {len(tasks)}[/green]")
        for aid, task in tasks.items():
            console.print(f"   [dim]{aid}:[/dim] {task[:60]}...")
        console.print("\n[yellow]🔍 DRY RUN - Orchestrator prompt:[/yellow]")
        console.print(Panel(prompt, title="Prompt"))
        return

    # =========================================================================
    # PRE-FLIGHT: Check for uncommitted changes
    # =========================================================================
    if is_dirty():
        console.print("\n[bold yellow]⚠️  Uncommitted changes detected![/bold yellow]")
        _, diff = sh("git status --short")
        console.print(f"[dim]{diff[:500]}[/dim]")

        if args.reset:
            git_reset()
        elif args.keep:
            console.print("[yellow]--keep: continuing with dirty state[/yellow]")
        else:
            console.print("[red]Cannot start dirty. Use --reset or --keep[/red]")
            sys.exit(1)

    # Check for API key (only needed for actual run)
    api_key = os.getenv("LLM_API_KEY")
    if not api_key:
        console.print("[red]ERROR: LLM_API_KEY environment variable is not set.[/red]")
        console.print("Get one from: https://app.all-hands.dev/settings/api-keys")
        sys.exit(1)
    assert api_key is not None  # for type checker

    # Configure LLM
    model = os.getenv("LLM_MODEL", "anthropic/claude-sonnet-4-5-20250929")
    base_url = os.getenv("LLM_BASE_URL")
    llm = LLM(
        model=model,
        api_key=SecretStr(api_key),
        base_url=base_url,
        usage_id="hans-orchestrator",
    )

    # Register agent types
    register_agent(
        name="seed_fixer",
        factory_func=create_seed_fixer_agent,
        description="Fixes csmith seed failures by diagnosing and patching compiler bugs",
    )
    register_agent(
        name="feature_hardener",
        factory_func=create_feature_hardener_agent,
        description="Hardens specific compiler features with edge case testing",
    )
    register_agent(
        name="program_porter",
        factory_func=create_program_porter_agent,
        description="Ports real-world C programs to compile with ralph-cc",
    )
    register_agent(
        name="diagnostician",
        factory_func=create_diagnostician_agent,
        description="Deep diagnosis of complex bugs that others got stuck on",
    )

    # Register delegation tool
    register_tool("DelegateTool", DelegateTool)

    # Display status
    console.print(Panel(f"[bold cyan]Phase: {phase}[/bold cyan]", title="HANS"))
    console.print(f"[green]📋 Tasks: {len(tasks)}[/green]")
    for aid, task in tasks.items():
        console.print(f"   [dim]{aid}:[/dim] {task[:60]}...")

    # =========================================================================
    # PRE-FLIGHT: Sync with remote
    # =========================================================================
    if not args.no_pull:
        console.print("\n[bold]📥 Syncing...[/bold]")
        git_pull()

    # Create orchestrator agent
    orchestrator = Agent(
        llm=llm,
        tools=[
            Tool(name="DelegateTool"),
            Tool(name=TerminalTool.name),
            Tool(name=FileEditorTool.name),
        ],
        agent_context=AgentContext(
            skills=[COMPILER_DEBUG_SKILL],
            system_message_suffix="You coordinate multiple sub-agents working in parallel. Use the delegate tool to spawn and assign tasks. Do NOT commit changes - the gather phase will handle that.",
        ),
    )

    # Run conversation
    cwd = os.getcwd()
    conversation = Conversation(
        agent=orchestrator,
        workspace=cwd,
        visualizer=DelegationVisualizer(name="HANS"),
    )

    console.print(
        f"\n[bold green]🚀 Starting HANS with {len(tasks)} parallel agents...[/bold green]"
    )
    console.print("=" * 60)

    conversation.send_message(prompt)  # type: ignore[attr-defined]
    conversation.run()  # type: ignore[attr-defined]

    # =========================================================================
    # GATHER: Verify and commit
    # =========================================================================
    console.print("\n" + "=" * 60)
    console.print("[bold]📦 Gather phase[/bold]")

    if is_dirty():
        _, diff = sh("git status --short")
        console.print(f"[dim]{diff[:300]}[/dim]")

        if run_tests() and not args.no_commit:
            # Stage changes first so generate_commit_message can see them
            sh("git add -A")
            console.print("[dim]Generating commit message...[/dim]")
            commit_msg = generate_commit_message(llm)
            console.print(f"[dim]{commit_msg[:200]}...[/dim]")
            # Commit (already staged) and push
            code, _ = sh(f'git commit -m "{commit_msg}"')
            if code == 0:
                console.print("[green]✓ Committed[/green]")
                code, _ = sh("git push origin main")
                console.print(
                    "[green]✓ Pushed[/green]"
                    if code == 0
                    else "[yellow]⚠ Push failed[/yellow]"
                )
        elif args.no_commit:
            console.print("[yellow]--no-commit: skipping[/yellow]")
        else:
            console.print("[red]Tests failed - not committing[/red]")
    else:
        console.print("[yellow]No changes[/yellow]")

    # Report costs
    cost = conversation.conversation_stats.get_combined_metrics().accumulated_cost  # type: ignore[attr-defined]
    console.print("=" * 60)
    console.print(f"[cyan]💰 Total cost: ${cost:.4f}[/cyan]")
    console.print("[bold green]✅ HANS complete[/bold green]")


if __name__ == "__main__":
    main()
