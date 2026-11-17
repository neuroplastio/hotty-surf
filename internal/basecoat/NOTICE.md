Basecoat 1.1.0 (basecoat-css on npm), Copyright (c) 2025 Ronan Berder, MIT
licensed; see LICENSE.md. These are its source files, unmodified:

- `components/*.css` and `base.css` (dist/components, dist/base) and
  `vega.css` (dist/styles), the default theme: ui/style/build.mjs builds
  ui/kit.css from them;
- `jinja/*.html.jinja` (templates/jinja), the macros ui's components
  render as, and the oracle their render tests compare against;
- `js/*.js` (dist/js, unminified, without all.js), the behaviour ui's
  components translate to Go, and which the differential tests run in
  Chromium. No Basecoat JavaScript ships in hotty-surf.

On a Basecoat bump, replace these files from the new package, rebuild
the kit (ui/style) and the oracle, and run the tests: what fails is what
changed.
