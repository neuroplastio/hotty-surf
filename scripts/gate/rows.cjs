// The rows gate: surfui.Rows against the host's own layout. Every story,
// at every density, in two themes, at 1.6× and 2× (and at a smaller root
// font), is laid out by Chromium and by hotty-blitz as `r=auto` does, and
// the rows it takes are compared with what surfui.Rows estimates from the
// tree and the cell size alone (stories -rows). It writes
// <out>/report.md and exits 1 where an estimate is further than --tol
// rows from Chromium, or --tol-blitz from Blitz.
//
//   node scripts/gate/rows.cjs --hotty <hotty> --fonts <fonts/> --stories <stories> --out <dir>
//        [--tol 1] [--tol-blitz 1] [--only <substr>] [--jobs N] [--measured <file>] [--known <file>]
//
// --known is the stories that are off for a reason that is no estimate's to
// fix: `<story> <density> <reason>` lines, each for every theme and scale,
// judged as legible.cjs's list is: one that no longer fails comes off it.
//
// The measures (both engines') are written to <out>/measured.tsv; --measured
// reads them instead of laying anything out, to compare the estimate as the
// code changes without Chromium.
//
// Chromium's count is gate.cjs's (what Blitz's Surface::laid_out_rows
// does: laid out one pixel tall, down to the root's bottom). Blitz's is
// its image's height in rows. The root font size varies in Chromium
// alone (hotty render has no flag for it): font-scale.test.cjs's
// reasoning, that Compact and Roomy follow the host's root font.
"use strict";
const fs = require("fs");
const os = require("os");
const path = require("path");
const { execFileSync, execFile } = require("child_process");

const THEMES = ["graphite", "paper"];
const DENSITIES = ["dense", "compact", "roomy"];
// scale, root font size (17 is hotty's), blitz: whether hotty render can draw it
const SCALES = [
  { scale: 1.6, rem: 17, blitz: true },
  { scale: 2, rem: 17, blitz: true },
  { scale: 1.6, rem: 13, blitz: false },
];
const FONT = "AdwaitaMono Nerd Font Propo";
const FONT_FILE = "AdwaitaMonoNerdFontPropo";
const ORIGIN = "https://rows.invalid";

const cell = (s) => ({ w: Math.round(10 * s), h: Math.round(21 * s) });

function args() {
  const a = { jobs: Math.max(2, Math.min(6, os.cpus().length >> 2)), tol: 1, "tol-blitz": 1 };
  const v = process.argv.slice(2);
  for (let i = 0; i < v.length; i += 2) a[v[i].replace(/^--/, "")] = v[i + 1];
  for (const k of ["hotty", "fonts", "stories", "out"]) if (!a[k]) throw new Error(`rows: --${k} is needed`);
  a.jobs = Number(a.jobs);
  a.tol = Number(a.tol);
  a["tol-blitz"] = Number(a["tol-blitz"]);
  return a;
}

const run = (cmd, argv) =>
  new Promise((resolve, reject) =>
    execFile(cmd, argv, { maxBuffer: 1 << 26 }, (err, stdout, stderr) =>
      err ? reject(new Error(`${cmd} ${argv.join(" ")}: ${stderr || err.message}`)) : resolve(stdout),
    ),
  );

async function pool(items, n, fn) {
  let next = 0;
  await Promise.all(Array.from({ length: Math.min(n, items.length) }, async (_, w) => { while (next < items.length) await fn(items[next++], w); }));
}

