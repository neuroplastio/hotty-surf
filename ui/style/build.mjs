import { readFileSync, readdirSync, writeFileSync, unlinkSync } from "node:fs";
import { execFileSync } from "node:child_process";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
const here = dirname(fileURLToPath(import.meta.url));
const vendor = resolve(here, "../../internal/basecoat");
const components = readdirSync(resolve(vendor, "components"))
  .filter((f) => f.endsWith(".css") && f !== "form.css")
  .map((f) => f.slice(0, -4))
  .sort();
let base = readFileSync(resolve(vendor, "base.css"), "utf8");
base = base
  .replace(
    "@custom-variant dark (&:is(html.dark *));",
    "@custom-variant dark (&:is(.dark *));",
  )
  .replace(
    '"Geist", "Geist Sans", ui-sans-serif, system-ui, sans-serif, "Apple Color Emoji", "Segoe UI Emoji", "Segoe UI Symbol", "Noto Color Emoji"',
    '"IBM Plex Sans", ui-sans-serif, system-ui, sans-serif',
  )
  .replace('"Geist Mono", ui-monospace', "ui-monospace");
writeFileSync(resolve(here, ".base-selected.css"), base);
const utilities =
  "flex flex-col flex-row flex-wrap grid items-center items-start justify-between justify-center grow shrink-0 gap-1 gap-2 gap-3 gap-4 p-1 p-2 p-3 p-4 p-6 px-2 px-3 px-4 py-1 py-2 py-3 m-0 mt-1 mt-2 mt-3 mt-4 mb-2 mb-4 w-full w-fit w-64 w-80 max-w-sm max-w-md text-xs text-sm text-base text-lg text-xl text-foreground text-muted-foreground text-destructive font-normal font-medium font-semibold";
