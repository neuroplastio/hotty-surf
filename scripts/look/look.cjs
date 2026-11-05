// The look loop: edit a .surf, a story or surfui/surf.css, render, compare,
// and look. Three steps, each usable alone (scripts/look/README.md):
//
//   node scripts/look/look.cjs story <name> [--theme dark|light] [--density compact]
//        [--width COLS] [--scale 1.6|2|both] [--engine chromium|blitz|both]
//        [--state hover|focus|within --at <selector>] [--crop <selector>]
//   node scripts/look/look.cjs ref <board.html> [--select <selector>] [--theme dark|light] [--scale …]
//   node scripts/look/look.cjs compare <a.png> <b.png> [--trim] [--name <name>]
//   node scripts/look/look.cjs run <name> --board <board.html> --select <selector> [any of the above]
//
// `story` draws one of internal/stories' stories (the gate's documents,
// written by internal/cmd/stories) in Chromium, as the gate does (the
// host's stylesheet in a layer below the document's, hotty-blitz's
// font, unhinted), and in Blitz, by `hotty render`. dark is the
// neuroplastio theme on the host's dark ground, light the paper theme.
// --state forces a pseudo-class's look (stories.Force) on what --at
// selects, in the document both engines draw. --crop cuts both shots to
// the box of what the selector picks, as Chromium lays it out.
//
// `ref` draws a board of the design (a page drawn with its kit.css), or
// the one element --select picks of it, in Chromium at the same scale;
// --theme puts the board in the kit's dark or light roles, and takes a
// board's own drawing in that theme where there is one (Components-light).
//
// `compare` puts two PNGs side by side with a third panel that marks
// where they differ (red), and prints their sizes, the share of pixels
// that differ and the box the difference covers. A pixel differs when a
// channel is more than --levels (8) outside what the other image has
// within 1 device px of it, so that antialiasing alone doesn't count.
// --trim cuts each image to what differs from its corner's colour
// first, so that a story's surface and a board's card line up.
//
// `run` is the three: the story in both engines, the reference, and the
// sheets story-in-Chromium against the reference and Blitz against
// Chromium, at each scale.
//
// Everything goes under .shots/look/. Options: --hotty (the hotty
// binary), --fonts (hotty-blitz's fonts/), --stories (the stories
// binary; built from internal/cmd/stories when not given), --design (the
// design's directory), --out.
"use strict";
const fs = require("fs");
const path = require("path");
const { execFileSync } = require("child_process");
const { createRequire } = require("module");

const gateRequire = createRequire(path.join(__dirname, "..", "gate", "package.json"));
const { PNG } = gateRequire("pngjs");
const { envelope } = require("../gate/diff.cjs");

const ROOT = path.resolve(__dirname, "..", "..");
const FONT = "AdwaitaMono Nerd Font Propo"; // hotty-blitz's own (fonts.rs)
const FONT_FILE = "AdwaitaMonoNerdFontPropo";
const ORIGIN = "https://look.invalid";
const THEMES = { dark: "neuroplastio", light: "paper" };

// What `hotty render --scale S` uses without --cell (as gate.cjs).
const cell = (s) => ({ w: Math.round(10 * s), h: Math.round(21 * s) });

function parse(argv) {
  const a = { _: [] };
  for (let i = 0; i < argv.length; i++) {
    const v = argv[i];
    if (v === "--trim") a.trim = true;
    else if (v.startsWith("--")) a[v.slice(2)] = argv[++i];
    else a._.push(v);
  }
  a.theme ??= "dark";
  a.density ??= "compact";
  a.engine ??= "both";
  a.levels = Number(a.levels ?? 8);
  a.scales = !a.scale || a.scale === "both" ? [1.6, 2] : [Number(a.scale)];
  a.out = path.resolve(a.out ?? path.join(ROOT, ".shots", "look"));
  a.design = path.resolve(a.design ?? process.env.DESIGN ?? path.join(ROOT, "..", "design"));
  if (!a.hotty) {
    const blitz = process.env.HOTTY_BLITZ ?? path.join(ROOT, "..", "hotty-blitz");
    a.hotty = process.env.HOTTY ?? path.join(blitz, "target", "release", "hotty");
    a.fonts ??= path.join(blitz, "crates", "hotty-blitz", "fonts");
  }
  if (!THEMES[a.theme]) throw new Error(`--theme is dark or light, not ${a.theme}`);
  return a;
}

