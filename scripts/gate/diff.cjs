// The gate's metric: how much of what Blitz draws Chromium doesn't draw,
// or the other way round, once what two engines never agree on is set
// aside: antialiasing, a stroke's shade, a baseline or a rounding a pixel
// or two off. gate.cjs says how its knobs were calibrated.
"use strict";

// Per channel, the lowest and highest value in a (2r+1)² window round each
// pixel: a row pass, then a column pass.
function envelope(img, r) {
  const { width: w, height: h, data } = img;
  const n = w * h * 4;
  const rowMin = new Uint8Array(n), rowMax = new Uint8Array(n);
  for (let y = 0; y < h; y++)
    for (let x = 0; x < w; x++) {
      const x0 = Math.max(0, x - r), x1 = Math.min(w - 1, x + r);
      for (let c = 0; c < 3; c++) {
        let lo = 255, hi = 0;
        for (let xx = x0; xx <= x1; xx++) {
          const v = data[(y * w + xx) * 4 + c];
          if (v < lo) lo = v;
          if (v > hi) hi = v;
        }
        rowMin[(y * w + x) * 4 + c] = lo;
        rowMax[(y * w + x) * 4 + c] = hi;
      }
    }
  const min = new Uint8Array(n), max = new Uint8Array(n);
  for (let y = 0; y < h; y++) {
    const y0 = Math.max(0, y - r), y1 = Math.min(h - 1, y + r);
    for (let x = 0; x < w; x++)
      for (let c = 0; c < 3; c++) {
        let lo = 255, hi = 0;
        for (let yy = y0; yy <= y1; yy++) {
          const i = (yy * w + x) * 4 + c;
          if (rowMin[i] < lo) lo = rowMin[i];
          if (rowMax[i] > hi) hi = rowMax[i];
        }
        min[(y * w + x) * 4 + c] = lo;
        max[(y * w + x) * 4 + c] = hi;
      }
  }
  return { min, max };
}

// Does pixel p of img lie within env at pixel q? Within `levels`, plus
// `ramp` of the range where the range is wide: across an edge one engine's
// strokes are a shade darker than the other's, so a stroke one pixel wide
// is a colour the other engine has nowhere near it.
function within(img, p, env, q, levels, ramp) {
  for (let c = 0; c < 3; c++) {
    const i = q * 4 + c, v = img.data[p * 4 + c];
    const lo = env.min[i], hi = env.max[i], t = levels + ramp * (hi - lo);
    if (v < lo - t || v > hi + t) return false;
  }
  return true;
}

