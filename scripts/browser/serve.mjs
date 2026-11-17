// `hotty-surf serve` in Chromium: hello's page with live reload. The app
// shows; typing and a click change its state; a Go save loads the next build
// into the same terminal with the state kept; a css save restyles in place;
// a save that doesn't build shows its report over the page and leaves the
// app running. ctl reload and ctl shot answer from serve, and a tab that
// closes is a leave: no save waits for it or reports it.
//
//   node scripts/browser/serve.mjs <hotty-surf> <app dir> [shots dir]
import { spawn, execFileSync } from "node:child_process";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { chromium } from "playwright-core";
import { surface, until } from "./lib.mjs";

const [bin, dir, shots] = process.argv.slice(2);
const sock = path.join(fs.mkdtempSync(path.join(os.tmpdir(), "hotty-surf-serve-")), "ctl.sock");
const env = { ...process.env, HOTTY_SURF_SOCK: sock };
const serve = spawn(bin, ["serve", "-addr", "127.0.0.1:0", dir], { stdio: ["ignore", "pipe", "inherit"], env });
// ctl is `hotty-surf ctl …`: its exit code and output.
const ctl = (...args) => {
  try {
    return { code: 0, out: execFileSync(bin, ["ctl", ...args], { env, encoding: "utf8" }) };
  } catch (e) {
    return { code: e.status, out: String(e.stdout) + String(e.stderr) };
  }
};
let log = "";
const url = await new Promise((resolve, reject) => {
  serve.stdout.on("data", (b) => {
    log += b;
    const m = log.match(/http:\/\/\S+\//);
    if (m) resolve(m[0]);
  });
  serve.on("exit", (code) => reject(new Error(`serve exited ${code}: ${log}`)));
});
const edit = (file, from, to) => {
  const p = path.join(dir, file);
  const s = fs.readFileSync(p, "utf8");
  if (!s.includes(from)) throw new Error(`${file} has no ${from}`);
  fs.writeFileSync(p, s.replace(from, to));
};
const shot = async (page, name) => shots && (await page.screenshot({ path: path.join(shots, name) }));

const browser = await chromium.launch();
try {
  const page = await browser.newPage({ viewport: { width: 1000, height: 600 } });
  const errors = [];
  page.on("pageerror", (e) => errors.push(String(e.stack ?? e)));
  await page.goto(url);
  let f = await surface(page, "Hello, world!", 180000); // the first build
  await f.getByPlaceholder("Your name").click();
  await page.keyboard.type("Ada");
  await f.getByRole("button", { name: "Wave" }).click();
  await until(async () => (await f.getByText(/Waved 1 times/).count()) > 0, "the wave");
  await shot(page, "1-state.png");

  // A second tab, closed: a leave.
  const other = await browser.newPage({ viewport: { width: 800, height: 400 } });
  await other.goto(url);
  await surface(other, "Hello, world!", 60000);
  // Leaving the page fires pagehide, as closing a tab does (Playwright's
  // close runs no unload handlers).
  await other.goto("about:blank");
  await other.close();

  // A Go save (a text save is one in a browser): the next build, with the
  // state handed over.
  const exits = await page.evaluate(() => hottySurfPage.exits);
  // What the next build writes the terminal: its first frame is a delta.
  await page.evaluate(() => {
    const write = hottySurf.write;
    hottySurf.sent = [];
    hottySurf.write = (b) => {
      hottySurf.sent.push(new TextDecoder().decode(b.slice(0, 24)));
      return write(b);
    };
  });
  const saved = Date.now();
  edit("page.templ", "Waved {", "Waves so far: {");
  await until(async () => (await page.evaluate(() => hottySurfPage.exits)) > exits, "the old build handing over", 120000);
  f = await surface(page, "Waves so far: 1", 120000);
  const took = Date.now() - saved;
  const sent = await page.evaluate(() => hottySurf.sent);
  if (sent.some((c) => c.includes("a=doc")) || !sent.some((c) => c.includes("a=delta"))) {
    throw new Error("the next build's first frame isn't a delta: " + JSON.stringify(sent));
  }
  if (!(await f.locator("h1").textContent()).includes("Hello, Ada!")) throw new Error("the name wasn't kept");
  if ((await f.getByPlaceholder("Your name").inputValue()) !== "Ada") throw new Error("the field wasn't kept");
  await shot(page, "2-go-save.png");

  // ctl, answered by serve: the save's report (the closed tab no failure),
  // and the newest page's document.
  const rel = ctl("reload");
  if (rel.code !== 0 || !rel.out.startsWith("ok: go") || rel.out.includes("page:")) {
    throw new Error(`ctl reload after the closed tab: ${rel.code} ${rel.out}`);
  }
  const html = path.join(path.dirname(sock), "shot.html");
  const sh = ctl("shot", "-html", "-o", html);
  if (sh.code !== 0 || !fs.readFileSync(html, "utf8").includes("Waves so far: 1")) {
    throw new Error(`ctl shot: ${sh.code} ${sh.out}`);
  }

  // A css save: restyled in place, by the same build.
  const build = await page.evaluate(() => hottySurfPage.running);
  edit("style/app.css", "*/", "*/\nh1 { color: rgb(200, 30, 30); }");
  await until(async () => (await f.locator("h1").evaluate((e) => getComputedStyle(e).color)) === "rgb(200, 30, 30)", "the restyle", 60000);
  if ((await page.evaluate(() => hottySurfPage.running)) !== build) throw new Error("a css save loaded a new build");
  await shot(page, "3-css-save.png");

  // A save that doesn't build: the report over the page, the app as it was.
  edit("app.go", "func NewApp()", "func NewApp() { nope }\nfunc newApp()");
  await until(async () => (await page.locator("#report").isVisible()) && (await page.locator("#report").textContent()).includes("build failed"), "the report", 120000);
  await f.getByRole("button", { name: "Wave" }).click();
  await until(async () => (await f.getByText(/Waves so far: 2/).count()) > 0, "the app still running");
  await shot(page, "4-build-failed.png");
  edit("app.go", "func NewApp() { nope }\nfunc newApp()", "func NewApp()");
  await until(async () => !(await page.locator("#report").isVisible()), "the report gone", 120000);
  f = await surface(page, "Waves so far: 2", 120000);

  if (errors.length) throw new Error("page errors: " + errors.join("; "));
  console.log(`ok: shown, state, go save with state (save to screen ${took} ms, a delta), css save in place, build failure reported, ctl answered, a closed tab left`);
} finally {
  await browser.close();
  serve.kill("SIGTERM");
}
