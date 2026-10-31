#!/usr/bin/env bash
# wasm-sizes.sh: the module of each program by each toolchain and its
# flags, as Markdown tables (vault/knowledge/measures.md's): bytes as
# built, and gzipped at -9, as a server would send them; then what of the
# templates' machinery each links.
#
# The programs are one counter, run through the runtime on hottytest's
# host in the process and clicked three times: examples/wasm draws it
# with Go builders, examples/wasm-surf by a .surf template, interpreted,
# and with -tags compiled, compiled. Each module is run under Node after
# it is built, and must print its three clicks.
#
# The modules go in bin/wasm/sizes/. Each build runs under CAP, one at a
# time, as make wasm's do (the Makefile says why). SIZES_ONLY=1 measures
# the last builds again.
#
#   GO=… NODE=… TINYGO=… CAP=… scripts/wasm-sizes.sh     (make wasm-sizes)
set -euo pipefail
cd "$(dirname "$0")/.."
GO=${GO:-mise x -- go}
NODE=${NODE:-mise x node@26.10.0 --}
TINYGO=${TINYGO:?the path to tinygo}
CAP=${CAP-}
out=bin/wasm/sizes
mkdir -p "$out"

progs=(builders interpreted compiled)
declare -A pkg tags title
pkg[builders]=./examples/wasm tags[builders]=
pkg[interpreted]=./examples/wasm-surf tags[interpreted]=
pkg[compiled]=./examples/wasm-surf tags[compiled]=compiled
title[builders]='`examples/wasm`, Go builders'
title[interpreted]='`examples/wasm-surf`, interpreted'
title[compiled]='`examples/wasm-surf`, compiled'
# BUILDS: the builds to make and run, of these (all by default).
read -ra builds <<<"${BUILDS:-go go-stripped tinygo tinygo-small}"

# The flags are words.
# shellcheck disable=SC2086
go_exec="$($GO env GOROOT)/lib/wasm/go_js_wasm_exec"
# shellcheck disable=SC2086
tinygo_exec="$($NODE "$TINYGO" env TINYGOROOT)/targets/wasm_exec_node.js"

# build PROG BUILD: PROG's module by BUILD, in $out/PROG-BUILD.wasm; the
# TinyGo build with debug info also writes its sizes by package
# (-size full) to $out/PROG-tinygo.size, and a native build of PROG is
# $out/PROG.native, for its symbols.
# shellcheck disable=SC2086
build() {
  local p=$1 b=$2 f="$out/$1-$2.wasm" t=()
  [ -z "${tags[$p]}" ] || t=(-tags "${tags[$p]}")
  case $b in
  go) GOOS=js GOARCH=wasm $CAP $GO build "${t[@]}" -o "$f" "${pkg[$p]}" ;;
  go-stripped) GOOS=js GOARCH=wasm $CAP $GO build "${t[@]}" -trimpath -ldflags='-s -w' -o "$f" "${pkg[$p]}" ;;
  tinygo) $CAP $NODE "$TINYGO" build -target=wasm "${t[@]}" -size full -o "$f" "${pkg[$p]}" >"$out/$p-tinygo.size" ;;
  tinygo-small) $CAP $NODE "$TINYGO" build -target=wasm "${t[@]}" -no-debug -opt=z -o "$f" "${pkg[$p]}" ;;
  native) $CAP $GO build "${t[@]}" -o "$out/$p.native" "${pkg[$p]}" ;;
  esac
}

# check PROG BUILD: the module runs under Node and prints its clicks.
# shellcheck disable=SC2086
check() {
  local f="$out/$1-$2.wasm" got
  case $2 in
  go*) got=$($CAP $NODE "$go_exec" "$f") ;;
  tinygo*) got=$($CAP $NODE node "$tinygo_exec" "$f") ;;
  esac
  [[ "$got" == *"3 clicks, 3 updates of 3 ops"* ]] || { echo "wasm-sizes: $f printed: $got" >&2; exit 1; }
}