// diff(a, b, o) compares two images of one size.
//
//  1. A pixel of one image matches when each of its channels lies within
//     `levels` (and `ramp`, across an edge) of the range the other image
//     has within `radius` px of it, tested both ways. A glyph antialiased
//     another way, or an edge part of a pixel off, lies between the
//     colours on either side of it; a border that isn't there, or a fill
//     of another colour, doesn't.
//  2. Each tile (a cell, in device px) is tried at every offset up to
//     `shift` px, as a whole, and keeps the one with fewest pixels that
//     don't match: a run of text a CSS px lower, or boxes a layout rounded
//     to whole pixels, match; a box in the wrong place, a missing one, a
//     wrong colour, a glyph that only one draws still don't. Then every
//     pixel of each image has to match where it is, at its tile's offset
//     or at a neighbouring tile's, so that a box's edge and the text a
//     pixel above it, in one tile, can each match.
//  3. A pixel that matches nowhere counts when one of its 8 neighbours
//     doesn't either: a lone speck at a glyph's edge doesn't, a line one
//     pixel wide does.
//
// It returns the counted pixels, their share of the image in % (`score`),
// the share that matched only at an offset (`moved`), and the diff image.
function diff(a, b, o) {
  const { width: w, height: h } = a;
  const ea = envelope(a, o.radius), eb = envelope(b, o.radius);
  const { levels, ramp } = o;
  // Does (x, y) match, with b moved by (dx, dy)?
  const match = (x, y, dx, dy) => {
    const xx = x + dx, yy = y + dy;
    if (xx < 0 || yy < 0 || xx >= w || yy >= h) return false;
    const p = y * w + x, q = yy * w + xx;
    return within(a, p, eb, q, levels, ramp) && within(b, q, ea, p, levels, ramp);
  };
  const tw = o.tile.w, th = o.tile.h;
  const cols = Math.ceil(w / tw), rows = Math.ceil(h / th);
  const best = new Array(cols * rows);
  for (let r = 0; r < rows; r++)
    for (let c = 0; c < cols; c++) {
      const x0 = c * tw, y0 = r * th, x1 = Math.min(w, x0 + tw), y1 = Math.min(h, y0 + th);
      const misses = (dx, dy, limit) => {
        let n = 0;
        for (let y = y0; y < y1; y++)
          for (let x = x0; x < x1; x++) if (!match(x, y, dx, dy) && ++n >= limit) return n;
        return n;
      };
      let t = { dx: 0, dy: 0, n: misses(0, 0, Infinity) };
      for (let d = 1; d <= o.shift && t.n > 0; d++)
        for (let dy = -d; dy <= d; dy++)
          for (let dx = -d; dx <= d; dx += Math.abs(dy) === d ? 1 : 2 * d) {
            const n = misses(dx, dy, t.n);
            if (n < t.n) t = { dx, dy, n };
          }
      best[r * cols + c] = t;
    }

  // Every pixel of each image has to be explained by the other, at one of
  // the offsets near it: tested one image at a time, so that a glyph only
  // one of them draws can't hide behind pixels the other has elsewhere.
  const state = new Uint8Array(w * h); // 0 same, 1 within, 2 moved, 3 differs
  const explained = (img, p, env, x, y, sign) => {
    if (within(img, p, env, p, levels, ramp)) return 0;
    const c = Math.floor(x / tw), r = Math.floor(y / th);
    for (let rr = Math.max(0, r - 1); rr <= Math.min(rows - 1, r + 1); rr++)
      for (let cc = Math.max(0, c - 1); cc <= Math.min(cols - 1, c + 1); cc++) {
        const t = best[rr * cols + cc];
        if (!t.dx && !t.dy) continue;
        const xx = x + sign * t.dx, yy = y + sign * t.dy;
        if (xx >= 0 && yy >= 0 && xx < w && yy < h && within(img, p, env, yy * w + xx, levels, ramp)) return 1;
      }
    return -1;
  };
  let moved = 0;
  for (let y = 0; y < h; y++)
    for (let x = 0; x < w; x++) {
      const p = y * w + x;
      const ka = explained(a, p, eb, x, y, 1), kb = explained(b, p, ea, x, y, -1);
      if (ka < 0 || kb < 0) state[p] = 3;
      else if (ka > 0 || kb > 0) (state[p] = 2), moved++;
      else if (a.data[p * 4] !== b.data[p * 4] || a.data[p * 4 + 1] !== b.data[p * 4 + 1] || a.data[p * 4 + 2] !== b.data[p * 4 + 2])
        state[p] = 1;
    }

  let counted = 0;
  const out = Buffer.alloc(w * h * 4);
  for (let y = 0; y < h; y++)
    for (let x = 0; x < w; x++) {
      const p = y * w + x;
      let kept = false;
      if (state[p] === 3)
        for (let dy = -1; dy <= 1 && !kept; dy++)
          for (let dx = -1; dx <= 1 && !kept; dx++) {
            if (!dx && !dy) continue;
            const xx = x + dx, yy = y + dy;
            if (xx >= 0 && yy >= 0 && xx < w && yy < h && state[yy * w + xx] === 3) kept = true;
          }
      // The diff image: Blitz faded; red counts, orange is a speck that
      // doesn't, blue matched only moved, yellow only within tolerance.
      const i = p * 4;
      const grey = 0.3 * a.data[i] + 0.59 * a.data[i + 1] + 0.11 * a.data[i + 2];
      const g = Math.round(255 - (255 - grey) * 0.15);
      let px = [g, g, g];
      if (kept) {
        counted++;
        px = [230, 0, 0];
      } else if (state[p] === 3) px = [255, 150, 0];
      else if (state[p] === 2) px = [150, 190, 255];
      else if (state[p] === 1) px = [Math.round(g * 0.97), Math.round(g * 0.92), 140];
      out[i] = px[0];
      out[i + 1] = px[1];
      out[i + 2] = px[2];
      out[i + 3] = 255;
    }
  return { score: (100 * counted) / (w * h), moved: (100 * moved) / (w * h), counted, data: out };
}

module.exports = { diff, envelope };
