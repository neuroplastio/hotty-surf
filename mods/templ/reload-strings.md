---
title: A program can make development mode read the strings again now
patch: 0001-feat-runtime-read-_templ.txt-files-again-on-demand.patch
upstream-fit: maybe
---

**What it changes.** In development mode, templ reuses a `_templ.txt`
file's strings for 100 ms before it checks the file again. The patch
adds `runtime.ReloadWatchedStrings` (and `StringLoader.Reload`), which
drops the strings read so far, so that the next render reads each file
again. Nothing changes for a program that doesn't call it.

**Why we carry it.** hotty-surf's reloader writes the strings itself and
asks the running process to render again. Without the patch, the process
sleeps until 101 ms after its latest render first, which a save right
after a render pays in full. With it, the process calls
`ReloadWatchedStrings` and renders at once.

**Why an overlay, not a go.mod replace.** The reloader builds a dev
build with `go build -overlay`, which maps templ's file in the module
cache to a patched copy in the reloader's own directory. go.mod stays as
it is, so an app builds against the same templ everywhere; release
builds never see the patch; and the module cache is only read. When the
app's templ isn't the version the patch is for, or the patch doesn't
apply, the build goes without it, the process waits as before, and the
report says why.

**For upstream: maybe.** templ's own watch mode doesn't need it: its
proxy reloads the browser after the same 100 ms. A tool that drives
development mode from inside the process (a live reload that keeps the
process, an editor preview) would use it, so it is useful as an API, but
upstream may prefer another shape for it.
