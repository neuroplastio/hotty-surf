// The legible gate: every story the gallery lists, drawn as the gallery
// draws it, measured in headless Chromium for what the eye can't read.
// F5's gate (gate.cjs) diffs stories.All's stories, each in a document of
// its own, Blitz against Chromium. Nothing looked at the rest of the
// gallery's catalog, or at a story in the gallery's canvas: a gallery
// that kept its canvas at its 20-cell floor drew the palette's 80-cell
// stories with every label cut to a letter and an ellipsis and their key
// caps run out of the dialog, and every check passed (2026-10-30).
//
//   node scripts/gate/legible.cjs --hotty <hotty> --fonts <hotty-blitz fonts/> --stories <stories> --out <dir>
//        [--densities compact,dense,roomy] [--themes neuroplastio] [--width N] [--only <substr>]
//        [--known <file>] [--jobs N]
//
// `stories -gallery` (internal/cmd/stories) writes each story of the gallery's
// catalog (each widget's own, stories.All's, the stories files'
// composites and screens) as the whole gallery with the story picked,
// its canvas at the story's declared width, in a 179×54-cell window,
// in both of the gallery's modes, interpreted and compiled. Each
// document is loaded here as gate.cjs loads one: with the host
// stylesheet `hotty css` gives at 1.6 (a high-DPI display's scale), in
// a layer below the document's, and
// hotty-blitz's font. A document the other mode wrote byte for byte is
// measured once (the canvas draws a story the same in both: the mode is
// the frame's). In each, it fails:
//
// - overflow: what draws (a text, or an element with a background, a
//   border or a shadow) past the right edge of a box round it: one that
//   clips (its overflow isn't visible), one that draws its edge, the
//   story's canvas, or the window. It names the nearest box it runs past,
//   and only the outermost of what runs past that. A scroll box counts:
//   the vocabulary's scroll boxes scroll down (.u-scroll), so what runs
//   past one's right edge is out of reach. A text cut by its own ellipsis
//   is the next check's.
// - ellipsis: a text its ellipsis cuts to fewer than MIN characters, or
//   to fewer than all of a text shorter than MIN ("Run" to "R…"). What
//   shows is the longest start of the text whose graphemes' Range rects
//   end before the room the ellipsis takes (a "…" measured in the text's
//   font), and the first character always: CSS Overflow 3 clips it rather
//   than ellipsing it.
//
// Each failure says where it is (the story, or the gallery's frame round
// it), the element's id (from the story's canvas on, in the story), its
// text and what of it shows, and writes <out>/report.md. --width draws
// every canvas N cells wide, not at its story's width: the calibration
// below. --known judges the run against a list of known failures:
// `<story id> <density> <kind> <element> <reason>` lines, each matching
// both modes. A failure off the list fails the run, and so does one on it
// that no longer fails, to come off it: the list is to be empty.
//
// Calibrated on 2026-10-31 (Chromium 153), two ways. With every canvas at
// --width 20, the maintainer's case: the palette's stories with rows
// (empty query, a query) fail, on key caps run out of the dialog and the
// selected row (16 to 30 px) and labels cut to a letter ("Pane" to "P",
// "Actions" to "A"); 28 of the 73 stories fail in all. "no match" passes,
// reading "No command m…". At their own widths, stories the eye reads
// whole pass: the palette's three at compact, button/states and
// field/states at every density. MIN is the maintainer's "about 4", TOL
// rounding.
//
// What it can't see: Blitz. Blitz lays out with Taffy and Parley, and
// `hotty render` gives pixels, not boxes (it has no layout dump), so the
// boxes here are Chromium's; F5's diff (gate.cjs) is what holds Blitz to
// Chromium, on stories.All's stories. Nor a text clipped vertically, a
// story's other themes (a theme changes colours, not sizes), the 2×
// scale, or a story's states matrix and cells UI.
"use strict";
const fs = require("fs");
const os = require("os");
const path = require("path");
const crypto = require("crypto");
const { execFileSync } = require("child_process");

const FONT = "AdwaitaMono Nerd Font Propo"; // as gate.cjs, which says too why IBM Plex Sans is loaded and where Blitz takes it from
const FONT_FILE = "AdwaitaMonoNerdFontPropo";
const ORIGIN = "https://legible.invalid";
const SCALE = 1.6; // a high-DPI display's
const CELL = { w: 16, h: 34 }; // `hotty render --scale 1.6`'s, as gate.cjs's cell(1.6)
const MIN = 4; // the characters an ellipsized text keeps at least
const TOL = 0.5; // CSS px a box may run past an edge: rounding

