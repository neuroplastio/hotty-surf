// The look loop speaks the theme vocabulary of each document producer.
const { test } = require('node:test');
const assert = require('node:assert/strict');
const { mkdtempSync, writeFileSync, readFileSync, rmSync } = require('node:fs');
const { tmpdir } = require('node:os');
const path = require('node:path');
const { spawnSync } = require('node:child_process');

test('passes themes in the producer vocabulary', () => {
  const dir = mkdtempSync(path.join(tmpdir(), 'look-theme-'));
  try {
    const producer = path.join(dir, 'producer');
    const record = path.join(dir, 'theme');
    writeFileSync(producer, `#!/usr/bin/env node
const fs = require('node:fs');
const args = process.argv.slice(2);
if (args[0] === '-list') console.log('example\\t10\\t2');
else { fs.writeFileSync(process.env.LOOK_TEST_RECORD, args[args.indexOf('-theme') + 1]); process.exit(42); }
`, { mode: 0o755 });
    for (const [kind, theme, expected] of [
      ['gallery', 'dark', 'dark'], ['gallery', 'light', 'light'],
      ['stories', 'dark', 'graphite'], ['stories', 'light', 'stone'],
    ]) {
      rmSync(record, { force: true });
      const result = spawnSync(process.execPath, [path.join(__dirname, 'look.cjs'), 'story', 'example',
        `--${kind}`, producer, '--theme', theme, '--hotty', producer, '--out', dir],
      { env: { ...process.env, LOOK_TEST_RECORD: record }, encoding: 'utf8' });
      assert.equal(result.status, 1, result.stderr);
      assert.match(result.stderr, /Command failed/);
      assert.equal(readFileSync(record, 'utf8'), expected, `${kind}: ${theme}`);
    }
  } finally {
    rmSync(dir, { recursive: true, force: true });
  }
});
