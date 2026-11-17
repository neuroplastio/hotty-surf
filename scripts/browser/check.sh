#!/bin/sh
# hotty-surf's page in Chromium (make browser): hello built by `hotty-surf
# build -wasm` (TinyGo) and served statically, then `hotty-surf serve` on a
# copy of hello outside the repository, with a Go, a css and a failed save.
# Screenshots go to .shots/browser. Needs node and Playwright's Chromium
# (npx playwright-core install chromium-headless-shell, once).
set -eu
root=$(cd "$(dirname "$0")/../.." && pwd)
GO=${GO:-go}
NODE=${NODE:-node}
out="$root/.shots/browser"
rm -rf "$out"
mkdir -p "$out"
cd "$root"
$GO build -o bin/hotty-surf ./cmd/hotty-surf

PATH="$root/bin:$PATH" hotty-surf build -wasm -o "$out/dist" examples/hello
$NODE scripts/browser/release.mjs "$out/dist" "$out/release.png"

# A module of its own that requires this one, so the saves stay out of the
# repository.
mod=$(mktemp -d)
trap 'rm -rf "$mod"' EXIT
cp -r examples/hello "$mod/hello"
rm -rf "$mod/hello/dist"
printf 'module example.com/apps\n\ngo 1.26.0\n\nrequire neuroplast.io/hotty-surf v0.0.0\n\nreplace neuroplast.io/hotty-surf => %s\n' "$root" >"$mod/go.mod"
cp go.sum "$mod/"
(cd "$mod" && GOFLAGS=-mod=mod GOPROXY=off $GO mod tidy)
$NODE scripts/browser/serve.mjs "$root/bin/hotty-surf" "$mod/hello" "$out"
