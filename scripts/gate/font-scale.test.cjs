// The same kit controls should follow the terminal's root font size.
const { test } = require('node:test');
const assert = require('node:assert/strict');
const { readFileSync } = require('node:fs');
const path = require('node:path');
const { chromium } = require('playwright');

test('compact controls scale with the terminal font', async () => {
  const browser = await chromium.launch({ headless: true });
  try {
    const page = await browser.newPage();
    const css = readFileSync(path.join(__dirname, '../../surfui/surf.css'), 'utf8');
    await page.setContent(`<style>${css}</style><div class="u-surface u-compact">
      <div class="u-nav"><div class="u-item" id="row">Example</div></div>
      <div class="u-switch" id="switch"><span class="u-knob"></span></div>
      <span class="u-icon" id="icon">x</span>
      <div class="u-item" id="plain">Plain</div><div class="u-item" id="with-button">Plain<button class="u-btn u-small u-bare">Reset</button></div></div>`);
    async function metrics(font, cell) {
      return page.evaluate(({font, cell}) => {
        document.documentElement.style.cssText = `font-size:${font}px;--hotty-cell-h:${cell}px;--hotty-cell-w:${cell / 2}px;--hotty-px:0.625px`;
        const row = getComputedStyle(document.querySelector('#row'));
        const sw = document.querySelector('#switch').getBoundingClientRect();
        const icon = document.querySelector('#icon').getBoundingClientRect();
        assertRows();
        function assertRows() {
          const a = document.querySelector('#plain').getBoundingClientRect().height;
          const b = document.querySelector('#with-button').getBoundingClientRect().height;
          if (Math.abs(a - b) > 0.02) throw new Error(`small bare button changes row height: ${a} vs ${b}`);
        }
        return [parseFloat(row.fontSize), parseFloat(row.paddingTop), sw.width, sw.height, icon.width];
      }, {font, cell});
    }
    const small = await metrics(12, 16);
    const base = await metrics(16, 16 * 4 / 3);
    for (let i = 0; i < base.length; i++) {
      assert.ok(Math.abs(small[i] / base[i] - 0.75) < 0.02,
        `metric ${i} did not scale: ${small[i]} vs ${base[i]}`);
    }
  } finally { await browser.close(); }
});
