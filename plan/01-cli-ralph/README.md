# 01 Cli Ralph

## Loop

Run simply with `openhands --headless -f plan/01-cli-ralph/RALPH.md`.

Log more data to file:

```sh
TERM=dumb openhands --headless -f plan/01-cli-ralph/RALPH.md --json > plan/01-cli-ralph/logs/`date +%Y%m%d-%H-%M-%S`.log
```

## Watching

```sh
container-user watch
```

## Setup

openhands mcp add --transport stdio container-use container-use -- stdio

Also should add tavily
