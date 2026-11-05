// F5's fidelity gate: every story, in every theme and density, at 1.6× and
// 2×, drawn by hotty-blitz (`hotty render`) and by headless Chromium with
// the same host stylesheet, cell size and font, and diffed. A case whose
// score is over the threshold fails, and the gate exits 1.
//
//   node scripts/gate/gate.cjs --hotty <hotty> --fonts <hotty-blitz fonts/> --stories <stories> --out <dir>
//        [--only <substr>] [--jobs N] [--inject <css>] [--baseline <file>] [--write-baseline <file>]
//
// --baseline judges the run against the known failures instead (`make
// check`'s way): each a Blitz gap on vault/knowledge/blitz-gaps.md, with
// its score. A case off the baseline may not fail; a case on it may fail
// as it did, and no worse than its score and the threshold; and a case on
// it that passes now, or that is no case any more, is to come off it. The
// scores repeat to the last digit from run to run. --write-baseline writes
// this run's failures as a baseline: when a Blitz fix lands, or a gap is
// listed.
//
// A story may ask that some of its boxes be one height: each with the same
// data-same="<name>" (button/small: a row holding a small button is as tall
// as one without it). The gate measures them in Chromium, where a group
// whose heights differ by more than 0.01 CSS px fails the run, baseline or
// not; the diff holds Blitz to Chromium.
//
// --inject adds CSS to Chromium's side alone, after the document's: a
// fault planted to see that the gate catches it (the calibration below).
//
// Writes <out>/<story>-<theme>-<density>-<scale>.png (Blitz | Chromium |
// diff) and <out>/report.md. The metric is diff.cjs; the calibration of its
// knobs is below.
"use strict";
const fs = require("fs");
const os = require("os");
const path = require("path");
const { execFileSync, execFile } = require("child_process");
const { Worker, isMainThread, parentPort } = require("worker_threads");
const { PNG } = require("pngjs");
const { diff } = require("./diff.cjs");

const THEMES = ["neuroplastio", "terminal", "paper"];
const DENSITIES = ["dense", "roomy", "compact"];
const SCALES = [1.6, 2];
const FONT = "AdwaitaMono Nerd Font Propo"; // hotty-blitz's own (fonts.rs)
const FONT_FILE = "AdwaitaMonoNerdFontPropo";
const ORIGIN = "https://gate.invalid";

// The metric's knobs (diff.cjs says what each does).
const RADIUS = 1; // device px an edge may move: antialiasing
const LEVELS = 3; // 0–255 a colour may round
const RAMP = 0.25; // and, across an edge, this share of the edge's contrast
const SHIFT_CSS = 2; // CSS px a cell's content may move: baselines, rounding
const THRESHOLD = 12; // CSS px² that may differ: a case over it fails
//
// Calibrated on 2026-10-29 (hotty built 2026-10-26, Chromium 153), two ways.
// - The stories as they are. Where the sheets show one look, the area that
//   differs is at most 8.2 CSS px² (colour emoji, resampled another way);
//   else under 4. The smallest difference the eye calls a difference, a ◉
//   from another fallback font, is 18; the select and the root a fraction
//   of a pixel short, 79 and up.
// - Faults planted in Chromium's side alone (--inject), each on its story
//   in all 12 variants: every variant failed. A chip 1 px further right
//   (184 and up), a fill mixing 9% of the foreground for 12% (626), no
//   ellipsis (48), muted text in the plain colour (580), no rule
//   (1173), a field without its padding (450), a radius of 0.12em for
//   0.25em (12.5, the closest call).
// 12 is 1.5× the noise and under the smallest fault planted. The score is
// an area in CSS px², not a share of the image, so that a big story is
// held to what a small one is. What it can't see: a glyph's weight (bold
// drawn regular moves each stem's edge by less than RADIUS), a radius off
// by less than about 0.1em, anything moved by less than SHIFT_CSS.

// What `hotty render --scale S` uses without --cell: a cell of 10×21 CSS
// px, rounded to device pixels.
function cell(scale) {
  return { w: Math.round(10 * scale), h: Math.round(21 * scale) };
}

function opts(scale) {
  return { radius: RADIUS, levels: LEVELS, ramp: RAMP, shift: Math.ceil(SHIFT_CSS * scale), tile: cell(scale) };
}