const slug = (s) => s.replace(/[^A-Za-z0-9._-]+/g, "-").replace(/^-+|-+$/g, "").slice(0, 120);

let browserP;
function browser() {
  const { chromium } = gateRequire("playwright");
  // Unhinted, as Blitz draws text (gate.cjs).
  browserP ??= chromium.launch({ args: ["--font-render-hinting=none"] });
  return browserP;
}

function storiesBin(a) {
  if (a.stories) return a.stories;
  const bin = path.join(ROOT, "bin", "stories");
  execFileSync("mise", ["x", "--", "go", "build", "-o", bin, "./internal/cmd/stories"], { cwd: ROOT, stdio: "inherit" });
  return (a.stories = bin);
}

function storyCols(a, name) {
  const list = execFileSync(storiesBin(a), ["-list"], { encoding: "utf8" }).trim().split("\n");
  const hit = list.map((l) => l.split("\t")).find(([n]) => n === name);
  if (!hit) throw new Error(`no story ${name}; the stories are:\n${list.map((l) => l.split("\t")[0]).join("\n")}`);
  return Number(hit[1]);
}

// The story's document as both engines draw it, with --state forced.
async function storyDoc(a, name) {
  const theme = THEMES[a.theme];
  const dir = path.join(a.out, "work", `${theme}-${a.density}`);
  execFileSync(storiesBin(a), ["-out", dir, "-theme", theme, "-density", a.density]);
  const file = path.join(dir, name.replaceAll("/", "-") + ".html");
  if (!a.state) return file;
  if (!a.at) throw new Error("--state needs --at <selector>");
  const page = await (await browser()).newPage();
  await page.setContent(fs.readFileSync(file, "utf8"));
  const n = await page.evaluate(
    ([sel, st]) => {
      const els = document.querySelectorAll(sel);
      for (const e of els) e.setAttribute("data-force", ((e.getAttribute("data-force") || "") + " " + st).trim());
      return els.length;
    },
    [a.at, a.state],
  );
  if (!n) throw new Error(`--at ${a.at}: nothing in ${name} matches`);
  const forced = file.replace(/\.html$/, `.${slug(a.state + "-" + a.at)}.html`);
  fs.writeFileSync(forced, "<!doctype html>" + (await page.evaluate(() => document.documentElement.outerHTML)));
  await page.close();
  return forced;
}

function hostCSS(a, s) {
  const c = cell(s);
  return execFileSync(a.hotty, ["css", "--cell", `${c.w}x${c.h}`, "--scale", String(s), "--font", FONT], { encoding: "utf8" });
}

// A Chromium page that serves ORIGIN's documents and hotty-blitz's fonts.
async function servingPage(a, s, scheme) {
  const ctx = await (await browser()).newContext({ deviceScaleFactor: s, colorScheme: scheme, viewport: { width: 800, height: 100 } });
  const docs = new Map();
  for (const f of ["Regular", "Bold", "Italic", "BoldItalic"])
    docs.set(`/fonts/${FONT_FILE}-${f}.woff2`, ["font/woff2", fs.readFileSync(path.join(a.fonts, `${FONT_FILE}-${f}.woff2`))]);
  await ctx.route(`${ORIGIN}/**`, (route) => {
    const hit = docs.get(new URL(route.request().url()).pathname);
    return hit ? route.fulfill({ status: 200, contentType: hit[0], body: hit[1] }) : route.fulfill({ status: 404 });
  });
  return { page: await ctx.newPage(), docs, ctx };
}

const fontCSS = [["Regular", 400, "normal"], ["Bold", 700, "normal"], ["Italic", 400, "italic"], ["BoldItalic", 700, "italic"]]
  .map(([f, w, st]) => `@font-face { font-family: "${FONT}"; src: url(/fonts/${FONT_FILE}-${f}.woff2) format("woff2"); font-weight: ${w}; font-style: ${st}; }`)
  .join("\n");

