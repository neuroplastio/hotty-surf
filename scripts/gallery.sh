#!/bin/sh
# gallery.sh [--build] [--web [PORT]] [GALLERY_FLAGS…]: the gallery of
# stories (cmd/surfgallery) in a new hottyterm window on this display,
# in an environment of its own; or with --web, in a browser tab, through
# the xterm addon's serve.py on 127.0.0.1:PORT (8792), until ctrl+c:
#
#   sh scripts/gallery.sh                      the gallery
#   sh scripts/gallery.sh -story notice-row    the first story with notice-row in its name
#   sh scripts/gallery.sh -states -story field the field in each of its states
#   sh scripts/gallery.sh -reference -story button/states
#                                              under it, as make gate drew it in Blitz and Chromium
#   sh scripts/gallery.sh --web                then open the URL it says
#   sh scripts/gallery.sh --build              build only, and print the file built
#
# make gallery runs it (GALLERY_FLAGS for its flags).
#
# It reloads as this checkout changes (gallery -dev): an edit of a
# stories file (internal/stories/*.surf), a template (internal/screens/*.surf) or the
# stylesheet (surfui/surf.css) is in the window at once, with the story,
# the toolbar and the controls as they were. A Go change needs the
# script again.
#
# In the window: the arrows move in the stories, Tab moves on to the
# story and its controls, the wheel scrolls each. ctrl+c, or closing the
# window, ends it.
#
# It builds cmd/surfgallery from the working tree, as it is now, into
# bin/surfgallery, under the Makefile's CAP, each time it runs. The
# window is the installed hottyterm's (SURF_HOTTYTERM, /usr/lib/hottyterm),
# which draws with its own Blitz. SURF_BLITZ_LIB=FILE draws with that
# Blitz instead, for patches not yet released: the terminal is then a
# copy of the hottyterm with FILE over its own, in
# ${XDG_CACHE_HOME:-$HOME/.cache}/hotty-surf/gallery/blitz-<FILE's checksum>/.
#
# The window's environment is this one without PLX_*, SURF_*, GHOSTTY_*
# and TERM_PROGRAM*, so that nothing of the terminal it was started
# from reaches the gallery's stories. hottyterm runs
# with --gtk-single-instance=false, so it never joins a running
# hottyterm, and in a session of its own (setsid), so it outlives the
# shell it was started from.
set -eu
here=$(cd "$(dirname "$0")" && pwd)
repo=$(cd "$here/.." && pwd)
die() {
  echo "gallery.sh: $*" >&2
  exit 1
}

# gobin is the directory of a go to build with: on PATH, else mise's
# (asked in the checkout, then in HOME, for a tree mise does not trust),
# or nothing.
gobin() {
  if command -v go >/dev/null 2>&1; then
    dirname "$(command -v go)"
  elif command -v mise >/dev/null 2>&1; then
    for at in "$repo" "$HOME"; do
      g=$(cd "$at" 2>/dev/null && mise which go 2>/dev/null) && [ -n "$g" ] && dirname "$g" && break
    done
  fi
  return 0
}

build=""
if [ "${1:-}" = --build ]; then
  build=1
  shift
fi

cache=${XDG_CACHE_HOME:-$HOME/.cache}/hotty-surf/gallery
term=${SURF_HOTTYTERM:-/usr/lib/hottyterm}
[ -x "$term/bin/hottyterm" ] || die "no hottyterm in $term (set SURF_HOTTYTERM)"
blitz=${SURF_BLITZ_LIB:-}
if [ -n "$blitz" ]; then
  [ -f "$blitz" ] || die "no Blitz at $blitz (SURF_BLITZ_LIB)"
  dir=$cache/blitz-$(cksum <"$blitz" | cut -d' ' -f1)
  if [ ! -x "$dir/bin/hottyterm" ]; then
    tmp=$cache/.build-$$
    rm -rf "$tmp" && mkdir -p "$tmp"
    trap 'rm -rf "$tmp"' EXIT
    cp -RL --preserve=mode,timestamps "$term/." "$tmp/"
    mkdir -p "$tmp/lib"
    cp -L --preserve=mode,timestamps "$blitz" "$tmp/lib/libhotty_blitz.so"
    rm -rf "$dir"
    mv "$tmp" "$dir"
    trap - EXIT
  fi
  term=$dir
fi

# The gallery is built from the working tree, as it is now, each time:
# go's cache makes it quick. The build runs under the Makefile's CAP
# (this machine's, if it sets one), or $CAP.
gobin=$(gobin)
[ -n "$gobin" ] || die "no go on PATH and no mise to find one"
cap=${CAP-$(make -s --no-print-directory -C "$repo" print-cap 2>/dev/null)}
gallery=$repo/bin/surfgallery
# shellcheck disable=SC2086 # cap is a command's words
(cd "$repo" && $cap "$gobin/go" build -o "$gallery" ./cmd/surfgallery)

if [ -n "$build" ]; then
  echo "$gallery"
  exit 0
fi

web="" port=8792
if [ "${1:-}" = --web ]; then
  web=1
  shift
  case ${1:-} in [0-9]*) port=$1 && shift ;; esac
fi

scrub=$(env | sed -n 's/^\(\(PLX_\|SURF_\|GHOSTTY_\|TERM_PROGRAM\)[A-Za-z0-9_]*\)=.*/-u \1/p')
cd "$HOME"
if [ -n "$web" ]; then
  addon=${SURF_XTERM_ADDON:-$(cd "$repo/../.." && pwd)/xterm-addon-hotty/main}
  [ -f "$addon/dist/demo.js" ] || die "no $addon/dist/demo.js: npm run build there first"
  echo "gallery.sh: open http://127.0.0.1:$port/ (ctrl+c here stops it)"
  # shellcheck disable=SC2086 # scrub is env's -u words
  exec env $scrub python3 "$addon/serve.py" --port "$port" -- "$gallery" -dev "$repo" "$@"
fi
# shellcheck disable=SC2086 # scrub is env's -u words
exec env $scrub setsid -f "$term/bin/hottyterm" --gtk-single-instance=false --title="gallery $repo" \
  -e "$gallery" -dev "$repo" "$@" >/dev/null 2>&1
