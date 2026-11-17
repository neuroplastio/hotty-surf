---
upstream: templ
url: https://github.com/a-h/templ
license: MIT
base: v0.3.1118
builds: hotty-surf's live reload (the reloader's dev builds only)
public: true
---

hotty-surf's live reload shows a save of a .templ file's text by writing
templ's development-mode strings and rendering again in the same process,
without a build. templ rereads those strings at most every 100 ms, so a
save that lands just after a render waited for that interval. The set
lets the process tell templ to read them now.

The set is the version go.mod pins, and neither go.mod nor an app's
go.work changes: the reloader mirrors templ's module from the module
cache into its own directory with the patches applied, and builds the
app's dev builds with GOWORK naming a go.work of its own that replaces
templ with the mirror (`internal/templmod`). Apply and check the set on
templ itself:

```sh
git clone --depth 1 --branch v0.3.1118 https://github.com/a-h/templ
cd templ && git am ../mods/templ/*.patch && go test ./runtime/
```

Against templ's latest, clone its default branch instead of the tag.
`go test ./internal/templmod/` checks that every patch has its page and
applies to the pinned version in the module cache.