function args() {
  const a = { jobs: Math.max(2, Math.min(6, os.cpus().length >> 2)), densities: "compact,dense,roomy", themes: "neuroplastio" };
  const v = process.argv.slice(2);
  for (let i = 0; i < v.length; i += 2) a[v[i].replace(/^--/, "")] = v[i + 1];
  for (const k of ["hotty", "fonts", "stories", "out"]) if (!a[k]) throw new Error(`legible: --${k} is needed`);
  if (a.known && a.width) throw new Error("legible: --known is for the stories at their own widths, not --width");
  a.jobs = Number(a.jobs);
  return a;
}

// readKnown is the known failures: `<story id> <density> <kind>
// <element> <reason…>` lines, with # comments.
function readKnown(file) {
  const out = [];
  for (const line of fs.readFileSync(file, "utf8").split("\n")) {
    const t = line.trim();
    if (!t || t.startsWith("#")) continue;
    const [story, density, kind, id, ...why] = t.split(/\s+/);
    if (!why.length || !["overflow", "ellipsis"].includes(kind))
      throw new Error(`${file}: not \`<story id> <density> <kind> <element> <reason>\`: ${t}`);
    out.push({ story, density, kind, id, why: why.join(" "), hit: false });
  }
  return out;
}

// measure runs in the page: the document's failures, by the rules above.
function measure({ canvas, min, tol }) {
  const html = document.documentElement;
  const root = document.getElementById(canvas);
  if (!root) return { error: `no element ${canvas}` };
  const styles = new Map();
  const style = (el) => {
    let s = styles.get(el);
    if (!s) styles.set(el, (s = getComputedStyle(el)));
    return s;
  };
  const opaque = (c) => {
    if (!c || c === "transparent") return false;
    let m = c.match(/^rgba\((.*)\)$/);
    if (m) return parseFloat(m[1].split(",")[3]) > 0;
    m = c.match(/\/\s*([\d.]+)%?\s*\)$/);
    return m ? parseFloat(m[1]) > 0 : true;
  };
  const clips = (el) => el === root || el === html || style(el).overflowX !== "visible";
  const paints = (el) => {
    const s = style(el);
    if (opaque(s.backgroundColor) || s.backgroundImage !== "none" || s.boxShadow !== "none") return true;
    return ["Top", "Right", "Bottom", "Left"].some(
      (d) => parseFloat(s[`border${d}Width`]) > 0 && s[`border${d}Style`] !== "none" && opaque(s[`border${d}Color`]),
    );
  };
  const box = (el) => clips(el) || paints(el);
  const ellipsis = (el) => el !== html && el !== root && style(el).overflowX !== "visible" && style(el).textOverflow === "ellipsis";
  const inline = (el) => style(el).display === "inline";
  // inlineIn says the path from n up to el is inline: el's
  // text-overflow cuts n.
  const inlineIn = (n, el) => {
    for (let a = n.parentElement; a && a !== el; a = a.parentElement) if (!inline(a)) return false;
    return true;
  };
  // A box's right edge: its padding box's, less a scrollbar.
  const edge = (el) => {
    if (el === html) return html.clientWidth;
    const r = el.getBoundingClientRect();
    if (!el.offsetWidth) return r.right;
    return r.right - (el.offsetWidth - el.clientLeft - el.clientWidth) * (r.width / el.offsetWidth);
  };
  const hidden = (el) => style(el).visibility !== "visible" || el.getClientRects().length === 0;
  // An element's id, from the canvas on; a path from the nearest id
  // where it has none. No space in it: a known failure's line is split
  // at spaces.
  const rel = (id) => (id === canvas ? "(canvas)" : id.startsWith(canvas + ".") ? id.slice(canvas.length + 1) : id);
  const name = (el) => rel(idPath(el));
  const idPath = (el) => {
    if (el.id) return el.id;
    const up = [];
    for (let a = el; a && !a.id; a = a.parentElement)
      up.unshift(a.localName + (a.className ? "." + String(a.className).trim().split(/\s+/).join(".") : ""));
    const at = el.closest("[id]");
    return (at ? at.id + ">" : "") + up.join(">");
  };
  const where = (el) => (el !== root && root.contains(el) ? "story" : "frame");
  const seg = new Intl.Segmenter(undefined, { granularity: "grapheme" });
  const range = document.createRange();
  const textsOf = (n) => {
    if (n.nodeType === Node.TEXT_NODE) return [n];
    const w = document.createTreeWalker(n, NodeFilter.SHOW_TEXT);
    const out = [];
    while (w.nextNode()) out.push(w.currentNode);
    return out;
  };
  // The graphemes of texts, each with its rect, in order.
  const graphemes = (texts) => {
    const out = [];
    for (const t of texts)
      for (const { segment, index } of seg.segment(t.data)) {
        if (out.length >= 400) return out;
        range.setStart(t, index);
        range.setEnd(t, index + segment.length);
        const r = range.getBoundingClientRect();
        out.push({ g: segment, left: r.left, right: r.right, w: r.width });
      }
    return out;
  };
  const tidy = (s) => s.replace(/\u0000+/g, "…").replace(/\s+/g, " ").trim();
  const count = (s) => [...seg.segment(s)].filter((x) => x.segment.trim()).length;
  const rectOf = (n) => {
    if (n.nodeType === Node.TEXT_NODE) {
      range.selectNodeContents(n);
      return range.getBoundingClientRect();
    }
    return n.getBoundingClientRect();
  };
  const fails = [];

  // Overflow: what draws (an element that paints, or a text) past the
  // right edge of a box round it, the nearest it runs past. A text cut
  // by its own ellipsis shows up to its box's edge, which is then the
  // ellipsis check's.
  const ink = [];
  for (const el of document.body.querySelectorAll("*"))
    if (!hidden(el) && (paints(el) || /^(img|svg|canvas|input|textarea|video)$/.test(el.localName))) ink.push(el);
  const tw = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT);
  while (tw.nextNode()) if (tw.currentNode.data.trim() && !hidden(tw.currentNode.parentElement)) ink.push(tw.currentNode);
  const past = new Map(); // what runs past → { box, by }
  for (const n of ink) {
    const r = rectOf(n);
    if (r.width === 0) continue;
    let right = r.right;
    const isText = n.nodeType === Node.TEXT_NODE;
    for (let a = n.parentElement; a; a = a.parentElement) {
      if (ellipsis(a) && (isText || inline(n)) && inlineIn(n, a)) {
        right = Math.min(right, edge(a));
        continue;
      }
      if (!box(a)) continue;
      const e = edge(a);
      if (right > e + tol) {
        past.set(n, { box: a, by: right - e });
        break;
      }
      if (clips(a)) right = Math.min(right, e);
    }
  }
  for (const [n, p] of past) {
    let outer = true;
    for (let a = n.parentElement; a && a !== p.box; a = a.parentElement) if (past.get(a)?.box === p.box) outer = false;
    if (!outer) continue;
    const el = n.nodeType === Node.TEXT_NODE ? n.parentElement : n;
    const gs = graphemes(textsOf(n));
    const e = edge(p.box);
    fails.push({
      kind: "overflow",
      where: where(el),
      id: name(el) + (n === el ? "" : "/text()"),
      box: name(p.box),
      by: p.by,
      text: tidy(gs.map((x) => x.g).join("")),
      visible: tidy(gs.map((x) => (x.w === 0 || x.right <= e + tol ? x.g : "\u0000")).join("").replace(/\u0000+$/, "")),
    });
  }

  // Ellipsis: each box its text-overflow cuts. It shows the longest start
  // of its text that leaves room for the ellipsis; and the first
  // character always, "clipped rather than ellipsed" (CSS Overflow 3).
  for (const el of document.body.querySelectorAll("*")) {
    if (!ellipsis(el) || hidden(el) || el.scrollWidth <= el.clientWidth) continue;
    const probe = document.createElement("span");
    probe.textContent = "…";
    probe.style.whiteSpace = "pre";
    el.appendChild(probe);
    const dots = probe.getBoundingClientRect().width;
    probe.remove();
    const e = edge(el);
    const gs = graphemes(textsOf(el).filter((t) => inlineIn(t, el)));
    let fit = 0;
    while (fit < gs.length && gs[fit].right <= e - dots + tol) fit++;
    if (fit === 0 && gs.length && gs[0].left < e) fit = 1;
    const full = tidy(gs.map((x) => x.g).join(""));
    const seen = tidy(gs.slice(0, fit).map((x) => x.g).join(""));
    const n = count(full), v = count(seen);
    if (v < Math.min(min, n)) fails.push({ kind: "ellipsis", where: where(el), id: name(el), text: full, visible: seen });
  }
  return { fails };
}

