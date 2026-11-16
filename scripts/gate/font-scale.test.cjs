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
      <span class="u-icon" id="icon">x</span><span class="u-glyph u-dot" id="dot"></span>
      <div class="u-item" id="plain">Plain</div><div class="u-item" id="with-button">Plain<button class="u-btn u-small u-bare">Reset</button></div></div>`);
    async function metrics(font, cell) {
      return page.evaluate(({font, cell}) => {
        document.documentElement.style.cssText = `font-size:${font}px;--hotty-cell-h:${cell}px;--hotty-cell-w:${cell / 2}px;--hotty-px:0.625px`;
        const row = getComputedStyle(document.querySelector('#row'));
        const sw = document.querySelector('#switch').getBoundingClientRect();
        const icon = document.querySelector('#icon').getBoundingClientRect();
        const dot = document.querySelector('#dot').getBoundingClientRect();
        if (Math.abs(dot.width - dot.height) > 0.02) throw new Error(`status dot is not round: ${dot.width} × ${dot.height}`);
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

test('region and window state select the current item fill', async () => {
  const browser = await chromium.launch({ headless: true });
  try {
    const page = await browser.newPage();
    const css = readFileSync(path.join(__dirname, '../../surfui/surf.css'), 'utf8');
    await page.setContent(`<style>${css}\n*::before{transition:none!important}</style><div id="root" class="u-surface"
      data-region="focused" data-window="focused"
      style="--u-transition:0ms;--u-selected:rgba(255, 255, 255, .04);--u-active:rgba(255, 255, 255, .1);--u-hover:rgba(255, 255, 255, .05)">
      <div class="u-list u-entries" tabindex="0"><div id="item" tabindex="0" class="u-item is-sel">Current</div></div>
      <input id="field" class="u-field"></div>`);
    await page.focus('.u-list');
    const state = () => page.evaluate(() => {
      const item = document.querySelector('#item');
      const style = getComputedStyle(item);
      return [style.backgroundColor, style.backgroundImage, getComputedStyle(item, '::before').opacity];
    });
    assert.deepEqual(await state(), ['rgba(255, 255, 255, 0.1)', 'none', '0']);
    await page.focus('#item');
    assert.equal(await page.locator('#item').evaluate(n => n.matches(':focus-visible')), true);
    assert.equal(await page.locator('#item').evaluate(n => getComputedStyle(n).boxShadow), 'none');
    await page.focus('#field');
    assert.notEqual(await page.locator('#field').evaluate(n => getComputedStyle(n).boxShadow), 'none');
    await page.focus('.u-list');
    await page.locator('#root').evaluate(n => n.setAttribute('data-window', 'unfocused'));
    assert.deepEqual(await state(), ['rgba(255, 255, 255, 0.04)', 'none', '0']);
    await page.hover('#item');
    let [color, image] = await state();
    assert.equal(color, 'rgba(255, 255, 255, 0.04)');
    assert.match(image, /linear-gradient\(rgba\(255, 255, 255, 0\.05\)/);
    await page.locator('#root').evaluate(n => n.setAttribute('data-window', 'focused'));
    [color, image] = await state();
    assert.equal(color, 'rgba(255, 255, 255, 0.1)');
    assert.match(image, /linear-gradient\(rgba\(255, 255, 255, 0\.05\)/);
    await page.locator('#root').evaluate(n => n.setAttribute('data-current-bar', 'on'));
    assert.equal((await state())[2], '1');
    await page.locator('#root').evaluate(n => n.setAttribute('data-window', 'unfocused'));
    assert.equal((await state())[2], '1');
    await page.locator('#root').evaluate(n => n.removeAttribute('data-current-bar'));
    assert.equal((await state())[2], '0');
    await page.locator('#root').evaluate(n => n.removeAttribute('data-region'));
    assert.equal((await state())[2], '1');
  } finally { await browser.close(); }
});
