// The page `hotty-surf build -wasm` writes, in Chromium: hello's, served
// as any static server serves it. The app shows, takes typing and a click,
// and draws them.
//
//   node scripts/browser/release.mjs <dist> [shot.png]
import http from "node:http";
import fs from "node:fs";
import path from "node:path";
import { chromium } from "playwright-core";
import { surface, until } from "./lib.mjs";

const [dist, shot] = process.argv.slice(2);
const types = { ".html": "text/html", ".js": "text/javascript", ".mjs": "text/javascript", ".css": "text/css", ".wasm": "application/wasm" };
const server = http.createServer((req, res) => {
  const name = decodeURIComponent(new URL(req.url, "http://x").pathname.slice(1)) || "index.html";
  const file = path.join(dist, path.normalize(name));
  if (!file.startsWith(path.resolve(dist)) || !fs.existsSync(file)) {
    res.writeHead(404).end();
    return;
  }
  res.writeHead(200, { "Content-Type": types[path.extname(file)] ?? "text/plain" });
  fs.createReadStream(file).pipe(res);
});
await new Promise((r) => server.listen(0, "127.0.0.1", r));
const url = `http://127.0.0.1:${server.address().port}/`;

const browser = await chromium.launch();
try {
  const page = await browser.newPage({ viewport: { width: 1000, height: 600 } });
  const errors = [];
  page.on("pageerror", (e) => errors.push(String(e)));
  await page.goto(url);
  const f = await surface(page, "Hello, world!");
  await f.getByPlaceholder("Your name").click();
  await page.keyboard.type("Ada");
  await until(async () => (await f.locator("h1").textContent()).includes("Hello, Ada!"), "the name typed");
  await f.getByRole("button", { name: "Wave" }).click();
  await until(async () => (await f.getByText(/Waved 1 times/).count()) > 0, "the wave");
  if (shot) await page.screenshot({ path: shot });
  if (errors.length) throw new Error("page errors: " + errors.join("; "));
  console.log("ok: typed, clicked, drawn");
} finally {
  await browser.close();
  server.close();
}