// The story in Chromium, as gate.cjs draws a case: rows counted as Blitz
// counts rows=auto, the shot cut to the rows' device px.
async function chromiumStory(a, file, cols, s, out) {
  const sc = cell(s);
  const { page, docs, ctx } = await servingPage(a, s, "dark");
  const doc = fs.readFileSync(file, "utf8");
  const style = `<style>${fontCSS}\n@layer hotty-host {\n${hostCSS(a, s)}\n}</style>`;
  docs.set("/story.html", ["text/html", doc.includes("<head>") ? doc.replace("<head>", "<head>" + style) : style + doc]);
  await page.setViewportSize({ width: (cols * sc.w) / s, height: 1 });
  await page.goto(`${ORIGIN}/story.html`, { waitUntil: "load" });
  await page.evaluate(() => document.fonts.ready);
  const bottom = await page.evaluate(() => {
    const r = document.documentElement;
    return Math.max(r.getBoundingClientRect().bottom + (parseFloat(getComputedStyle(r).marginBottom) || 0), r.scrollHeight);
  });
  const rows = Math.min(1000, Math.max(1, Math.ceil(Math.ceil(Math.max(0, bottom - 0.5) * s) / sc.h)));
  const w = cols * sc.w, h = rows * sc.h;
  await page.setViewportSize({ width: Math.ceil(w / s), height: Math.ceil(h / s) });
  await page.evaluate(() => document.fonts.ready);
  const shot = PNG.sync.read(await page.screenshot({ animations: "disabled" }));
  const png = new PNG({ width: w, height: h });
  PNG.bitblt(shot, png, 0, 0, w, h, 0, 0);
  fs.writeFileSync(out, PNG.sync.write(png));
  // The box --crop picks, in device px, for both engines' shots.
  let box = null;
  if (a.crop) {
    const r = await page.locator(a.crop).first().boundingBox();
    if (!r) throw new Error(`--crop ${a.crop}: nothing in the story matches`);
    const x = Math.floor(r.x * s), y = Math.floor(r.y * s);
    box = { x, y, w: Math.min(w, Math.ceil((r.x + r.width) * s)) - x, h: Math.min(h, Math.ceil((r.y + r.height) * s)) - y };
  }
  await ctx.close();
  return { rows, box };
}

function blitzStory(a, file, cols, s, out, rows) {
  execFileSync(a.hotty, ["render", file, "-o", out, "--cols", String(cols), "--scale", String(s), "--font", FONT, "--rows", rows ? String(rows) : "auto"]);
}

async function story(a, name) {
  if (!name) throw new Error("story: which story?");
  if (!fs.existsSync(a.hotty)) throw new Error(`no hotty at ${a.hotty}: build hotty-blitz (make blitz release), or set --hotty`);
  const cols = Number(a.width ?? storyCols(a, name));
  const file = await storyDoc(a, name);
  const base = [slug(name), a.theme, a.density, cols, ...(a.state ? [slug(a.state + "-" + a.at)] : []), ...(a.crop ? ["crop-" + slug(a.crop)] : [])].join("-");
  const shots = {};
  fs.mkdirSync(a.out, { recursive: true });
  for (const s of a.scales) {
    const c = path.join(a.out, `${base}-${s}.chromium.png`), b = path.join(a.out, `${base}-${s}.blitz.png`);
    let rows, box;
    if (a.engine !== "blitz") ({ rows, box } = await chromiumStory(a, file, cols, s, c));
    // Blitz at Chromium's rows when both draw, so that the two line up;
    // its own rows=auto otherwise.
    if (a.engine !== "chromium") blitzStory(a, file, cols, s, b, rows);
    if (box)
      for (const f of [c, b]) {
        if (!fs.existsSync(f)) continue;
        const img = PNG.sync.read(fs.readFileSync(f));
        const out = new PNG({ width: box.w, height: box.h });
        PNG.bitblt(img, out, box.x, box.y, Math.min(box.w, img.width - box.x), Math.min(box.h, img.height - box.y), 0, 0);
        fs.writeFileSync(f, PNG.sync.write(out));
      }
    shots[s] = { chromium: a.engine !== "blitz" ? c : null, blitz: a.engine !== "chromium" ? b : null };
    for (const f of Object.values(shots[s])) if (f) console.log(f);
  }
  return shots;
}

