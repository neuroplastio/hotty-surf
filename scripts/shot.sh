#!/bin/sh
# shot.sh [-w SECS] [-k KEYS | -c X,Y | -m X,Y]… [-d SECS] [-r WxH] NAME SCALES -- PROGRAM [ARGS…]
#
# Runs PROGRAM in a fresh hottyterm window on the private headless sway
# display, once for each scale in SCALES (one, or a comma list: 1.6,2), and
# screenshots the display into .shots/:
#
#   .shots/NAME-SCALE.png     SECS (-w) after the window opens
#   .shots/NAME-SCALE-N.png   SECS (-d) after the Nth -k's keys, -c's click or
#                             -m's move
#
#   -w SECS  wait before the first shot (default 3)
#   -k KEYS  one wtype run after the shot before it, its arguments as one
#            shell-quoted string: -k 'hello -k Return', -k '-M ctrl b -m ctrl'.
#            Repeatable; each makes a shot.
#   -c X,Y   a left click at X,Y in logical pixels of the output (the -r size
#            over the scale), with a keyboard held around it, as drive.sh's
#            click: Ghostty takes a click on an unfocused window as a focus
#            click and doesn't report it. Repeatable, in turn with -k; each
#            makes a shot.
#   -m X,Y   the pointer moved to X,Y (logical pixels, as -c), to hover
#            there. Repeatable, in turn with -k and -c; each makes a shot.
#   -d SECS  wait after each -k before its shot (default 1)
#   -r WxH   the output's size in device pixels (default 2400x1500): 1500x937
#            logical at scale 1.6, 1200x750 at 2
#
# Keep PROGRAM running past the last shot: hottyterm closes its window when
# the program exits. Its output goes to .shots/NAME-SCALE.log.
#
# Safety: every PLX_* variable is unset first, and the script refuses to run
# if any remains, so nothing reaches a running plx. DISPLAY and
# HYPRLAND_INSTANCE_SIGNATURE are unset and WAYLAND_DISPLAY is the headless
# display, so nothing reaches the real one. hottyterm runs with
# --gtk-single-instance=false, so it never joins a running hottyterm,
# and the script kills only the hottyterm it started and that one's
# children. The display is shared, so the run holds
# ${XDG_CACHE_HOME:-$HOME/.cache}/hotty/sway/lock (flock -o, so a sway
# started under it does not keep the lock).
#
# Environment: HOTTYTERM (default /usr/bin/hottyterm), HOTTY_BLITZ (the
# hotty-blitz checkout whose scripts/headless.sh and drive.sh it uses; by
# default one beside this checkout, ../hotty-blitz or ../../hotty-blitz/main).
set -eu

root=$(cd "$(dirname "$0")/.." && pwd)
HOTTYTERM=${HOTTYTERM:-/usr/bin/hottyterm}
if [ -z "${HOTTY_BLITZ:-}" ]; then
  for d in "$root/../hotty-blitz" "$root/../../hotty-blitz/main"; do
    [ -f "$d/Cargo.toml" ] && { HOTTY_BLITZ=$(cd "$d" && pwd); break; }
  done
fi
HOTTY_BLITZ=${HOTTY_BLITZ:?no hotty-blitz checkout beside this one: set HOTTY_BLITZ}
CACHE="${XDG_CACHE_HOME:-$HOME/.cache}/hotty/sway"

usage() {
  sed -n '2s/^# /usage: /p' "$0" >&2
  exit 2
}

# Every PLX_* variable the calling terminal hands down goes, before
# anything runs.
for v in $(env | sed -n 's/^\(PLX_[A-Za-z0-9_]*\)=.*/\1/p'); do
  unset "$v"
done
if env | grep -q '^PLX_'; then
  echo "shot.sh: PLX_* variables remain; refusing to run:" >&2
  env | grep '^PLX_' | cut -d= -f1 >&2
  exit 1
fi

# The rest runs holding the display's lock.
if [ -z "${HOTTY_UI_SHOT_LOCKED:-}" ]; then
  mkdir -p "$CACHE"
  HOTTY_UI_SHOT_LOCKED=1 exec flock -o -w 600 "$CACHE/lock" sh "$0" "$@"
fi
unset HOTTY_UI_SHOT_LOCKED

wait_first=3
wait_keys=1
res=2400x1500
keys=""
while getopts w:k:c:m:d:r: opt; do
  case $opt in
    w) wait_first=$OPTARG ;;
    k) keys="${keys}keys $OPTARG
" ;;
    c) keys="${keys}click $(echo "$OPTARG" | tr ',' ' ')
" ;;
    m) keys="${keys}move $(echo "$OPTARG" | tr ',' ' ')
" ;;
    d) wait_keys=$OPTARG ;;
    r) res=$OPTARG ;;
    *) usage ;;
  esac
