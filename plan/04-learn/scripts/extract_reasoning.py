#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.12"
# dependencies = [
# ]
# ///
"""
Extract agent reasoning and thinking blocks from session logs.
Shows what the agent was thinking at key decision points.
"""
import json
import re
import sys
from pathlib import Path
from dataclasses import dataclass
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
            pass


def extract_reasoning(log_path: Path):
    """Extract reasoning blocks with context."""
    events = list(parse_log_file(log_path))
    
    reasoning_samples = []
    
    for i, event in enumerate(events):
        if event.kind == "ActionEvent":
            reasoning = event.data.get("reasoning_content")
            thinking_blocks = event.data.get("thinking_blocks", [])
            action = event.data.get("action", {})
            summary = event.data.get("summary", "")
            
            reasoning_text = ""
            if reasoning:
                reasoning_text = reasoning
            elif thinking_blocks:
                for block in thinking_blocks:
                    if isinstance(block, dict) and "thinking" in block:
                        reasoning_text += block["thinking"] + "\n"
            
            if reasoning_text:
                # Look ahead for observation
                outcome = None
                if i + 1 < len(events) and events[i + 1].kind == "ObservationEvent":
                    obs = events[i + 1].data.get("observation", {})
                    if isinstance(obs, dict):
                        exit_code = obs.get("exit_code")
                        output = obs.get("output", "")[:200]
                        outcome = {"exit_code": exit_code, "output_preview": output}
                
                reasoning_samples.append({
                    "timestamp": event.timestamp,
                    "action_kind": action.get("kind", ""),
                    "summary": summary,
                    "reasoning": reasoning_text[:500],
                    "outcome": outcome
                })
    
    return reasoning_samples


def main():
    if len(sys.argv) < 2:
        # Analyze all logs
        log_dir = Path("plan/03-pop-ralph/logs")
        log_files = sorted(log_dir.glob("*.log"), key=lambda p: p.stat().st_size, reverse=True)[:3]
    else:
        log_files = [Path(sys.argv[1])]
    
    for log_path in log_files:
        print(f"\n{'='*60}")
        print(f"=== {log_path.name} ===")
        print(f"{'='*60}\n")
        
        samples = extract_reasoning(log_path)
        
        # Filter to interesting cases - failures or retry patterns
        interesting = []
        for s in samples:
            outcome = s.get("outcome")
            if outcome and outcome.get("exit_code") and outcome["exit_code"] != 0:
                interesting.append(("FAILURE", s))
            elif "retry" in s["reasoning"].lower() or "again" in s["reasoning"].lower():
                interesting.append(("RETRY", s))
            elif "explore" in s["reasoning"].lower() or "look" in s["reasoning"].lower():
                interesting.append(("EXPLORE", s))
        
        print(f"Total reasoning blocks: {len(samples)}")
        print(f"Interesting cases: {len(interesting)}\n")
        
        for category, s in interesting[:10]:
            print(f"[{category}] {s['timestamp']}")
            print(f"  Action: {s['action_kind']}")
            print(f"  Summary: {s['summary']}")
            print(f"  Reasoning: {s['reasoning'][:200]}...")
            if s['outcome']:
                print(f"  Outcome: exit={s['outcome']['exit_code']}")
            print()


if __name__ == "__main__":
    main()
