#!/bin/sh
# render.sh FILE.html [COLS]
#
# Draws FILE.html as a hotty host would, with hotty-blitz's `hotty render`,
# at scale 1.6 (a high-DPI display's) and at 2, COLS columns wide (80 by
# default) and as many rows as it takes, into .shots/<name>-1.6.png and
# .shots/<name>-2.png, <name> being FILE's name without .html.
#
# No display is needed. `hotty render` has no network policy, so images
# stay blank. HOTTY overrides the binary; by default it is the release
# build of a hotty-blitz checkout beside this one (../hotty-blitz or
# ../../hotty-blitz/main).
set -eu
if [ -z "${HOTTY:-}" ]; then
  here=$(cd "$(dirname "$0")/.." && pwd)
  for d in "$here/../hotty-blitz" "$here/../../hotty-blitz/main"; do
    [ -f "$d/Cargo.toml" ] && { HOTTY=$(cd "$d" && pwd)/target/release/hotty; break; }
  done
fi
HOTTY=${HOTTY:?no hotty-blitz checkout beside this one: set HOTTY}
[ $# -ge 1 ] || { echo "usage: $0 FILE.html [COLS]" >&2; exit 2; }
file=$1
cols=${2:-80}
[ -f "$file" ] || { echo "$file: no such file" >&2; exit 2; }
[ -x "$HOTTY" ] || { echo "$HOTTY: not built (cargo build --release in hotty-blitz)" >&2; exit 2; }
root=$(cd "$(dirname "$0")/.." && pwd)
name=$(basename "$file" .html)
mkdir -p "$root/.shots"
for scale in 1.6 2; do
  out="$root/.shots/$name-$scale.png"
  "$HOTTY" render "$file" -o "$out" --cols "$cols" --rows auto --scale "$scale"
  echo "$out"
done
