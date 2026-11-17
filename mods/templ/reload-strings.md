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

**Why a dev-only go.work replace.** The reloader builds the app's dev
builds with GOWORK naming a go.work of its own: the app's go.work as it
is, or a fresh one using the app's module, plus a replace of templ with a
mirror of templ's module (symlinks into the module cache, and this file
patched). An overlay would be lighter, but Go refuses one that replaces a
file under GOMODCACHE. go.mod and the app's go.work stay as they are, so
an app builds against the same templ everywhere else; release builds
never see the patch; and the module cache is only read. When the app
vendors its modules, its templ isn't the version the patch is for, or
templ is replaced already, the build goes without the patch, the process
waits as before, and the report says why.

**For upstream: maybe.** templ's own watch mode doesn't need it: its
proxy reloads the browser after the same 100 ms. A tool that drives
development mode from inside the process (a live reload that keeps the
process, an editor preview) would use it, so it is useful as an API, but
upstream may prefer another shape for it.
