# 2026-11-09: the module path is neuroplast.io/hotty-surf

The maintainer had neuroplast.io answer `go get` for every repository of
the org, and moved the libraries under it, the lowest first: cel-lite
v0.2.0, myelin v0.3.0 (myelinkey v0.2.0), hotty-go v0.7.1, then this
module. Done by Infra Agent at the maintainer's request. No board item,
and no change to how anything looks.

## What changed

- **The module is `neuroplast.io/hotty-surf`**, and every import,
  generated file, doc and the CI's importable check say so. The
  repository stays github.com/neuroplastio/hotty-surf.
- **It requires** `neuroplast.io/cel-lite` v0.2.0,
  `neuroplast.io/hotty-go/...` v0.7.1 and
  `neuroplast.io/myelin/myelinkey` v0.2.0: the same code as before at
  the new paths.
- **The next release is v0.3.0.** v0.2.0 and earlier stay at the old
  path, and a program moves when it upgrades.

## How to see it

`make check`; `go list -m` prints `neuroplast.io/hotty-surf`.