// A board, or one element of it, in Chromium with its kit.css.
async function ref(a, board) {
  if (!board) throw new Error("ref: which board?");
  let file = path.isAbsolute(board) || fs.existsSync(board) ? path.resolve(board) : path.join(a.design, board);
  // A board drawn in each theme (Components-dark, Components-light): the
  // theme's own.
  const own = file.replace(/-(dark|light)\.html$/, `-${a.theme}.html`);
  if (fs.existsSync(own)) file = own;
  if (!fs.existsSync(file)) throw new Error(`no board ${file}: unpack the design into --design (${a.design})`);
  const html = fs.readFileSync(file, "utf8");
  const kit = fs.readFileSync(path.join(path.dirname(file), "kit.css"), "utf8");
  const want = a.theme === "light" ? "t-light" : "t-dark";
  const doc = html.replace(/\bt-(dark|light)\b/g, want).replace(/<link rel="stylesheet" href="kit.css">/, () => `<style>${kit}</style>`);
  fs.mkdirSync(a.out, { recursive: true });
  const shots = {};
  for (const s of a.scales) {
    const { page, docs, ctx } = await servingPage(a, s, a.theme);
    docs.set("/board.html", ["text/html", doc]);
    await page.setViewportSize({ width: 1600, height: 1000 });
    await page.goto(`${ORIGIN}/board.html`, { waitUntil: "load" });
    await page.evaluate(() => document.fonts.ready);
    const out = path.join(a.out, `ref-${slug(path.basename(file, ".html"))}-${a.theme}${a.select ? "-" + slug(a.select) : ""}-${s}.png`);
    if (a.select) {
      const el = page.locator(a.select).first();
      if (!(await el.count())) throw new Error(`--select ${a.select}: nothing in ${board} matches`);
      await el.screenshot({ path: out, animations: "disabled" });
    } else await page.screenshot({ path: out, fullPage: true, animations: "disabled" });
    await ctx.close();
    console.log(out);
    shots[s] = out;
  }
  return shots;
}

// What differs from the image's top-left pixel: its content's box.
function contentBox(img, levels) {
  const { width: w, height: h, data } = img;
  let x0 = w, y0 = h, x1 = -1, y1 = -1;
  for (let y = 0; y < h; y++)
    for (let x = 0; x < w; x++) {
      const i = (y * w + x) * 4;
      if (Math.abs(data[i] - data[0]) > levels || Math.abs(data[i + 1] - data[1]) > levels || Math.abs(data[i + 2] - data[2]) > levels) {
        if (x < x0) x0 = x;
        if (x > x1) x1 = x;
        if (y < y0) y0 = y;
        if (y > y1) y1 = y;
      }
    }
  return x1 < 0 ? { x: 0, y: 0, w, h } : { x: x0, y: y0, w: x1 - x0 + 1, h: y1 - y0 + 1 };
}

function crop(img, b, pad) {
  const out = new PNG({ width: b.w + 2 * pad, height: b.h + 2 * pad });
  for (let i = 0; i < out.data.length; i += 4) out.data.set(img.data.subarray(0, 4), i); // the corner's colour
  PNG.bitblt(img, out, b.x, b.y, b.w, b.h, pad, pad);
  return out;
}

// Both at one size, the smaller grown with its corner's colour.
function grow(img, w, h) {
  if (img.width === w && img.height === h) return img;
  const out = new PNG({ width: w, height: h });
  for (let i = 0; i < out.data.length; i += 4) out.data.set(img.data.subarray(0, 4), i);
  PNG.bitblt(img, out, 0, 0, img.width, img.height, 0, 0);
  return out;
}

function within(img, p, env, levels) {
  for (let c = 0; c < 3; c++) {
    const v = img.data[p * 4 + c];
    if (v < env.min[p * 4 + c] - levels || v > env.max[p * 4 + c] + levels) return false;
  }
  return true;
}

