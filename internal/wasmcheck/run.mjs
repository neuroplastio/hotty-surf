// node run.mjs <wasm_exec.js> <module.wasm> <events.json> <out>: runs an
// app built for js on hotty-surf's page glue (globalThis.hottySurf), sends
// each event and waits for the writes to settle, and writes the stream.
import fs from 'node:fs';
import { createRequire } from 'node:module';
const [execJS, wasmPath, evPath, outPath] = process.argv.slice(2);
createRequire(import.meta.url)(execJS);
const chunks = [];
let last = performance.now();
globalThis.hottySurf = { cols: 160, rows: 50, write: (u8) => { chunks.push(Buffer.from(u8)); last = performance.now(); } };
const go = new Go();
const { instance } = await WebAssembly.instantiate(fs.readFileSync(wasmPath), go.importObject);
go.run(instance);
const wait = (ms) => new Promise((r) => setTimeout(r, ms));
const settle = async () => { do { await wait(20); } while (performance.now() - last < 20); };
while (!globalThis.hottySurf.input) await wait(5);
await settle();
const t0 = performance.now();
for (const e of JSON.parse(fs.readFileSync(evPath, 'utf8'))) {
  globalThis.hottySurf.input(new TextEncoder().encode(e));
  await settle();
}
fs.writeFileSync(outPath, Buffer.concat(chunks));
console.log(JSON.stringify({ writes: chunks.length, bytes: Buffer.concat(chunks).length }));
globalThis.hottySurf.quit();
await wait(50);
process.exit(0);
