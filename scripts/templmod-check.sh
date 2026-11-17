#!/bin/sh
# templmod-check.sh vet|lint: vets or lints internal/devchild's
# surftemplmod file (the call into mods/templ's patch) against the
# patched templ, through the reloader's own go.work for this repository.
# Plain vet and lint can't compile it: plain templ lacks what it calls.
set -eu
GO=${GO:-go}
out=$(mktemp -d)
trap 'rm -rf "$out"' EXIT
work=$($GO run ./internal/templmod/work "$out")
case $1 in
vet) GOWORK=$work $GO vet -tags surfdev,surftemplmod ./internal/devchild/ ;;
lint) GOWORK=$work $GO run "$STATICCHECK" -tags surfdev,surftemplmod ./internal/devchild/ ;;
*) echo "templmod-check.sh vet|lint" >&2; exit 2 ;;
esac