async function compare(a, fa, fb, name) {
  if (!fa || !fb) throw new Error("compare: two PNGs");
  let A = PNG.sync.read(fs.readFileSync(fa)), B = PNG.sync.read(fs.readFileSync(fb));
  const sizes = `${A.width}×${A.height} and ${B.width}×${B.height}`;
  if (a.trim) {
    A = crop(A, contentBox(A, a.levels), 8);
    B = crop(B, contentBox(B, a.levels), 8);
  }
  const w = Math.max(A.width, B.width), h = Math.max(A.height, B.height);
  A = grow(A, w, h);
  B = grow(B, w, h);
  const eA = envelope(A, 1), eB = envelope(B, 1);
  const D = new PNG({ width: w, height: h });
  let n = 0, x0 = w, y0 = h, x1 = -1, y1 = -1;
  for (let p = 0; p < w * h; p++) {
    const x = p % w, y = (p / w) | 0;
    const g = (A.data[p * 4] + A.data[p * 4 + 1] + A.data[p * 4 + 2]) / 3;
    const off = !within(A, p, eB, a.levels) || !within(B, p, eA, a.levels);
    D.data.set(off ? [255, 40, 40, 255] : [g * 0.35, g * 0.35, g * 0.35, 255], p * 4);
    if (off) {
      n++;
      if (x < x0) x0 = x;
      if (x > x1) x1 = x;
      if (y < y0) y0 = y;
      if (y > y1) y1 = y;
    }
  }
  const share = (100 * n) / (w * h);
  const box = n ? `${x0},${y0} ${x1 - x0 + 1}×${y1 - y0 + 1}` : "none";
  const line = `${name ?? path.basename(fa) + " vs " + path.basename(fb)}: sizes ${sizes}${a.trim ? ` (trimmed to ${w}×${h})` : ""}; ${share.toFixed(2)}% of pixels differ (${n}); the difference covers ${box} device px`;
  // The sheet: A | B | diff, side by side, or one under another where
  // they're wide, with the numbers over them.
  const gap = 12, wide = w > 2 * h;
  const S = wide ? new PNG({ width: w + 2 * gap, height: 3 * h + 4 * gap }) : new PNG({ width: 3 * w + 4 * gap, height: h + 2 * gap });
  for (let i = 0; i < S.data.length; i += 4) S.data.set([30, 30, 30, 255], i);
  [A, B, D].forEach((img, k) => PNG.bitblt(img, S, 0, 0, w, h, wide ? gap : gap + k * (w + gap), wide ? gap + k * (h + gap) : gap));
  const out = path.join(a.out, `cmp-${slug(name ?? path.basename(fa, ".png") + "--" + path.basename(fb, ".png"))}.png`);
  fs.mkdirSync(a.out, { recursive: true });
  // The numbers as a caption, drawn by Chromium over the sheet.
  const page = await (await browser()).newPage({ deviceScaleFactor: 1 });
  const uri = "data:image/png;base64," + PNG.sync.write(S).toString("base64");
  const esc = (t) => t.replace(/[&<>]/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;" })[c]);
  const [la, lb] = [a["label-a"] ?? path.basename(fa), a["label-b"] ?? path.basename(fb)];
  await page.setViewportSize({ width: Math.max(600, S.width), height: 100 });
  await page.setContent(`<!doctype html><style>body{margin:0;background:#1e1e1e;color:#eee;font:14px/1.35 sans-serif}
    .t{padding:4px 12px}img{display:block}</style>
    <div class=t>${esc(line)}</div><div class=t>${wide ? "top to bottom: " : "left to right: "}${esc(la)} · ${esc(lb)} · where they differ (red)</div><img src="${uri}">`);
  await page.screenshot({ path: out, fullPage: true });
  await page.close();
  console.log(line);
  console.log(out);
  return { out, share, n, box, sizes };
}

async function run(a, name) {
  if (!a.board || !a.select) throw new Error("run: --board and --select");
  const st = await story(a, name);
  const rf = await ref(a, a.board);
  const res = [];
  for (const s of a.scales) {
    const tag = [slug(name), a.theme, a.density, s, ...(a.crop ? [slug(a.crop)] : [])].join("-");
    if (st[s].chromium)
      res.push(await compare({ ...a, trim: true, "label-a": `${name}, Chromium ${s}`, "label-b": `${path.basename(rf[s])}, ${a.select}` }, st[s].chromium, rf[s], `${tag}-vs-board`));
    if (st[s].chromium && st[s].blitz)
      res.push(await compare({ ...a, trim: false, "label-a": `${name}, Blitz ${s}`, "label-b": `${name}, Chromium ${s}` }, st[s].blitz, st[s].chromium, `${tag}-blitz-vs-chromium`));
  }
  return res;
}

(async () => {
  const [cmd, ...rest] = process.argv.slice(2);
  const a = parse(rest);
  switch (cmd) {
    case "story": await story(a, a._[0]); break;
    case "ref": await ref(a, a._[0]); break;
    case "compare": await compare(a, a._[0], a._[1], a.name); break;
    case "run": await run(a, a._[0]); break;
    default:
      console.error("usage: look.cjs story|ref|compare|run … (scripts/look/README.md)");
      process.exit(2);
  }
  if (browserP) await (await browserP).close();
})().catch(async (e) => {
  console.error(e.message ?? e);
  if (browserP) await (await browserP).close().catch(() => {});
  process.exit(1);
});