// A diff worker: reads both images, diffs them, writes the diff image.
if (!isMainThread) {
  parentPort.on("message", ({ blitz, chromium, out, scale }) => {
    let A = PNG.sync.read(fs.readFileSync(blitz));
    let B = PNG.sync.read(fs.readFileSync(chromium));
    if (A.width !== B.width) {
      parentPort.postMessage({ error: `widths ${A.width} and ${B.width}` });
      return;
    }
    const h = Math.max(A.height, B.height);
    A = toHeight(A, h);
    B = toHeight(B, h);
    const d = diff(A, B, opts(scale));
    const png = new PNG({ width: A.width, height: h });
    png.data = d.data;
    fs.writeFileSync(out, PNG.sync.write(png));
    // The area that differs, in CSS px², whatever the scale and the story's size.
    parentPort.postMessage({ area: d.counted / (scale * scale), share: d.score, moved: d.moved });
  });
  return;
}

// The image, h rows tall: cut, or grown with transparent rows.
function toHeight(img, h) {
  if (img.height === h) return img;
  const data = Buffer.alloc(img.width * h * 4);
  img.data.copy(data, 0, 0, Math.min(img.data.length, data.length));
  return { width: img.width, height: h, data };
}

function args() {
  const a = { jobs: Math.max(2, Math.min(8, os.cpus().length >> 1)) };
  const v = process.argv.slice(2);
  for (let i = 0; i < v.length; i += 2) a[v[i].replace(/^--/, "")] = v[i + 1];
  for (const k of ["hotty", "fonts", "stories", "out"]) if (!a[k]) throw new Error(`gate: --${k} is needed`);
  a.jobs = Number(a.jobs);
  return a;
}

function run(cmd, argv) {
  return new Promise((resolve, reject) =>
    execFile(cmd, argv, { maxBuffer: 1 << 26 }, (err, stdout, stderr) =>
      err ? reject(new Error(`${cmd} ${argv.join(" ")}: ${stderr || err.message}`)) : resolve(stdout),
    ),
  );
}

// readBaseline is a baseline's scores by case: `<case> <score>` lines,
// with # comments.
function readBaseline(file) {
  const m = new Map();
  for (const line of fs.readFileSync(file, "utf8").split("\n")) {
    const t = line.trim();
    if (!t || t.startsWith("#")) continue;
    const [id, score] = t.split(/\s+/);
    if (!id || !Number.isFinite(Number(score))) throw new Error(`${file}: not \`<case> <score>\`: ${t}`);
    m.set(id, Number(score));
  }
  return m;
}

function writeBaseline(file, failed, hotty) {
  const head = [
    "# F5's known failures (scripts/gate/gate.cjs --baseline): every case here",
    "# is a Blitz gap on vault/knowledge/blitz-gaps.md, with its score in CSS",
    "# px². Written by --write-baseline, again only when a Blitz fix lands or",
    "# a gap is listed. Last written with " + hotty + ".",
  ];
  const rows = failed
    .slice()
    .sort((x, y) => x.id.localeCompare(y.id))
    .map((c) => `${c.id}\t${c.area.toFixed(1)}`);
  fs.writeFileSync(file, [...head, ...rows, ""].join("\n"));
}

async function pool(items, n, fn) {
  let next = 0;
  await Promise.all(
    Array.from({ length: Math.min(n, items.length) }, async (_, w) => {
      while (next < items.length) await fn(items[next++], w);
    }),
  );
}

// One worker thread per job, each taking one diff at a time.
function diffWorkers(n) {
  const ws = Array.from({ length: n }, () => new Worker(__filename));
  return {
    run: (w, msg) =>
      new Promise((resolve, reject) => {
        ws[w].once("message", (r) => (r.error ? reject(new Error(r.error)) : resolve(r)));
        ws[w].postMessage(msg);
      }),
    close: () => Promise.all(ws.map((w) => w.terminate())),
  };
}

// Loads a case's document in Chromium and counts its rows as Blitz counts
// `rows=auto` (Surface::laid_out_rows): laid out one pixel tall, down to
// the root's bottom or what overflows it, half a pixel forgiven.
async function chromiumLoad(page, url, sc, cols) {
  await page.setViewportSize({ width: (cols * sc.cell.w) / sc.scale, height: 1 });
  await page.goto(url, { waitUntil: "load" });
  await page.evaluate(() => document.fonts.ready);
  const bottom = await page.evaluate(() => {
    const r = document.documentElement;
    const m = parseFloat(getComputedStyle(r).marginBottom) || 0;
    return Math.max(r.getBoundingClientRect().bottom + m, r.scrollHeight);
  });
  const px = Math.ceil(Math.max(0, bottom - 0.5) * sc.scale);
  return Math.min(1000, Math.max(1, Math.ceil(px / sc.cell.h)));
}

