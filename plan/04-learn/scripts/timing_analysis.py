#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.12"
# dependencies = [
# ]
# ///
"""
Analyze timing and action sequences in OpenHands session logs.
"""
import json
import re
import sys
from pathlib import Path
from dataclasses import dataclass
from datetime import datetime
from typing import Iterator
from collections import defaultdict


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
            pass


def analyze_timing(log_path: Path):
    """Analyze action timing patterns."""
    events = list(parse_log_file(log_path))
    
    # Track action-observation pairs and their timing
    action_times = []
    last_action_time = None
    last_action_type = None
    
    for event in events:
        ts = parse_timestamp(event.timestamp) if event.timestamp else None
        
        if event.kind == "ActionEvent":
            action = event.data.get("action", {})
            action_kind = action.get("kind", "")
            last_action_time = ts
            last_action_type = action_kind
        
        elif event.kind == "ObservationEvent" and last_action_time and ts:
            duration = (ts - last_action_time).total_seconds()
            action_times.append({
                "action": last_action_type,
                "duration": duration,
                "timestamp": last_action_time.isoformat()
            })
            last_action_time = None
    
    return action_times


def main():
    log_dir = Path(sys.argv[1]) if len(sys.argv) > 1 else Path("plan/03-pop-ralph/logs")
    
    all_times = defaultdict(list)
    total_by_file = {}
    
    for log_file in sorted(log_dir.glob("*.log")):
        times = analyze_timing(log_file)
        if not times:
            continue
            
        file_total = sum(t["duration"] for t in times)
        total_by_file[log_file.name] = {
            "total_seconds": file_total,
            "action_count": len(times)
        }
        
        for t in times:
            all_times[t["action"]].append(t["duration"])
    
    print("=== TIMING ANALYSIS ===\n")
    
    print("## Session totals:")
    for fname, info in sorted(total_by_file.items()):
        mins = info["total_seconds"] / 60
        print(f"  {fname}: {mins:.1f}min ({info['action_count']} actions)")
    
    print("\n## Action type statistics:")
    for action_type, durations in sorted(all_times.items()):
        avg = sum(durations) / len(durations) if durations else 0
        total = sum(durations)
        print(f"\n  {action_type}:")
        print(f"    Count: {len(durations)}")
        print(f"    Total: {total:.1f}s")
        print(f"    Avg: {avg:.2f}s")
        if durations:
            print(f"    Max: {max(durations):.2f}s")
        
        # Find slow actions
        slow = [d for d in durations if d > 10]
        if slow:
            print(f"    Slow (>10s): {len(slow)}")


if __name__ == "__main__":
    main()
