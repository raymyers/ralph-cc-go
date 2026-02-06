#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.12"
# dependencies = [
# ]
# ///
"""
Find the slowest commands in session logs.
"""
import json
import re
import sys
from pathlib import Path
from dataclasses import dataclass
from datetime import datetime
from typing import Iterator


@dataclass
class LogEvent:
    kind: str
    timestamp: str | None
    data: dict


def parse_timestamp(ts: str) -> datetime | None:
    try:
        return datetime.fromisoformat(ts.replace('Z', '+00:00'))
    except:
        return None


def parse_log_file(path: Path) -> Iterator[LogEvent]:
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
            pass


def find_slow_commands(log_dir: Path, threshold: float = 5.0):
    """Find commands that took longer than threshold seconds."""
    slow_commands = []
    
    for log_file in sorted(log_dir.glob("*.log")):
        events = list(parse_log_file(log_file))
        
        last_action = None
        last_action_time = None
        last_command = None
        
        for event in events:
            ts = parse_timestamp(event.timestamp) if event.timestamp else None
            
            if event.kind == "ActionEvent":
                action = event.data.get("action", {})
                if action.get("kind") == "TerminalAction":
                    last_action = action
                    last_action_time = ts
                    last_command = action.get("command", "")
                else:
                    last_action = None
            
            elif event.kind == "ObservationEvent" and last_action and last_action_time and ts:
                duration = (ts - last_action_time).total_seconds()
                if duration > threshold:
                    obs = event.data.get("observation", {})
                    exit_code = obs.get("exit_code") if isinstance(obs, dict) else None
                    output = obs.get("output", "")[:500] if isinstance(obs, dict) else ""
                    
                    slow_commands.append({
                        "file": log_file.name,
                        "command": last_command,
                        "duration": duration,
                        "exit_code": exit_code,
                        "output_preview": output
                    })
                last_action = None
    
    return slow_commands


def main():
    log_dir = Path(sys.argv[1]) if len(sys.argv) > 1 else Path("plan/03-pop-ralph/logs")
    threshold = float(sys.argv[2]) if len(sys.argv) > 2 else 5.0
    
    slow = find_slow_commands(log_dir, threshold)
    slow.sort(key=lambda x: -x["duration"])
    
    print(f"=== SLOW COMMANDS (>{threshold}s) ===\n")
    print(f"Total: {len(slow)}\n")
    
    for cmd in slow[:20]:
        print(f"[{cmd['file']}] {cmd['duration']:.1f}s (exit={cmd['exit_code']})")
        print(f"  Command: {cmd['command'][:100]}")
        if cmd['output_preview']:
            # Show first few lines of output
            lines = cmd['output_preview'].split('\n')[:5]
            for line in lines:
                print(f"    {line[:80]}")
        print()


if __name__ == "__main__":
    main()