// The loaded document, rows tall: a viewport of whole CSS px (Chromium's
// are), cut to the rows' device px here, since a clip in CSS px at 1.6×
// loses the last device row.
async function chromiumShot(page, sc, cols, rows, file) {
  const w = cols * sc.cell.w, h = rows * sc.cell.h;
  await page.setViewportSize({ width: Math.ceil(w / sc.scale), height: Math.ceil(h / sc.scale) });
  await page.evaluate(() => document.fonts.ready);
  const shot = PNG.sync.read(await page.screenshot({ animations: "disabled" }));
  if (shot.width < w || shot.height < h) throw new Error(`${file}: a shot of ${shot.width}×${shot.height}, short of ${w}×${h}`);
  const png = new PNG({ width: w, height: h });
  PNG.bitblt(shot, png, 0, 0, w, h, 0, 0);
  fs.writeFileSync(file, PNG.sync.write(png));
}

function escapeHTML(s) {
  return s.replace(/[&<>"]/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" })[c]);
}

// The side-by-side: Blitz | Chromium | diff, under a line with the verdict.
async function sheet(page, file, c, files) {
  const imgs = files.map((f) => fs.readFileSync(f));
  const uri = (buf) => "data:image/png;base64," + buf.toString("base64");
  const col = (t, buf) => `<div><div class=h>${escapeHTML(t)}</div><img src="${uri(buf)}"></div>`;
  const html = `<!doctype html><style>
    body { margin: 0; background: #1e1e1e; color: #eee; font: 15px/1.3 sans-serif; }
    .t { padding: 6px 10px; } .t b { color: ${c.pass ? "#7c7" : "#f66"}; } .t i { color: #fc6; font-style: normal; }
    .r { display: flex; gap: 12px; padding: 0 10px 10px; align-items: flex-start; }
    .h { padding: 2px 0 4px; color: #bbb; } img { display: block; }
  </style><div class=t>${escapeHTML(c.id)} &nbsp; <b>${c.pass ? "pass" : "FAIL"}</b> &nbsp;
    differs ${c.area.toFixed(1)} CSS px² (threshold ${THRESHOLD}) &nbsp; moved ${c.moved.toFixed(1)}% &nbsp;
    rows: Blitz ${c.rowsBlitz}, Chromium ${c.rowsChromium}${c.note ? ` &nbsp; <i>${escapeHTML(c.note)}</i>` : ""}</div>
  <div class=r>${col("hotty-blitz", imgs[0])}${col(c.chromium, imgs[1])}${col("diff: red counts; orange, a speck that doesn't; blue, matched moved; yellow, within tolerance", imgs[2])}</div>`;
  const w = imgs.reduce((s, b) => s + b.readUInt32BE(16), 0) + 50; // PNG IHDR width
  await page.setViewportSize({ width: Math.max(600, w), height: 100 });
  await page.setContent(html, { waitUntil: "load" });
  await page.screenshot({ path: file, fullPage: true });
}

