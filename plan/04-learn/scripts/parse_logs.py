#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.12"
# dependencies = [
# ]
# ///
"""
Parse OpenHands log files and extract JSON events.
Handles the mixed format with text headers and JSON blocks.
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
    
    # Find all JSON Event blocks
    # Pattern: --JSON Event--\n{...json...}\n--JSON Event-- or end
    pattern = r'--JSON Event--\s*\n(\{.*?\})\s*(?=--JSON Event--|$)'
    
    # Use DOTALL to match across newlines
    for match in re.finditer(pattern, content, re.DOTALL):
        json_str = match.group(1)
        # Handle control characters that may be in the JSON
        json_str = json_str.encode('utf-8', errors='replace').decode('utf-8')
        try:
            data = json.loads(json_str, strict=False)
            yield LogEvent(
                kind=data.get("kind", "unknown"),
                timestamp=data.get("timestamp"),
                data=data
            )
        except json.JSONDecodeError as e:
            # Try to extract minimal info
            kind_match = re.search(r'"kind":\s*"([^"]+)"', json_str)
            ts_match = re.search(r'"timestamp":\s*"([^"]+)"', json_str)
            if kind_match:
                yield LogEvent(
                    kind=kind_match.group(1),
                    timestamp=ts_match.group(1) if ts_match else None,
                    data={"_parse_error": str(e), "_raw_length": len(json_str)}
                )
            else:
                print(f"Warning: Failed to parse JSON block: {e}", file=sys.stderr)


def main():
    if len(sys.argv) < 2:
        print("Usage: parse_logs.py <logfile> [--summary]")
        sys.exit(1)
    
    log_path = Path(sys.argv[1])
    summary_mode = "--summary" in sys.argv
    
    events = list(parse_log_file(log_path))
    
    if summary_mode:
        print(f"Total events: {len(events)}")
        kinds = {}
        for e in events:
            kinds[e.kind] = kinds.get(e.kind, 0) + 1
        print("\nEvent types:")
        for k, v in sorted(kinds.items(), key=lambda x: -x[1]):
            print(f"  {k}: {v}")
    else:
        for event in events:
            print(json.dumps(event.data, indent=2))


if __name__ == "__main__":
    main()
