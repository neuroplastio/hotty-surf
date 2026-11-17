// hotty-surf's page: xterm.js with xterm-addon-hotty, and the app, built
// for js/wasm, running in it. The app's run.App speaks to globalThis.hottySurf
// (run/wasm.go): the page gives it write, cols, rows and caps, and the app
// sets input, resize and quit on it. The bytes are the terminal stream's,
// both ways.
//
// Served by `hotty-surf serve` (<html data-dev>), the page also reloads the
// app as its files are saved: each build's app dials the reloader back
// over a WebSocket (dev/wire) as the instance page:<key>, and dev/events
// says when a new build is up (restart) and what the last save did
// (report). The page tells serve that it goes (dev/leave, its key as
// text). Where the app is, and at what size, the app says on its own wire
// (reload.Placed), for ctl shot.

import { Terminal } from "./xterm.mjs";
import { FitAddon } from "./addon-fit.mjs";
import { HottyAddon } from "./hotty-xterm.js";

const el = document.getElementById("term");
const term = new Terminal({
  fontFamily: '"JetBrains Mono", "DejaVu Sans Mono", monospace',
  fontSize: 15,
  cursorBlink: false,
  vtExtensions: { kittyKeyboard: true },
  theme: { background: "#1d1f21", foreground: "#c5c8c6" },
});
const fit = new FitAddon();
term.loadAddon(fit);
term.loadAddon(new HottyAddon());
term.open(el);
fit.fit();
new ResizeObserver(() => fit.fit()).observe(el);
term.focus();
// The app has the whole terminal, as run.App takes it in one: the
// alternate screen (no scrollback to move the surface), no cursor.
term.write("\x1b[?1049h\x1b[?25l");

const mac = /Mac|iPhone|iPad/.test(navigator.platform);
const dev = document.documentElement.hasAttribute("data-dev");
const glue = {
  write(bytes) {
    term.write(bytes);
  },
  cols: term.cols,
  rows: term.rows,
  caps: { mod: mac ? "super" : "ctrl" },
};
globalThis.hottySurf = glue;
const encoder = new TextEncoder();
// alive is the running app's: what it set on glue, and its wire, are called
// only while it runs. An app that exited (a dev build handing over) is
// gone, and a call into it would throw.
let alive = { ok: false };
term.onData((data) => alive.ok && glue.input?.(encoder.encode(data)));
term.onBinary((data) => alive.ok && glue.input?.(Uint8Array.from(data, (c) => c.charCodeAt(0))));
term.onResize(({ cols, rows }) => {
  glue.cols = cols;
  glue.rows = rows;
  if (alive.ok) glue.resize?.(cols, rows);
});
let running = 0; // the build whose app runs, 0 for none
let exits = 0; // how many apps exited

// start runs the app's wasm; the promise ends when the app does.
async function start(build) {
  running = build;
  const go = new Go();
  // Both wasm_exec.js set go.exited the moment the app exits, a microtask
  // before go.run's promise settles: a frame or a key in between would
  // call into the exited instance.
  let ended = false;
  // Go's wasm_exec.js leaves the timers the app scheduled (a time.After
  // still waiting) when it exits, and one firing later resumes the exited
  // instance, which throws. TinyGo's checks; Go's needs them cleared.
  const exit = go.exit;
  go.exit = (code) => {
    go._scheduledTimeouts?.forEach((t) => clearTimeout(t));
    go._scheduledTimeouts?.clear();
    exit.call(go, code);
  };
  const me = (alive = {
    get ok() {
      return !ended && !go.exited;
    },
  });
  const url = dev ? `app.wasm?build=${build}` : "app.wasm";
  const { instance } = await WebAssembly.instantiateStreaming(fetch(url), go.importObject);
  await go.run(instance);
  ended = true;
  exits++;
  glue.input = glue.resize = glue.quit = undefined;
  if (glue.dev?.socket) {
    glue.dev.socket.close();
    glue.dev.socket = null;
  }
  if (running === build) running = 0;
  if (!dev) term.write("\r\n\x1b[2m[the app exited]\x1b[0m\r\n");
}

if (!dev) {
  start(1).catch(fail);
} else {
  // The instance's key, the page's own: a reload of the page is a leave
  // and a new page, so a late leave can't end the next page's wire.
  const key = "page:" + Math.random().toString(36).slice(2, 10);
  // The app's end of the reload wire: run/dev_js.go opens it, and gets
  // the bytes the reloader sends through onData.
  glue.dev = {
    key,
    socket: null,
    open(onData, onClose) {
      const me = alive;
      const url = new URL("dev/wire", location.href);
      url.searchParams.set("key", key);
      url.protocol = location.protocol === "https:" ? "wss:" : "ws:";
      const ws = new WebSocket(url);
      ws.binaryType = "arraybuffer";
      ws.onmessage = (m) => me.ok && onData(new Uint8Array(m.data));
      ws.onclose = () => me.ok && onClose();
      glue.dev.socket = ws;
      return new Promise((resolve, reject) => {
        ws.onopen = () => resolve(ws);
        ws.onerror = () => reject(new Error("the reload wire didn't open"));
      });
    },
  };
  const report = document.getElementById("report");
  const events = new EventSource("dev/events");
  events.addEventListener("restart", (e) => {
    const build = Number(e.data);
    if (build !== running) start(build).catch(fail);
  });
  events.addEventListener("report", (e) => {
    report.textContent = e.data;
    report.style.display = e.data ? "block" : "none";
  });
  // A page that goes away (closed, reloaded) says so: a leave, not a
  // crash, and the reloader doesn't wait for it to come back. Its wire is
  // closing, so serve says it on the wire for it.
  addEventListener("pagehide", () => {
    navigator.sendBeacon("dev/leave", key);
    if (alive.ok) glue.quit?.();
  });
}

function fail(err) {
  term.write(`\r\n\x1b[31m${String(err)}\x1b[0m\r\n`);
}

// For tests: what the page is doing.
globalThis.hottySurfPage = { term, get running() { return running; }, get exits() { return exits; } };