(async () => {
  const { chromium } = require("playwright");
  const t0 = Date.now();
  const a = args();
  const out = path.resolve(a.out);
  const work = path.join(out, "work");
  fs.rmSync(out, { recursive: true, force: true });
  fs.mkdirSync(work, { recursive: true });

  const stories = execFileSync(a.stories, ["-list"], { encoding: "utf8" })
    .trim()
    .split("\n")
    .map((l) => {
      const [name, cols] = l.split("\t");
      return { name, file: name.replaceAll("/", "-"), cols: Number(cols) };
    });
  for (const t of THEMES)
    for (const d of DENSITIES) execFileSync(a.stories, ["-out", path.join(work, `${t}-${d}`), "-theme", t, "-density", d]);

  // The host stylesheet `hotty render` uses at each scale. Blitz gives it
  // the user origin; in Chromium it goes in a cascade layer below every
  // author rule, which is where SPEC §8 puts it and the same for the
  // cascade. (As an unlayered author sheet, `button:enabled` would beat
  // `.u-btn` in Chromium alone.)
  //
  // The font: hotty-blitz ships AdwaitaMono Nerd Font Propo and makes it
  // the first `monospace` (crates/hotty-blitz/src/fonts.rs), where Chromium
  // takes the system's. So both are told its name, as a host whose
  // terminal names its font is (`--font`, which draws the same in Blitz),
  // and Chromium loads the faces from hotty-blitz's fonts/.
  //
  // So does IBM Plex Sans, which the kit draws Compact and Roomy in
  // (400, 500, 600), from hotty-blitz 0.6.9's fonts/. That makes Chromium's
  // side machine-independent. Blitz's is not yet: HOTTY is v0.5.8, which
  // takes Plex from the system, so a machine without Plex falls back to
  // Inter or Noto in Blitz while Chromium draws Plex, and Compact and Roomy
  // cases fail. HOTTY moves to 0.6.9 in a re-baseline after the kit.
  const hostCSS = {};
  for (const s of SCALES) {
    const c = cell(s);
    hostCSS[s] = execFileSync(a.hotty, ["css", "--cell", `${c.w}x${c.h}`, "--scale", String(s), "--font", FONT], { encoding: "utf8" });
  }
  const faces = [["Regular", 400, "normal"], ["Bold", 700, "normal"], ["Italic", 400, "italic"], ["BoldItalic", 700, "italic"]];
  const plex = [["Regular", 400], ["Medium", 500], ["SemiBold", 600]];
  const havePlex = plex.every(([f]) => fs.existsSync(path.join(a.fonts, `IBMPlexSans-${f}.woff2`)));
  if (!havePlex) console.error(`gate: no IBM Plex Sans in ${a.fonts}: Chromium takes the system's Plex or a fallback; point HOTTY_FONTS at hotty-blitz 0.6.9's fonts/`);
  const fontCSS = faces
    .map(([f, w, st]) => `@font-face { font-family: "${FONT}"; src: url(/fonts/${FONT_FILE}-${f}.woff2) format("woff2"); font-weight: ${w}; font-style: ${st}; }`)
    .concat(havePlex ? plex.map(([f, w]) => `@font-face { font-family: "IBM Plex Sans"; src: url(/fonts/IBMPlexSans-${f}.woff2) format("woff2"); font-weight: ${w}; font-style: normal; }`) : [])
    .join("\n");
  const docs = new Map(); // what ORIGIN serves: path → [type, body]
  for (const [f] of faces) {
    const file = path.join(a.fonts, `${FONT_FILE}-${f}.woff2`);
    docs.set(`/fonts/${FONT_FILE}-${f}.woff2`, ["font/woff2", fs.readFileSync(file)]);
  }
  if (havePlex) for (const [f] of plex) docs.set(`/fonts/IBMPlexSans-${f}.woff2`, ["font/woff2", fs.readFileSync(path.join(a.fonts, `IBMPlexSans-${f}.woff2`))]);
  const serve = (id, file, s) => {
    const doc = fs.readFileSync(file, "utf8");
    const style = `<style>${fontCSS}\n@layer hotty-host {\n${hostCSS[s]}\n}</style>`;
    let html = doc.includes("<head>") ? doc.replace("<head>", "<head>" + style) : style + doc;
    if (a.inject) html = html.replace("</head>", `<style>${a.inject}</style></head>`);
    docs.set(`/${id}.html`, ["text/html", html]);
    return `${ORIGIN}/${id}.html`;
  };

  // Notes on cases that are wrong in Chromium too: our CSS's fault, not a
  // Blitz gap. notes.json maps a case id's prefix to the note.
  const notesFile = path.join(__dirname, "notes.json");
  const notes = fs.existsSync(notesFile) ? JSON.parse(fs.readFileSync(notesFile, "utf8")) : {};
  const noteFor = (id) =>
    Object.entries(notes)
      .filter(([k]) => id.startsWith(k))
      .map(([, v]) => v)
      .join(" ");

  const cases = [];
  for (const st of stories)
    for (const t of THEMES)
      for (const d of DENSITIES)
        for (const s of SCALES) {
          const id = `${st.file}-${t}-${d}-${s}`;
          if (a.only && !id.includes(a.only)) continue;
          cases.push({ id, story: st.name, theme: t, density: d, scale: s, cols: st.cols, note: noteFor(id), html: path.join(work, `${t}-${d}`, st.file + ".html") });
        }

  // Unhinted, as Blitz draws text: hinted, Chromium rounds a web font's
  // advances to whole pixels (10.07 px → 10), and a line drifts from
  // Blitz's by a pixel every 14 letters.
  const browser = await chromium.launch({ args: ["--font-render-hinting=none"] });
  const chromiumName = `Chromium ${browser.version()}`;
  // Which hotty: the binary's build time, and its checkout's commit, which
  // may be newer than the build.
  let hottyRev = "unknown";
  try {
    const dir = path.resolve(path.dirname(a.hotty), "../..");
    hottyRev = execFileSync("git", ["-C", dir, "log", "-1", "--format=%h"], { encoding: "utf8" }).trim();
  } catch {}
  const hottyBuilt = fs.statSync(a.hotty).mtime.toISOString().slice(0, 16).replace("T", " ");

  // Chromium pages, one per job and scale, dark as the host's theme is
  // (`prefers-color-scheme` follows the terminal's, SPEC §8). They load
  // from ORIGIN, which nothing but this script serves.
  const pages = {};
  for (const s of SCALES) {
    const ctx = await browser.newContext({ deviceScaleFactor: s, colorScheme: "dark", viewport: { width: 800, height: 100 } });
    await ctx.route(`${ORIGIN}/**`, (route) => {
      const hit = docs.get(new URL(route.request().url()).pathname);
      return hit ? route.fulfill({ status: 200, contentType: hit[0], body: hit[1] }) : route.fulfill({ status: 404 });
    });
    pages[s] = [];
    for (let w = 0; w < a.jobs; w++) pages[s].push(await ctx.newPage());
  }
  const sheetCtx = await browser.newContext({ deviceScaleFactor: 1 });
  const sheetPages = [];
  for (let w = 0; w < a.jobs; w++) sheetPages.push(await sheetCtx.newPage());
  const diffs = diffWorkers(a.jobs);

  await pool(cases, a.jobs, async (c, w) => {
    const sc = { scale: c.scale, cell: cell(c.scale) };
    const page = pages[c.scale][w];
    const blitzPNG = path.join(work, c.id + ".blitz.png");
    const chromePNG = path.join(work, c.id + ".chromium.png");
    const diffPNG = path.join(work, c.id + ".diff.png");
    const base = ["render", c.html, "-o", blitzPNG, "--cols", String(c.cols), "--scale", String(c.scale), "--font", FONT];
    const url = serve(c.id, c.html, c.scale);
    const [, rowsC] = await Promise.all([run(a.hotty, [...base, "--rows", "auto"]), chromiumLoad(page, url, sc, c.cols)]);
    const rowsB = Math.round(fs.readFileSync(blitzPNG).readUInt32BE(20) / sc.cell.h);
    // Rows that differ are layouts that differ: both are drawn at the
    // taller, and the diff shows where.
    const rows = Math.max(rowsB, rowsC);
    if (rows !== rowsB) await run(a.hotty, [...base, "--rows", String(rows)]);
    await chromiumShot(page, sc, c.cols, rows, chromePNG);
    c.uneven = await page.evaluate(() => {
      const groups = {};
      for (const e of document.querySelectorAll("[data-same]")) (groups[e.dataset.same] ??= []).push(e.getBoundingClientRect().height);
      return Object.entries(groups)
        .filter(([, h]) => Math.max(...h) - Math.min(...h) > 0.01)
        .map(([k, h]) => `${k}: ${h.map((x) => x.toFixed(2)).join(", ")} CSS px`);
    });
    const d = await diffs.run(w, { blitz: blitzPNG, chromium: chromePNG, out: diffPNG, scale: c.scale }).catch((e) => {
      throw new Error(`${c.id}: ${e.message}`);
    });
    Object.assign(c, d, { rowsBlitz: rowsB, rowsChromium: rowsC, chromium: chromiumName, pass: d.area <= THRESHOLD });
    await sheet(sheetPages[w], path.join(out, c.id + ".png"), c, [blitzPNG, chromePNG, diffPNG]);
    process.stdout.write(`${c.pass ? "pass" : "FAIL"} ${c.area.toFixed(1).padStart(7)}  ${c.share.toFixed(3).padStart(6)}%  moved ${c.moved.toFixed(1).padStart(4)}%  ${c.id}\n`);
  });
  await diffs.close();
  await browser.close();

  const secs = ((Date.now() - t0) / 1000).toFixed(0);
  const failed = cases.filter((c) => !c.pass);
  const known = cases.filter((c) => c.note);
  const uneven = cases.filter((c) => c.uneven.length).map((c) => `uneven: ${c.id}: ${c.uneven.join("; ")}`);
  // Against the baseline: what it doesn't know, what got worse, and what
  // is to come off it.
  let against = null;
  if (a.baseline) {
    const base = readBaseline(a.baseline);
    against = [];
    for (const c of cases) {
      const b = base.get(c.id);
      if (b === undefined) {
        if (!c.pass) against.push(`new: ${c.id} fails, ${c.area.toFixed(1)}`);
      } else if (c.pass) {
        against.push(`fixed: ${c.id} passes now, ${c.area.toFixed(1)}: take it off the baseline`);
      } else if (c.area > b + THRESHOLD) {
        against.push(`worse: ${c.id}, ${b.toFixed(1)} on the baseline, ${c.area.toFixed(1)} now`);
      }
    }
    if (!a.only)
      for (const id of base.keys())
        if (!cases.some((c) => c.id === id)) against.push(`gone: ${id} is no case now: take it off the baseline`);
  }
  const lines = [
    "# F5 gate: hotty-blitz next to Chromium",
    "",
    `${new Date().toISOString().slice(0, 16).replace("T", " ")} UTC: \`hotty\` built ${hottyBuilt} UTC (checkout at \`${hottyRev}\`), ${chromiumName}. ` +
      `${cases.length} cases, ${failed.length} over the threshold, in ${secs} s.`,
    "",
    `A pixel differs when a channel lies more than ${LEVELS} levels (and ${RAMP * 100}% of an edge's contrast) ` +
      `outside what the other engine has within ${RADIUS} device px of it, after each cell's content may move ` +
      `up to ${SHIFT_CSS} CSS px; it counts when a neighbour differs too. The score is the area that differs, in ` +
      `CSS px²; over **${THRESHOLD}** fails. scripts/gate/diff.cjs is the metric, and gate.cjs says how it was calibrated.`,
    "",
    "Each sheet is Blitz | Chromium | diff (red: counts; orange: a speck that doesn't; blue: matched only moved; " +
      "yellow: within tolerance). Rows B/C: what each engine lays the story out to with rows=auto (when they differ, " +
      "both are drawn at the taller). Moved: the share of the image that matched only moved. " +
      `Wrong in Chromium too (our CSS, not a Blitz gap): ${known.length ? known.map((c) => c.id).join(", ") : "none noted"}.`,
    "",
    ...(uneven.length ? [`Boxes to be one height that aren't (data-same), in Chromium:`, ...uneven.map((x) => `- ${x}`), ""] : []),
    ...(against
      ? [
          `Against the baseline (\`${a.baseline}\`): ` +
            (against.length ? `${against.length} to look at:` : "every failure is a known one, none worse."),
          ...against.map((x) => `- ${x}`),
          "",
        ]
      : []),
    "| case | rows B/C | moved | share | score (CSS px²) | result | note |",
    "| --- | --- | ---: | ---: | ---: | --- | --- |",
    ...cases
      .slice()
      .sort((x, y) => y.area - x.area || x.id.localeCompare(y.id))
      .map(
        (c) =>
          `| [${c.id}](${c.id}.png) | ${c.rowsBlitz}/${c.rowsChromium} | ${c.moved.toFixed(1)}% | ${c.share.toFixed(3)}% | ${c.area.toFixed(1)} | ${c.pass ? "pass" : "**FAIL**"} | ${c.note} |`,
      ),
    "",
  ];
  fs.writeFileSync(path.join(out, "report.md"), lines.join("\n"));
  if (a["write-baseline"])
    writeBaseline(a["write-baseline"], failed, `hotty built ${hottyBuilt} UTC (checkout at ${hottyRev}), ${chromiumName}`);
  console.log(`${cases.length} cases, ${failed.length} failed, ${secs} s: ${path.join(a.out, "report.md")}`);
  for (const x of uneven) console.log(x);
  if (against) {
    for (const x of against) console.log(x);
    console.log(`against the baseline: ${against.length ? `${against.length} to look at` : "as known"}`);
    process.exit(against.length || uneven.length ? 1 : 0);
  }
  process.exit(failed.length || uneven.length ? 1 : 0);
})().catch((e) => {
  console.error(e);
  process.exit(2);
});
