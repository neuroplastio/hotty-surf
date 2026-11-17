#!/bin/sh
# The terminal hotty-surf's page runs an app in (build -wasm, serve):
# xterm.js, its fit addon and xterm-addon-hotty, as their npm packages
# ship them, checked against the integrity npm records, into
# internal/page/assets. Run it to bump a version; the files are committed,
# since the command embeds them.
#
#   sh scripts/page-assets.sh
set -eu

# name version integrity (npm's sha512) and the files taken from package/
XTERM="@xterm/xterm 6.1.0-beta.304 sha512-Wq9d4qFYslYQBIan+riXotdyurtREc6v+HWdAwV6Y2JSDI9eJ1dkWm/fzYb2rCGgGnM3baiYuSK4Z6OLNKLzDw=="
FIT="@xterm/addon-fit 0.12.0-beta.301 sha512-MukRyJLAFQrW/++aM209jBZ8G9J/Ioe4SGS6c3ba87v+zYuX/I66JrDB7sVfykQ9t7DlT+c1MWl1jTlX2kyx1w=="
HOTTY="@neuroplastio/xterm-addon-hotty 0.8.0 sha512-MPodAk9s/Rz9cPkK9mGpH33jTXBjOAu+aklbI+AL5gtVFj2dMBCQ7Lx7ZYtOG/zEjN/FuHs6hrOLHEYruepQ4g=="

out=$(cd "$(dirname "$0")/.." && pwd)/internal/page/assets
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

# fetch <name> <version> <integrity>: the package, unpacked in $tmp/<base>
fetch() {
	base=${1##*/}
	url="https://registry.npmjs.org/$1/-/$base-$2.tgz"
	curl -fsSL "$url" -o "$tmp/$base.tgz"
	got="sha512-$(openssl dgst -sha512 -binary "$tmp/$base.tgz" | base64 -w0)"
	if [ "$got" != "$3" ]; then
		echo "$1@$2: integrity $got, want $3" >&2
		exit 1
	fi
	mkdir "$tmp/$base"
	tar -xzf "$tmp/$base.tgz" -C "$tmp/$base"
}

# shellcheck disable=SC2086
fetch $XTERM
# shellcheck disable=SC2086
fetch $FIT
# shellcheck disable=SC2086
fetch $HOTTY

cp "$tmp/xterm/package/lib/xterm.mjs" "$tmp/xterm/package/css/xterm.css" "$out/"
cp "$tmp/xterm/package/LICENSE" "$out/xterm.LICENSE"
cp "$tmp/addon-fit/package/lib/addon-fit.mjs" "$out/"
cp "$tmp/addon-fit/package/LICENSE" "$out/addon-fit.LICENSE"
cp "$tmp/xterm-addon-hotty/package/dist/hotty-xterm.js" "$out/"
cp "$tmp/xterm-addon-hotty/package/LICENSE" "$out/hotty-xterm.LICENSE"
# Source maps point at files that aren't shipped.
sed -i '/^\/\/# sourceMappingURL=/d' "$out/xterm.mjs" "$out/addon-fit.mjs" "$out/hotty-xterm.js"
{
	echo "$XTERM" | cut -d' ' -f1,2
	echo "$FIT" | cut -d' ' -f1,2
	echo "$HOTTY" | cut -d' ' -f1,2
} >"$out/VERSIONS"
echo "assets: $out"
