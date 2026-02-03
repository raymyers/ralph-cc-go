# Plan of Plans

```sh
function ralph3() {
    local logfile="plan/03-pop-ralph/logs/$(date +%Y%m%d-%H-%M-%S).log"
    echo "ralph3: writing to $logfile"
    env TERM=dumb openhands --headless -f plan/03-pop-ralph/RALPH.md --json > "$logfile"
}

function save_plan() {
    git add plan && git diff --cached --quiet plan || git commit plan -m "Update plan"
}

```

## Watching

```sh
 watch -n 10 -c "(grep -m 1 -C2 '\[ \]' plan/02-e2e-ralph/PLAN.md || true) && echo '\n--- untracked ---' && git ls-files --others --exclude-standard && echo '--- git diff ---' && git diff --stat && echo '--- git log ---' && git log --oneline -n 10"
 ```