[ -n "${SIZES_ONLY:-}" ] || for p in "${progs[@]}"; do
  for b in "${builds[@]}"; do
    build "$p" "$b"
    check "$p" "$b"
  done
  build "$p" native
done

# mib FILE: its size in MiB, as built and gzipped at -9.
mib() {
  awk -v b="$(wc -c <"$1")" -v z="$(gzip -9c "$1" | wc -c)" 'BEGIN { printf "%.2f / %.2f", b / 1048576, z / 1048576 }'
}

# shellcheck disable=SC2086
gov=$($GO env GOVERSION)
# shellcheck disable=SC2086
tgv=$($NODE "$TINYGO" version | awk '{print $3}')
declare -A how=(
  [go]="Go ${gov#go} | (none)"
  [go-stripped]="Go ${gov#go} | \`-trimpath -ldflags='-s -w'\`"
  [tinygo]="TinyGo $tgv | (none: \`-opt=z\`, with debug info)"
  [tinygo-small]="TinyGo $tgv | \`-no-debug -opt=z\`"
)
echo "Module, MiB as built / gzip -9:"
echo
printf '| Toolchain | Flags |'
for p in "${progs[@]}"; do printf ' %s |' "${title[$p]}"; done
printf '\n| --- | --- |'
for _ in "${progs[@]}"; do printf ' ---: |'; done
echo
for b in "${builds[@]}"; do
  printf '| %s |' "${how[$b]}"
  for p in "${progs[@]}"; do printf ' %s |' "$(mib "$out/$p-$b.wasm")"; done
  echo
done

# What each links of the templates' machinery, in KiB by package: from
# TinyGo's -size full of the build with debug info, its flash (code,
# rodata and data: what the module holds); and from the symbols of a
# native Go build of the same program (go tool nm reads no wasm module),
# but for fips140's 32 MiB DRBG buffer, which takes no bytes of the file.
parts=(github.com/neuroplastio/cel-lite github.com/neuroplastio/hotty-surf github.com/neuroplastio/hotty-surf/pkg/markup crypto)
# tinygoKiB PROG PKG: PKG's flash in PROG's TinyGo report (its columns:
# code rodata data bss | flash ram | package); crypto is every crypto
# package's.
tinygoKiB() {
  awk -v want="$2" '
    $NF == want || (want == "crypto" && $NF ~ /^crypto(\/|$)/) { n += $6 }
    END { printf "%.1f", n / 1024 }' "$out/$1-tinygo.size"
}
# goKiB PROG PKG: the size of PKG's symbols in PROG's native build.
goKiB() {
  # shellcheck disable=SC2086
  $GO tool nm -size "$out/$1.native" | awk -v want="$2" '
    $3 ~ /^[TtRrDdBb]$/ {
      s = $4; sub(/^type:\.?(eq\.)?\*?/, "", s); sub(/^go:/, "", s)
      p = s; if (match(p, /^.*\//)) { pre = substr(p, 1, RLENGTH); rest = substr(p, RLENGTH + 1) } else { pre = ""; rest = p }
      sub(/\..*$/, "", rest); p = pre rest
      if (p == want || (want == "crypto" && p ~ /^crypto(\/|$)/ && $2 < 1048576)) n += $2
    }
    END { printf "%.1f", n / 1024 }'
}
echo
echo "What each links, KiB by package (TinyGo: flash; Go: symbols):"
echo
printf '| Program | Toolchain |'
for x in "${parts[@]}"; do printf ' `%s` |' "${x#github.com/neuroplastio/}"; done
printf '\n| --- | --- |'
for _ in "${parts[@]}"; do printf ' ---: |'; done
echo
for p in "${progs[@]}"; do
  printf '| %s | TinyGo |' "${title[$p]}"
  for x in "${parts[@]}"; do printf ' %s |' "$(tinygoKiB "$p" "$x")"; done
  printf '\n| %s | Go, native |' "${title[$p]}"
  for x in "${parts[@]}"; do printf ' %s |' "$(goKiB "$p" "$x")"; done
  echo
done
