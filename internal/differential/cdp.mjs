// Drives Basecoat's own scripts in headless Chromium over the Chrome
// DevTools Protocol (a bare WebSocket, no Playwright or Puppeteer): for each
// case of a component's manifest (written by ./record), opens its page,
// does its steps as real input (Input.dispatchMouseEvent at the element's
// centre, Input.dispatchKeyEvent), and after the load and after every step
// records the root's outerHTML, the focused element's id (or tag) and the
// events the page heard. Adapted from the spike's select.js
// harness.
//
//   node cdp.mjs <port> <manifest.json> <pages dir> <out.json>
import { readFileSync, writeFileSync } from 'node:fs';

const [port, manifestPath, pages, outPath] = process.argv.slice(2);
const spec = JSON.parse(readFileSync(manifestPath, 'utf8'));

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

async function target() {
  for (let i = 0; i < 50; i++) {
    try {
      const list = await (await fetch(`http://127.0.0.1:${port}/json/list`)).json();
      const page = list.find((t) => t.type === 'page');
      if (page) return page.webSocketDebuggerUrl;
    } catch (_) { /* not up yet */ }
    await sleep(200);
  }
  throw new Error('no page target');
}

const ws = new WebSocket(await target());
await new Promise((r, j) => { ws.onopen = r; ws.onerror = j; });
let next = 0;
const pending = new Map();
const waiters = [];
ws.onmessage = (m) => {
  const msg = JSON.parse(m.data);
  if (msg.id !== undefined) {
    const p = pending.get(msg.id);
    pending.delete(msg.id);
    if (msg.error) p.reject(new Error(JSON.stringify(msg.error)));
    else p.resolve(msg.result);
  } else {
    for (let i = waiters.length - 1; i >= 0; i--) {
      if (waiters[i].method === msg.method) { waiters[i].resolve(msg.params); waiters.splice(i, 1); }
    }
  }
};
const send = (method, params = {}) => new Promise((resolve, reject) => {
  const id = ++next;
  pending.set(id, { resolve, reject });
  ws.send(JSON.stringify({ id, method, params }));
});
const once = (method) => new Promise((resolve) => waiters.push({ method, resolve }));
const evaluate = async (expression) => {
  const r = await send('Runtime.evaluate', { expression, returnByValue: true, awaitPromise: true });
  if (r.exceptionDetails) throw new Error(JSON.stringify(r.exceptionDetails));
  return r.result.value;
};

// hotty's key names (SPEC §10.4) as DevTools key events.
const KEYS = {
  arrow_down: { key: 'ArrowDown', code: 'ArrowDown', windowsVirtualKeyCode: 40 },
  arrow_up: { key: 'ArrowUp', code: 'ArrowUp', windowsVirtualKeyCode: 38 },
  arrow_left: { key: 'ArrowLeft', code: 'ArrowLeft', windowsVirtualKeyCode: 37 },
  arrow_right: { key: 'ArrowRight', code: 'ArrowRight', windowsVirtualKeyCode: 39 },
  home: { key: 'Home', code: 'Home', windowsVirtualKeyCode: 36 },
  end: { key: 'End', code: 'End', windowsVirtualKeyCode: 35 },
  escape: { key: 'Escape', code: 'Escape', windowsVirtualKeyCode: 27 },
  tab: { key: 'Tab', code: 'Tab', windowsVirtualKeyCode: 9 },
  'shift+tab': { key: 'Tab', code: 'Tab', windowsVirtualKeyCode: 9, modifiers: 8 },
  enter: { key: 'Enter', code: 'Enter', windowsVirtualKeyCode: 13, text: '\r' },
  space: { key: ' ', code: 'Space', windowsVirtualKeyCode: 32, text: ' ' },
};

async function key(name) {
  const k = KEYS[name];
  if (!k) throw new Error(`no key ${name}`);
  await send('Input.dispatchKeyEvent', { type: k.text ? 'keyDown' : 'rawKeyDown', ...k });
  await send('Input.dispatchKeyEvent', { type: 'keyUp', ...k, text: undefined });
}

// settle waits for the page's finite animations (a toast sliding in), so
// that a click lands where the element was measured, at most a second.
const settle = () => evaluate(`Promise.race([
  Promise.all(document.getAnimations()
    .filter(a => a.effect && a.effect.getComputedTiming().iterations !== Infinity)
    .map(a => a.finished.catch(() => {}))),
  new Promise(r => setTimeout(r, 1000)),
]).then(() => true)`);

async function click(id, sel) {
  await settle();
  const box = await evaluate(`(() => {
    const el = ${sel ? `document.querySelector(${JSON.stringify(sel)})` : `document.getElementById(${JSON.stringify(id)})`};
    if (!el) return null;
    const r = el.getBoundingClientRect();
    return { x: r.left + r.width / 2, y: r.top + r.height / 2, w: r.width, h: r.height };
  })()`);
  if (!box || box.w === 0 || box.h === 0) return `not shown: ${id || sel}`;
  const at = { x: box.x, y: box.y };
  await send('Input.dispatchMouseEvent', { type: 'mouseMoved', ...at });
  await send('Input.dispatchMouseEvent', { type: 'mousePressed', ...at, button: 'left', buttons: 1, clickCount: 1 });
  await send('Input.dispatchMouseEvent', { type: 'mouseReleased', ...at, button: 'left', buttons: 0, clickCount: 1 });
  return '';
}

const SNAP = (sel) => `new Promise(r => requestAnimationFrame(() => requestAnimationFrame(() => r({
  html: document.querySelector(${JSON.stringify(sel)}).outerHTML,
  focus: document.activeElement && document.activeElement !== document.body ? (document.activeElement.id || document.activeElement.tagName) : '',
  events: window.__events.splice(0),
}))))`;

await send('Page.enable');
await send('Runtime.enable');
const out = {};
for (const c of spec.cases) {
  const loaded = once('Page.loadEventFired');
  await send('Page.navigate', { url: `file://${pages}/${c.name}.html` });
  await loaded;
  // Park the pointer away from everything, as a fresh page would have it.
  await send('Input.dispatchMouseEvent', { type: 'mouseMoved', x: 1000, y: 700 });
  if (spec.ready) {
    for (let i = 0; i < 50 && !(await evaluate(`!!document.querySelector(${JSON.stringify(spec.ready)})`)); i++) await sleep(20);
  }
  await settle();
  const snaps = [{ step: 'load', ...(await evaluate(SNAP(c.root))) }];
  for (const s of c.steps) {
    let note = '';
    if (s.click || s.select) note = await click(s.click, s.select);
    else await key(s.key);
    await sleep(20);
    snaps.push({ step: s.click || s.select ? `click ${s.click || s.select}` : `key ${s.key}`, note, ...(await evaluate(SNAP(c.root))) });
  }
  out[c.name] = snaps;
}
writeFileSync(outPath, JSON.stringify(out, null, 1) + '\n');
console.log(`${Object.keys(out).length} cases → ${outPath}`);
ws.close();
