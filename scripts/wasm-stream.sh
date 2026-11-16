#!/bin/sh
# wasm-stream.sh: Tasks' release wasm (TinyGo) writes the stream Go's js
# build writes, for the same events (internal/wasmcheck), in Node. Run it
# under the machine's CAP (make print-cap); it needs node.
set -e
dir=$(mktemp -d)
trap 'rm -rf "$dir"' EXIT
mise x -- go run ./internal/wasmcheck >"$dir/events.json"
GOOS=js GOARCH=wasm mise x -- go build -o "$dir/go.wasm" ./examples/tasks
mise x -- tinygo build -no-debug -target wasm -o "$dir/tinygo.wasm" ./examples/tasks
goexec="$(mise x -- go env GOROOT)/lib/wasm/wasm_exec.js"
tgexec="$(mise x -- tinygo env TINYGOROOT)/targets/wasm_exec.js"
node internal/wasmcheck/run.mjs "$goexec" "$dir/go.wasm" "$dir/events.json" "$dir/go.stream"
node internal/wasmcheck/run.mjs "$tgexec" "$dir/tinygo.wasm" "$dir/events.json" "$dir/tinygo.stream"
cmp "$dir/go.stream" "$dir/tinygo.stream"
echo "the same stream: $(wc -c <"$dir/go.stream") bytes; TinyGo's wasm $(wc -c <"$dir/tinygo.wasm") bytes, $(gzip -9c "$dir/tinygo.wasm" | wc -c) gzipped"
