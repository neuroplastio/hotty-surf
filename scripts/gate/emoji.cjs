// hotty-blitz 0.7.7+ bundles OpenMoji ahead of the system's emoji font.
// Give Chromium the same fallback, after named text faces and before
// generic families that would otherwise pick the machine's emoji font.
"use strict";
const fs = require("fs");
const path = require("path");

const FILE = "OpenMoji-color-glyf_colr_0.woff2";
const URL = `/fonts/${FILE}`;
const CSS = `@font-face { font-family: "OpenMoji"; src: url(${URL}) format("woff2"); }`;

function serveEmoji(fonts, served) {
  served.set(URL, ["font/woff2", fs.readFileSync(path.join(fonts, FILE))]);
}

async function emojiFallback(page) {
  await page.evaluate(() => {
    const families = [...document.querySelectorAll("*")].map((el) => [el, getComputedStyle(el).fontFamily]);
    for (const [el, family] of families) {
      const generic = /(^|,\s*)(serif|sans-serif|monospace|system-ui|emoji)(?=,|$)/;
      el.style.fontFamily = generic.test(family)
        ? family.replace(generic, '$1"OpenMoji", $2')
        : `${family}, "OpenMoji"`;
    }
  });
}

module.exports = { CSS, serveEmoji, emojiFallback };
