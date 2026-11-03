# Q-0001 — A colour that comes from data: named colours, declared once

Status: answered, built in ff11eb0..142738a (hues; script CSS tokens, the last bullet, are left for a leg of their own)
Blocks: hue on text, dot, glyph and icon; script CSS (after `<icon>`)
Raised: 2026-11-03

## What I need decided

A view wants a colour of its own data's on its text and dots, such as each
agent in a list drawn in the colour its user gave it. surf's colours today
are the look's: the tones (accent, ok, warn, error, info, muted, plain) and
the tokens a theme sets. A second, related ask: a script adding its own CSS
to a view.

## Why I cannot decide it

A colour shape is what every view that uses it builds on, and it decides
whether themes keep control of colour. Unwinding it is a migration.

## Options

- Any colour, bound inline (`:colour="a.hex"`). Themes lose control, and
  nothing keeps text readable on both grounds.
- A fixed categorical palette in the look, bound by name.
- Named colours the view declares, bound by name, which a theme may
  override.

## Answer

The maintainer, 2026-11-03: "I don't want any-color support by default, so
themes can potentially stay in control. But in some cases like in the
feedback request (agents have colors), the set of colors is defined by the
script. So, can we just have color set defined as "classes" in CSS and let
script render elements that refer to them?" Then, on the shape below: "as
written".

```html
<palette>
  <hue name="coral" dark="#ff8e7a" light="#b4321f" cells="9"/>
  <hue name="teal"  dark="#4fd1c5" light="#0f766e" cells="6"/>
</palette>

<row :each="a in agents">
  <dot :hue="a.colour"/>
  <text :hue="a.colour">{{ a.name }}</text>
</row>
```

- **A hue is a name**, and `hue` goes on text, dot, glyph and icon. An
  element takes a tone or a hue, not both. A literal name that no palette
  declares is refused at load; a bound one draws the default ink and warns
  once. No hex appears anywhere but in a palette.
- **On a host,** each hue is a custom property, `--u-hue-<name>`, on the
  surface (its dark value, and its light one under
  `prefers-color-scheme: light`), and an element that takes it has the
  class `u-hue-<name>`.
- **Where it lives:** `<palette>` in a `.surf` file, so Go and Lua, the
  interpreted and the compiled mode all read it. A set known only at run
  time (a colour from a user's configuration) is declared with
  `surf.Palette{…}` in Go and `surf.palette{ coral = { dark = "…", light
  = "…", cells = 9 } }` in Lua. Declaring again replaces the set and sends
  it again, as `SetCSS` does.
- **Pairs are recommended, not required.** A single `value=` is checked
  against both grounds, and few mid tones pass both.
- **Contrast:** text needs 4.5:1 and a dot 3:1, against the dark ground and
  paper's.
  - A palette in a `.surf` file that fails is refused at load, with the
    number: `coral's dark #a33 is 2.9:1 on the dark ground; text needs
    4.5`.
  - A palette declared at run time warns once, and on the failing theme
    that hue's text is drawn in the default ink; its dots keep it. Text
    stays legible whatever the data says.
- **A theme overrides a name** by setting `--u-hue-<name>` in its own rule,
  as it sets `--u-accent` (`.u-surface.u-theme-paper { --u-hue-coral: …
  }`). A host's stylesheet does the same. surf doesn't check an override's
  contrast: the theme owns it.
- **The cells:** each hue names one of the terminal's 16 colours
  (`cells="9"`), by default the nearest chromatic one by hue angle. The
  terminal's palette draws it, so the terminal's theme keeps control in
  the cells as well. There is no truecolor.
- **Script CSS is tokens only.** A view may set a `--u-*` token, and only
  to a declared hue or another token (`--u-accent: hue(coral)`). No raw
  CSS comes from a script.
