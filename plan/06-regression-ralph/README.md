```sh
function ralph6() {
    local logfile="plan/06-regression-ralph/logs/$(date +%Y%m%d-%H-%M-%S).log"
    echo "ralph6: writing to $logfile"
    env TERM=dumb openhands --headless -f plan/06-regression-ralph/RALPH.md --json > "$logfile"
}
```
