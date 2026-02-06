#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.12"
# dependencies = [
# ]
# ///
"""
Analyze an OpenHands session to identify efficiency patterns.
"""
import json
import re
import sys
from pathlib import Path
from dataclasses import dataclass, field
from datetime import datetime
from typing import Iterator
from collections import defaultdict


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
        except json.JSONDecodeError as e:
            kind_match = re.search(r'"kind":\s*"([^"]+)"', json_str)
            ts_match = re.search(r'"timestamp":\s*"([^"]+)"', json_str)
            if kind_match:
                yield LogEvent(
                    kind=kind_match.group(1),
                    timestamp=ts_match.group(1) if ts_match else None,
                    data={"_parse_error": str(e)}
                )


@dataclass
class SessionAnalysis:
    log_file: str
    total_events: int = 0
    action_count: int = 0
    message_count: int = 0
    observation_count: int = 0
    
    # Action types
    terminal_actions: int = 0
    file_editor_actions: int = 0
    file_read_actions: int = 0
    container_actions: int = 0
    other_actions: int = 0
    
    # Pattern detection
    repeated_commands: list = field(default_factory=list)
    failed_commands: list = field(default_factory=list)
    long_outputs: list = field(default_factory=list)
    exploration_actions: list = field(default_factory=list)
    
    # Timeline
    start_time: str | None = None
    end_time: str | None = None
    
    # Tasks attempted
    task_context: str = ""


def analyze_session(log_path: Path) -> SessionAnalysis:
    """Analyze a session log file."""
    analysis = SessionAnalysis(log_file=log_path.name)
    events = list(parse_log_file(log_path))
    analysis.total_events = len(events)
    
    command_history = []
    file_views = []
    
    for event in events:
        if event.timestamp:
            if not analysis.start_time:
                analysis.start_time = event.timestamp
            analysis.end_time = event.timestamp
        
        if event.kind == "MessageEvent":
            analysis.message_count += 1
            # Extract task context from first message
            if not analysis.task_context and "llm_message" in event.data:
                msg = event.data.get("llm_message", {})
                content = msg.get("content", [])
                if content and isinstance(content, list):
                    for c in content:
                        if isinstance(c, dict) and "text" in c:
                            text = c["text"][:500]
                            if "RALPH.md" in text or "PLAN" in text:
                                analysis.task_context = text[:200]
                            break
        
        elif event.kind == "ActionEvent":
            analysis.action_count += 1
            action = event.data.get("action", {})
            action_kind = action.get("kind", "")
            
            if action_kind == "TerminalAction":
                analysis.terminal_actions += 1
                cmd = action.get("command", "")
                command_history.append(cmd)
                
                # Check for exploration patterns
                if any(x in cmd for x in ["ls", "find", "cat", "head", "tail", "grep"]):
                    analysis.exploration_actions.append(cmd[:100])
            
            elif action_kind == "FileEditorAction":
                analysis.file_editor_actions += 1
                cmd = action.get("command", "")
                path = action.get("path", "")
                if cmd == "view":
                    file_views.append(path)
            
            elif "container" in action_kind.lower():
                analysis.container_actions += 1
            
            else:
                analysis.other_actions += 1
        
        elif event.kind == "ObservationEvent":
            analysis.observation_count += 1
            obs = event.data.get("observation", {})
            
            # Check for failures
            if isinstance(obs, dict):
                exit_code = obs.get("exit_code")
                if exit_code and exit_code != 0:
                    output = obs.get("output", "")[:200]
                    analysis.failed_commands.append({
                        "exit_code": exit_code,
                        "output_preview": output
                    })
                
                # Check for long outputs
                output = obs.get("output", "")
                if len(output) > 5000:
                    analysis.long_outputs.append({
                        "length": len(output),
                        "preview": output[:100]
                    })
    
    # Detect repeated commands
    cmd_counts = defaultdict(int)
    for cmd in command_history:
        # Normalize command for comparison
        normalized = cmd.strip()[:80]
        cmd_counts[normalized] += 1
    
    for cmd, count in cmd_counts.items():
        if count >= 2:
            analysis.repeated_commands.append({"command": cmd, "count": count})
    
    return analysis


def main():
    if len(sys.argv) < 2:
        print("Usage: analyze_session.py <logfile>")
        sys.exit(1)
    
    log_path = Path(sys.argv[1])
    analysis = analyze_session(log_path)
    
    print(f"=== Session Analysis: {analysis.log_file} ===\n")
    print(f"Time: {analysis.start_time} -> {analysis.end_time}")
    print(f"\nEvent counts:")
    print(f"  Total: {analysis.total_events}")
    print(f"  Messages: {analysis.message_count}")
    print(f"  Actions: {analysis.action_count}")
    print(f"  Observations: {analysis.observation_count}")
    
    print(f"\nAction breakdown:")
    print(f"  Terminal: {analysis.terminal_actions}")
    print(f"  FileEditor: {analysis.file_editor_actions}")
    print(f"  Container: {analysis.container_actions}")
    print(f"  Other: {analysis.other_actions}")
    
    if analysis.task_context:
        print(f"\nTask context:\n  {analysis.task_context[:150]}...")
    
    if analysis.failed_commands:
        print(f"\nFailed commands ({len(analysis.failed_commands)}):")
        for f in analysis.failed_commands[:5]:
            print(f"  exit={f['exit_code']}: {f['output_preview'][:60]}...")
    
    if analysis.repeated_commands:
        print(f"\nRepeated commands:")
        for r in sorted(analysis.repeated_commands, key=lambda x: -x['count'])[:5]:
            print(f"  {r['count']}x: {r['command'][:60]}")
    
    if analysis.long_outputs:
        print(f"\nLong outputs ({len(analysis.long_outputs)}):")
        for l in analysis.long_outputs[:3]:
            print(f"  {l['length']} chars: {l['preview'][:50]}...")
    
    print(f"\nExploration actions: {len(analysis.exploration_actions)}")


if __name__ == "__main__":
    main()
