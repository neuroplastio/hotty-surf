#!/bin/sh
# canary.sh: applies each set under mods/ to its upstream's latest and runs
# its tests (gov P-7's canary). A red canary is the set's owner's work: the
# patch is redone on the new upstream, or dropped where upstream made it
# unnecessary, and the set's base moves forward when it's green.
#
#   templ: its default branch, mods/templ/*.patch applied, then
#   go test ./runtime/.
#
# git apply, not git am: the same apply without the commit, which would
# need an identity in a throwaway clone.
set -eu
GO=${GO:-go}
root=$(cd "$(dirname "$0")/.." && pwd)
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

git clone --quiet --depth 1 https://github.com/a-h/templ "$work/templ"
cd "$work/templ"
echo "templ at $(git rev-parse --short HEAD) ($(git log -1 --format=%cs))"
for p in "$root"/mods/templ/*.patch; do
	git apply --index "$p"
done
$GO test ./runtime/
echo "mods/templ: green on templ's latest"