(async () => {
  const { chromium } = require("playwright");
  const t0 = Date.now();
  const a = args();
  const out = path.resolve(a.out);
  const work = path.join(out, "work");
  if (!a.measured) fs.rmSync(out, { recursive: true, force: true });
  fs.mkdirSync(work, { recursive: true });

  const stories = execFileSync(a.stories, ["-list"], { encoding: "utf8" }).trim().split("\n").map((l) => {
    const [name, cols] = l.split("\t");
    return { name, file: name.replaceAll("/", "-"), cols: Number(cols) };
  });
  for (const t of THEMES)
    for (const d of DENSITIES) execFileSync(a.stories, ["-out", path.join(work, `${t}-${d}`), "-theme", t, "-density", d]);

  const faces = [["Regular", 400, "normal"], ["Bold", 700, "normal"], ["Italic", 400, "italic"], ["BoldItalic", 700, "italic"]];
  const plex = [["Regular", 400], ["Medium", 500], ["SemiBold", 600]];
  const fontCSS = faces
    .map(([f, w, st]) => `@font-face { font-family: "${FONT}"; src: url(/fonts/${FONT_FILE}-${f}.woff2) format("woff2"); font-weight: ${w}; font-style: ${st}; }`)
    .concat(plex.map(([f, w]) => `@font-face { font-family: "IBM Plex Sans"; src: url(/fonts/IBMPlexSans-${f}.woff2) format("woff2"); font-weight: ${w}; font-style: normal; }`))
    .join("\n");
  const served = new Map();
  for (const [f] of faces) served.set(`/fonts/${FONT_FILE}-${f}.woff2`, ["font/woff2", fs.readFileSync(path.join(a.fonts, `${FONT_FILE}-${f}.woff2`))]);
  for (const [f] of plex) served.set(`/fonts/IBMPlexSans-${f}.woff2`, ["font/woff2", fs.readFileSync(path.join(a.fonts, `IBMPlexSans-${f}.woff2`))]);

  // The cases, with surfui.Rows's answer for each.
  const cases = [];
  for (const sc of SCALES) {
    const c = cell(sc.scale);
    // At a smaller root font the host's cell shrinks with it.
    const k = sc.rem / 17;
    const cw = (c.w / sc.scale) * k, ch = (c.h / sc.scale) * k;
    const hostCSS = execFileSync(a.hotty, ["css", "--cell", `${c.w}x${c.h}`, "--scale", String(sc.scale), "--font", FONT], { encoding: "utf8" }) +
      (sc.rem === 17 ? "" : `:root { font-size: ${sc.rem}px; --hotty-cell-w: ${cw}px; --hotty-cell-h: ${ch}px; line-height: ${ch}px; }\n`);
    for (const t of THEMES)
      for (const d of DENSITIES) {
        const est = new Map(
          execFileSync(a.stories, ["-rows", "-density", d, "-theme", t, "-cellw", String(cw), "-cellh", String(ch), "-scale", String(sc.scale), "-rem", String(sc.rem)], { encoding: "utf8" })
            .trim().split("\n").map((l) => l.split("\t")),
        );
        for (const st of stories) {
          const id = `${st.file}-${t}-${d}-${sc.scale}${sc.rem === 17 ? "" : "-rem" + sc.rem}`;
          if (a.only && !id.includes(a.only)) continue;
          cases.push({ id, st, t, d, sc, cw, ch, c, est: Number(est.get(st.name)), hostCSS, html: path.join(work, `${t}-${d}`, st.file + ".html") });
        }
      }
  }

  const known = a.measured
    ? new Map(fs.readFileSync(a.measured, "utf8").trim().split("\n").map((l) => { const [id, c, b] = l.split("\t"); return [id, { chromium: Number(c), blitz: b === undefined || b === "" ? undefined : Number(b) }]; }))
    : null;
  const browser = known ? null : await chromium.launch({ args: ["--font-render-hinting=none"] });
  const chromiumName = known ? "measures of " + a.measured : `Chromium ${browser.version()}`;
  const ctxs = new Map();
  const page = async (sc, w) => {
    const key = sc.scale + ":" + w;
    if (!ctxs.has(key)) {
      const ctx = await browser.newContext({ deviceScaleFactor: sc.scale, colorScheme: "dark", viewport: { width: 800, height: 100 } });
      await ctx.route(`${ORIGIN}/**`, (route) => {
        const hit = served.get(new URL(route.request().url()).pathname);
        return hit ? route.fulfill({ status: 200, contentType: hit[0], body: hit[1] }) : route.fulfill({ status: 404 });
      });
      ctxs.set(key, await ctx.newPage());
    }
    return ctxs.get(key);
  };

  if (known) for (const c of cases) Object.assign(c, known.get(c.id) ?? { chromium: NaN });
  else await pool(cases, a.jobs, async (c, w) => {
    const doc = fs.readFileSync(c.html, "utf8");
    served.set(`/${c.id}.html`, ["text/html", doc.replace("<head>", `<head><style>${fontCSS}\n@layer hotty-host {\n${c.hostCSS}\n}</style>`)]);
    const p = await page(c.sc, w);
    await p.setViewportSize({ width: Math.ceil(((c.st.cols * c.c.w) / c.sc.scale) * (c.sc.rem / 17)), height: 1 });
    await p.goto(`${ORIGIN}/${c.id}.html`, { waitUntil: "load" });
    await p.evaluate(() => document.fonts.ready);
    const bottom = await p.evaluate(() => {
      const r = document.documentElement;
      const m = parseFloat(getComputedStyle(r).marginBottom) || 0;
      return Math.max(r.getBoundingClientRect().bottom + m, r.scrollHeight);
    });
    const px = Math.ceil(Math.max(0, bottom - 0.5) * c.sc.scale);
    c.chromium = Math.min(1000, Math.max(1, Math.ceil(px / Math.round(c.ch * c.sc.scale))));
    if (c.sc.blitz) {
      const png = path.join(work, c.id + ".png");
      await run(a.hotty, ["render", c.html, "-o", png, "--cols", String(c.st.cols), "--scale", String(c.sc.scale), "--font", FONT, "--rows", "auto"]);
      c.blitz = Math.round(fs.readFileSync(png).readUInt32BE(20) / c.c.h);
      fs.rmSync(png);
    }
  });
  if (browser) await browser.close();
  if (!known) fs.writeFileSync(path.join(out, "measured.tsv"), cases.map((c) => `${c.id}\t${c.chromium}\t${c.blitz ?? ""}`).join("\n") + "\n");

  const knownList = [];
  if (a.known)
    for (const line of fs.readFileSync(a.known, "utf8").split("\n")) {
      const t = line.trim();
      if (!t || t.startsWith("#")) continue;
      const [story, density, ...why] = t.split(/\s+/);
      knownList.push({ story, density, why: why.join(" "), hit: false });
    }
  const bad = [];
  const worst = { chromium: 0, blitz: 0 };
  const byDensity = {};
  for (const c of cases) {
    c.dc = c.est - c.chromium;
    c.db = c.blitz === undefined ? undefined : c.est - c.blitz;
    worst.chromium = Math.max(worst.chromium, Math.abs(c.dc));
    if (c.db !== undefined) worst.blitz = Math.max(worst.blitz, Math.abs(c.db));
    const s = (byDensity[c.d] ??= { n: 0, exact: 0, within: 0, max: 0 });
    s.n++;
    if (c.dc === 0) s.exact++;
    if (Math.abs(c.dc) <= a.tol) s.within++;
    s.max = Math.max(s.max, Math.abs(c.dc));
    const over = Math.abs(c.dc) > a.tol || (c.db !== undefined && Math.abs(c.db) > a["tol-blitz"]);
    const k = knownList.find((x) => x.story === c.st.name && x.density === c.d);
    if (over && k) {
      k.hit = true;
      c.known = k.why;
    } else if (over) bad.push(c);
  }
  const stale = knownList.filter((k) => !k.hit && cases.some((c) => c.st.name === k.story && c.d === k.density));
  const secs = ((Date.now() - t0) / 1000).toFixed(0);
  const lines = [
    "# Rows gate: surfui.Rows against the host's layout",
    "",
    `${new Date().toISOString().slice(0, 16).replace("T", " ")} UTC, ${chromiumName}, ${cases.length} cases in ${secs} s. ` +
      `Tolerance: ${a.tol} row from Chromium, ${a["tol-blitz"]} from Blitz. ` +
      `${bad.length} over it${knownList.length ? `, ${cases.filter((c) => c.known).length} known` : ""}. Worst: ${worst.chromium} rows from Chromium, ${worst.blitz} from Blitz.`,
    "",
    "| density | cases | exact (Chromium) | within tolerance | worst |",
    "| --- | ---: | ---: | ---: | ---: |",
    ...Object.entries(byDensity).map(([d, s]) => `| ${d} | ${s.n} | ${s.exact} | ${s.within} | ${s.max} |`),
    "",
    "| case | estimate | Chromium | Blitz | off by (Chromium) | known |",
    "| --- | ---: | ---: | ---: | ---: | --- |",
    ...cases.slice().sort((x, y) => Math.abs(y.dc) - Math.abs(x.dc) || x.id.localeCompare(y.id)).map((c) => `| ${c.id} | ${c.est} | ${c.chromium} | ${c.blitz ?? ""} | ${c.dc} | ${c.known ?? ""} |`),
    "",
  ];
  fs.writeFileSync(path.join(out, "report.md"), lines.join("\n"));
  for (const c of bad) console.log(`FAIL ${c.id}: estimate ${c.est}, Chromium ${c.chromium}, Blitz ${c.blitz ?? "-"}`);
  for (const [d, s] of Object.entries(byDensity)) console.log(`${d}: ${s.n} cases, ${s.exact} exact, ${s.within} within ${a.tol}, worst ${s.max}`);
  console.log(`${cases.length} cases, ${bad.length} over tolerance, ${secs} s: ${path.join(a.out, "report.md")}`);
  for (const k of stale) console.log(`fixed: ${k.story} ${k.density}: take it off ${a.known}`);
  process.exit(bad.length || stale.length ? 1 : 0);
})().catch((e) => {
  console.error(e);
  process.exit(2);
});
