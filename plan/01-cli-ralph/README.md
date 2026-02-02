# 01 Cli Ralph

## Loop

Run simply with `openhands --headless -f plan/01-cli-ralph/RALPH.md`.

Log more data to file:

```sh
TERM=dumb openhands --headless -f plan/01-cli-ralph/RALPH.md --json > plan/01-cli-ralph/logs/`date +%Y%m%d-%H-%M-%S`.log
```

With other conveniences:

```sh
function ralph1() {
    local logfile="plan/01-cli-ralph/logs/$(date +%Y%m%d-%H-%M-%S).log"
    echo "ralph1: writing to $logfile" >&2
    env TERM=dumb openhands --headless -f plan/01-cli-ralph/RALPH.md --json > "$logfile"
}

function save_plan() {
    git add plan && git diff --cached --quiet plan || git commit plan -m "Update plan"
}

function ralph4() {
    save_plan
    ralph1
    ralph1
    ralph1
    ralph1
}


```

## Watching

```sh
watch -n 10 -c "git diff --stat && git log --oneline -n 10"

watch -n 10 -c "(grep -m 1 -A1 '\[ \]' plan/01-cli-ralph/PLAN.md || true) && echo '--- untracked ---' && git ls-files --others --exclude-standard && echo '--- git diff ---' && git diff --stat && echo '--- git log ---' && git log --oneline -n 10"
```


```sh
container-user watch
```

## Setup

openhands mcp add --transport stdio container-use container-use -- stdio

Also should add tavily