done
shift $((OPTIND - 1))
[ $# -ge 4 ] && [ "$3" = "--" ] || usage
name=$1
scales=$2
shift 3

[ -x "$HOTTYTERM" ] || { echo "shot.sh: $HOTTYTERM is not there" >&2; exit 1; }

# ours is the headless sway headless.sh started: alive, run with its config,
# with its display and IPC sockets.
ours() {
  pid=$(cat "$CACHE/pid" 2>/dev/null) || return 1
  [ -n "$pid" ] && [ "$(cat "/proc/$pid/comm" 2>/dev/null)" = sway ] || return 1
  tr '\0' ' ' <"/proc/$pid/cmdline" | grep -q -- "$CACHE/config" || return 1
  disp=$(cat "$CACHE/display" 2>/dev/null) || return 1
  [ -S "$XDG_RUNTIME_DIR/$disp" ] && [ -S "$XDG_RUNTIME_DIR/sway-ipc.$(id -u).$pid.sock" ]
}
if ! ours; then
  sh "$HOTTY_BLITZ/scripts/headless.sh" start
  sleep 1
  ours || { echo "shot.sh: the headless sway did not come up (see $CACHE/sway.log)" >&2; exit 1; }
fi
sock="$XDG_RUNTIME_DIR/sway-ipc.$(id -u).$pid.sock"

# Nothing below reaches the real display.
unset DISPLAY HYPRLAND_INSTANCE_SIGNATURE XDG_ACTIVATION_TOKEN DESKTOP_STARTUP_ID
export WAYLAND_DISPLAY="$disp" GDK_BACKEND=wayland

swaymsg() {
  LD_LIBRARY_PATH="$CACHE/root/usr/lib" "$CACHE/root/usr/bin/swaymsg" -s "$sock" "$@" >/dev/null
}

# term is the hottyterm this run started, "" for none.
term=""
descendants() {
  for c in $(pgrep -P "$1" 2>/dev/null); do
    echo "$c"
    descendants "$c"
  done
}
stop_term() {
  [ -n "$term" ] || return 0
  kids=$(descendants "$term")
  kill "$term" 2>/dev/null || true
  i=0
  while kill -0 "$term" 2>/dev/null && [ $i -lt 20 ]; do
    sleep 0.1
    i=$((i + 1))
  done
  kill -9 "$term" 2>/dev/null || true
  wait "$term" 2>/dev/null || true
  for k in $kids; do kill "$k" 2>/dev/null || true; done
  term=""
}
trap stop_term EXIT
trap 'exit 130' INT TERM

mkdir -p "$root/.shots"
# The display has a keyboard, so a focused window, only while a wtype runs:
# without one held around the shot, a window shot after keys is unfocused
# (a hollow caret, and the program told it lost focus).
WTYPE="${XDG_CACHE_HOME:-$HOME/.cache}/hotty/wtype/root/usr/bin/wtype"
shoot() {
  if ! kill -0 "$term" 2>/dev/null; then
    echo "shot.sh: hottyterm exited before the shot $1; its log:" >&2
    tail -20 "$log" >&2
    exit 1
  fi
  kb=""
  if [ -x "$WTYPE" ]; then
    "$WTYPE" -s 1200 &
    kb=$!
    sleep 0.6
  fi
  grim -o HEADLESS-1 "$1"
  [ -z "$kb" ] || wait "$kb" 2>/dev/null || true
  echo "$1"
}

for scale in $(echo "$scales" | tr ',' ' '); do
  swaymsg output HEADLESS-1 resolution "$res" scale "$scale"
  log="$root/.shots/$name-$scale.log"
  for try in 1 2; do
    "$HOTTYTERM" --gtk-single-instance=false --window-decoration=false --cursor-style-blink=false -e "$@" >"$log" 2>&1 &
    term=$!
    sleep "$wait_first"
    kill -0 "$term" 2>/dev/null && break
    # The first launch after sway starts has failed to create its display.
    if [ $try = 1 ] && grep -qi 'display' "$log"; then
      wait "$term" 2>/dev/null || true
      term=""
      continue
    fi
    break
  done
  shoot "$root/.shots/$name-$scale.png"
  n=0
  while IFS= read -r k; do
    [ -n "$k" ] || continue
    n=$((n + 1))
    case $k in
      move\ *)
        mx=${k#move }
        lw=$(echo "$res $scale" | awk '{ split($1, d, "x"); printf "%d %d", d[1] / $2, d[2] / $2 }')
        # shellcheck disable=SC2086 # the logical width and height, split
        WAYLAND_DISPLAY="$disp" python3 "$HOTTY_BLITZ/scripts/vpointer.py" $lw move "${mx%% *}" "${mx#* }"
        ;;
      click\ *)
        cx=${k#click }
        cy=${cx#* }
        cx=${cx%% *}
        lw=$(echo "$res $scale" | awk '{ split($1, d, "x"); printf "%d %d", d[1] / $2, d[2] / $2 }')
        "$WTYPE" -s 1500 &
        ckb=$!
        sleep 0.4
        # shellcheck disable=SC2086 # the logical width and height, split
        WAYLAND_DISPLAY="$disp" python3 "$HOTTY_BLITZ/scripts/vpointer.py" $lw move "$cx" "$cy" sleep 0.1 click
        wait "$ckb" 2>/dev/null || true
        ;;
      *) eval "sh \"\$HOTTY_BLITZ/scripts/drive.sh\" $k" ;;
    esac
    sleep "$wait_keys"
    shoot "$root/.shots/$name-$scale-$n.png"
  done <<EOF
$keys
EOF
  stop_term
done
