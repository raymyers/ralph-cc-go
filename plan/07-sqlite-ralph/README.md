```sh
function ralph7() {
    local logfile="plan/07-sqlite-ralph/logs/$(date +%Y%m%d-%H-%M-%S).log"
    echo "ralph7: writing to $logfile"
    env TERM=dumb openhands --headless -f plan/07-sqlite-ralph/RALPH.md --json > "$logfile"
}

function save_plan() {
    git add plan && git diff --cached --quiet plan || git commit plan -m "Update plan"
}

```