const input = `@import "tailwindcss" source(none);\n@import "./.base-selected.css";\n${components.map((x) => `@import "../../internal/basecoat/components/${x}.css";`).join("\n")}\n@import "../../internal/basecoat/vega.css";\n@source "../*.templ";\n@source "../shell.go";\n@source inline("${utilities}");\n@theme { --font-sans: "IBM Plex Sans", ui-sans-serif, system-ui, sans-serif; --font-weight-bold: 600; --font-weight-extrabold: 600; --font-weight-black: 600; }\n@layer base { html,body { font-family: var(--font-sans); font-weight: 400; } b,strong { font-weight:600; } }\n@layer base {\n /* Controls: a hand, and their words are a label, not text to select. */\n button, [role=button], [role=tab], [role=switch], [role=menuitem], [role=menuitemcheckbox], [role=menuitemradio], [role=option], [role=checkbox], [role=radio], summary, select, input[type=checkbox], input[type=radio], label:has(> input[type=checkbox], > input[type=radio]), [data-on~=click] { cursor: pointer; user-select: none; }\n :is(button, [role=button], [role=tab], [role=switch], [role=menuitem], [role=option], input, select):is(:disabled, [aria-disabled=true]) { cursor: not-allowed; }\n /* Labels that are chrome, not content: no text cursor, nothing to select. */\n .badge, .kbd, .kit-progress, [role=tablist], .label { cursor: default; user-select: none; }\n}\n@layer components {\n .dark, .light { --color-background:var(--background); --color-foreground:var(--foreground); --color-card:var(--card); --color-card-foreground:var(--card-foreground); --color-popover:var(--popover); --color-popover-foreground:var(--popover-foreground); --color-primary:var(--primary); --color-primary-foreground:var(--primary-foreground); --color-secondary:var(--secondary); --color-secondary-foreground:var(--secondary-foreground); --color-muted:var(--muted); --color-muted-foreground:var(--muted-foreground); --color-accent:var(--accent); --color-accent-foreground:var(--accent-foreground); --color-destructive:var(--destructive); --color-border:var(--border); --color-input:var(--input); --color-ring:var(--ring); --color-sidebar:var(--sidebar); --color-sidebar-foreground:var(--sidebar-foreground); --color-sidebar-primary:var(--sidebar-primary); --color-sidebar-primary-foreground:var(--sidebar-primary-foreground); --color-sidebar-accent:var(--sidebar-accent); --color-sidebar-accent-foreground:var(--sidebar-accent-foreground); --color-sidebar-border:var(--sidebar-border); --color-sidebar-ring:var(--sidebar-ring); --color-chart-1:var(--chart-1); --color-chart-2:var(--chart-2); --color-chart-3:var(--chart-3); --color-chart-4:var(--chart-4); --color-chart-5:var(--chart-5); }
 .kit-surface { min-height:100vh; background:var(--background); color:var(--foreground); font-family:var(--font-sans); }\n .kit-palette-group { gap:0; } .kit-palette-group .item { padding:.25rem .5rem; }
 .kit-palette-group .kit-palette-line { display:flex; flex-direction:row; align-items:baseline; gap:.4rem; }
 .kit-palette-line > span:first-child { margin-right:.5rem; } .kit-palette-line > span.muted { margin-left:.5rem; }
 .kit-separator { width:100%; height:1px; background:var(--border); }\n .kit-switch { display:inline-flex; align-items:center; width:2.25rem; height:1.25rem; border-radius:9999px; background:var(--input); border:1px solid var(--border); padding:2px; }\n .kit-switch[aria-checked="true"] { background:var(--primary); }\n .kit-switch > span { width:.875rem; height:.875rem; border-radius:9999px; background:var(--background); }\n .kit-switch[aria-checked="true"] > span { margin-left:auto; }\n .kit-skeleton { height:1rem; width:12rem; }
 .kit-progress { height:.5rem; border-radius:9999px; background:var(--muted); }\n .kit-progress > span { display:block; height:100%; border-radius:9999px; background:var(--primary); }\n .kit-popup { position:absolute; top:100%; left:0; z-index:20; background:var(--popover); color:var(--popover-foreground); border:1px solid var(--border); border-radius:var(--radius-md); padding:.25rem; min-width:12rem; }\n .kit-popup[hidden], .kit-popup:not([data-open]), .kit-popup[data-open="false"] { display:none; }\n .kit-popup [role=option], .kit-popup [role=menuitem] { padding:.5rem; }\n .kit-selected, .item[data-selected]:not([data-selected="false"]) { background:var(--accent); color:var(--accent-foreground); }
 /* hotty-blitz paints no border on a table row: the cells carry it. */
 .table :is(th, td) { border-bottom:1px solid var(--border); } .table tbody tr:last-child > td { border-bottom:0; }
 /* A popup's items are controls: a hand, where Basecoat (as shadcn) keeps the arrow. */
 .dropdown-menu :is([role=menuitem],[role=menuitemcheckbox],[role=menuitemradio]):not([aria-disabled=true]), .command [role=menuitem]:not([aria-disabled=true]), .select:not(select) [role=option]:not([aria-disabled=true]) { cursor:pointer; }
 /* Hover on a command's items and on the select.js port's options, as on a menu's: Basecoat has none once its JS has started (the
    port marks the select data-select-initialized; a command has no rule at all), since its JS moves .active on mouseover, and
    native sends the program no pointer moves. The host paints :hover itself. */
 .command [role=menuitem]:not([aria-disabled=true]):hover, .select:not(select) [role=option]:not([aria-disabled=true]):not(:disabled):hover { background-color:var(--color-accent); color:var(--color-accent-foreground); }
 /* Basecoat pads a select's group on top of the popover's, so its options sat 4px right of the ungrouped ones; shadcn's line up. */
 .select:not(select) [role=listbox] [role=group] { padding-inline:0; }
 .accent { color:var(--primary); } .bold { font-weight:600; } .muted { color:var(--muted-foreground); }\n}\n`;
writeFileSync(resolve(here, ".input.css"), input);
try {
  execFileSync(
    resolve(process.env.HOTTY_SURF_NODE_MODULES || resolve(here, "node_modules"), ".bin/tailwindcss"),
    [
      "-i",
      resolve(here, ".input.css"),
      "-o",
      resolve(here, "../kit.css"),
      "--minify",
    ],
    { stdio: "inherit" },
  );
} finally {
  for (const f of [".input.css", ".base-selected.css"])
    unlinkSync(resolve(here, f));
}
