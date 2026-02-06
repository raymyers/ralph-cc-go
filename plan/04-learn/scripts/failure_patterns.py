#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.12"
# dependencies = [
# ]
# ///
"""
Extract and categorize failure patterns from OpenHands session logs.
"""
import json
import re
import sys
from pathlib import Path
from dataclasses import dataclass, field
from collections import defaultdict
from typing import Iterator


@dataclass
class LogEvent:
    kind: str
    timestamp: str | None
    data: dict


def parse_log_file(path: Path) -> Iterator[LogEvent]:
    """Parse a log file and yield JSON events."""
    content = path.read_text()
    pattern = r'--JSON Event--\s*\n(\{.*?\})\s*(?=--JSON Event--|$)'
    
    for match in re.finditer(pattern, content, re.DOTALL):
        json_str = match.group(1)
        json_str = json_str.encode('utf-8', errors='replace').decode('utf-8')
        try:
            data = json.loads(json_str, strict=False)
            yield LogEvent(
                kind=data.get("kind", "unknown"),
                timestamp=data.get("timestamp"),
                data=data
            )
        except json.JSONDecodeError:
            kind_match = re.search(r'"kind":\s*"([^"]+)"', json_str)
            ts_match = re.search(r'"timestamp":\s*"([^"]+)"', json_str)
            if kind_match:
                yield LogEvent(
                    kind=kind_match.group(1),
                    timestamp=ts_match.group(1) if ts_match else None,
                    data={"_parse_error": True}
                )


def extract_failures(log_dir: Path) -> dict:
    """Extract all failure patterns across logs."""
    failures = {
        "build_failures": [],
        "test_failures": [],
        "exploration_waste": [],
        "retry_loops": [],
        "missing_knowledge": []
    }
    
    for log_file in sorted(log_dir.glob("*.log")):
        events = list(parse_log_file(log_file))
        
        last_command = None
        last_action_idx = -1
        command_attempts = defaultdict(int)
        
        for i, event in enumerate(events):
            if event.kind == "ActionEvent":
                action = event.data.get("action", {})
                action_kind = action.get("kind", "")
                
                if action_kind == "TerminalAction":
                    cmd = action.get("command", "")
                    last_command = cmd
                    last_action_idx = i
                    
                    # Track command patterns
                    cmd_key = cmd.strip()[:60]
                    command_attempts[cmd_key] += 1
                    
                    # Detect exploration patterns
                    if any(x in cmd for x in ["find ", "grep -r", "ls -la", "cat "]):
                        if "compcert" in cmd or ".v" in cmd or "/coq/" in cmd:
                            failures["exploration_waste"].append({
                                "file": log_file.name,
                                "command": cmd[:100],
                                "category": "exploring_compcert_coq"
                            })
            
            elif event.kind == "ObservationEvent":
                obs = event.data.get("observation", {})
                if isinstance(obs, dict):
                    exit_code = obs.get("exit_code")
                    output = obs.get("output", "")
                    
                    if exit_code and exit_code != 0:
                        # Categorize failure
                        failure_info = {
                            "file": log_file.name,
                            "exit_code": exit_code,
                            "command": last_command[:100] if last_command else "unknown",
                            "output_preview": output[:300] if output else ""
                        }
                        
                        if last_command:
                            if "go build" in last_command or "make build" in last_command:
                                failures["build_failures"].append(failure_info)
                            elif "go test" in last_command or "make test" in last_command or "make check" in last_command:
                                failures["test_failures"].append(failure_info)
        
        # Detect retry loops
        for cmd, count in command_attempts.items():
            if count >= 3:
                failures["retry_loops"].append({
                    "file": log_file.name,
                    "command": cmd,
                    "count": count
                })
    
    return failures


def main():
    log_dir = Path(sys.argv[1]) if len(sys.argv) > 1 else Path("plan/03-pop-ralph/logs")
    
    failures = extract_failures(log_dir)
    
    print("=== FAILURE PATTERN ANALYSIS ===\n")
    
    print(f"## Build Failures ({len(failures['build_failures'])})")
    for f in failures["build_failures"][:10]:
        print(f"  [{f['file']}] exit={f['exit_code']}")
        print(f"    cmd: {f['command'][:70]}")
        if "error" in f['output_preview'].lower():
            # Extract error message
            error_lines = [l for l in f['output_preview'].split('\n') if 'error' in l.lower()][:2]
            for err in error_lines:
                print(f"    err: {err[:80]}")
        print()
    
    print(f"\n## Test Failures ({len(failures['test_failures'])})")
    for f in failures["test_failures"][:10]:
        print(f"  [{f['file']}] exit={f['exit_code']}")
        print(f"    cmd: {f['command'][:70]}")
        print()
    
    print(f"\n## Retry Loops ({len(failures['retry_loops'])})")
    for r in sorted(failures["retry_loops"], key=lambda x: -x["count"])[:15]:
        print(f"  {r['count']}x [{r['file']}]: {r['command'][:60]}")
    
    print(f"\n## Exploration Waste ({len(failures['exploration_waste'])})")
    categories = defaultdict(list)
    for e in failures["exploration_waste"]:
        categories[e["category"]].append(e)
    for cat, items in categories.items():
        print(f"  {cat}: {len(items)} occurrences")
        for item in items[:3]:
            print(f"    - {item['command'][:60]}")


if __name__ == "__main__":
    main()