(async () => {
  const { chromium } = require("playwright");
  const t0 = Date.now();
  const a = args();
  const out = path.resolve(a.out);
  const work = path.join(out, "work");
  fs.rmSync(out, { recursive: true, force: true });
  fs.mkdirSync(work, { recursive: true });

  // The documents: each theme and density's, in both modes.
  const docs = [];
  for (const theme of a.themes.split(","))
    for (const density of a.densities.split(",")) {
      const dir = path.join(work, `${theme}-${density}`);
      const argv = ["-gallery", "-out", dir, "-theme", theme, "-density", density];
      if (a.width) argv.push("-width", String(a.width));
      execFileSync(a.stories, argv);
      for (const line of fs.readFileSync(path.join(dir, "index.tsv"), "utf8").trim().split("\n")) {
        const [mode, id, file, cols, canvas, wcols, wrows, section, title] = line.split("\t");
        if (a.only && !`${id} ${title}`.includes(a.only)) continue;
        docs.push({ mode, id, theme, density, section, title, cols: +cols, canvas: +canvas, wcols: +wcols, wrows: +wrows, file: path.join(dir, file) });
      }
    }

  // What ORIGIN serves: the fonts, and each document with the host's
  // stylesheet, as gate.cjs serves them: AdwaitaMono and IBM Plex Sans
  // from --fonts. Chromium's side is machine-independent so; Blitz's is
  // not yet (HOTTY is v0.5.8, which takes Plex from the system), until
  // the re-baseline after the kit. gate.cjs says more.
  const hostCSS = execFileSync(a.hotty, ["css", "--cell", `${CELL.w}x${CELL.h}`, "--scale", String(SCALE), "--font", FONT], { encoding: "utf8" });
  const faces = [["Regular", 400, "normal"], ["Bold", 700, "normal"], ["Italic", 400, "italic"], ["BoldItalic", 700, "italic"]];
  const plex = [["Regular", 400], ["Medium", 500], ["SemiBold", 600]];
  const havePlex = plex.every(([f]) => fs.existsSync(path.join(a.fonts, `IBMPlexSans-${f}.woff2`)));
  if (!havePlex) console.error(`legible: no IBM Plex Sans in ${a.fonts}: Chromium takes the system's Plex or a fallback; point HOTTY_FONTS at hotty-blitz 0.6.9's fonts/`);
  const fontCSS = faces
    .map(([f, w, st]) => `@font-face { font-family: "${FONT}"; src: url(/fonts/${FONT_FILE}-${f}.woff2) format("woff2"); font-weight: ${w}; font-style: ${st}; }`)
    .concat(havePlex ? plex.map(([f, w]) => `@font-face { font-family: "IBM Plex Sans"; src: url(/fonts/IBMPlexSans-${f}.woff2) format("woff2"); font-weight: ${w}; font-style: normal; }`) : [])
    .join("\n");
  const served = new Map();
  for (const [f] of faces) served.set(`/fonts/${FONT_FILE}-${f}.woff2`, ["font/woff2", fs.readFileSync(path.join(a.fonts, `${FONT_FILE}-${f}.woff2`))]);
  if (havePlex) for (const [f] of plex) served.set(`/fonts/IBMPlexSans-${f}.woff2`, ["font/woff2", fs.readFileSync(path.join(a.fonts, `IBMPlexSans-${f}.woff2`))]);
  const style = `<style>${fontCSS}\n@layer hotty-host {\n${hostCSS}\n}</style>`;

  // A document the other mode wrote byte for byte is measured once.
  const byHash = new Map();
  for (const d of docs) {
    const src = fs.readFileSync(d.file, "utf8");
    d.hash = crypto.createHash("sha256").update(src).digest("hex").slice(0, 16);
    if (!byHash.has(d.hash)) {
      byHash.set(d.hash, d);
      served.set(`/${d.hash}.html`, ["text/html", src.replace("<head>", "<head>" + style)]);
    }
  }
  const unique = [...byHash.values()];

  const browser = await chromium.launch({ args: ["--font-render-hinting=none"] });
  const chromiumName = `Chromium ${browser.version()}`;
  const ctx = await browser.newContext({ deviceScaleFactor: SCALE, colorScheme: "dark", viewport: { width: 800, height: 100 } });
  await ctx.route(`${ORIGIN}/**`, (route) => {
    const hit = served.get(new URL(route.request().url()).pathname);
    return hit ? route.fulfill({ status: 200, contentType: hit[0], body: hit[1] }) : route.fulfill({ status: 404 });
  });
  const pages = [];
  for (let w = 0; w < a.jobs; w++) pages.push(await ctx.newPage());
  const results = new Map(); // hash → failures
  let next = 0;
  await Promise.all(
    pages.map(async (page) => {
      while (next < unique.length) {
        const d = unique[next++];
        await page.setViewportSize({ width: Math.ceil((d.wcols * CELL.w) / SCALE), height: Math.ceil((d.wrows * CELL.h) / SCALE) });
        await page.goto(`${ORIGIN}/${d.hash}.html`, { waitUntil: "load" });
        await page.evaluate(() => document.fonts.ready);
        const r = await page.evaluate(measure, { canvas: `r.kpage.kmain.kmid.kc-${d.id}`, min: MIN, tol: TOL });
        if (r.error) throw new Error(`${d.id} (${d.theme} ${d.density}): ${r.error}`);
        results.set(d.hash, r.fails);
      }
    }),
  );
  await browser.close();

  const ran = new Set(docs.map((d) => `${d.id} ${d.density}`));
  const known = a.known ? readKnown(a.known).filter((k) => ran.has(`${k.story} ${k.density}`)) : null;
  const rows = [];
  for (const d of docs)
    for (const f of results.get(d.hash)) {
      const k = known && known.find((x) => x.story === d.id && x.density === d.density && x.kind === f.kind && x.id === f.id);
      if (k) k.hit = true;
      rows.push({ ...f, el: f.id, story: d.id, mode: d.mode, theme: d.theme, density: d.density, known: k ? k.why : "" });
    }
  const fresh = rows.filter((r) => !r.known);
  const stale = known ? known.filter((k) => !k.hit) : [];
  const failing = new Set(rows.map((r) => `${r.mode} ${r.story} ${r.theme} ${r.density}`));
  const secs = ((Date.now() - t0) / 1000).toFixed(0);
  const cut = (s, n = 60) => (s.length > n ? s.slice(0, n - 1) + "…" : s);
  const md = (s) => String(s).replace(/\|/g, "\\|").replace(/\n/g, " ");
  const kind = (r) => r.kind + (r.kind === "overflow" ? ` ${r.by.toFixed(1)}px past ${r.box}` : "");
  const lines = [
    "# Legible gate: the gallery's stories, measured in Chromium",
    "",
    `${new Date().toISOString().slice(0, 16).replace("T", " ")} UTC, ${chromiumName}, at ${SCALE}×. ` +
      `${docs.length} documents (${unique.length} distinct: a mode's that the other wrote byte for byte is measured once), ` +
      `${failing.size} failing, ${rows.length} failures, in ${secs} s.` +
      (a.width ? ` Every canvas at --width ${a.width}, not its story's width.` : ""),
    "",
    `A failure is a text, or an element that paints, past the right edge of a box round it that clips or paints ` +
      `(overflow: by how many CSS px, past which box, and what of it shows), or a text its ellipsis cuts to fewer ` +
      `than ${MIN} characters, or to fewer than all of a shorter one (ellipsis). ` +
      "Where: the story, or the gallery's frame round it. scripts/gate/legible.cjs says how it measures, and what it can't see.",
    "",
    ...(known
      ? [
          `Against the known failures (\`${a.known}\`): ${fresh.length} new, ${stale.length} to come off it.`,
          ...stale.map((k) => `- fixed: ${k.story} ${k.density} ${k.kind} ${k.id}: take it off the list`),
          "",
        ]
      : []),
    "| story | mode | theme, density | where | kind | element | text | visible | known |",
    "| --- | --- | --- | --- | --- | --- | --- | --- | --- |",
    ...rows.map(
      (r) =>
        `| ${md(r.story)} | ${r.mode} | ${r.theme} ${r.density} | ${r.where} | ${kind(r)} | \`${md(r.el)}\` | ` +
        `${md(cut(r.text))} | ${md(cut(r.visible))} | ${md(r.known)} |`,
    ),
    "",
  ];
  fs.writeFileSync(path.join(out, "report.md"), lines.join("\n"));
  for (const r of rows)
    console.log(
      `${r.known ? "known" : "FAIL "} ${r.story} ${r.mode} ${r.theme}-${r.density} ${r.where} ${kind(r)} ${r.el} ` +
        `${JSON.stringify(cut(r.text))} shows ${JSON.stringify(cut(r.visible))}`,
    );
  for (const k of stale) console.log(`fixed: ${k.story} ${k.density} ${k.kind} ${k.id}: take it off ${a.known}`);
  console.log(`${docs.length} documents, ${failing.size} failing, ${rows.length} failures (${fresh.length} new), ${secs} s: ${path.join(a.out, "report.md")}`);
  process.exit(fresh.length || stale.length ? 1 : 0);
})().catch((e) => {
  console.error(e);
  process.exit(2);
});
