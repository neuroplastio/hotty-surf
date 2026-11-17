#!/bin/sh
# The browser half of ui's differential tests: writes the pages of every
# spec in ui/testdata/differential (internal/differential/record), starts
# a headless Chromium on a profile of its own, drives each component's
# cases over the DevTools protocol (internal/differential/cdp.mjs), writes
# what it saw to ui/testdata/differential/<component>.browser.json, and
# closes it. `make differential` runs it under CAP; then `make test`
# compares. Needs chromium and node; run it again on a Basecoat bump or a
# new case.
set -e
cd "$(dirname "$0")/.."
R=$(pwd)
C=${XDG_CACHE_HOME:-$HOME/.cache}/hotty-surf/differential
CHROMIUM=${CHROMIUM:-chromium}
rm -rf "$C"
mkdir -p "$C/profile"
mise x -- go run ./internal/differential/record ui/testdata/differential "$C/pages"
# Port 0: Chromium picks a free one and writes it to DevToolsActivePort, so
# a run never meets the port of one before it still closing.
"$CHROMIUM" --headless=new --remote-debugging-port=0 \
  --user-data-dir="$C/profile" --no-first-run --no-default-browser-check \
  --disable-gpu --disable-extensions --window-size=1280,800 \
  --allow-file-access-from-files about:blank >"$C/chromium.log" 2>&1 &
P=$!
trap 'kill $P 2>/dev/null; wait $P 2>/dev/null; rm -rf "$C/profile"' EXIT INT TERM
i=0
until [ -s "$C/profile/DevToolsActivePort" ]; do
  i=$((i + 1))
  if [ $i -gt 100 ]; then echo "differential: chromium didn't start ($C/chromium.log)" >&2; exit 1; fi
  sleep 0.1
done
PORT=$(head -n1 "$C/profile/DevToolsActivePort")
for m in "$C"/pages/*.manifest.json; do
  comp=$(basename "$m" .manifest.json)
  node "$R/internal/differential/cdp.mjs" "$PORT" "$m" "$C/pages/$comp" \
    "$R/ui/testdata/differential/$comp.browser.json"
done
