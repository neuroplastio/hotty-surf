// src/hostcss.ts
var FONT_FAMILY = "AdwaitaMono Nerd Font Propo";
var SANS_FAMILY = "IBM Plex Sans";
var SANS_FACES = [
  ["IBMPlexSans-Regular.woff2", 400, "normal"],
  ["IBMPlexSans-Medium.woff2", 500, "normal"],
  ["IBMPlexSans-SemiBold.woff2", 600, "normal"]
];
var FONT_FACES = [
  ["AdwaitaMonoNerdFontPropo-Regular.woff2", 400, "normal"],
  ["AdwaitaMonoNerdFontPropo-Bold.woff2", 700, "normal"],
  ["AdwaitaMonoNerdFontPropo-Italic.woff2", 400, "italic"],
  ["AdwaitaMonoNerdFontPropo-BoldItalic.woff2", 700, "italic"]
];
function fontFaces(fonts) {
  if (!fonts) return [];
  const faces = [
    ...FONT_FACES.map(([file, weight, style2]) => [FONT_FAMILY, file, weight, style2]),
    ...SANS_FACES.map(([file, weight, style2]) => [SANS_FAMILY, file, weight, style2])
  ];
  return faces.map(
    ([family, file, weight, style2]) => `@font-face { font-family: "${family}"; src: url("${fonts}${file}") format("woff2"); font-weight: ${weight}; font-style: ${style2}; font-display: swap; }`
  );
}
var DEFAULT_FG = "#ffffff";
var DEFAULT_BG = "#000000";
var ANSI_KEYS = ["black", "red", "green", "yellow", "blue", "magenta", "cyan", "white", "brightBlack", "brightRed", "brightGreen", "brightYellow", "brightBlue", "brightMagenta", "brightCyan", "brightWhite"];
var DEFAULT_ANSI = ["#2e3436", "#cc0000", "#4e9a06", "#c4a000", "#3465a4", "#75507b", "#06989a", "#d3d7cf", "#555753", "#ef2929", "#8ae234", "#fce94f", "#729fcf", "#ad7fa8", "#34e2e2", "#eeeeec"];
function palette(theme) {
  const fg = theme?.foreground ?? DEFAULT_FG;
  const bg = theme?.background ?? DEFAULT_BG;
  const ansi = ANSI_KEYS.map((k, i) => theme?.[k] ?? DEFAULT_ANSI[i]);
  return { fg, bg, ansi, dark: luminance(bg) < 0.5 };
}
function boxBackground(p) {
  return `color-mix(in srgb, color-mix(in srgb, ${p.fg} 10%, ${p.bg}) 99.5%, transparent)`;
}
function hostCss(theme, m) {
  const p = palette(theme);
  const lines = [
    ...fontFaces(m.fonts ?? ""),
    "@layer hotty-host {",
    ":root {",
    `  color-scheme: ${p.dark ? "dark" : "light"};`,
    `  --hotty-fg: ${p.fg};`,
    `  --hotty-bg: ${p.bg};`,
    ...p.ansi.map((c, i) => `  --hotty-ansi-${i}: ${c};`),
    // The terminal's blue: plain blue is hard to read on a dark background.
    `  --hotty-accent: var(--hotty-ansi-${p.dark ? 12 : 4});`,
    `  --hotty-cell-w: ${m.cellW}px;`,
    `  --hotty-cell-h: ${m.cellH}px;`,
    `  --hotty-px: ${m.px}px;`,
    `  --hotty-font: ${fontStack(m.fontFamily, !!m.fonts)};`,
    "  font-family: var(--hotty-font);",
    `  font-size: ${m.fontSize}px;`,
    // An explicit line-height, so that `1rlh` is one terminal row.
    `  line-height: ${m.cellH}px;`,
    "  color: var(--hotty-fg);",
    "  background: var(--hotty-bg);",
    "  accent-color: var(--hotty-accent);",
    "}",
    "body { margin: 0; }",
    // Controls in the terminal's colours (SPEC §8), not the browser's: ansi-8
    // is the dim grey of borders, buttons, placeholders and what is disabled.
    // :where() keeps the base rule below the state rules after it, such as
    // :disabled.
    ":where(input:not([type=checkbox], [type=radio], [type=range]), textarea,",
    "       select, button) {",
    "  color: var(--hotty-fg);",
    "  background: var(--hotty-bg);",
    "  border: 1px solid var(--hotty-ansi-8);",
    "}",
    "button:enabled, input:is([type=button], [type=submit], [type=reset]):enabled {",
    "  background: var(--hotty-ansi-8);",
    "}",
    "::placeholder, :disabled { color: var(--hotty-ansi-8); }",
    // The ring is the controls' (SPEC §8): links, buttons, fields, selects,
    // a details element's summary, and an editing host, which is a text
    // field (§10.2). A focused container (an element with tabindex, a
    // scrolling region, a list) has none, not even the browser's own, which
    // its user-agent sheet gives every :focus-visible element.
    ":focus-visible { outline: none; }",
    ":is(a[href], button, input, select, textarea, summary, [contenteditable]):focus-visible {",
    "  outline: 1px solid var(--hotty-accent);",
    "}",
    ":any-link { color: var(--hotty-accent); }",
    "::selection { color: var(--hotty-bg); background: var(--hotty-accent); }",
    // An element that opts in to drags selects no text, whatever the
    // document's CSS (SPEC §9.1, §11): important in the host's layer, the
    // first, wins over every rule of the document's.
    "[data-on~=drag], [data-on~=drag] * { -webkit-user-select: none !important; user-select: none !important; }",
    // The rows a document needs are measured with no scrollbar of the root's
    // in the way (`r=auto`, `fit`, SPEC §5.2).
    `:root[${MEASURE}] { overflow: hidden !important; }`,
    "}"
  ];
  return lines.join("\n");
}
var MEASURE = "data-xterm-hotty-measure";
var CLIP = "data-xterm-hotty-clip";
var UNCLIPPED = "data-xterm-hotty-unclipped";
function scrollCss(axes, pageScrolls) {
  if (!axes) {
    return [
      "@layer hotty-host {",
      `:root { overflow: hidden; touch-action: ${rootTouchAction(axes, pageScrolls)}; }`,
      "* { scrollbar-width: none !important; }",
      "::-webkit-scrollbar { display: none !important; }",
      "}"
    ].join("\n");
  }
  const lines = ["@layer hotty-host {", `:root { touch-action: ${rootTouchAction(axes, pageScrolls)}; }`];
  if (axes !== 3) {
    const clipped = axes === 1 ? "overflow-x" : "overflow-y";
    lines.push(`:root, :root:not([${UNCLIPPED}]) [${CLIP}] { ${clipped}: hidden !important; }`);
  }
  lines.push("}");
  return lines.join("\n");
}
function rootTouchAction(axes, pageScrolls) {
  if (pageScrolls) return "manipulation";
  return axes === 3 ? "pan-x pan-y" : axes === 1 ? "pan-y" : axes === 2 ? "pan-x" : "none";
}
function fontStack(family, ours = false) {
  const f = family.trim().replace(/(^|,)\s*monospace\s*$/, "");
  return [f, ours ? `"${FONT_FAMILY}"` : "", "monospace"].filter(Boolean).join(", ");
}
function luminance(color) {
  let hex = color.trim().replace(/^#/, "");
  if (hex.length === 3) hex = hex.replace(/./g, (c) => c + c);
  if (!/^[0-9a-f]{6}/i.test(hex)) return 0;
  const [r, g, b] = [0, 2, 4].map((i) => {
    const c = parseInt(hex.slice(i, i + 2), 16) / 255;
    return c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4;
  });
  return 0.2126 * r + 0.7152 * g + 0.0722 * b;
}

// src/msgpack.ts
var Float = class {
  value;
  constructor(value) {
    this.value = value;
  }
};
var float = (value) => new Float(value);
var utf8 = new TextEncoder();
var Writer = class {
  buf = new Uint8Array(256);
  view = new DataView(this.buf.buffer);
  n = 0;
  /** `b` as msgpack: a view of the buffer, good until the next call. */
  body(b) {
    this.n = 0;
    this.value(b);
    return this.buf.subarray(0, this.n);
  }
  value(v) {
    if (typeof v === "string") this.str(v);
    else if (typeof v === "number") this.int(v);
    else if (typeof v === "boolean") this.byte(v ? 195 : 194);
    else if (v instanceof Float) this.float(v.value);
    else if (Array.isArray(v)) {
      this.head(144, 0, 220, 221, v.length);
      for (const x of v) {
        if (x == null) throw new TypeError("an array holds no nil");
        this.value(x);
      }
    } else {
      const proto = v === null || typeof v !== "object" ? void 0 : Object.getPrototypeOf(v);
      if (proto !== Object.prototype && proto !== null) throw new TypeError(`${String(v)} is no value a body holds`);
      const fields = Object.entries(v).filter((f) => f[1] != null);
      this.head(128, 0, 222, 223, fields.length);
      for (const [k, x] of fields) {
        if (!k.isWellFormed()) throw new TypeError(`the key ${JSON.stringify(k)} holds a lone surrogate`);
        this.str(k);
        this.value(x);
      }
    }
  }
  /** An int in its smallest form. Only whole numbers within 2^53 − 1 of
   * zero are ints (SPEC §3.3): anything else is a bug where it was made. */
  int(x) {
    if (!Number.isSafeInteger(x)) throw new RangeError(`${x} is not an int (a float goes in a Float)`);
    this.room(9);
    const d = this.view;
    if (x >= -32 && x < 128) this.buf[this.n++] = x & 255;
    else if (x >= 0) {
      if (x < 256) d.setUint8(this.tag(204, 1), x);
      else if (x < 65536) d.setUint16(this.tag(205, 2), x);
      else if (x < 4294967296) d.setUint32(this.tag(206, 4), x);
      else d.setBigUint64(this.tag(207, 8), BigInt(x));
    } else if (x >= -128) d.setInt8(this.tag(208, 1), x);
    else if (x >= -32768) d.setInt16(this.tag(209, 2), x);
    else if (x >= -2147483648) d.setInt32(this.tag(210, 4), x);
    else d.setBigInt64(this.tag(211, 8), BigInt(x));
  }
  float(x) {
    this.room(9);
    this.view.setFloat64(this.tag(203, 8), x);
  }
  str(s) {
    this.room(5 + 3 * s.length);
    const at = this.n + 5;
    const { written } = utf8.encodeInto(s, this.buf.subarray(at));
    this.head(160, 217, 218, 219, written, 31);
    this.buf.copyWithin(this.n, at, at + written);
    this.n += written;
  }
  /** A string's, an array's or a map's header: its fix form up to `fixMax`,
   * then 8 bits (strings only), 16 and 32. */
  head(fix, c8, c16, c32, len, fixMax = 15) {
    this.room(5);
    if (len <= fixMax) this.buf[this.n++] = fix | len;
    else if (c8 && len < 256) this.view.setUint8(this.tag(c8, 1), len);
    else if (len < 65536) this.view.setUint16(this.tag(c16, 2), len);
    else this.view.setUint32(this.tag(c32, 4), len);
  }
  byte(x) {
    this.room(1);
    this.buf[this.n++] = x;
  }
  /** A form's first byte, `code`, and `size` bytes after it, where the
   * caller writes them: it has made room for both. */
  tag(code, size) {
    this.buf[this.n] = code;
    const at = this.n + 1;
    this.n = at + size;
    return at;
  }
  room(k) {
    if (this.n + k <= this.buf.length) return;
    let size = this.buf.length * 2;
    while (size < this.n + k) size *= 2;
    const b = new Uint8Array(size);
    b.set(this.buf.subarray(0, this.n));
    this.buf = b;
    this.view = new DataView(b.buffer);
  }
};
var MAX_DEPTH = 32;
var MAX_INT = BigInt(Number.MAX_SAFE_INTEGER);
var strict = new TextDecoder("utf-8", { fatal: true });
var Timestamp = class {
  seconds;
  nanoseconds;
  constructor(seconds, nanoseconds) {
    this.seconds = seconds;
    this.nanoseconds = nanoseconds;
  }
};
function read(bytes) {
  const b = bytes;
  const d = new DataView(b.buffer, b.byteOffset, b.byteLength);
  let at = 0;
  const take = (n) => {
    if (n > b.length - at) throw new RangeError("msgpack: cut short");
    const i = at;
    at += n;
    return i;
  };
  const big = (x) => {
    if (x > MAX_INT || x < -MAX_INT) throw new RangeError(`msgpack: ${x} is beyond 2^53 - 1`);
    return Number(x);
  };
  const str2 = (n) => {
    const i = take(n);
    try {
      return strict.decode(b.subarray(i, i + n));
    } catch {
      throw new TypeError("msgpack: a str that is not UTF-8");
    }
  };
  const bin = (n) => {
    const i = take(n);
    return b.slice(i, i + n);
  };
  const ext = (n) => {
    const type = d.getInt8(take(1));
    const i = take(n);
    if (type !== -1) throw new TypeError(`msgpack: the extension ${type}`);
    let seconds;
    let nanoseconds;
    if (n === 4) [seconds, nanoseconds] = [BigInt(d.getUint32(i)), 0];
    else if (n === 8) {
      const x = d.getBigUint64(i);
      [seconds, nanoseconds] = [x & 0x3ffffffffn, Number(x >> 34n)];
    } else if (n === 12) [seconds, nanoseconds] = [d.getBigInt64(i + 4), d.getUint32(i)];
    else throw new RangeError(`msgpack: a timestamp of ${n} bytes`);
    if (nanoseconds >= 1e9) throw new RangeError(`msgpack: a timestamp's ${nanoseconds} nanoseconds`);
    return new Timestamp(big(seconds), nanoseconds);
  };
  const array = (n, depth) => {
    if (depth === MAX_DEPTH) throw new RangeError("msgpack: nested too deep");
    if (n > b.length - at) throw new RangeError("msgpack: cut short");
    const out = [];
    for (let i = 0; i < n; i++) out.push(value(depth + 1));
    return out;
  };
  const map = (n, depth) => {
    if (depth === MAX_DEPTH) throw new RangeError("msgpack: nested too deep");
    if (2 * n > b.length - at) throw new RangeError("msgpack: cut short");
    const out = /* @__PURE__ */ Object.create(null);
    for (let i = 0; i < n; i++) {
      const k = value(depth + 1);
      if (typeof k !== "string") throw new TypeError("msgpack: a map key that is not a str");
      if (k in out) throw new TypeError(`msgpack: the key ${JSON.stringify(k)} is given twice`);
      out[k] = value(depth + 1);
    }
    return out;
  };
  const value = (depth) => {
    const c = d.getUint8(take(1));
    if (c <= 127) return c;
    if (c >= 224) return c - 256;
    if (c <= 143) return map(c & 15, depth);
    if (c <= 159) return array(c & 15, depth);
    if (c <= 191) return str2(c & 31);
    switch (c) {
      case 192:
        throw new TypeError("msgpack: a nil");
      case 194:
        return false;
      case 195:
        return true;
      case 196:
        return bin(d.getUint8(take(1)));
      case 197:
        return bin(d.getUint16(take(2)));
      case 198:
        return bin(d.getUint32(take(4)));
      case 199:
        return ext(d.getUint8(take(1)));
      case 200:
        return ext(d.getUint16(take(2)));
      case 201:
        return ext(d.getUint32(take(4)));
      case 202:
        return new Float(d.getFloat32(take(4)));
      case 203:
        return new Float(d.getFloat64(take(8)));
      case 204:
        return d.getUint8(take(1));
      case 205:
        return d.getUint16(take(2));
      case 206:
        return d.getUint32(take(4));
      case 207:
        return big(d.getBigUint64(take(8)));
      case 208:
        return d.getInt8(take(1));
      case 209:
        return d.getInt16(take(2));
      case 210:
        return d.getInt32(take(4));
      case 211:
        return big(d.getBigInt64(take(8)));
      case 212:
        return ext(1);
      case 213:
        return ext(2);
      case 214:
        return ext(4);
      case 215:
        return ext(8);
      case 216:
        return ext(16);
      case 217:
        return str2(d.getUint8(take(1)));
      case 218:
        return str2(d.getUint16(take(2)));
      case 219:
        return str2(d.getUint32(take(4)));
      case 220:
        return array(d.getUint16(take(2)), depth);
      case 221:
        return array(d.getUint32(take(4)), depth);
      case 222:
        return map(d.getUint16(take(2)), depth);
      case 223:
        return map(d.getUint32(take(4)), depth);
    }
    throw new RangeError(`msgpack: 0x${c.toString(16)} is no msgpack form`);
  };
  const v = value(0);
  if (at !== b.length) throw new RangeError("msgpack: bytes after the value");
  return v;
}

// src/keys.ts
var ACTIONS = /* @__PURE__ */ new Set([
  "char-backward",
  "char-forward",
  "word-backward",
  "word-forward",
  "line-start",
  "line-end",
  "delete-char-backward",
  "delete-char-forward",
  "delete-word-backward",
  "delete-word-forward",
  "delete-to-line-start",
  "delete-to-line-end",
  "line-previous",
  "line-next",
  "page-up",
  "page-down",
  "input-start",
  "input-end",
  "select-all",
  "newline",
  "submit",
  "program",
  "prefix",
  "default",
  "copy",
  "cut",
  "paste",
  "undo",
  "redo",
  "scroll-up",
  "scroll-down",
  "scroll-left",
  "scroll-right",
  "scroll-page-up",
  "scroll-page-down",
  "scroll-half-page-up",
  "scroll-half-page-down",
  "scroll-start",
  "scroll-end"
]);
var SCROLL_ACTIONS = new Set([...ACTIONS].filter((a) => a.startsWith("scroll-")));
var MULTILINE_ACTIONS = /* @__PURE__ */ new Set(["line-previous", "line-next", "page-up", "page-down", "newline"]);
var MOVES = /* @__PURE__ */ new Set([
  "char-backward",
  "char-forward",
  "word-backward",
  "word-forward",
  "line-start",
  "line-end",
  "line-previous",
  "line-next",
  "page-up",
  "page-down",
  "input-start",
  "input-end"
]);
var ROW_ACTIONS = /* @__PURE__ */ new Set(["line-previous", "line-next", "page-up", "page-down"]);
var BACKWARD = /* @__PURE__ */ new Set(["char-backward", "word-backward", "line-start", "line-previous", "page-up", "input-start"]);
var INSERT = "insert";
var TERMINAL_KEYS = [
  "arrow_left=char-backward ctrl+b=char-backward arrow_right=char-forward ctrl+f=char-forward",
  "alt+arrow_left=word-backward ctrl+arrow_left=word-backward alt+b=word-backward",
  "alt+arrow_right=word-forward ctrl+arrow_right=word-forward alt+f=word-forward",
  "home=line-start end=line-end ctrl+e=line-end",
  "backspace=delete-char-backward ctrl+h=delete-char-backward",
  "delete=delete-char-forward ctrl+d=delete-char-forward",
  "alt+backspace=delete-word-backward ctrl+w=delete-word-backward ctrl+backspace=delete-word-backward",
  "alt+delete=delete-word-forward alt+d=delete-word-forward ctrl+delete=delete-word-forward",
  "ctrl+u=delete-to-line-start ctrl+k=delete-to-line-end",
  "arrow_up=line-previous ctrl+p=line-previous arrow_down=line-next ctrl+n=line-next",
  "page_up=page-up page_down=page-down",
  "alt+<=input-start ctrl+home=input-start alt+>=input-end ctrl+end=input-end",
  "mod+a=select-all",
  "ctrl+m=newline"
].join(" ");
var segmenter = new Intl.Segmenter(void 0, { granularity: "grapheme" });
function chars(s) {
  return Array.from(segmenter.segment(s), (g) => g.segment);
}
var WHITE_SPACE = /* @__PURE__ */ new Set([9, 10, 11, 12, 13, 32, 133, 160, 5760, 8232, 8233, 8239, 8287, 12288]);
function isSpace(c) {
  const cp = c.codePointAt(0) ?? 0;
  return WHITE_SPACE.has(cp) || cp >= 8192 && cp <= 8202;
}
function isBreak(c) {
  return c === "\n" || c === "\r" || c === "\r\n";
}
var MODIFIERS = ["super", "ctrl", "alt", "shift"];
var MODIFIER_NAMES = {
  super: "super",
  cmd: "super",
  command: "super",
  ctrl: "ctrl",
  control: "ctrl",
  alt: "alt",
  opt: "alt",
  option: "alt",
  shift: "shift"
};
var PHYSICAL = new Set(
  "backquote backslash bracket_left bracket_right comma digit_0 digit_1 digit_2 digit_3 digit_4 digit_5 digit_6 digit_7 digit_8 digit_9 equal intl_backslash intl_ro intl_yen key_a key_b key_c key_d key_e key_f key_g key_h key_i key_j key_k key_l key_m key_n key_o key_p key_q key_r key_s key_t key_u key_v key_w key_x key_y key_z minus period quote semicolon slash alt_left alt_right backspace caps_lock context_menu control_left control_right enter meta_left meta_right shift_left shift_right space tab convert kana_mode non_convert delete end help home insert page_down page_up arrow_down arrow_left arrow_right arrow_up num_lock numpad_0 numpad_1 numpad_2 numpad_3 numpad_4 numpad_5 numpad_6 numpad_7 numpad_8 numpad_9 numpad_add numpad_backspace numpad_clear numpad_clear_entry numpad_comma numpad_decimal numpad_divide numpad_enter numpad_equal numpad_memory_add numpad_memory_clear numpad_memory_recall numpad_memory_store numpad_memory_subtract numpad_multiply numpad_paren_left numpad_paren_right numpad_subtract numpad_separator numpad_up numpad_down numpad_right numpad_left numpad_begin numpad_home numpad_end numpad_insert numpad_delete numpad_page_up numpad_page_down escape f1 f2 f3 f4 f5 f6 f7 f8 f9 f10 f11 f12 f13 f14 f15 f16 f17 f18 f19 f20 f21 f22 f23 f24 f25 fn fn_lock print_screen scroll_lock pause browser_back browser_favorites browser_forward browser_home browser_refresh browser_search browser_stop eject launch_app_1 launch_app_2 launch_mail media_play_pause media_select media_stop media_track_next media_track_previous power sleep audio_volume_down audio_volume_mute audio_volume_up wake_up copy cut paste".split(" ")
);
var OTHER = {
  up: "arrow_up",
  down: "arrow_down",
  left: "arrow_left",
  right: "arrow_right",
  pageup: "page_up",
  pagedown: "page_down",
  pgup: "page_up",
  pgdn: "page_down",
  esc: "escape"
};
var ALIAS = /^<[A-Za-z0-9_-]+>$/;
function lower(c) {
  return Array.from(c, (x) => {
    const l = x.toLowerCase();
    return Array.from(l).length === 1 ? l : x;
  }).join("");
}
function upper(c) {
  return Array.from(c, (x) => {
    const u = x.toUpperCase();
    return Array.from(u).length === 1 ? u : x;
  }).join("");
}
function isCapital(c) {
  return Array.from(c).length === 1 && lower(c) !== c && upper(lower(c)) === c;
}
function isLetter(c) {
  return upper(c) !== c || isCapital(c);
}
function isChar(v) {
  if (v === "") return false;
  const cp = v.codePointAt(0);
  return chars(v).length === 1 && cp > 32 && cp !== 127 && !(cp >= 128 && cp < 160);
}
function physical(part) {
  const low = part.toLowerCase();
  if (PHYSICAL.has(low)) return low;
  if (OTHER[low] !== void 0) return OTHER[low];
  let out = "";
  for (let i = 0; i < part.length; i++) {
    const c = part[i];
    if (!/[A-Za-z0-9]/.test(c)) return null;
    if (i > 0 && (/[A-Z]/.test(c) || /[0-9]/.test(c))) out += "_";
    out += c.toLowerCase();
  }
  return PHYSICAL.has(out) ? out : null;
}
function join(mods, value) {
  const m = new Set(mods);
  return [...["mod", ...MODIFIERS].filter((x) => m.has(x)), value].join("+");
}
function read2(text, mod, keep) {
  if (text === "" || ALIAS.test(text)) return null;
  const parts = [];
  let i = 0;
  for (; ; ) {
    const j = text.indexOf("+", i);
    if (j < 0) {
      parts.push(text.slice(i));
      break;
    }
    parts.push(text.slice(i, j));
    i = j + 1;
    if (i === text.length) {
      if (parts[parts.length - 1] !== "") return null;
      break;
    }
  }
  const mods = /* @__PURE__ */ new Set();
  const order = [];
  const written = /* @__PURE__ */ new Set();
  const named = /* @__PURE__ */ new Set();
  let value = null;
  for (const p of parts) {
    const low = p.toLowerCase();
    let m = MODIFIER_NAMES[low];
    if (low === "meta") return null;
    if (low === "mod") {
      if (mod === null) return null;
      if (named.has("mod") || named.has("super") || named.has("ctrl")) return null;
      named.add("mod");
      m = mod;
      written.add(m);
    } else if (m !== void 0) {
      if ((m === "super" || m === "ctrl") && named.has("mod")) return null;
      named.add(m);
    }
    if (m !== void 0) {
      if (mods.has(m)) return null;
      mods.add(m);
      order.push(m);
      continue;
    }
    if (value !== null) return null;
    if (p === "") value = "+";
    else if (isChar(p)) value = lower(p);
    else {
      value = physical(p);
      if (value === null) return null;
    }
  }
  if (value === null) {
    if (order.length === 0) return null;
    value = order[order.length - 1];
    mods.delete(value);
    written.delete(value);
  }
  if (keep && written.size) return join([...[...mods].filter((m) => !written.has(m)), "mod"], value);
  return join(mods, value);
}
function parseKey(name) {
  return read2(name, null, false);
}
function splitKey(k) {
  const parts = k.split("+");
  const mods = [];
  while (parts.length > 1 && MODIFIERS.includes(parts[0])) mods.push(parts.shift());
  const value = parts.join("+");
  const kind = PHYSICAL.has(value) ? "physical" : MODIFIERS.includes(value) ? "modifier" : "character";
  return { mods, value, kind };
}
function keyText(key) {
  const k = parseKey(key);
  if (k === null) return null;
  const { mods, value, kind } = splitKey(k);
  if (mods.some((m) => m !== "shift")) return null;
  if (value === "space") return " ";
  if (kind !== "character") return null;
  return mods.includes("shift") && isLetter(value) ? upper(value) : value;
}
var ARROWS = { arrow_left: "\u2190", arrow_right: "\u2192", arrow_up: "\u2191", arrow_down: "\u2193" };
function displayKey(name) {
  const k = parseKey(name);
  if (k === null) return name;
  const { mods, value } = splitKey(k);
  if (ARROWS[value] !== void 0) return [...mods, ARROWS[value]].join("+");
  if (value === "tab") {
    const shifted = mods[mods.length - 1] === "shift";
    return [...shifted ? mods.slice(0, -1) : mods, shifted ? "\u21E7\u21E5" : "\u21E5"].join("+");
  }
  return k;
}
var EDIT_ACTIONS = ["select-all", "copy", "cut", "paste", "undo", "redo"];
function readKey(p, name) {
  return read2(name, p.mod ?? "ctrl", false);
}
function defaultEdit(mod = "ctrl") {
  const e = {};
  EDIT_ACTIONS.slice(0, 5).forEach((a, i) => e[a] = [`mod+${"acxvz"[i]}`]);
  e.redo = ["mod+shift+z"];
  if (mod === "ctrl") e.redo.push("ctrl+y");
  for (const a of EDIT_ACTIONS) e[a] = e[a].map((k) => read2(k, mod, false));
  return e;
}
function editOf(p) {
  return p.edit ?? defaultEdit(p.mod);
}
function editAction(p, name) {
  const m = new Keymap(false, p.mod);
  const edit = editOf(p);
  for (const a of EDIT_ACTIONS) for (const k of edit[a] ?? []) m.bind(k, a);
  return m.bindings.get(name) ?? null;
}
var NOT_KEYS = /* @__PURE__ */ new Set(["Unidentified", "Dead", "Process", "Compose", "Shift", "Control", "Alt", "AltGraph", "Meta", "OS", "Super", "Hyper", "Fn", "FnLock", "CapsLock", "NumLock", "ScrollLock", "Symbol", "SymbolLock"]);
function held(e) {
  return [e.metaKey && "super", e.ctrlKey && "ctrl", e.altKey && "alt", e.shiftKey && "shift"].filter(Boolean);
}
function pressNames(e) {
  if (NOT_KEYS.has(e.key)) return [];
  const mods = held(e);
  const out = [];
  const code = e.code ? physical(e.code) : null;
  if (code !== null) out.push(join(mods, code));
  if (e.key === " ") out.push(join(mods, "space"));
  else if (isChar(e.key)) {
    const c = lower(e.key);
    out.push(join(mods.filter((m) => m !== "shift" || isLetter(e.key)), c));
  }
  return [...new Set(out)];
}
var C0 = {
  0: "ctrl+space",
  8: "ctrl+h",
  9: "tab",
  13: "enter",
  27: "escape",
  127: "backspace",
  28: "ctrl+\\",
  29: "ctrl+]",
  30: "ctrl+^",
  31: "ctrl+_"
};
var FINAL = { A: "arrow_up", B: "arrow_down", C: "arrow_right", D: "arrow_left", H: "home", F: "end" };
var TILDE = { 1: "home", 7: "home", 4: "end", 8: "end", 2: "insert", 3: "delete", 5: "page_up", 6: "page_down" };
var CODES = { 9: "tab", 13: "enter", 27: "escape", 8: "backspace", 127: "backspace", 32: "space" };
var KITTY = {
  57409: ".",
  57410: "/",
  57411: "*",
  57412: "-",
  57413: "+",
  57414: "enter",
  57415: "=",
  57417: "arrow_left",
  57418: "arrow_right",
  57419: "arrow_up",
  57420: "arrow_down",
  57421: "page_up",
  57422: "page_down",
  57423: "home",
  57424: "end",
  57425: "insert",
  57426: "delete"
};
for (let i = 0; i < 10; i++) KITTY[57399 + i] = String(i);
function charKey(c, mods) {
  const m = new Set(mods);
  if (c === " ") return join(m, "space");
  if (isCapital(c)) {
    m.add("shift");
    c = lower(c);
  }
  return join(m, c);
}
function withMods(name, add) {
  if (name === null) return null;
  const { mods, value } = splitKey(name);
  return join([...mods, ...add], value);
}
function modsOf(field) {
  const [ms = "", es = ""] = (field ?? "").split(":");
  const m = ms === "" ? 1 : Number(ms);
  const e = es === "" ? 1 : Number(es);
  if (!Number.isInteger(m) || !Number.isInteger(e) || e === 3) return null;
  const bits = m < 2 ? 0 : m - 1;
  const out = /* @__PURE__ */ new Set();
  if (bits & 1) out.add("shift");
  if (bits & 2) out.add("alt");
  if (bits & 4) out.add("ctrl");
  if (bits & (8 | 32)) out.add("super");
  return out;
}
function codeKey(code, mods, shifted, text) {
  if (CODES[code]) return join(mods, CODES[code]);
  if (code >= 57344) return KITTY[code] ? join(mods, KITTY[code]) : null;
  if (code < 32 || code === 127 || code > 1114111) return null;
  const c = String.fromCodePoint(code);
  if (isLetter(c)) return join(mods, lower(c));
  if (mods.has("shift")) {
    const rest = new Set(mods);
    rest.delete("shift");
    if (shifted) return charKey(String.fromCodePoint(shifted), rest);
    if (text) return charKey(text, rest);
  }
  return join(mods, c);
}
function csi(params, final) {
  if (/^[<=>?]/.test(params)) return null;
  const fields = params.split(";");
  const num = (s) => s === void 0 || s === "" ? 0 : Number(s);
  if (final === "u") {
    const codes = fields[0].split(":").map(num);
    const mods = modsOf(fields[1]);
    if (mods === null || codes.some((c) => !Number.isInteger(c))) return null;
    const text = fields[2]?.split(":").filter(Boolean).map((x) => String.fromCodePoint(Number(x))).join("");
    return codeKey(codes[0], mods, codes[1] || void 0, text || void 0);
  }
  if (final === "~") {
    const n = num(fields[0]);
    if (n === 27 && fields.length >= 3) {
      const mods2 = modsOf(fields[1]);
      const c = num(fields[2]);
      return mods2 === null || !Number.isInteger(c) ? null : codeKey(c, mods2);
    }
    const mods = modsOf(fields[1]);
    return TILDE[n] && mods !== null ? join(mods, TILDE[n]) : null;
  }
  if (FINAL[final]) {
    const mods = modsOf(fields[1]);
    return mods === null ? null : join(mods, FINAL[final]);
  }
  if (final === "Z") return "shift+tab";
  return null;
}
function one(s, i) {
  const c = s.charCodeAt(i);
  if (c < 32 || c === 127) return [C0[c] ?? `ctrl+${String.fromCharCode(c + 96)}`, 1];
  const cl = segmenter.segment(s.slice(i)).containing(0).segment;
  return [cl === " " ? "space" : isChar(cl) ? charKey(cl, []) : null, cl.length];
}
function inputKeys(s) {
  const out = [];
  let i = 0;
  while (i < s.length) {
    const start = i;
    let key;
    if (s.charCodeAt(i) !== 27) {
      let n;
      [key, n] = one(s, i);
      i += n;
    } else if (i + 1 === s.length) {
      key = "escape";
      i += 1;
    } else if ((s[i + 1] === "[" || s[i + 1] === "O") && i + 2 < s.length) {
      if (s[i + 1] === "O") {
        key = "ABCDHF".includes(s[i + 2]) ? FINAL[s[i + 2]] : null;
        i += 3;
      } else {
        let j = i + 2;
        while (j < s.length && s.charCodeAt(j) >= 48 && s.charCodeAt(j) <= 63) j++;
        const params = s.slice(i + 2, j);
        while (j < s.length && s.charCodeAt(j) >= 32 && s.charCodeAt(j) <= 47) j++;
        if (j === s.length) {
          key = null;
          i = j;
        } else {
          key = s.charCodeAt(j) >= 64 && s.charCodeAt(j) <= 126 ? csi(params, s[j]) : null;
          i = j + 1;
        }
      }
    } else {
      const [k, n] = one(s, i + 1);
      key = withMods(k, ["alt"]);
      i += 1 + n;
    }
    out.push({ key, data: s.slice(start, i) });
  }
  return out;
}
var UNBOUND = /* @__PURE__ */ new Set(["tab", "shift+tab", "escape"]);
var Keymap = class {
  /** By key, canonical with `mod` read. */
  bindings = /* @__PURE__ */ new Map();
  /** Each key's last binding as written, `mod` kept. */
  written = /* @__PURE__ */ new Map();
  multiline;
  mod;
  /** A resolved field keeps the host's keymap for `default`. */
  defaults = null;
  constructor(multiline = false, mod = "ctrl") {
    this.multiline = multiline;
    this.mod = mod;
  }
  /** Binds a key, which may name `mod`, to an action. */
  bind(key, action) {
    const k = read2(key, this.mod, false);
    if (k === null) return;
    this.bindings.set(k, action);
    this.written.set(k, read2(key, this.mod, true));
  }
  /** Each key once, where it was first bound, as its last binding wrote
   * it, with its last action. */
  format() {
    return Array.from(this.bindings, ([k, a]) => `${this.written.get(k) ?? k}=${a}`).join(" ");
  }
  /** The action a key is bound to: its own binding, or for a key with
   * `shift` that is not a character and has none, the binding of the key
   * without it (SPEC §10.2). */
  action(name) {
    const k = parseKey(name);
    if (k === null || UNBOUND.has(k) && !(k === "tab" && this.defaults)) return void 0;
    const a = this.bindings.get(k);
    if (a !== void 0) return a;
    const { mods, value, kind } = splitKey(k);
    if (!mods.includes("shift") || kind === "character") return void 0;
    return this.bindings.get(join(mods.filter((m) => m !== "shift"), value));
  }
  /** Whether the keymap binds the key, with the fallback of `action`. A
   * press with two names (SPEC §10.4) is asked of its physical key first:
   * what follows is of the first name that is bound. */
  binds(name) {
    return this.action(name) !== void 0;
  }
  /** Whether the key is the program's (SPEC §10.2, *Keys for the
   * program*): bound to `program`, or to `prefix`, which is `program` and
   * more. It reaches the program before the focused element uses it and
   * before the surface scrolls with it. */
  program(name) {
    const a = this.action(name);
    return a === "program" || a === "prefix";
  }
  /** Whether the key begins a prefix (SPEC §10.2, §10.5): bound to
   * `prefix`. Every key after it reaches the program too. */
  prefix(name) {
    return this.action(name) === "prefix";
  }
  /** The scroll action the key is bound to, or null (SPEC §10.2,
   * *Scrolling keys*): outside a text field, a key the element does not
   * use scrolls with it. */
  scroll(name) {
    const a = this.action(name);
    return a !== void 0 && SCROLL_ACTIONS.has(a) ? a : null;
  }
  /** What the field does with a key: an action, `INSERT` for a key it types
   * (`keyText`), or null when the key is not the field's. */
  lookup(name) {
    const k = parseKey(name);
    if (k === null || UNBOUND.has(k)) return null;
    let a = this.action(k);
    if (a === "default") a = this.defaults?.action(k);
    if (a !== void 0) return a === "program" || a === "prefix" || MULTILINE_ACTIONS.has(a) && !this.multiline ? null : a;
    return keyText(k) !== null ? INSERT : null;
  }
  /** Whether the field selects with a key (SPEC §10.2, *Shift selects*):
   * it looks up a move, and the key's name has `shift`. The field then
   * moves its caret and keeps the selection's anchor. */
  selects(name) {
    const k = parseKey(name);
    if (k === null) return false;
    const a = this.lookup(k);
    return a !== null && MOVES.has(a) && splitKey(k).mods.includes("shift");
  }
};
function* bindingsOf(mod, value) {
  for (const b of value.split(/[ \t\n\f\r]+/)) {
    const i = b.lastIndexOf("=");
    if (i < 0) continue;
    const key = b.slice(0, i);
    const k = read2(key, mod, false);
    const a = b.slice(i + 1);
    if (k === null || !ACTIONS.has(a) || UNBOUND.has(k) && !(k === "tab" && (a === "program" || a === "default"))) continue;
    yield [key, a];
  }
}
function elementKeymap(p, ...values) {
  const m = new Keymap(false, p.mod);
  for (const v of values) for (const [k, a] of bindingsOf(m.mod, v)) m.bind(k, a);
  return m;
}
var DEFAULT_KEYS = [
  ["arrow_left", "char-backward"],
  ["arrow_right", "char-forward"],
  ["alt+arrow_left", "word-backward"],
  ["alt+arrow_right", "word-forward"],
  ["home", "line-start"],
  ["end", "line-end"],
  ["backspace", "delete-char-backward"],
  ["delete", "delete-char-forward"],
  ["alt+backspace", "delete-word-backward"],
  ["alt+delete", "delete-word-forward"],
  ["arrow_up", "line-previous"],
  ["arrow_down", "line-next"],
  ["page_up", "page-up"],
  ["page_down", "page-down"]
];
var CTRL_KEYS = [
  ["ctrl+arrow_left", "word-backward"],
  ["ctrl+arrow_right", "word-forward"],
  ["ctrl+home", "input-start"],
  ["ctrl+end", "input-end"],
  ["ctrl+backspace", "delete-word-backward"],
  ["ctrl+delete", "delete-word-forward"]
];
var SUPER_KEYS = [
  ["super+arrow_left", "line-start"],
  ["super+arrow_right", "line-end"],
  ["super+arrow_up", "input-start"],
  ["super+arrow_down", "input-end"],
  ["super+backspace", "delete-to-line-start"],
  ["ctrl+a", "line-start"],
  ["ctrl+e", "line-end"],
  ["ctrl+b", "char-backward"],
  ["ctrl+f", "char-forward"],
  ["ctrl+p", "line-previous"],
  ["ctrl+n", "line-next"],
  ["ctrl+h", "delete-char-backward"],
  ["ctrl+d", "delete-char-forward"],
  ["ctrl+k", "delete-to-line-end"]
];
function resolve(p, multiline, ...values) {
  const m = new Keymap(multiline, p.mod);
  for (const [k, a] of DEFAULT_KEYS) m.bind(k, a);
  m.bind("enter", multiline ? "newline" : "submit");
  for (const [k, a] of m.mod === "super" ? SUPER_KEYS : CTRL_KEYS) m.bind(k, a);
  const edit = editOf(p);
  for (const a of EDIT_ACTIONS) for (const k of edit[a] ?? []) m.bind(k, a);
  m.defaults = new Keymap(multiline, m.mod);
  for (const [k, a] of m.bindings) m.defaults.bind(k, a);
  for (const v of values) for (const [k, a] of bindingsOf(m.mod, v)) if (!SCROLL_ACTIONS.has(a)) m.bind(k, a);
  return m;
}
function lineOf(t, p) {
  if (!t.multiline) return [0, t.chars.length];
  let s = p;
  while (s > 0 && !isBreak(t.chars[s - 1])) s--;
  let e = p;
  while (e < t.chars.length && !isBreak(t.chars[e])) e++;
  return [s, e];
}
function wordBack(t, p) {
  if (t.password) return 0;
  while (p > 0 && isSpace(t.chars[p - 1])) p--;
  while (p > 0 && !isSpace(t.chars[p - 1])) p--;
  return p;
}
function wordForward(t, p) {
  if (t.password) return t.chars.length;
  while (p < t.chars.length && isSpace(t.chars[p])) p++;
  while (p < t.chars.length && !isSpace(t.chars[p])) p++;
  return p;
}
function plan(t, action, rows, goal = null, extend = false) {
  if (MULTILINE_ACTIONS.has(action) && !t.multiline) return { kind: "none" };
  const n = t.chars.length;
  const selected = t.start < t.end;
  if (action === "select-all") return { kind: "move", anchor: 0, to: n };
  if (extend && MOVES.has(action)) {
    const anchor = t.caret === t.start ? t.end : t.start;
    const moved = plan({ ...t, start: t.caret, end: t.caret }, action, rows, goal);
    return moved.kind === "move" ? { ...moved, anchor } : moved;
  }
  if (action.startsWith("delete-")) {
    if (selected) return { kind: "replace", from: t.start, to: t.end, text: "" };
    const p2 = t.start;
    const [s2, e2] = lineOf(t, p2);
    const [from, to] = {
      "delete-char-backward": [Math.max(p2 - 1, 0), p2],
      "delete-char-forward": [p2, Math.min(p2 + 1, n)],
      "delete-word-backward": [wordBack(t, p2), p2],
      "delete-word-forward": [p2, wordForward(t, p2)],
      "delete-to-line-start": [s2, p2],
      "delete-to-line-end": [p2, e2]
    }[action] ?? [p2, p2];
    return from < to ? { kind: "replace", from, to, text: "" } : { kind: "none" };
  }
  if (action === "newline") return { kind: "replace", from: t.start, to: t.end, text: "\n" };
  const p = BACKWARD.has(action) ? t.start : t.end;
  if (selected && (action === "char-backward" || action === "char-forward")) return { kind: "move", to: p };
  const [s, e] = lineOf(t, p);
  switch (action) {
    case "char-backward":
      return { kind: "move", to: Math.max(p - 1, 0) };
    case "char-forward":
      return { kind: "move", to: Math.min(p + 1, n) };
    case "word-backward":
      return { kind: "move", to: wordBack(t, p) };
    case "word-forward":
      return { kind: "move", to: wordForward(t, p) };
    case "line-start":
      return { kind: "move", to: s };
    case "line-end":
      return { kind: "move", to: e };
    case "input-start":
      return { kind: "move", to: 0 };
    case "input-end":
      return { kind: "move", to: n };
    case "line-previous":
    case "line-next":
    case "page-up":
    case "page-down": {
      if (!rows) return { kind: "none" };
      const by = { "line-previous": -1, "line-next": 1, "page-up": -rows.page, "page-down": rows.page }[action];
      const r = rows.move(p, by, goal);
      return { kind: "move", to: r.to, goal: r.goal };
    }
  }
  return { kind: "none" };
}

// src/delta.ts
var DeltaError = class extends Error {
  code;
  i;
  constructor(code, detail, i) {
    super(detail);
    this.code = code;
    this.i = i;
  }
};
var OPS = /* @__PURE__ */ new Set(["morph", "inner", "replace", "append", "prepend", "before", "after", "remove", "attr", "unattr", "text", "var", "reveal", "scroll"]);
var KEYED = /* @__PURE__ */ new Set(["attr", "unattr", "var"]);
function readDelta(bytes) {
  let body;
  try {
    body = read(bytes);
  } catch (e) {
    throw new DeltaError("EINVAL", `the delta does not decode: ${e.message}`);
  }
  const list = isMap(body) ? body.ops : void 0;
  if (!Array.isArray(list)) throw new DeltaError("EINVAL", "a delta is a map with a list of ops");
  if (list.length === 0) throw new DeltaError("EINVAL", "a delta has no ops");
  return list.map((o, i) => {
    const bad = (why) => new DeltaError("EINVAL", `op ${i}: ${why}`, i);
    if (!isMap(o)) throw bad("not a map");
    const op = o.op;
    if (typeof op !== "string" || !OPS.has(op)) throw bad(`unknown op ${JSON.stringify(op)}`);
    const field = (name, need) => {
      const x = o[name];
      if (x !== void 0 && typeof x !== "string") throw bad(`${name} is not a str`);
      if (need && !x) throw bad(`op=${op} needs ${name}`);
      return x;
    };
    const t = field("t", op !== "morph" && op !== "scroll") || void 0;
    const k = field("k", KEYED.has(op));
    const v = field("v", false);
    if (op === "reveal" && v !== void 0 && v !== "start" && v !== "end") throw bad(`op=reveal takes start, end or nothing, not ${JSON.stringify(v)}`);
    if (op === "scroll" && (v === void 0 || !SCROLL_ACTIONS.has(v))) throw bad(`op=scroll needs a scroll action, not ${JSON.stringify(v)}`);
    return { op, t, k: KEYED.has(op) ? k : void 0, v: op === "remove" || op === "unattr" ? void 0 : v };
  });
}
var isMap = (x) => typeof x === "object" && x !== null && !Array.isArray(x) && !(x instanceof Uint8Array) && Object.getPrototypeOf(x) === null;
var Deltas = class {
  doc;
  resolver;
  /** Scrolls an element into view: the surface's to do (`reveal`). */
  revealer;
  /** Hears that the delta set a control's value: the surface puts a text
   * field's caret at the end (SPEC §6.2). */
  valued;
  scroller;
  constructor(doc, resolver, revealer, valued = () => {
  }, scroller = () => {
  }) {
    this.doc = doc;
    this.resolver = resolver;
    this.revealer = revealer;
    this.valued = valued;
    this.scroller = scroller;
  }
  /** A changed attribute's property, as `syncProperty`, told to `valued`
   * when it set the value. */
  sync(el, name, value, focused) {
    if (syncProperty(el, name, value, focused)) this.valued(el);
  }
  /** Parses a fragment inert, cleaned and resolved. */
  parse(html) {
    const t = this.doc.createElement("template");
    t.innerHTML = html;
    this.resolver.adopt(t.content);
    return Array.from(t.content.childNodes);
  }
  /**
   * Applies the ops in order, as one unit (SPEC §6.1). An op whose target
   * is missing changes nothing and the ops after it still apply; the first
   * op that failed is returned, named by its index.
   */
  apply(ops) {
    let failed = null;
    ops.forEach((o, i) => {
      try {
        this.applyOp(o.op, o.t, o.k, o.v ?? "");
      } catch (e) {
        failed ??= e instanceof DeltaError ? new DeltaError(e.code, e.message, i) : new DeltaError("EINVAL", e instanceof Error ? e.message : String(e), i);
      }
    });
    return failed;
  }
  applyOp(op, target, key, payload) {
    const find2 = () => {
      const el = target ? this.doc.getElementById(target) : null;
      if (!el) throw new DeltaError("ENOTARGET", target ?? "");
      return el;
    };
    switch (op) {
      case "morph": {
        if (target === void 0) {
          const missing = [];
          for (const k of this.parse(payload)) {
            if (k.nodeType !== 1) continue;
            const id = k.getAttribute("id");
            const old = id ? this.doc.getElementById(id) : null;
            if (old) this.morphNode(old, k);
            else missing.push(id ?? "(element without id)");
          }
          if (missing.length) throw new DeltaError("ENOTARGET", missing.join(","));
          return;
        }
        const t = find2();
        const frag = this.parse(payload);
        const elements = frag.filter((n) => n.nodeType === 1);
        if (elements.length === 1) {
          this.morphNode(t, elements[0]);
        } else {
          t.before(...frag);
          t.remove();
        }
        return;
      }
      case "inner":
        this.morphChildren(find2(), this.parse(payload));
        return;
      case "replace": {
        const t = find2();
        t.before(...this.parse(payload));
        t.remove();
        return;
      }
      case "append":
      case "prepend": {
        const t = find2();
        const fresh = [];
        for (const k of this.parse(payload)) {
          const id = k.nodeType === 1 ? k.getAttribute("id") : null;
          const old = id ? this.doc.getElementById(id) : null;
          if (old && old.parentNode === t) this.morphNode(old, k);
          else fresh.push(k);
        }
        if (op === "append") t.append(...fresh);
        else t.prepend(...fresh);
        return;
      }
      case "before":
        find2().before(...this.parse(payload));
        return;
      case "after":
        find2().after(...this.parse(payload));
        return;
      case "remove":
        find2().remove();
        return;
      case "attr": {
        const t = find2();
        const name = key;
        this.resolver.set(t, name, payload);
        this.sync(t, name, payload, this.doc.activeElement === t);
        return;
      }
      case "unattr": {
        const t = find2();
        const name = key;
        this.resolver.remove(t, name);
        this.sync(t, name, null, this.doc.activeElement === t);
        return;
      }
      case "text": {
        const t = find2();
        if (t.localName === "style") this.resolver.setCss(t, payload);
        else if (t.childNodes.length === 1 && t.firstChild.nodeType === 3) t.firstChild.data = payload;
        else t.textContent = payload;
        return;
      }
      case "var": {
        const t = find2();
        const k = key;
        const name = k.startsWith("--") ? k : `--${k}`;
        t.style?.setProperty(name, payload);
        return;
      }
      case "scroll":
        this.scroller(target ? find2() : this.doc.documentElement, payload);
        return;
      case "reveal": {
        this.revealer(find2(), payload);
        return;
      }
      default:
        throw new DeltaError("EINVAL", `unknown op=${op}`);
    }
  }
  /** Makes `old` (live) look like `neu` (parsed), keeping `old` where possible. Returns the node now in place. */
  morphNode(old, neu) {
    if (old.nodeType === 3 && neu.nodeType === 3) {
      if (old.data !== neu.data) old.data = neu.data;
      return old;
    }
    if (old.nodeType === 8 && neu.nodeType === 8) return old;
    if (old.nodeType === 1 && neu.nodeType === 1) {
      const a = old;
      const b = neu;
      const ia = a.getAttribute("id");
      const ib = b.getAttribute("id");
      if (a.localName === b.localName && a.namespaceURI === b.namespaceURI && (ia === ib || ia === null || ib === null)) {
        this.syncAttributes(a, b);
        if (a.localName === "style") {
          const css = this.resolver.cssOf(b);
          if (this.resolver.cssOf(a) !== css) this.resolver.setCss(a, css);
        } else {
          const kids = a.localName === "template" ? b.content.childNodes : b.childNodes;
          this.morphChildren(a.localName === "template" ? a.content : a, Array.from(kids));
        }
        return a;
      }
    }
    old.replaceWith(neu);
    return neu;
  }
  syncAttributes(old, neu) {
    const oldAttrs = this.resolver.attributes(old);
    const newAttrs = this.resolver.attributes(neu);
    const focused = this.doc.activeElement === old;
    for (const [name, value] of newAttrs) {
      if (focused && name === "value") continue;
      const had = oldAttrs.find(([n]) => n === name);
      if (!had || had[1] !== value) {
        this.resolver.set(old, name, value);
        this.sync(old, name, value, focused);
      }
    }
    for (const [name] of oldAttrs) {
      if (!newAttrs.some(([n]) => n === name)) {
        this.resolver.remove(old, name);
        this.sync(old, name, null, focused);
      }
    }
  }
  /** Morphs the children of `parent` (live) into `kids` (parsed). */
  morphChildren(parent, kids) {
    const oldKids = Array.from(parent.childNodes);
    const idOf = (n) => n.nodeType === 1 ? n.getAttribute("id") : null;
    const newIds = new Set(kids.map(idOf).filter((x) => x !== null));
    const used = /* @__PURE__ */ new Set();
    const plan2 = [];
    let cursor = 0;
    for (const nk of kids) {
      const id = idOf(nk);
      let match = null;
      if (id !== null) {
        match = oldKids.find((ok) => !used.has(ok) && idOf(ok) === id) ?? null;
      } else {
        for (let i = cursor; i < oldKids.length; i++) {
          const ok = oldKids[i];
          if (used.has(ok)) continue;
          const okId = idOf(ok);
          let same2 = false;
          if (okId !== null && newIds.has(okId)) same2 = false;
          else if (ok.nodeType === 1 && nk.nodeType === 1)
            same2 = okId === null && ok.localName === nk.localName && ok.namespaceURI === nk.namespaceURI;
          else same2 = ok.nodeType === nk.nodeType && (ok.nodeType === 3 || ok.nodeType === 8);
          if (same2) {
            match = ok;
            cursor = i + 1;
            break;
          }
        }
      }
      if (match) used.add(match);
      plan2.push([match, nk]);
    }
    for (const ok of oldKids) if (!used.has(ok)) ok.remove();
    plan2.forEach(([keep, nk], i) => {
      const want = keep ? this.morphNode(keep, nk) : nk;
      const current = parent.childNodes[i] ?? null;
      if (current !== want) move(parent, want, current);
    });
  }
};
function move(parent, node, before) {
  const p = parent;
  if (node.parentNode === parent && typeof p.moveBefore === "function") {
    try {
      p.moveBefore(node, before);
      return;
    } catch {
    }
  }
  parent.insertBefore(node, before);
}
function syncProperty(el, name, value, focused) {
  if (focused) return false;
  const n = name.toLowerCase();
  if (n === "value" && "value" in el && (el.localName === "input" || el.localName === "textarea" || el.localName === "select")) {
    el.value = value ?? "";
    return true;
  } else if (n === "checked" && el.localName === "input") {
    el.checked = value !== null;
  } else if (n === "selected" && el.localName === "option") {
    el.selected = value !== null;
  }
  return false;
}

// src/field.ts
var FieldError = class extends Error {
  code;
  constructor(code, detail) {
    super(detail);
    this.code = code;
  }
};
var invalid = (why) => new FieldError("EINVAL", why);
var isMap2 = (x) => typeof x === "object" && x !== null && !Array.isArray(x) && !(x instanceof Uint8Array) && Object.getPrototypeOf(x) === null;
var isInt = (x) => typeof x === "number" && Number.isSafeInteger(x);
function readField(bytes) {
  let body;
  try {
    body = read(bytes);
  } catch (e) {
    throw invalid(`the body does not decode: ${e.message}`);
  }
  if (!isMap2(body)) throw invalid("the body is a map");
  const { rev, value, text, from, to, sel, checked } = body;
  if (rev === void 0) throw invalid("rev is missing");
  if (!isInt(rev)) throw invalid("rev is not an int");
  if (value !== void 0 && typeof value !== "string") throw invalid("value is not a str");
  if (text !== void 0 && typeof text !== "string") throw invalid("text is not a str");
  for (const [name, x] of [["from", from], ["to", to]]) {
    if (x !== void 0 && !isInt(x)) throw invalid(`${name} is not an int`);
  }
  if (sel !== void 0 && !(Array.isArray(sel) && sel.length === 2 && sel.every(isInt))) throw invalid("sel is not two ints");
  if (checked !== void 0 && typeof checked !== "boolean") throw invalid("checked is not a bool");
  if ([value, text, from, to, sel, checked].every((x) => x === void 0)) throw invalid("no field but rev");
  if (value !== void 0 && (text !== void 0 || from !== void 0 || to !== void 0)) throw invalid("value comes with text, from or to");
  if (from === void 0 !== (to === void 0)) throw invalid("from and to come together");
  if (from !== void 0 && text === void 0) throw invalid("from and to come with text");
  if (from !== void 0 && from > to) throw invalid("from is past to");
  const change = { rev };
  if (value !== void 0) change.value = value;
  if (text !== void 0) change.text = text;
  if (from !== void 0) [change.from, change.to] = [from, to];
  if (sel !== void 0) change.sel = [sel[0], sel[1]];
  if (checked !== void 0) change.checked = checked;
  return change;
}
var utf8Length = (c) => c < 128 ? 1 : c < 2048 ? 2 : c < 65536 ? 3 : 4;
function unitAt(s, b) {
  if (b < 0) return null;
  let bytes = 0;
  let i = 0;
  while (bytes < b) {
    if (i >= s.length) return null;
    const c = s.codePointAt(i);
    bytes += utf8Length(c);
    i += c > 65535 ? 2 : 1;
  }
  return bytes === b ? i : null;
}
function byteAt(s, u) {
  let bytes = 0;
  for (let i = 0; i < u && i < s.length; ) {
    const c = s.codePointAt(i);
    bytes += utf8Length(c);
    i += c > 65535 ? 2 : 1;
  }
  return bytes;
}
var segmenter2 = new Intl.Segmenter(void 0, { granularity: "grapheme" });
function clusterStart(s, u) {
  const g = u < s.length ? segmenter2.segment(s).containing(u) : void 0;
  return g ? g.index : u;
}
function wireSel(value, s) {
  const [anchor, caret] = s.backward ? [s.end, s.start] : [s.start, s.end];
  return [byteAt(value, anchor), byteAt(value, caret)];
}

// src/network.ts
var DIRECTIVES = ["img-src", "media-src", "font-src", "style-src"];
function source(s) {
  const v = s.trim().toLowerCase();
  if (v === "https:") return v;
  let u;
  try {
    u = new URL(v);
  } catch {
    return null;
  }
  if (u.protocol !== "https:" && u.protocol !== "http:") return null;
  if (u.pathname !== "/" && u.pathname !== "" || u.search || u.hash || u.username || u.password) return null;
  if (!/^https?:\/\/[^/]+\/?$/.test(v)) return null;
  return u.origin;
}
function clean(p) {
  const out = {};
  for (const d of DIRECTIVES) {
    const srcs = (p?.[d] ?? []).map(source).filter((s) => s !== null);
    if (srcs.length) out[d] = [...new Set(srcs)];
  }
  return out;
}
function parse(content) {
  const p = {};
  for (const part of content.split(";")) {
    const [name, ...srcs] = part.trim().split(/\s+/);
    const d = (name ?? "").toLowerCase();
    if (DIRECTIVES.includes(d)) p[d] = [...p[d] ?? [], ...srcs];
  }
  return clean(p);
}
function matches(src, url) {
  if (src === "https:") return url.protocol === "https:";
  return url.origin === src;
}
function intersect(host, doc) {
  const out = {};
  for (const d of DIRECTIVES) {
    const h = host[d] ?? [];
    const res = /* @__PURE__ */ new Set();
    for (const s of doc[d] ?? []) {
      if (s === "https:") {
        for (const hs of h) if (hs === "https:" || hs.startsWith("https://")) res.add(hs);
      } else if (h.some((hs) => matches(hs, new URL(s)))) {
        res.add(s);
      }
    }
    if (res.size) out[d] = [...res];
  }
  return out;
}
function allows(p, d, url) {
  let u;
  try {
    u = new URL(url);
  } catch {
    return false;
  }
  return (p[d] ?? []).some((s) => matches(s, u));
}
function cspSources(p, d) {
  return (p[d] ?? []).join(" ");
}
function directiveFor(tag, attr, rel = "") {
  if (attr === "srcset") return "img-src";
  if (attr === "poster") return "img-src";
  if (attr === "src") {
    if (tag === "img" || tag === "input" || tag === "image") return "img-src";
    if (tag === "video" || tag === "audio" || tag === "source" || tag === "track") return "media-src";
  }
  if ((attr === "href" || attr === "xlink:href") && (tag === "image" || tag === "use" || tag === "feimage")) return "img-src";
  if (attr === "href" && tag === "link" && rel.split(/\s+/).includes("stylesheet")) return "style-src";
  return null;
}

// src/resources.ts
var INVALID = "about:invalid";
var NO_BASE = "https://hotty.invalid/";
function network(value, ctx, d) {
  if (!ctx) return INVALID;
  let abs;
  try {
    abs = new URL(value, ctx.base).href;
  } catch {
    return INVALID;
  }
  return !abs.startsWith(NO_BASE) && allows(ctx.policy, d, abs) ? abs : INVALID;
}
var URL_ATTRS = /* @__PURE__ */ new Set(["src", "href", "poster", "background", "data", "action", "formaction", "xlink:href", "srcset", "cite", "longdesc", "lowsrc", "manifest", "codebase", "usemap", "ping"]);
function isUrlAttr(name) {
  return URL_ATTRS.has(name.toLowerCase());
}
var Store = class {
  entries = /* @__PURE__ */ new Map();
  used = 0;
  quota;
  /** Called with the names whose URL changed, so surfaces can re-resolve. */
  onChange = () => {
  };
  constructor(quota = 64 << 20) {
    this.quota = quota;
  }
  /** End the session: no bytes, URLs or quota usage carry into the next. */
  clear() {
    for (const e of this.entries.values()) if (e.url) URL.revokeObjectURL(e.url);
    this.entries.clear();
    this.used = 0;
  }
  put(name, mime, bytes) {
    const old = this.entries.get(name);
    const next = this.used - (old?.bytes.length ?? 0) + bytes.length;
    if (next > this.quota) throw new Error(`resource store over quota (${next} > ${this.quota} bytes)`);
    this.used = next;
    this.entries.set(name, { mime, bytes, url: null, deps: /* @__PURE__ */ new Set() });
    this.changed(name, old);
  }
  remove(name) {
    const old = this.entries.get(name);
    if (!old) return false;
    this.used -= old.bytes.length;
    this.entries.delete(name);
    this.changed(name, old);
    return true;
  }
  /** The `blob:` URL for a resource, or `null` if there is none (yet). */
  url(name) {
    const e = this.entries.get(name);
    if (!e) return null;
    if (e.url === null) {
      let bytes = e.bytes;
      if (e.mime.startsWith("text/css")) {
        e.deps.clear();
        const css = new TextDecoder().decode(e.bytes);
        bytes = new TextEncoder().encode(this.rewriteCss(css, e.deps));
      }
      e.url = URL.createObjectURL(new Blob([bytes], { type: e.mime }));
    }
    return e.url;
  }
  /** CSS text with every reference resolved; `names` collects the resources.
   * Without a context (a stylesheet resource, shared by surfaces with
   * different policies) nothing from the network is allowed. */
  rewriteCss(css, names, ctx) {
    const one2 = (raw, d) => {
      const v = raw.trim();
      if (/^cid:/i.test(v)) {
        names.add(v.slice(4));
        return this.url(v.slice(4)) ?? INVALID;
      }
      if (/^data:/i.test(v) || /^blob:/i.test(v) || v.startsWith("#")) return raw;
      for (const dir of d) {
        const r = network(v, ctx, dir);
        if (r !== INVALID) return r;
      }
      return INVALID;
    };
    return css.replace(/@import\s+(?:url\(\s*)?(['"])([^'"]*)\1\s*\)?/gi, (_m, _q, u) => `@import "${one2(u, ["style-src"])}"`).replace(/url\(\s*(['"]?)([^'")]*)\1\s*\)/gi, (_m, _q, u) => `url("${one2(u, ["img-src", "font-src"])}")`);
  }
  /** Resolves one attribute value; `names` collects the resources it names.
   * `d` is the directive that covers the attribute, if it fetches. */
  resolveUrl(value, names, ctx, d) {
    const v = value.trim();
    if (/^cid:/i.test(v)) {
      const name = v.slice(4);
      names.add(name);
      return this.url(name) ?? INVALID;
    }
    if (/^data:/i.test(v) || /^blob:/i.test(v) || v === "" || v.startsWith("#")) return value;
    if (d) return network(v, ctx, d);
    return /^[a-z][a-z0-9+.-]*:/i.test(v) ? INVALID : value;
  }
  changed(name, old) {
    const names = /* @__PURE__ */ new Set([name]);
    for (const [other, e] of this.entries) {
      if (e.deps.has(name)) {
        if (e.url) URL.revokeObjectURL(e.url);
        e.url = null;
        names.add(other);
      }
    }
    this.onChange(names);
    if (old?.url) URL.revokeObjectURL(old.url);
  }
};

// src/menu.ts
var MENU_DEPTH = 8;
function readMenu(bytes) {
  const v = read(bytes);
  if (!isMap3(v)) throw new Error("a menu is a map");
  const label = str(v.label, "a menu's label");
  return { label, items: items(v.items, 1) };
}
function items(v, depth) {
  if (v === void 0) return [];
  if (!Array.isArray(v)) throw new Error("items are an array");
  if (depth > MENU_DEPTH && v.length) throw new Error(`a menu nests at most ${MENU_DEPTH} deep`);
  return v.map((i) => {
    if (!isMap3(i)) throw new Error("a menu item is a map");
    if (i.enabled !== void 0 && typeof i.enabled !== "boolean") throw new Error("an item's enabled is a bool");
    return {
      key: str(i.key, "an item's key"),
      label: str(i.label, "an item's label"),
      name: str(i.name, "an item's name"),
      enabled: i.enabled === true,
      items: i.items === void 0 ? void 0 : items(i.items, depth + 1)
    };
  });
}
function readSeq(bytes) {
  if (!bytes.length) return [];
  const v = read(bytes);
  if (!isMap3(v)) throw new Error("a show is a map");
  const seq = v.seq === void 0 ? [] : v.seq;
  if (!Array.isArray(seq) || !seq.every((k) => typeof k === "string")) throw new Error("a show's seq is an array of str");
  return seq;
}
function reach(menu, seq, readKey2) {
  let label = menu.label;
  let level = menu.items;
  for (const typed of seq) {
    const k = readKey2(typed);
    const next = k === null ? void 0 : level.find((i) => isGroup(i) && i.key !== "" && readKey2(i.key) === k);
    if (!next) return null;
    label = next.label;
    level = next.items;
  }
  return { label, items: level };
}
function shownLevel(menu, seq, readKey2) {
  const level = reach(menu, seq, readKey2);
  const keyed = level?.items.filter((i) => i.key !== "");
  return level && keyed.length ? { label: level.label, items: keyed } : null;
}
function hides(items2, typed, readKey2) {
  const ks = [typed].flat().map(readKey2).filter((k) => k !== null);
  return ks.length === 0 || ks.includes("escape") || !items2.some((i) => i.key !== "" && ks.some((k) => k === readKey2(i.key)));
}
var isGroup = (i) => i.items !== void 0 && i.items.length > 0;
function find(menu, name) {
  if (name === "") return null;
  const walk = (level) => {
    for (const i of level) {
      if (i.name === name) return i;
      const inner = i.items && walk(i.items);
      if (inner) return inner;
    }
    return null;
  };
  return walk(menu.items);
}
function entries(menu, shown) {
  const out = [];
  const walk = (level, path, keys) => {
    for (const i of level) {
      const here = [...path, i.label || i.name || i.key];
      const reached = keys && i.key !== "" ? [...keys, shown(i.key)] : null;
      if (i.name !== "") out.push({ name: i.name, label: here.join(" \u203A "), keys: reached ? reached.join(" ") : "", enabled: i.enabled });
      if (i.items) walk(i.items, here, reached);
    }
  };
  walk(menu.items, [], []);
  return out;
}
var Menus = class {
  normal = null;
  alternate = null;
  get(screen) {
    return this[screen];
  }
  set(screen, menu) {
    this[screen] = menu;
  }
  /** The screen changed to `screen`. */
  change(screen) {
    if (screen === "normal") this.alternate = null;
  }
  /** A full reset. */
  clear() {
    this.normal = this.alternate = null;
  }
};
var str = (v, what) => {
  if (v !== void 0 && typeof v !== "string") throw new Error(`${what} is a str`);
  return v ?? "";
};
var isMap3 = (x) => typeof x === "object" && x !== null && !Array.isArray(x) && Object.getPrototypeOf(x) === null;

// src/touch.ts
var DRAG_SLOP = 8;
var LONG_PRESS_MS = 500;
var FLING_DECAY = 0.996;
var REST_MS = 80;
function touchPans(value) {
  const v = value.trim();
  if (!v || v === "auto" || v === "manipulation") return [true, true];
  const words = v.split(/\s+/);
  return [words.some((w) => /^pan-(x|left|right)$/.test(w)), words.some((w) => /^pan-(y|up|down)$/.test(w))];
}
function panBlocked(pans, dx, dy) {
  const ax = Math.abs(dx), ay = Math.abs(dy);
  return ax > ay ? !pans.x : ay > ax ? !pans.y : false;
}
var MAX_TOUCH_AREAS = 256;
function readTouchAreas(bytes) {
  if (!bytes.length) return [];
  const v = read(bytes);
  if (!isMap4(v)) throw new Error("a=touch's body is a map");
  const areas = v.areas;
  if (!Array.isArray(areas)) throw new Error("areas are an array");
  if (areas.length > MAX_TOUCH_AREAS) throw new Error(`at most ${MAX_TOUCH_AREAS} areas`);
  return areas.map((a) => {
    if (!isMap4(a)) throw new Error("an area is a map");
    const int = (k, least) => {
      const n = a[k];
      if (typeof n !== "number" || !Number.isInteger(n)) throw new Error(`an area's ${k} is an int`);
      if (n < least) throw new Error(`an area's ${k} is at least ${least}`);
      return n;
    };
    const drag = a.drag;
    if (drag !== "x" && drag !== "y" && drag !== "xy") throw new Error("an area's drag is x, y or xy");
    return { c: int("c", 0), r: int("r", 0), w: int("w", 1), h: int("h", 1), x: drag !== "y", y: drag !== "x" };
  });
}
function areaAt(areas, col, row) {
  for (let i = areas.length - 1; i >= 0; i--) {
    const a = areas[i];
    if (col >= a.c && col - a.c < a.w && row >= a.r && row - a.r < a.h) return a;
  }
  return null;
}
function areaDrags(area, dx, dy) {
  return panBlocked({ x: !area.x, y: !area.y }, dx, dy);
}
var isMap4 = (x) => typeof x === "object" && x !== null && !Array.isArray(x) && Object.getPrototypeOf(x) === null;
var Touch = class {
  touch = null;
  fling = 0;
  off = [];
  win;
  host;
  exclusive;
  engage;
  /**
   * `exclusive`: every touch here is this Touch's (the cells: xterm.js's own
   * touch handling, which listens on the document, never sees it). Otherwise
   * (a surface) a touch is left alone until it moves. `engage`, if given,
   * says on a touch's start whether it is this Touch's at all: one it
   * declines is the browser's, from start to end.
   */
  constructor(target, win, host, exclusive = false, engage) {
    this.win = win;
    this.host = host;
    this.exclusive = exclusive;
    this.engage = engage;
    const on = (type, f, passive) => {
      const l = (e) => f(e);
      target.addEventListener(type, l, { capture: true, passive });
      this.off.push(() => target.removeEventListener(type, l, { capture: true }));
    };
    on("touchstart", (e) => this.start(e), true);
    on("touchmove", (e) => this.move(e), false);
    on("touchend", (e) => this.end(e), false);
    on("touchcancel", (e) => {
      this.keep(e);
      this.drop();
    }, true);
  }
  dispose() {
    this.stopFling();
    for (const f of this.off) f();
  }
  /** The touch under way is the host's from now to its lift (a long press
   * it took, SPEC §9.1): it scrolls nothing, and its lift is cancelled, so
   * the browser makes no click of it. */
  take() {
    if (this.touch) this.touch.held = true;
  }
  keep(e) {
    if (this.exclusive) e.stopPropagation();
  }
  /** The touch under way ends without its lift: a drag the host holds
   * hears it. */
  drop() {
    const s = this.touch;
    this.touch = null;
    if (s?.held) this.host.dragCancel?.();
  }
  start(e) {
    this.keep(e);
    this.stopFling();
    const t = e.touches.length === 1 ? e.touches[0] : void 0;
    if (!t || this.engage && !this.engage(e)) {
      this.drop();
      return;
    }
    const [x, y] = this.host.toPage(t.clientX, t.clientY);
    const begin = {
      target: e.target,
      x: t.clientX,
      y: t.clientY,
      time: e.timeStamp,
      shiftKey: e.shiftKey,
      ctrlKey: e.ctrlKey,
      altKey: e.altKey,
      metaKey: e.metaKey
    };
    this.touch = { id: t.identifier, x0: x, y0: y, x, y, t: e.timeStamp, t0: e.timeStamp, vx: 0, vy: 0, dragging: false, held: false, begin };
  }
  move(e) {
    this.keep(e);
    const s = this.touch;
    if (!s) return;
    if (e.touches.length !== 1) {
      this.drop();
      return;
    }
    const t = [...e.changedTouches].find((c) => c.identifier === s.id);
    if (!t) return;
    if (s.held) {
      e.preventDefault();
      this.host.dragMove?.(t.clientX, t.clientY, e);
      return;
    }
    const [x, y] = this.host.toPage(t.clientX, t.clientY);
    if (!s.dragging && Math.hypot(x - s.x0, y - s.y0) < DRAG_SLOP) return;
    if (!s.dragging && this.host.claim) {
      const claimed = this.host.claim(s.x0 - x, s.y0 - y, e, s.begin);
      if (!claimed) {
        this.touch = null;
        return;
      }
      if (claimed === "drag") {
        s.held = true;
        e.preventDefault();
        return;
      }
    }
    s.dragging = true;
    e.preventDefault();
    const dx = x - s.x;
    const dy = y - s.y;
    const dt = Math.max(1, e.timeStamp - s.t);
    s.vx = 0.7 * (dx / dt) + 0.3 * s.vx;
    s.vy = 0.7 * (dy / dt) + 0.3 * s.vy;
    s.x = x;
    s.y = y;
    s.t = e.timeStamp;
    this.host.scroll(-dx, -dy, x, y);
  }
  end(e) {
    this.keep(e);
    const s = this.touch;
    const t = s && [...e.changedTouches].find((c) => c.identifier === s.id);
    if (!s || !t) return;
    this.touch = null;
    if (s.held) {
      e.preventDefault();
      this.host.dragEnd?.(t.clientX, t.clientY, e);
      return;
    }
    if (s.dragging) {
      if (e.timeStamp - s.t < REST_MS) this.startFling(-s.vx, -s.vy, s.x, s.y);
      return;
    }
    if (this.host.tap && e.timeStamp - s.t0 < LONG_PRESS_MS) {
      e.preventDefault();
      this.host.tap(s.x, s.y);
    }
  }
  startFling(vx, vy, x, y) {
    let last = this.win.performance.now();
    const step2 = (now) => {
      const dt = now - last;
      last = now;
      const decay = Math.pow(FLING_DECAY, dt);
      vx *= decay;
      vy *= decay;
      if (Math.abs(vx) < 0.02 && Math.abs(vy) < 0.02) {
        this.fling = 0;
        return;
      }
      this.host.scroll(vx * dt, vy * dt, x, y);
      this.fling = this.win.requestAnimationFrame(step2);
    };
    this.fling = this.win.requestAnimationFrame(step2);
  }
  /** A key or a program's scroll op takes over from this momentum. */
  stopFling() {
    if (this.fling) this.win.cancelAnimationFrame(this.fling);
    this.fling = 0;
  }
};

// src/steps.ts
function parseSteps(value) {
  if (value === null) return null;
  const words = value.trim().split(/\s+/);
  if (words.length > 2 || !words.every((w) => /^[0-9]+$/.test(w))) return null;
  const [x, y = 0] = words.map(Number);
  if (!Number.isSafeInteger(x) || !Number.isSafeInteger(y) || x === 0 && y === 0) return null;
  return [x, y];
}
function stepAt(offset, size, count) {
  if (!(size > 0)) return 0;
  const at = Math.min(Math.max(offset / size, 0), 1) * count;
  return Math.min(Math.floor(at + 0.5), count);
}

// src/view.ts
var style = (el) => el.ownerDocument.defaultView.getComputedStyle(el);
function slack(edge, cell, dpr) {
  return 0.5 / dpr + Math.abs(edge / cell) / 60;
}
function containsPositioned(s, pos) {
  if (pos === "absolute" && s.position !== "static") return true;
  const v = (p) => s.getPropertyValue(p);
  return v("transform") !== "none" || v("perspective") !== "none" || v("filter") !== "none" || /\b(layout|paint|strict|content)\b/.test(v("contain")) || /\b(transform|perspective|filter)\b/.test(v("will-change")) || /\b(size|inline-size)\b/.test(v("container-type"));
}
function holders(el) {
  const doc = el.ownerDocument;
  const root = doc.documentElement;
  const rs = style(root);
  const fromBody = rs.overflowX === "visible" && rs.overflowY === "visible";
  const boxes = [];
  let pos = style(el).position;
  for (let a = el.parentElement; a && a !== root; a = a.parentElement) {
    const s = style(a);
    if (s.display === "contents") continue;
    if ((pos === "absolute" || pos === "fixed") && !containsPositioned(s, pos)) continue;
    pos = s.position;
    if (a === doc.body && fromBody) continue;
    boxes.push(a);
  }
  return { boxes, viewport: pos !== "fixed" };
}
function clips(s, axis) {
  if (s.display === "inline") return false;
  return (axis === "x" ? s.overflowX : s.overflowY) !== "visible" || /\b(paint|strict|content)\b/.test(s.getPropertyValue("contain"));
}
function port(el) {
  const root = el.ownerDocument.documentElement;
  if (el === root) return { left: 0, top: 0, right: root.clientWidth, bottom: root.clientHeight };
  const r = el.getBoundingClientRect();
  const left = r.left + el.clientLeft;
  const top = r.top + el.clientTop;
  return { left, top, right: left + el.clientWidth, bottom: top + el.clientHeight };
}
function scrolls(box, axis, axes) {
  if (!(axes & (axis === "y" ? 1 : 2))) return false;
  const doc = box.ownerDocument;
  const prop = axis === "y" ? "overflowY" : "overflowX";
  if (box !== doc.documentElement) return /auto|scroll/.test(style(box)[prop]);
  const rs = style(box);
  const fromBody = rs.overflowX === "visible" && rs.overflowY === "visible";
  return !/hidden|clip/.test((fromBody ? style(doc.body) : rs)[prop]);
}
function range(box, axis) {
  const span = Math.max(0, axis === "y" ? box.scrollHeight - box.clientHeight : box.scrollWidth - box.clientWidth);
  return axis === "x" && style(box).direction === "rtl" ? [-span, 0] : [0, span];
}
function nearest(e0, e1, p0, p1, tol) {
  const before = e0 < p0 - tol;
  const after = e1 > p1 + tol;
  if (before === after) return 0;
  const fits = e1 - e0 <= p1 - p0;
  return before === fits ? e0 - p0 : e1 - p1;
}
function reveal(el, where, axes, tol) {
  const doc = el.ownerDocument;
  const { boxes, viewport } = holders(el);
  if (viewport) boxes.push(doc.documentElement);
  const b = el.getBoundingClientRect();
  let dx = 0;
  let dy = 0;
  for (const box of boxes) {
    const along = ["y", "x"].filter((axis) => scrolls(box, axis, axes));
    if (!along.length) continue;
    const p = port(box);
    const to = { left: box.scrollLeft, top: box.scrollTop };
    let moves = false;
    for (const axis of along) {
      const [e0, e1, p0, p1] = axis === "y" ? [b.top - dy, b.bottom - dy, p.top, p.bottom] : [b.left - dx, b.right - dx, p.left, p.right];
      const by = axis === "y" && where === "start" ? e0 - p0 : axis === "y" && where === "end" ? e1 - p1 : nearest(e0, e1, p0, p1, tol);
      if (Math.abs(by) <= tol) continue;
      const [min, max] = range(box, axis);
      const at = axis === "y" ? box.scrollTop : box.scrollLeft;
      const next = Math.min(max, Math.max(min, at + by));
      if (next === at) continue;
      if (axis === "y") [to.top, dy] = [next, dy + next - at];
      else [to.left, dx] = [next, dx + next - at];
      moves = true;
    }
    if (moves) box.scrollTo({ left: to.left, top: to.top, behavior: "auto" });
  }
}
function rendered(el) {
  const check = el.checkVisibility;
  if (check) return check.call(el, { visibilityProperty: true, checkVisibilityCSS: true });
  return el.getClientRects().length > 0 && style(el).visibility === "visible";
}
function shows(el, shown, tol) {
  const clip = { ...shown };
  for (const box of holders(el).boxes) {
    const s = style(box);
    const [x, y] = [clips(s, "x"), clips(s, "y")];
    if (!x && !y) continue;
    const p = port(box);
    if (x) [clip.left, clip.right] = [Math.max(clip.left, p.left), Math.min(clip.right, p.right)];
    if (y) [clip.top, clip.bottom] = [Math.max(clip.top, p.top), Math.min(clip.bottom, p.bottom)];
  }
  const b = el.getBoundingClientRect();
  const along = (axis, e0, e1, c0, c1) => {
    const t = Math.max(tol(e0, axis), tol(e1, axis));
    if (e1 - e0 > 2 * t) return Math.min(e1, c1) - Math.max(e0, c0) > t;
    const m = (e0 + e1) / 2;
    return m >= c0 - t && m < c1 - t;
  };
  return along("x", b.left, b.right, clip.left, clip.right) && along("y", b.top, b.bottom, clip.top, clip.bottom);
}
function atEnd(doc, axes) {
  const root = doc.documentElement;
  if (!scrolls(root, "y", axes)) return true;
  return root.scrollTop >= root.scrollHeight - root.clientHeight - 1;
}

// src/resolver.ts
var DROP = /* @__PURE__ */ new Set(["script", "iframe", "frame", "frameset", "object", "embed", "applet", "base", "meta", "portal", "fencedframe"]);
var DROP_ATTRS = /* @__PURE__ */ new Set(["srcdoc", "autofocus", "http-equiv", "ping", "nonce", "integrity"]);
var Resolver = class {
  /** The program's value of each attribute the resolver rewrote. */
  orig = /* @__PURE__ */ new WeakMap();
  /** The program's text of each `<style>` whose CSS names a resource. */
  css = /* @__PURE__ */ new WeakMap();
  /** Elements that name resources, and which. */
  refs = /* @__PURE__ */ new Map();
  /** Attributes the host set and the program did not write (a detached
   * surface's `disabled`, SPEC §5.5): live, never reported. */
  own = /* @__PURE__ */ new WeakMap();
  store;
  /** The document's base URL and effective network policy (set per a=doc). */
  ctx;
  constructor(store) {
    this.store = store;
  }
  /** Cleans and resolves an inert subtree, in place. */
  adopt(root) {
    const walk = (node) => {
      if (node.nodeType === 1) {
        const el = node;
        if (!this.allowed(el)) {
          el.remove();
          return;
        }
        this.cleanAttributes(el);
        if (el.localName === "style") this.setCss(el, el.textContent ?? "");
      }
      for (const child of Array.from(node.childNodes)) walk(child);
      if (node.nodeType === 1 && node.localName === "template") {
        walk(node.content);
      }
    };
    walk(root);
  }
  allowed(el) {
    const name = el.localName;
    if (DROP.has(name)) return false;
    if (name === "link") {
      const rel = (el.getAttribute("rel") ?? "").toLowerCase().split(/\s+/);
      return rel.includes("stylesheet");
    }
    return true;
  }
  cleanAttributes(el) {
    for (const a of Array.from(el.attributes)) {
      const n = a.name.toLowerCase();
      if (n.startsWith("on") || DROP_ATTRS.has(n)) {
        el.removeAttribute(a.name);
      } else if (isUrlAttr(n) || n === "style") {
        this.set(el, a.name, a.value);
      }
    }
  }
  /** An attribute as the program wrote it. */
  get(el, name) {
    if (this.own.get(el)?.has(name)) return null;
    return this.orig.get(el)?.get(name) ?? el.getAttribute(name);
  }
  /** Every attribute as the program wrote it, in order. */
  attributes(el) {
    const o = this.orig.get(el);
    const own = this.own.get(el);
    const out = [];
    for (const a of Array.from(el.attributes)) if (!own?.has(a.name)) out.push([a.name, o?.get(a.name) ?? a.value]);
    return out;
  }
  /** Sets an attribute of the host's own, which the program does not see. */
  setOwn(el, name, value) {
    el.setAttribute(name, value);
    let own = this.own.get(el);
    if (!own) this.own.set(el, own = /* @__PURE__ */ new Set());
    own.add(name);
  }
  /** Gives an attribute the program wrote a live value of the host's,
   * while the program's stays the one reported, until `unmask`. */
  mask(el, name, live) {
    const value = this.get(el, name);
    if (value === null) return;
    let o = this.orig.get(el);
    if (!o) this.orig.set(el, o = /* @__PURE__ */ new Map());
    o.set(name, value);
    el.setAttribute(name, live);
  }
  /** The program's value again, after `mask`. */
  unmask(el, name) {
    const o = this.orig.get(el);
    const value = o?.get(name);
    if (value === void 0) return;
    o.delete(name);
    el.setAttribute(name, value);
  }
  /** Removes an attribute, if it is the host's own. */
  removeOwn(el, name) {
    if (!this.own.get(el)?.delete(name)) return;
    el.removeAttribute(name);
  }
  /** Sets an attribute to the program's value, resolving URLs and CSS. */
  set(el, name, value) {
    const n = name.toLowerCase();
    if (n.startsWith("on") || DROP_ATTRS.has(n)) return;
    const names = /* @__PURE__ */ new Set();
    let live = value;
    if (n === "style") live = this.store.rewriteCss(value, names, this.ctx);
    else if (isUrlAttr(n)) {
      const d = directiveFor(el.localName, n, el.getAttribute("rel") ?? "");
      live = n === "srcset" ? this.srcset(value, names) : this.store.resolveUrl(value, names, this.ctx, d);
    }
    if ((el.localName === "a" || el.localName === "area") && n === "href") live = value;
    try {
      el.setAttribute(name, live);
    } catch {
      return;
    }
    this.own.get(el)?.delete(n);
    this.remember(el, name, value, live, names);
  }
  remove(el, name) {
    el.removeAttribute(name);
    this.orig.get(el)?.delete(name);
    this.own.get(el)?.delete(name.toLowerCase());
  }
  /** A `<style>`'s CSS as the program wrote it. */
  cssOf(el) {
    return this.css.get(el) ?? el.textContent ?? "";
  }
  setCss(el, text) {
    const names = /* @__PURE__ */ new Set();
    const live = this.store.rewriteCss(text, names, this.ctx);
    if (el.textContent !== live) el.textContent = live;
    if (live !== text) this.css.set(el, text);
    else this.css.delete(el);
    if (names.size > 0) this.track(el, names);
  }
  /** Re-resolves everything that names one of `names` (a resource changed);
   * whether anything did. */
  refresh(names) {
    let any = false;
    for (const [el, uses] of Array.from(this.refs)) {
      if (!el.isConnected) {
        this.refs.delete(el);
        continue;
      }
      if (![...uses].some((u) => names.has(u))) continue;
      any = true;
      this.refs.delete(el);
      const o = this.orig.get(el);
      if (o) for (const [attr, value] of Array.from(o)) this.set(el, attr, value);
      const css = this.css.get(el);
      if (css !== void 0) this.setCss(el, css);
    }
    return any;
  }
  srcset(value, names) {
    return value.split(",").map((part) => {
      const [url, ...rest] = part.trim().split(/\s+/);
      return [this.store.resolveUrl(url ?? "", names, this.ctx, "img-src"), ...rest].join(" ");
    }).join(", ");
  }
  remember(el, name, value, live, names) {
    let o = this.orig.get(el);
    if (value !== live) {
      if (!o) this.orig.set(el, o = /* @__PURE__ */ new Map());
      o.set(name, value);
    } else {
      o?.delete(name);
    }
    if (names.size > 0) this.track(el, names);
  }
  track(el, names) {
    const uses = this.refs.get(el) ?? /* @__PURE__ */ new Set();
    for (const n of names) uses.add(n);
    this.refs.set(el, uses);
  }
};

// src/history.ts
var LIMIT = 100;
var History = class {
  /** The states to go back to, the latest last. */
  undo = [];
  /** The states undone, to go forward to. */
  redo = [];
  /** The run the last edit made, which an edit of the same kind goes on
   * with while the field is as that edit left it. */
  run = null;
  /** The field as the last edit, undo or redo left it. A value that
   * differs was set by the program, which clears the history. */
  after = null;
};
function stateOf(el) {
  const n = el.value.length;
  return { value: el.value, start: el.selectionStart ?? n, end: el.selectionEnd ?? n, backward: el.selectionDirection === "backward" };
}
var same = (a, b) => a.value === b.value && a.start === b.start && a.end === b.end;
function clear(h) {
  h.undo = [];
  h.redo = [];
  h.run = null;
  h.after = null;
}
function edited(h, before, after, run) {
  if (after.value === before.value) {
    if (run === null) h.run = null;
    return;
  }
  if (h.after !== null && h.after.value !== before.value) clear(h);
  const r = run === "deleting" && before.start !== before.end ? null : run;
  if (r === null || h.run !== r || h.after === null || !same(h.after, before)) {
    h.undo.push(before);
    if (h.undo.length > LIMIT) h.undo.shift();
  }
  h.redo = [];
  h.run = r;
  h.after = after;
}
function step(h, now, redo) {
  if (h.after === null || h.after.value !== now.value) {
    clear(h);
    return null;
  }
  const [from, to] = redo ? [h.redo, h.undo] : [h.undo, h.redo];
  const s = from.pop();
  if (s === void 0) return null;
  to.push(h.after);
  h.after = s;
  h.run = null;
  return s;
}
function runOf(inputType) {
  if (inputType === "insertText" || inputType === "insertLineBreak" || inputType === "insertParagraph") return "typing";
  if (inputType === "deleteContentBackward" || inputType === "deleteContentForward") return "deleting";
  return null;
}

// src/surface.ts
function csp(host, fonts = "") {
  const with_ = (fixed, d) => [fixed, cspSources(host, d)].filter(Boolean).join(" ");
  return [
    "default-src 'none'",
    with_("img-src data: blob:", "img-src"),
    with_("media-src data: blob:", "media-src"),
    with_(["font-src data: blob:", cspPath(fonts)].filter(Boolean).join(" "), "font-src"),
    with_("style-src 'unsafe-inline' blob:", "style-src"),
    "form-action 'none'",
    "base-uri http: https:"
  ].join("; ");
}
function cspPath(url) {
  return url.replace(/[\s;,]/g, (c) => "%" + c.charCodeAt(0).toString(16).toUpperCase().padStart(2, "0"));
}
function baseOf(doc) {
  const href = doc.querySelector("base[href]")?.getAttribute("href") ?? "";
  try {
    const u = new URL(href);
    if (u.protocol === "http:" || u.protocol === "https:") return u.href;
  } catch {
  }
  return NO_BASE;
}
var SCROLL_ACTIONS2 = {
  "scroll-up": { axis: "y", sign: -1, by: "line" },
  "scroll-down": { axis: "y", sign: 1, by: "line" },
  "scroll-left": { axis: "x", sign: -1, by: "line" },
  "scroll-right": { axis: "x", sign: 1, by: "line" },
  "scroll-page-up": { axis: "y", sign: -1, by: "page" },
  "scroll-page-down": { axis: "y", sign: 1, by: "page" },
  "scroll-half-page-up": { axis: "y", sign: -1, by: "half" },
  "scroll-half-page-down": { axis: "y", sign: 1, by: "half" },
  "scroll-start": { axis: "y", sign: -1, by: "end" },
  "scroll-end": { axis: "y", sign: 1, by: "end" }
};
var SCROLL_ENDS = typeof window !== "undefined" && "onscrollend" in window;
var SCROLL_REST_MS = 150;
var SCROLL_REST_LONG_MS = 600;
var LINE_PX = 40;
var PAGE_FRACTION = 0.875;
var FIELD_TYPES = /* @__PURE__ */ new Set(["text", "email", "password", "search", "tel", "url", "number"]);
var DATE_TYPES = /* @__PURE__ */ new Set(["date", "datetime-local", "month", "time", "week"]);
var NO_CARET = /* @__PURE__ */ new Set(["email", "number"]);
var NUMBER_CHARS = /^[0-9.,+\-eE]$/;
var mac = typeof navigator !== "undefined" && /Mac|iPhone|iPad/.test(navigator.platform || navigator.userAgent);
var LAYOUT_STYLES = [
  "font-family",
  "font-size",
  "font-weight",
  "font-style",
  "font-variant",
  "font-stretch",
  "font-feature-settings",
  "font-kerning",
  "letter-spacing",
  "word-spacing",
  "line-height",
  "text-transform",
  "text-indent",
  "tab-size",
  "white-space",
  "overflow-wrap",
  "word-break",
  "direction",
  "text-align",
  "padding-left",
  "padding-right",
  "padding-top",
  "padding-bottom"
];
var CONTROLS = "input, select, textarea, button";
var HYPERLINK = "data-xterm-hotty-hyperlink";
var DETACHED_CSS = `@layer hotty-host {
* { cursor: auto !important; }
a[href], a[href] * { cursor: text !important; }
a[href][${HYPERLINK}], a[href][${HYPERLINK}] * { cursor: pointer !important; }
}`;
var Surface = class {
  name;
  box;
  frame;
  doc;
  resolver;
  deltas;
  cols = 80;
  rows = 24;
  /** The part of the surface on screen, in cells (SPEC §5.2). */
  win = { x: 0, y: 0, w: 80, h: 24 };
  autoRows = false;
  host;
  hostStyle;
  /** Set while the program moves focus, so no `focus` event echoes back. */
  programFocus = false;
  /** The document's base URL (SPEC §7.3). */
  base = NO_BASE;
  keyboard = false;
  /** `a=focus` came while the page's focus was outside the terminal: the
   * surface has the keyboard, and takes the browser's focus, at this
   * element id ("" for none), once the terminal has it. Null when nothing
   * waits. */
  waiting = null;
  /** The element the program knows has focus: the last `focus` named it,
   * or the program's `a=focus` gave it (SPEC §10.1). */
  told = null;
  /** Detached (SPEC §5.5): nothing in the surface reaches the program. */
  detachedState = false;
  /** A press is on something that takes no focus (SPEC §10.1): the focus
   * the browser gives the frame for it is not the surface's. */
  pressNothing = false;
  /** The element a press focuses, until the browser is done with it. */
  pressed = null;
  /** A press of another button than the primary one, until the browser is
   * done with it: it neither takes nor gives back the keyboard. */
  auxPress = false;
  settling = null;
  /** A drag under way (SPEC §9.1). */
  drag = null;
  /** A drag just ended at a release: it reported its own click, so the
   * browser's, which follows in the same task, is not reported. */
  dragReleased = false;
  /** The program asked for presses (`p=1`, SPEC §5.2): the surface's, held
   * until it sends `p=0`. */
  presses = false;
  /** And for `fit` (`f=1`), held in the same way. */
  fits = false;
  /** A mouse's or a pen's pointerdown came first: the mousedown that
   * follows is the same press, not a tap's. */
  pointerPress = false;
  /** The pointer of a press with Alt held (SPEC §9.2), until its release:
   * the gesture is the program's. */
  altPointer = null;
  /** Its button: 0, or 2 for a secondary press (§9). */
  altButton = 0;
  finger = null;
  /** A touch that may become a long press on an element that takes it
   * (SPEC §9.1): where and when it began, and the timer that makes it one.
   * `done` once it has: the touch is the surface's to its lift. */
  longPress = null;
  /** A secondary press (SPEC §9) taken at its pointerdown: the mousedown it
   * makes is cancelled and the `contextmenu` that follows is the surface's
   * to cancel. Cleared by the next pointerdown. */
  secondaryTaken = false;
  /** The last thing the user pressed, until a key: what a `contextmenu`
   * with no secondary pointerdown before it came from. The keyboard's (the
   * Menu key, Shift+F10) is a key for the program (§10.2), not a press. */
  lastDown = null;
  /** Such a gesture just ended at a release: the browser's click that
   * follows in the same task is not the surface's. */
  altReleased = false;
  /** One cell, in CSS pixels: a drag's detail counts cells (SPEC §9.1). */
  cellW = 9;
  cellH = 17;
  css = "";
  /** Keys whose keydown went to the program, so their keyup follows. */
  forwarded = /* @__PURE__ */ new Set();
  /** A run of row moves in a field (SPEC §10.2): the caret it left, and the
   * place along the row it keeps. Anything else that moves the caret ends
   * it. */
  goal = null;
  /** Each text field's history of edits (SPEC §10.2, undo). */
  histories = /* @__PURE__ */ new WeakMap();
  /** The edit under way, from its `beforeinput` or from `replace`, which
   *  its `input` puts in the field's history. */
  pending = null;
  /** `replace` is editing: the edit is pending already. */
  editing = false;
  /** The field shown as `text` while focused (`NO_CARET`). */
  masked = null;
  /** Each control's `rev` (SPEC §6.4): the changes the user made to it
   * since it entered the document. A radio group's is its buttons'
   * highest (`revOf`). */
  revs = /* @__PURE__ */ new WeakMap();
  /** Text fields the user edited since they were focused or last sent
   * `change`: focus leaving commits only theirs (SPEC §6.4, no echo). */
  dirty = /* @__PURE__ */ new WeakSet();
  /** A text field's value when it last committed, or was focused: a
   * commit sends `change` only for a value changed since (SPEC §9). The
   * browser keeps its own, which the `submit` action cannot move. */
  committed = /* @__PURE__ */ new WeakMap();
  /** Controls whose `input` counted a pick or a value, which the `change`
   * after it carries without counting it again. */
  picked = /* @__PURE__ */ new WeakSet();
  /** A text field's selection while it is not focused (SPEC §10.2), which
   * `a=focus` gives back; none is the caret at the end of its value. */
  kept = /* @__PURE__ */ new WeakMap();
  /** What the program last heard of a field with `data-on~=caret`: its
   * selection, in bytes, and its `rev` (SPEC §9, `caret`). None is the
   * caret at the end and `rev` 0, as the field entered the document. */
  heard = /* @__PURE__ */ new WeakMap();
  /** The field whose selection the next frame checks (`caret`), and that
   * frame. */
  caretField = null;
  caretFrame = 0;
  /** The field an input method composes text in: `a=field` is stale
   * meanwhile (SPEC §6.4). */
  composing = null;
  /** `a=field` is changing a control: the events that fire are not the
   * user's. */
  applying = false;
  /** Tab is moving focus: the field it reaches selects its text. */
  tabbing = false;
  /** A prefix is on (SPEC §10.5): every key reaches the program, whatever is
   * focused, until the program sends `a=focus` or `a=blur`, the surface
   * loses the keyboard, or the user focuses an element with a press. */
  prefix = false;
  /** The rows the program heard last, while the placement asked for `fit`
   * (`f=1`, SPEC §5.2); null when it did not. */
  fitRows = null;
  /** The height may have changed since the last check. */
  fitDirty = false;
  /** The frame a check waits for. */
  fitFrame = 0;
  fitObserver = null;
  /** The axes the document asked to scroll along (`scroll`, SPEC §5.1): 1
   * vertically, 2 horizontally, 3 both; 0, nothing scrolls. */
  axes = 0;
  /** Where the touch drag under way goes, once it has a direction: "self",
   * the document, but scrolled by the surface, since the document's
   * `touch-action` would stop the browser's own pan (SPEC §9.1: it changes
   * nothing there). */
  touchRoute = "terminal";
  /** The box a touch drag scrolls itself ("self"). */
  touchBox = null;
  /** The box the wheel gesture under way scrolls, if it went to the
   * document: the gesture moves no other (`scrollWheel`). */
  wheelBox = null;
  /** Some element may carry `CLIP`. */
  clipping = false;
  /** Placed (SPEC §5.2), from `a=place` until the placement goes: drawn or
   * not, as the terminal scrolls it in and out of view (§5.4). */
  placed = false;
  /** What the program heard last of which elements are in view (`view`,
   * SPEC §9.5), as `checkView` compares it, across placements; null when it
   * has heard nothing of this document. */
  viewSent = null;
  /** What is in view may have changed since the last check. */
  viewDirty = false;
  /** The frame a check waits for. */
  viewFrame = 0;
  /** What notices the elements that report whether they are in view
   * changing size, or coming into the window or leaving it, whatever moved
   * them, once one asks; and the elements they watch. */
  viewObservers = null;
  viewWatched = /* @__PURE__ */ new Set();
  /** The boxes that scroll now (the document, for its root), until each
   * comes to rest (`scrollend`), or, where the browser tells no rest, until
   * none has moved for a while (`restTimer`). */
  moving = /* @__PURE__ */ new Set();
  /** A smooth key/op scroll starts its next step at this destination,
   * not at the position the browser is drawing (SPEC §6.1). */
  scrollDestinations = /* @__PURE__ */ new Map();
  restTimer = null;
  constructor(name, host, hostCss2, scheme) {
    this.name = name;
    this.host = host;
    this.box = document.createElement("div");
    this.box.className = "hotty-surface";
    this.box.dataset.surface = name;
    Object.assign(this.box.style, {
      position: "absolute",
      left: "0px",
      top: "0px",
      overflow: "hidden",
      pointerEvents: "auto",
      visibility: "hidden"
    });
    this.frame = document.createElement("iframe");
    this.frame.setAttribute("sandbox", "allow-same-origin allow-forms");
    this.frame.setAttribute("title", `hotty surface ${name}`);
    Object.assign(this.frame.style, {
      border: "0",
      position: "absolute",
      left: "0px",
      top: "0px",
      width: "100%",
      height: "100%",
      display: "block",
      colorScheme: scheme
    });
    this.box.append(this.frame);
    host.layer.append(this.box);
    const doc = this.frame.contentDocument;
    if (!doc) throw new Error("surface iframe has no document");
    doc.open();
    doc.write(
      `<!doctype html><html><head><meta charset="utf-8"><meta http-equiv="Content-Security-Policy" content="${csp(host.policy, host.fonts)}"><base href="${NO_BASE}"><style id="hotty-host"></style></head><body></body></html>`
    );
    doc.close();
    this.doc = doc;
    this.hostStyle = doc.getElementById("hotty-host");
    this.css = hostCss2;
    this.restyle();
    this.resolver = new Resolver(host.store);
    this.deltas = new Deltas(doc, this.resolver, (el, where) => this.reveal(el, where), (el) => this.valueSet(el), (el, action) => this.scrollAction(el, action));
    this.listen();
  }
  setHostCss(css, scheme) {
    this.css = css;
    this.restyle();
    if (this.frame.style.colorScheme !== scheme) this.frame.style.colorScheme = scheme;
    this.relayout();
  }
  restyle() {
    const css = [this.css, scrollCss(this.axes, this.host.pageScrolls), ...this.detachedState ? [DETACHED_CSS] : []].join("\n");
    if (this.hostStyle.textContent !== css) this.hostStyle.textContent = css;
    this.clip();
  }
  /** Replaces the whole document (`a=doc`): its head's styles and its body.
   * `detached` (`d=1`) detaches the surface first; without it, the surface
   * is the program's again (SPEC §5.1, §5.5). `axes` (`scroll`) are the
   * axes it scrolls along, 0 for none (§5.3). */
  setDocument(html, detached = false, axes = 0) {
    this.scrollDestinations.clear();
    if (detached) this.detach();
    const scrolled = this.axes !== 0;
    this.axes = axes;
    const parsed = new DOMParser().parseFromString(html, "text/html");
    this.base = baseOf(parsed);
    const request = parse(parsed.querySelector('meta[name="hotty-network" i]')?.getAttribute("content") ?? "");
    this.resolver.ctx = { base: this.base, policy: intersect(this.host.policy, request) };
    const ours = this.doc.head.querySelector("base");
    if (ours && ours.getAttribute("href") !== this.base) ours.setAttribute("href", this.base);
    this.resolver.adopt(parsed.documentElement);
    const head = this.doc.head;
    for (const n of Array.from(head.childNodes)) {
      const el = n;
      const ours2 = el.nodeType === 1 && (el.id === "hotty-host" || el.localName === "meta" || el.localName === "base");
      if (!ours2) n.remove();
    }
    head.append(...Array.from(parsed.head.childNodes));
    this.copyAttributes(parsed.documentElement, this.doc.documentElement);
    this.copyAttributes(parsed.body, this.doc.body);
    this.doc.body.replaceChildren(...Array.from(parsed.body.childNodes));
    this.detachedState = detached;
    this.sync();
    if (scrolled) {
      this.doc.documentElement.scrollTo(0, 0);
      this.doc.body.scrollTo(0, 0);
    }
    this.viewSent = null;
    this.relayout();
  }
  /** A delta (SPEC §6.1): its ops in order, as one unit, and the first that
   * failed, if any. On a detached surface, the controls it adds are
   * disabled too, and so is one whose `disabled` it removes. */
  delta(ops) {
    if (ops.some((o) => o.op !== "scroll")) this.scrollDestinations.clear();
    try {
      return this.deltas.apply(ops);
    } finally {
      if (this.detachedState) this.sync();
      else this.clip();
      this.relayout();
    }
  }
  // --- Ownership (SPEC §5.5) -----------------------------------------------
  get detached() {
    return this.detachedState;
  }
  /** `a=detach`: nothing in the surface reaches the program any more. A
   * surface that has the keyboard gives it back silently, with no `change`
   * and no `blur`, and its form controls act disabled. */
  detach() {
    if (!this.detachedState) {
      this.detachedState = true;
      this.dropDrag();
      const had = this.keyboard && this.waiting === null || this.frameFocused();
      this.prefix = false;
      this.keyboard = false;
      this.waiting = null;
      this.told = null;
      this.focusedControl()?.blur();
      if (had) this.giveBack();
    }
    this.sync();
  }
  /** Disables the form controls of a detached surface, as if each had the
   * `disabled` attribute, or enables those it disabled, and marks its
   * hyperlinks for the host's stylesheet. The attributes are the host's
   * own: the document reports the program's (SPEC §16). */
  sync() {
    const detached = this.detachedState;
    for (const el of Array.from(this.doc.querySelectorAll(CONTROLS))) {
      if (!detached) this.resolver.removeOwn(el, "disabled");
      else if (!el.hasAttribute("disabled")) this.resolver.setOwn(el, "disabled", "");
    }
    for (const a of Array.from(this.doc.querySelectorAll("a"))) {
      if (detached && this.isHyperlink(a)) {
        if (!a.hasAttribute(HYPERLINK)) this.resolver.setOwn(a, HYPERLINK, "");
      } else this.resolver.removeOwn(a, HYPERLINK);
    }
    this.restyle();
  }
  copyAttributes(from, to) {
    for (const [name] of this.resolver.attributes(to)) {
      if (!from.hasAttribute(name)) this.resolver.remove(to, name);
    }
    for (const [name, value] of this.resolver.attributes(from)) this.resolver.set(to, name, value);
  }
  /** Rows the content needs at `cols` columns (`r=auto`). */
  contentRows(cols, cellW, cellH) {
    Object.assign(this.box.style, { display: "block", visibility: "hidden" });
    return this.neededRows(cols, cellW, cellH);
  }
  /**
   * Rows the document needs at `cols` columns: `r=auto`'s, and `fit`'s
   * (SPEC §5.2). The frame is laid out at the width a placement gives it
   * and a height of 1px, so the viewport adds nothing, then restored in the
   * same task: nothing is drawn in between.
   */
  neededRows(cols, cellW, cellH) {
    const f = this.frame.style;
    const [width, height] = [f.width, f.height];
    const root = this.doc.documentElement;
    if (this.axes) this.resolver.setOwn(root, MEASURE, "");
    Object.assign(f, { width: `${Math.round(cols * cellW)}px`, height: "1px" });
    const h = root.scrollHeight;
    Object.assign(f, { width, height });
    if (this.axes) this.resolver.removeOwn(root, MEASURE);
    return Math.max(1, Math.min(1e3, Math.ceil(h / cellH)));
  }
  /**
   * In a document that scrolls along one axis only, the elements whose
   * `overflow` along the other is `auto` or `scroll` are marked (`CLIP`), so
   * that the host stylesheet clips that axis as `overflow: hidden` does
   * (SPEC §5.3): no scrollbar, and nothing the user does scrolls it. CSS
   * cannot select by a computed value, so the addon reads it, with its own
   * rule off (`UNCLIPPED`), each time the document or its style may have
   * changed: a document, a delta, the host stylesheet. Style only, no
   * layout, and only for a document with `scroll` of 1 or 2.
   */
  clip() {
    const prop = this.axes === 1 ? "overflowX" : this.axes === 2 ? "overflowY" : null;
    if (!prop && !this.clipping) return;
    const marked = new Set(this.doc.querySelectorAll(`[${CLIP}]`));
    const want = /* @__PURE__ */ new Set();
    if (prop) {
      const root = this.doc.documentElement;
      const win = this.frame.contentWindow;
      this.resolver.setOwn(root, UNCLIPPED, "");
      for (const el of [this.doc.body, ...Array.from(this.doc.body.querySelectorAll("*"))]) {
        if (/auto|scroll/.test(win.getComputedStyle(el)[prop])) want.add(el);
      }
      this.resolver.removeOwn(root, UNCLIPPED);
    }
    for (const el of marked) if (!want.has(el)) this.resolver.removeOwn(el, CLIP);
    for (const el of want) if (!marked.has(el)) this.resolver.setOwn(el, CLIP, "");
    this.clipping = want.size > 0;
  }
  // --- Fit (SPEC §5.2: f=1 on a=place) --------------------------------------
  /**
   * `f=1` on the surface: from `rows`, the placement's own, the program
   * hears `fit` whenever the rows the document needs differ from those it
   * heard last. `null`: the surface did not ask, or has no placement.
   *
   * What can change the height asks for a check in the next frame drawn:
   * a document, a delta, a resource (`refresh`), the host stylesheet (cell
   * size, font), an image, stylesheet or font loading, and, for whatever
   * else does (the user opening a `<details>`, say), the root's and the
   * body's boxes changing size. One check per frame, so one `fit` at most.
   */
  setFit(rows) {
    this.fitRows = rows;
    if (rows === null) {
      this.fitObserver?.disconnect();
      this.fitObserver = null;
      if (this.fitFrame) this.frame.ownerDocument.defaultView?.cancelAnimationFrame(this.fitFrame);
      this.fitFrame = 0;
      this.fitDirty = false;
      return;
    }
    if (!this.fitObserver) {
      this.fitObserver = new ResizeObserver(() => this.refit());
      this.fitObserver.observe(this.doc.documentElement);
      this.fitObserver.observe(this.doc.body);
    }
    this.refit();
  }
  /** The document's height may have changed: a check in the next frame
   * drawn. A surface out of view draws none; it checks once shown. */
  refit() {
    if (this.fitRows === null) return;
    this.fitDirty = true;
    if (this.fitFrame || this.box.style.display === "none") return;
    this.fitFrame = this.frame.ownerDocument.defaultView.requestAnimationFrame(() => {
      this.fitFrame = 0;
      this.checkFit();
    });
  }
  checkFit() {
    if (this.fitRows === null || !this.fitDirty || this.box.style.display === "none") return;
    this.fitDirty = false;
    if (this.detachedState) return;
    const rows = this.neededRows(this.cols, this.cellW, this.cellH);
    if (rows === this.fitRows) return;
    this.fitRows = rows;
    this.emit("fit", "", { r: rows });
  }
  /** A resource changed (`a=res`, `a=del`): what names it resolves again. */
  refresh(names) {
    if (this.resolver.refresh(names)) this.relayout();
  }
  /** The layout may have changed: `fit` and `view` check it in the next
   * frame drawn. */
  relayout() {
    this.refit();
    this.requestView();
  }
  // --- Reveal and view (SPEC §6.1, §9.5) -----------------------------------
  /**
   * A placement begins (`a=place`, SPEC §5.2). Its first `view` goes out
   * once the document is laid out in it and drawn, if what it carries
   * differs from what the program last heard (§9.5). Placing again, a new
   * window, is a new placement.
   */
  beginPlacement() {
    this.endPlacement();
    this.placed = true;
    this.requestView();
  }
  /** The placement goes (`a=hide`, `a=del`, its line gone): no `view` until
   * the next, which reports how what it carries differs from what the
   * program last heard (SPEC §9.5). */
  endPlacement() {
    this.scrollDestinations.clear();
    this.placed = false;
    this.viewDirty = false;
    if (this.viewFrame) this.frame.ownerDocument.defaultView?.cancelAnimationFrame(this.viewFrame);
    this.viewFrame = 0;
    this.viewObservers?.size.disconnect();
    this.viewObservers?.window.disconnect();
    this.viewObservers = null;
    this.viewWatched.clear();
    this.moving.clear();
    if (this.restTimer) clearTimeout(this.restTimer);
    this.restTimer = null;
  }
  /**
   * `reveal` (SPEC §6.1): scrolls the boxes that hold `el`, along the axes
   * the document scrolls. A surface not placed has nothing in view to bring
   * it into, and scrolls nothing. One placed but out of view (§5.4) is laid
   * out unseen for it, as `r=auto` measures, and its offsets stay as it is
   * hidden again (as hiding keeps them).
   */
  reveal(el, where) {
    if (!this.placed || !this.axes) return;
    this.scrollDestinations.clear();
    const box = this.box.style;
    const out = box.display === "none";
    const visibility = box.visibility;
    if (out) Object.assign(box, { display: "block", visibility: "hidden" });
    try {
      reveal(el, where, this.axes, 0.5 / (this.frame.ownerDocument.defaultView?.devicePixelRatio || 1));
    } finally {
      if (out) Object.assign(box, { display: "none", visibility });
    }
  }
  /**
   * What is in view may have changed (SPEC §9.5): a check in the next frame
   * drawn, once every scroll under way has come to rest. One check per
   * frame, so one `view` at most. A placement out of view draws no frame,
   * and checks once it is shown again.
   */
  requestView() {
    if (!this.placed) return;
    this.viewDirty = true;
    if (this.viewFrame || this.moving.size || this.box.style.display === "none") return;
    this.viewFrame = this.frame.ownerDocument.defaultView.requestAnimationFrame(() => {
      this.viewFrame = 0;
      this.checkView();
    });
  }
  /** A box scrolled (SPEC §9.5): what is in view waits for it to rest. */
  viewScrolled(e) {
    if (!this.placed || !e.target) return;
    this.moving.add(e.target);
    this.viewDirty = true;
    if (this.restTimer) clearTimeout(this.restTimer);
    this.restTimer = setTimeout(() => this.rested(), SCROLL_ENDS ? SCROLL_REST_LONG_MS : SCROLL_REST_MS);
  }
  /** A box came to rest (`scrollend`): once all have, what is in view is
   * checked. */
  scrollEnded(e) {
    const el = e.target === this.doc ? this.doc.scrollingElement : e.target;
    if (el && isElement(el)) this.scrollDestinations.delete(el);
    if (e.target && this.moving.delete(e.target) && !this.moving.size) this.rested();
  }
  rested() {
    if (this.restTimer) clearTimeout(this.restTimer);
    this.restTimer = null;
    this.moving.clear();
    this.scrollDestinations.clear();
    this.requestView();
  }
  /**
   * `view` (SPEC §9.5), if what it would carry changed since the program
   * last heard it: every element with `view` in its `data-on` and an id
   * that is in view, with its area, and `end`. A document with none that
   * asks sends none. A detached surface sends none (§5.5).
   */
  checkView() {
    if (!this.placed || !this.viewDirty || this.moving.size || this.box.style.display === "none") return;
    this.viewDirty = false;
    if (this.detachedState) return;
    const asks = Array.from(this.doc.querySelectorAll("[data-on~=view][id]")).filter((el) => el.id !== "");
    this.watchViews(asks);
    if (!asks.length) {
      this.viewSent = null;
      return;
    }
    const inView = /* @__PURE__ */ Object.create(null);
    const shown = this.shownRect();
    const dpr = this.frame.ownerDocument.defaultView?.devicePixelRatio || 1;
    const tol = (edge, axis) => slack(edge, axis === "x" ? this.cellW : this.cellH, dpr);
    for (const el of asks) {
      if (el.id in inView || !el.id.isWellFormed() || !rendered(el) || !shows(el, shown, tol)) continue;
      inView[el.id] = this.areaOf(el);
    }
    const end = atEnd(this.doc, this.axes);
    const key = JSON.stringify([end, Object.keys(inView).sort().map((id) => [id, inView[id]])]);
    if (key === this.viewSent) return;
    this.viewSent = key;
    this.emit("view", "", { in: inView, end });
  }
  /** The placement's window (SPEC §5.2) within the viewport, without its
   * scrollbars, in the frame's pixels: what the user can see of the
   * document. */
  shownRect() {
    const root = this.doc.documentElement;
    const left = Math.round(this.win.x * this.cellW);
    const top = this.win.y * this.cellH;
    return {
      left,
      top,
      right: Math.min(root.clientWidth, left + Math.round(this.win.w * this.cellW)),
      bottom: Math.min(root.clientHeight, top + this.win.h * this.cellH)
    };
  }
  /** The observers watch the elements that ask, as deltas add and remove
   * them, and the root and the body, from the first that asks in the
   * placement. */
  watchViews(asks) {
    if (!asks.length && !this.viewObservers) return;
    if (!this.viewObservers) {
      const w = this.win;
      const [left, top] = [Math.round(w.x * this.cellW), w.y * this.cellH];
      const [right, bottom] = [Math.round(this.cols * this.cellW) - left - Math.round(w.w * this.cellW), (this.rows - w.y - w.h) * this.cellH];
      this.viewObservers = {
        size: new ResizeObserver(() => this.requestView()),
        window: new IntersectionObserver(() => this.requestView(), { root: this.doc, rootMargin: `${-top}px ${-right}px ${-bottom}px ${-left}px` })
      };
      this.viewObservers.size.observe(this.doc.documentElement);
      this.viewObservers.size.observe(this.doc.body);
    }
    const o = this.viewObservers;
    const now = new Set(asks);
    for (const el of this.viewWatched) {
      if (now.has(el)) continue;
      o.size.unobserve(el);
      o.window.unobserve(el);
      this.viewWatched.delete(el);
    }
    for (const el of now) {
      if (this.viewWatched.has(el)) continue;
      o.size.observe(el);
      o.window.observe(el);
      this.viewWatched.add(el);
    }
  }
  /** The surface is `cols`×`rows` cells, and the box shows `win` of it:
   *  the document keeps the whole size, offset by the window's corner. */
  setSize(cols, rows, cellW, cellH, win = { x: 0, y: 0, w: cols, h: rows }) {
    this.cols = cols;
    this.rows = rows;
    this.win = win;
    this.cellW = cellW;
    this.cellH = cellH;
    Object.assign(this.frame.style, {
      left: `${-Math.round(win.x * cellW)}px`,
      top: `${-win.y * cellH}px`,
      width: `${Math.round(cols * cellW)}px`,
      height: `${rows * cellH}px`
    });
    this.box.style.width = `${Math.round(win.w * cellW)}px`;
    this.box.style.height = `${win.h * cellH}px`;
  }
  /** The placement's z (SPEC §5.2): CSS z-index on the box, so overlapping
   *  boxes stack by it, and by their order in the layer at the same z. */
  setZ(z) {
    this.box.style.zIndex = String(z);
  }
  /** Shows the surface with its top-left corner at (x, y) in the screen's pixels. */
  show(x, y) {
    this.box.style.left = `${x}px`;
    this.box.style.top = `${y}px`;
    this.box.style.visibility = "visible";
    this.box.style.display = "block";
    if (this.fitDirty) this.refit();
    if (this.viewDirty) this.requestView();
  }
  /** Out of view (its line left the screen): a drag under way ends
   * (SPEC §9.1). */
  hide() {
    this.cancelDrag();
    this.dropAlt();
    this.box.style.display = "none";
  }
  /** No placement (hidden, or not placed yet): the document stays, and the
   *  browser skips its style, layout and paint until it is placed again. */
  park() {
    this.box.style.display = "none";
  }
  destroy() {
    this.setFit(null);
    this.endPlacement();
    this.dropDrag();
    this.dropAlt();
    if (this.settling) clearTimeout(this.settling);
    if (this.caretFrame) this.frame.ownerDocument.defaultView?.cancelAnimationFrame(this.caretFrame);
    this.prefix = false;
    if (this.keyboard && this.waiting === null || this.frameFocused()) this.host.focusTerminal();
    this.waiting = null;
    this.box.remove();
  }
  // --- Keyboard (PROTOCOL §8, Q-0005) -------------------------------------
  hasKeyboard() {
    return this.keyboard;
  }
  /** The text selected in the surface, for the host's copy keys (SPEC
   * §11): the focused text field's selection, never a password field's, or
   * else the document's. */
  selectedText() {
    const el = this.doc.activeElement;
    if (el && (el.localName === "input" || el.localName === "textarea")) {
      const type = (this.resolver.get(el, "type") ?? el.type ?? "").toLowerCase();
      if (type === "password") return "";
      try {
        const [a, b] = [el.selectionStart, el.selectionEnd];
        return a === null || b === null ? "" : el.value.slice(a, b);
      } catch {
        return "";
      }
    }
    return this.doc.getSelection()?.toString() ?? "";
  }
  /**
   * A key for the selection in the document outside a text field (SPEC
   * §11): null when there is none, or a focused text field holds the
   * selection; else its text when the keymap of the element whose text it
   * starts in, read as a text field's, binds the key to `copy`, and false
   * when it does not, the key going on as if nothing were selected.
   */
  selectionCopy(e) {
    const active = this.doc.activeElement;
    if (isValueField(active) || active?.isContentEditable) return null;
    const sel = this.doc.getSelection();
    if (!sel || sel.isCollapsed || sel.rangeCount === 0) return null;
    const text = sel.toString();
    if (text === "") return null;
    const start = sel.getRangeAt(0).startContainer;
    const el = start.nodeType === 1 ? start : start.parentElement;
    if (!el) return null;
    const keymap = resolve(this.host.keys.platform, false, ...keysValues(el));
    const named = pressNames(e).find((n) => keymap.binds(n));
    return named !== void 0 && keymap.lookup(named) === "copy" ? text : false;
  }
  /** Drops the document's selection: the user pressed elsewhere (SPEC §11). */
  clearSelection() {
    const sel = this.doc.getSelection();
    if (sel && sel.rangeCount > 0) sel.removeAllRanges();
  }
  /** `a=focus`: the surface takes the keyboard, at `target` if given. With
   * `now` false, the page's focus is outside the terminal, and stays there:
   * the surface has the keyboard, and takes the browser's focus when the
   * terminal does (`takeFocus`). */
  focus(target, now = true) {
    const el = target ? this.doc.getElementById(target) : null;
    if (target && !el) throw new Error(target);
    this.prefix = false;
    this.waiting = null;
    if (!now) {
      this.keyboard = true;
      this.waiting = target ?? "";
      return;
    }
    this.programFocus = true;
    try {
      if (!this.frameFocused()) {
        this.frame.focus({ preventScroll: true });
        this.frame.contentWindow?.focus();
      }
      const had = this.focusedControl();
      const to = el ?? (had ? null : this.tabbable()[0]);
      if (to) {
        to.focus({ preventScroll: true });
        this.reveal(to, "");
      }
      this.keyboard = true;
      this.told = this.focusedControl();
      if (this.told && this.told !== had && this.isTextField(this.told)) this.restoreSelection(this.told);
    } finally {
      setTimeout(() => this.programFocus = false, 0);
    }
  }
  /** `a=blur`, or the keyboard leaving: the edited control commits first. */
  blur() {
    this.pressed = null;
    this.prefix = false;
    if (this.dropWait(true)) return;
    this.commit();
    if (this.keyboard) this.giveBack();
  }
  /** The terminal has the page's focus, and not by a click: the focus that
   * waited moves now, as the program's (no echo). Its element may have gone
   * meanwhile; the surface takes the focus all the same. */
  takeFocus() {
    if (this.waiting === null) return;
    const target = this.waiting;
    this.waiting = null;
    const el = target ? this.doc.getElementById(target) : null;
    this.focus(el ? target : void 0);
  }
  /** A focus that waits ends without moving: a click elsewhere gives the
   * keyboard back (SPEC §10.1), and says so with `blur` when `say`;
   * another `a=focus` moves it, and the program knows. Whether one waited. */
  dropWait(say) {
    if (this.waiting === null) return false;
    this.waiting = null;
    this.keyboard = false;
    this.told = null;
    if (say) this.emit("blur", "");
    return true;
  }
  /** The browser's focus goes from the frame to the terminal. The frame lets
   * go of it first: Firefox can leave the frame focused otherwise. */
  giveBack() {
    this.frame.blur();
    this.host.focusTerminal();
  }
  commit() {
    const active = this.focusedControl();
    if (active) active.blur();
  }
  focusedControl() {
    const a = this.doc.activeElement;
    return a && a !== this.doc.body && a !== this.doc.documentElement ? a : null;
  }
  /** Whether the browser's focus is in this surface's frame. */
  frameFocused() {
    return this.frame.ownerDocument.activeElement === this.frame;
  }
  /**
   * A press (SPEC §10.1). On an element that takes focus, the focus that
   * follows gives the surface the keyboard (`onFocusIn`). On anything else,
   * or anywhere in a detached surface, it takes nothing: a surface that had
   * the keyboard gives it back (its control commits, then `blur`), and the
   * focus the browser gives the frame anyway goes back to the terminal.
   */
  onPress(e) {
    if (e.type === "pointerdown" && e.pointerType === "touch") return;
    if (e.button !== 0) {
      this.auxPress = true;
      setTimeout(() => this.auxPress = false, 0);
      return;
    }
    this.pressAt(e);
  }
  /** What a click (SPEC §10.1) at `e`'s target does to the keyboard: a
   * press of the primary button, a tap, or a touch that becomes a drag. */
  pressAt(e) {
    this.prefix = false;
    this.host.clicked(this.name);
    const el = this.detachedState ? null : focusTargetAt(e, this.hyper);
    if (this.waiting !== null) {
      this.waiting = null;
      if (el) this.keyboard = false;
    }
    if (el) {
      this.pressed = el;
      this.settle();
      return;
    }
    this.pressNothing = true;
    if (this.keyboard) {
      this.commit();
      this.keyboard = false;
      this.told = null;
      this.emit("blur", "");
    }
    this.settle();
  }
  /**
   * Once the browser is done with a press (its default action focuses the
   * frame): an element that takes focus has it, even where the browser does
   * not focus one on a click (a button on a Mac, say); and focus the
   * surface does not hold goes back to the terminal. The frame keeps it
   * until then, so text selection starts as usual.
   */
  settle() {
    if (this.settling) return;
    this.settling = setTimeout(() => {
      this.settling = null;
      const el = this.pressed;
      this.pressed = null;
      this.pressNothing = false;
      if (el && !this.keyboard && !this.detachedState && el.isConnected && this.doc.activeElement !== el) {
        this.frame.focus({ preventScroll: true });
        el.focus({ preventScroll: true });
      }
      if (!this.keyboard && this.frameFocused()) this.giveBack();
    }, 0);
  }
  /** An element got focus: the surface takes the keyboard if the element is
   * one that takes focus, and the focus is neither the program's (no echo)
   * nor a detached surface's. The program hears where the user put focus,
   * each time it is another element than the one it knows (SPEC §10.1). */
  onFocusIn(e) {
    const el = isElement(e.target) ? e.target : null;
    if (this.programFocus) {
      this.told = el;
      return;
    }
    if (this.auxPress) return;
    if (this.detachedState || this.pressNothing || !el || !takesFocus(el, this.hyper)) {
      this.settle();
      return;
    }
    const gained = !this.keyboard;
    this.keyboard = true;
    if (!gained && el === this.told) return;
    this.told = el;
    this.emit("focus", el.closest('[id]:not([id=""])')?.id ?? "");
  }
  tabbable() {
    const all = this.doc.querySelectorAll(
      "a[href], button, input:not([type=hidden]), select, textarea, summary, [tabindex], [contenteditable=''], [contenteditable=true]"
    );
    return Array.from(all).filter((el) => el.tabIndex >= 0 && !el.disabled && el.getClientRects().length > 0);
  }
  controlKind(el) {
    if (!el) return "none";
    const tag = el.localName;
    if (tag === "textarea" || el.isContentEditable) return "textarea";
    if (tag === "select") return "select";
    if (tag === "input") {
      const type = this.inputType(el);
      if (FIELD_TYPES.has(type)) return "text";
      if (DATE_TYPES.has(type)) return "date";
      return "activatable";
    }
    if (tag === "button" || tag === "a" || tag === "summary") return "activatable";
    return "none";
  }
  /** Whether the focused element uses the key; if not, the program gets it. */
  consumes(e, kind) {
    const plain2 = !e.ctrlKey && !e.altKey && !e.metaKey;
    const k = e.key;
    const printable = [...k].length === 1;
    if (k === "Escape") return false;
    switch (kind) {
      case "date":
        return (plain2 || e.shiftKey && !e.ctrlKey && !e.altKey && !e.metaKey) && (printable || ["Backspace", "Delete", "ArrowLeft", "ArrowRight", "ArrowUp", "ArrowDown", "Home", "End", "PageUp", "PageDown", "Enter"].includes(k));
      case "select":
        return plain2 && (printable || ["ArrowUp", "ArrowDown", "Home", "End", "PageUp", "PageDown", "Enter", " "].includes(k));
      case "activatable":
        return plain2 && (k === " " || k === "Enter");
      default:
        return false;
    }
  }
  onKey(e) {
    if (e.isComposing || e.defaultPrevented) return;
    this.host.typed(e);
    if (this.host.copyKey(e)) {
      e.stopPropagation();
      return;
    }
    if (!this.keyboard) {
      this.settle();
      if (this.host.browserKey(e)) return;
      this.forward(e);
      return;
    }
    if (listOpen(this.focusedControl())) return;
    if (this.prefix) {
      if (!this.host.browserKey(e)) this.forward(e);
      return;
    }
    this.plainKey(e);
  }
  /**
   * A key for the focused element (SPEC §10.2): Tab, a text field's keys,
   * what the element's keymap gives the program, what the element uses, the
   * scroll actions and the keys the document scrolls with; the rest is the
   * program's.
   */
  plainKey(e) {
    const active = this.focusedControl();
    const kind = this.controlKind(active);
    if (e.key === "Tab" && !e.ctrlKey && !e.altKey && !e.metaKey) {
      if (!e.shiftKey && active && (kind === "text" || kind === "textarea") && this.keymapOf(active, kind === "textarea").program("tab")) {
        this.toProgram(e, this.host.encodeKey(e));
        return;
      }
      const order = this.tabbable();
      const at = order.indexOf(this.doc.activeElement);
      const leaving = order.length === 0 || (e.shiftKey ? at <= 0 : at === order.length - 1 || at < 0);
      if (leaving) {
        e.preventDefault();
        this.blur();
      } else {
        this.tabbing = true;
        setTimeout(() => this.tabbing = false, 0);
      }
      return;
    }
    if (kind === "text" || kind === "textarea") {
      this.scrollDestinations.clear();
      this.fieldKey(e, active, kind === "textarea");
      return;
    }
    if (this.host.browserKey(e)) return;
    if (active && this.programKey(e, active)) return;
    if (this.consumes(e, kind)) {
      this.scrollDestinations.clear();
      return;
    }
    if (active && this.scrollActionKey(e, active)) return;
    if (this.axes && this.scrollKey(e)) return;
    this.forward(e);
  }
  /**
   * A key the focused element's keymap gives the program (SPEC §10.2, *Keys
   * for the program*), outside a text field: it reaches the program before
   * the element uses it (a select's arrows) and before the document
   * scrolls with it. The keymap is each `data-keys` from the root down to
   * the element, with no default, and only its `program` and `prefix`
   * bindings count; a `prefix` begins one (§10.5). The key is named as in a
   * text field (`fieldKey`).
   */
  programKey(e, el) {
    const values = keysValues(el);
    if (!values.some((v) => v.includes("program") || v.includes("prefix"))) return false;
    const keymap = elementKeymap(this.host.keys.platform, ...values);
    const data = this.host.encodeKey(e);
    const keys = asked(keymap, e, data);
    if (!keys.some((k) => keymap.program(k))) return false;
    this.toProgram(e, data);
    if (keys.some((k) => keymap.prefix(k))) this.prefix = true;
    return true;
  }
  /**
   * A key the focused element's keymap binds to a scroll action (SPEC
   * §10.2, *Scrolling keys*), outside a text field, and that the element
   * does not use. Along an axis the document scrolls, it scrolls the
   * nearest box from the element outward that can still move that way, as
   * a gesture goes (§5.3), and never the terminal: where nothing can move,
   * the key is used and does nothing. Along an axis the document does not
   * scroll, it goes on as if its keymap did not bind it. The key is named
   * as in a text field (`fieldKey`).
   */
  scrollActionKey(e, el) {
    const values = keysValues(el);
    if (!values.some((v) => v.includes("scroll-"))) return false;
    const keymap = elementKeymap(this.host.keys.platform, ...values);
    const data = this.host.encodeKey(e);
    const action = asked(keymap, e, data).map((k2) => keymap.scroll(k2)).find((a) => a !== null);
    const k = action ? SCROLL_ACTIONS2[action] : void 0;
    if (!k || !(this.axes & (k.axis === "y" ? 1 : 2))) return false;
    e.preventDefault();
    e.stopPropagation();
    this.scrollAction(el, action);
    return true;
  }
  /** A delta's or a key's scroll action, confined to this document. */
  scrollAction(el, action) {
    if (!this.placed) return;
    const k = SCROLL_ACTIONS2[action];
    if (!k || !(this.axes & (k.axis === "y" ? 1 : 2))) return;
    this.finger?.stopFling();
    const box = this.box.style;
    const out = box.display === "none";
    const visibility = box.visibility;
    if (out) Object.assign(box, { display: "block", visibility: "hidden" });
    try {
      const to = this.scroller(el, k.axis, k.sign);
      if (typeof to !== "string") this.scrollBox(to, k);
    } finally {
      if (out) Object.assign(box, { display: "none", visibility });
    }
  }
  /** Scrolls `to` as a key does (`KeyScroll`), by Chromium's amounts. */
  scrollBox(to, k) {
    this.finger?.stopFling();
    const y = k.axis === "y";
    const port2 = y ? to.clientHeight : to.clientWidth;
    const by = k.by === "line" ? LINE_PX : k.by === "page" ? Math.max(1, port2 * PAGE_FRACTION) : k.by === "half" ? Math.max(1, port2 / 2) : y ? to.scrollHeight : to.scrollWidth;
    const from = this.scrollDestinations.get(to) ?? { left: to.scrollLeft, top: to.scrollTop };
    const next = { ...from };
    const style2 = to.ownerDocument.defaultView.getComputedStyle(to);
    if (y) next.top = Math.max(0, Math.min(to.scrollHeight - to.clientHeight, from.top + k.sign * by));
    else {
      const span = to.scrollWidth - to.clientWidth;
      const [min, max] = style2.direction === "rtl" ? [-span, 0] : [0, span];
      next.left = Math.max(min, Math.min(max, from.left + k.sign * by));
    }
    if (style2.scrollBehavior === "smooth") this.scrollDestinations.set(to, next);
    else this.scrollDestinations.delete(to);
    to.scrollTo(next);
  }
  /** A key the program has, as xterm.js encoded it (`data`; null when it
   * cannot say, and the key goes through it). Its release follows. */
  toProgram(e, data) {
    e.preventDefault();
    e.stopPropagation();
    if (e.type === "keydown" && this.host.command(e)) return;
    this.forwarded.add(e.code);
    if (data === null) this.host.key(e);
    else if (data) this.host.sendKey(data);
  }
  // --- Text fields (SPEC §10.2, §10.4) ------------------------------------
  /** An input's type, as the browser has it; the program's for a field
   * that is `text` while focused (`NO_CARET`). */
  inputType(el) {
    return el === this.masked ? (this.resolver.get(el, "type") ?? "").toLowerCase() : el.type;
  }
  /** Whether a text field has text selected. */
  selects(el) {
    if (isValueField(el)) return el.selectionStart !== el.selectionEnd;
    const sel = this.doc.getSelection();
    return !!sel && !sel.isCollapsed && el.contains(sel.anchorNode);
  }
  /**
   * A key in a text field, offered at most twice (SPEC §10.4). A key that
   * types no text is offered first as pressed, named from the DOM's event
   * by its physical key and its character (`pressNames`), before xterm.js
   * translates it (on a Mac, Option+ArrowLeft is ESC b to the program).
   * Otherwise it is named as the program would read it: from
   * what xterm.js would send the program for it, in the encoding the
   * program enabled, read as SPEC §10.4 says. The field's keymap (the
   * default, then each `data-keys` from the root to the field) says what
   * the field does with each key; what it does not use is the program's,
   * sent as xterm.js encoded it.
   */
  fieldKey(e, el, multiline) {
    if (this.host.browserKey(e)) return;
    const keymap = this.keymapOf(el, multiline);
    const cuts = this.selects(el) && !(el.localName === "input" && this.inputType(el) === "password");
    const lookup = (key) => {
      const use = keymap.lookup(key);
      return (use === "copy" || use === "cut") && !cuts ? null : use;
    };
    const named = typesNoText(e) ? pressNames(e).find((n) => keymap.binds(n)) : void 0;
    const first = named !== void 0 && lookup(named) !== null;
    const data = first ? "" : this.host.encodeKey(e);
    const keys = first ? [{ key: named, data: "" }] : named !== void 0 ? [{ key: named, data: data ?? "" }] : typedKeys(e, data);
    const uses = keys.map((k) => k.key === null ? null : lookup(k.key));
    const extend = keys.map((k) => k.key !== null && keymap.selects(k.key));
    if (uses.every((u) => u === null)) {
      const program = keys.some((k) => k.key !== null && keymap.program(k.key));
      if (!program && this.axes && this.scrollKey(e)) return;
      this.toProgram(e, data);
      if (keys.some((k) => k.key !== null && keymap.prefix(k.key))) this.prefix = true;
      return;
    }
    const number = el.localName === "input" && this.inputType(el) === "number";
    if (keys.length === 1) {
      const [use, key] = [uses[0], keys[0].key];
      const typed = use === INSERT && !(number && !NUMBER_CHARS.test(keyText(key) ?? ""));
      const enter = use === "submit" && e.key === "Enter" && el.localName === "input";
      const paste = use === "paste" && first && (mac ? e.metaKey && !e.ctrlKey : e.ctrlKey && !e.metaKey) && !e.altKey;
      if (typed || enter || paste) {
        this.goal = null;
        return;
      }
    }
    e.preventDefault();
    e.stopPropagation();
    keys.forEach((k, i) => {
      const use = uses[i];
      if (use === null) {
        if (k.data) this.host.sendKey(k.data);
      } else if (use === INSERT) {
        const text = keyText(k.key) ?? "";
        if (text && (!number || NUMBER_CHARS.test(text))) this.edit(el, multiline, { kind: "type", text });
      } else if (use === "submit") {
        this.goal = null;
        this.commitField(el);
        submitFrom(el);
      } else if (use === "undo" || use === "redo") this.undo(el, use === "redo");
      else if (use === "copy") this.host.writeClipboard(this.selectedText());
      else if (use === "cut") this.cut(el);
      else if (use === "paste") this.pasteFrom(el, multiline);
      else this.edit(el, multiline, { kind: "action", action: use, extend: extend[i] });
    });
  }
  /** A field's keymap (SPEC §10.2): the default, then the `data-keys` of
   * each element from the root down to the field. */
  keymapOf(el, multiline) {
    return resolve(this.host.keys.platform, multiline, ...keysValues(el));
  }
  /** Does an action in a field, or types text in it. */
  edit(el, multiline, what) {
    if (what.kind === "action" && MULTILINE_ACTIONS.has(what.action) && !multiline) return;
    if (el.localName === "input" || el.localName === "textarea") this.editValue(el, multiline, what);
    else this.editHost(el, what);
  }
  /** An input's or a textarea's value: the action's plan, applied with
   * the editing commands, so that the browser sends `input` (and `change`
   * when focus leaves) as for typing. */
  editValue(el, multiline, what) {
    const cl = chars(el.value);
    const at = [0];
    for (const c of cl) at.push(at[at.length - 1] + c.length);
    const index = (cu) => {
      let i = 0;
      while (i < cl.length && at[i + 1] <= cu) i++;
      return i;
    };
    const start = index(el.selectionStart ?? el.value.length);
    const end = index(el.selectionEnd ?? el.value.length);
    const caret = el.selectionDirection === "backward" ? start : end;
    if (what.kind === "type") {
      this.goal = null;
      this.replace(el, at[start], at[end], what.text, "typing");
      return;
    }
    const action = what.action;
    const run = action === "newline" ? "typing" : action === "delete-char-backward" || action === "delete-char-forward" ? "deleting" : null;
    if (run === null) {
      const h = this.histories.get(el);
      if (h) h.run = null;
    }
    const password = el.localName === "input" && this.inputType(el) === "password";
    const text = { chars: cl, start, end, caret, multiline, password };
    const g = this.goal;
    const goal = ROW_ACTIONS.has(action) && g && g.el === el && g.caret === caret ? g.x : null;
    const rows = multiline && ROW_ACTIONS.has(action) ? this.textareaRows(el, cl) : null;
    const p = plan(text, action, rows, goal, what.extend);
    this.goal = p.kind === "move" && p.goal !== void 0 ? { el, caret: p.to, x: p.goal } : null;
    if (p.kind === "move") {
      const a = p.anchor ?? p.to;
      el.setSelectionRange(at[Math.min(a, p.to)], at[Math.max(a, p.to)], a > p.to ? "backward" : "forward");
    } else if (p.kind === "replace") this.replace(el, at[p.from], at[p.to], p.text, run);
  }
  /** Replaces code units `from` to `to` of a field's value with `text`, the
   * caret after it, as typing does: an edit in the `run` for the field's
   * history (SPEC §10.2, undo). */
  replace(el, from, to, text, run) {
    this.flushCaret(el);
    const before = stateOf(el);
    el.setSelectionRange(from, to);
    if (from === to && !text) return;
    this.pending = { el, before, run };
    this.editing = true;
    try {
      const done = text ? this.doc.execCommand("insertText", false, text) : this.doc.execCommand("delete", false);
      if (done) return;
      el.setRangeText(text, from, to, "end");
      const win = this.frame.contentWindow;
      el.dispatchEvent(new win.InputEvent("input", { bubbles: true, inputType: text ? "insertText" : "deleteContent", data: text || null }));
    } finally {
      this.editing = false;
      this.pending = null;
    }
  }
  /**
   * A textarea's rows as it lays them out (SPEC §10.2: a line that wraps is
   * several rows): a hidden copy of its text, wrapped as it wraps, with
   * every character in an element of its own, measured where each
   * position is. The place along a row is a position's x.
   */
  textareaRows(el, cl) {
    const win = this.frame.contentWindow;
    const cs = win.getComputedStyle(el);
    const copy = this.doc.createElement("hotty-rows");
    for (const p of LAYOUT_STYLES) copy.style.setProperty(p, cs.getPropertyValue(p));
    const width = el.clientWidth - parseFloat(cs.paddingLeft) - parseFloat(cs.paddingRight);
    Object.assign(copy.style, {
      display: "block",
      position: "absolute",
      left: "0px",
      top: "0px",
      visibility: "hidden",
      boxSizing: "content-box",
      width: `${Math.max(width, 1)}px`,
      height: "auto",
      border: "0",
      margin: "0"
    });
    const cells = [...cl, "\u200B"].map((c) => {
      const s = this.doc.createElement("hotty-c");
      s.style.cssText = "display:inline;margin:0;padding:0;border:0;font:inherit;letter-spacing:inherit";
      s.textContent = c;
      return s;
    });
    copy.append(...cells);
    this.doc.body.append(copy);
    const pos = cells.map((s) => {
      const r = s.getClientRects()[0] ?? s.getBoundingClientRect();
      return { x: r.left, y: Math.round(r.top) };
    });
    copy.remove();
    const tops = [...new Set(pos.map((p) => p.y))].sort((a, b) => a - b);
    const line = tops.length > 1 ? tops[1] - tops[0] : parseFloat(cs.lineHeight) || parseFloat(cs.fontSize) * 1.2;
    const shown = el.clientHeight - parseFloat(cs.paddingTop) - parseFloat(cs.paddingBottom);
    const n = cl.length;
    return {
      page: Math.max(1, Math.round(shown / line)),
      move(from, by, goal) {
        const g = goal ?? pos[from].x;
        const r = tops.indexOf(pos[from].y) + by;
        if (r < 0) return { to: 0, goal: g };
        if (r >= tops.length) return { to: n, goal: g };
        let best = -1;
        for (let i = 0; i <= n; i++) {
          if (pos[i].y !== tops[r]) continue;
          if (best < 0 || Math.abs(pos[i].x - g) < Math.abs(pos[best].x - g)) best = i;
        }
        return { to: best, goal: g };
      }
    };
  }
  /**
   * An editing host (`contenteditable`): the actions with the selection's
   * own moves, a character at a time where they need to see the text, so
   * that words and lines are SPEC §10.2's, then the editing commands.
   */
  editHost(host, what) {
    this.goal = null;
    const sel = this.doc.getSelection();
    if (!sel || !sel.rangeCount) return;
    const d = this.doc;
    if (what.kind === "type") {
      d.execCommand("insertText", false, what.text);
      return;
    }
    const action = what.action;
    if (action === "newline") {
      d.execCommand("insertLineBreak", false);
      return;
    }
    if (action === "select-all") {
      sel.selectAllChildren(host);
      return;
    }
    const within = () => sel.focusNode !== null && host.contains(sel.focusNode);
    const next = (dir) => {
      const [an, ao, fn, fo] = [sel.anchorNode, sel.anchorOffset, sel.focusNode, sel.focusOffset];
      sel.collapse(fn, fo);
      sel.modify("extend", dir, "character");
      const c = within() ? sel.toString() : "";
      sel.setBaseAndExtent(an, ao, fn, fo);
      return c;
    };
    const walk = (alter2, dir, go) => {
      for (let i = 0; i < 1e5; i++) {
        const c = next(dir);
        if (!c || !go(c)) return;
        sel.modify(alter2, dir, "character");
      }
    };
    const back = (alter2, unit) => {
      if (unit === "word") {
        walk(alter2, "backward", isSpace);
        walk(alter2, "backward", (c) => !isSpace(c));
      } else if (unit === "line") walk(alter2, "backward", (c) => !isBreak(c));
      else sel.modify(alter2, "backward", "character");
    };
    const forward = (alter2, unit) => {
      if (unit === "word") {
        walk(alter2, "forward", isSpace);
        walk(alter2, "forward", (c) => !isSpace(c));
      } else if (unit === "line") walk(alter2, "forward", (c) => !isBreak(c));
      else sel.modify(alter2, "forward", "character");
    };
    if (action.startsWith("delete-")) {
      if (sel.isCollapsed) {
        const unit = action.includes("word") ? "word" : action.includes("line") ? "line" : "char";
        if (action.endsWith("backward") || action.endsWith("start")) back("extend", unit);
        else forward("extend", unit);
      }
      if (!sel.isCollapsed) d.execCommand("delete", false);
      return;
    }
    const backward = ["char-backward", "word-backward", "line-start", "line-previous", "page-up", "input-start"].includes(action);
    const alter = what.extend ? "extend" : "move";
    if (!what.extend && !sel.isCollapsed) {
      if (backward) sel.collapseToStart();
      else sel.collapseToEnd();
      if (action === "char-backward" || action === "char-forward") return;
    }
    const by = { "char-backward": "char", "char-forward": "char", "word-backward": "word", "word-forward": "word", "line-start": "line", "line-end": "line" }[action];
    if (by) {
      if (backward) back(alter, by);
      else forward(alter, by);
      return;
    }
    if (action === "input-start" || action === "input-end") {
      sel.modify(alter, backward ? "backward" : "forward", "documentboundary");
      return;
    }
    if (ROW_ACTIONS.has(action)) {
      const rows = action.startsWith("page") ? Math.max(1, Math.round(host.clientHeight / (parseFloat(this.frame.contentWindow.getComputedStyle(host).lineHeight) || 16))) : 1;
      for (let i = 0; i < rows; i++) {
        const [fn, fo] = [sel.focusNode, sel.focusOffset];
        sel.modify(alter, backward ? "backward" : "forward", "line");
        if (sel.focusNode === fn && sel.focusOffset === fo) {
          sel.modify(alter, backward ? "backward" : "forward", "documentboundary");
          return;
        }
      }
    }
  }
  /** A field that hides its caret (`NO_CARET`) is `text` while focused:
   * from the press that focuses it, so that the caret lands where the
   * press does, or from the focus. */
  maskField(target, focusing) {
    const el = target;
    if (!el || el.nodeType !== 1 || el.localName !== "input" || el === this.masked || this.detachedState) return;
    const type = (this.resolver.get(el, "type") ?? "").toLowerCase();
    if (!NO_CARET.has(type) || el.disabled) return;
    this.unmaskField();
    this.masked = el;
    this.resolver.mask(el, "type", "text");
    if (!el.hasAttribute("inputmode")) this.resolver.setOwn(el, "inputmode", type === "number" ? "decimal" : "email");
    if (focusing) el.setSelectionRange(this.tabbing ? 0 : el.value.length, el.value.length);
  }
  unmaskField() {
    const el = this.masked;
    if (!el) return;
    this.masked = null;
    this.resolver.removeOwn(el, "inputmode");
    this.resolver.unmask(el, "type");
  }
  /**
   * A key a browser scrolls with, in a document that scrolls (SPEC §5.3):
   * it scrolls the innermost box, from the focused element outward (from
   * the root when none is), that can still move that way. Where none can,
   * the key goes on to the program, as every key the surface does not use
   * does (§10.2), unless `overscroll-behavior` stops it. The addon scrolls
   * the box itself: the browser's own action could be the focused
   * element's instead (a radio button's arrows, a number field's).
   */
  scrollKey(e) {
    const k = keyScroll(e);
    if (!k) return false;
    const to = this.scroller(this.focusedControl(), k.axis, k.sign);
    if (to === "terminal") return false;
    e.preventDefault();
    if (to === "stop") return true;
    this.scrollBox(to, k);
    return true;
  }
  forward(e) {
    e.preventDefault();
    e.stopPropagation();
    if (e.type === "keydown" && this.host.command(e)) return;
    this.forwarded.add(e.code);
    this.host.key(e);
  }
  /** The release of a key the program had: the program hears it too, if it
   *  asked the terminal for releases (SPEC §10.3). */
  onKeyUp(e) {
    if (!this.forwarded.delete(e.code)) return;
    e.preventDefault();
    e.stopPropagation();
    this.host.key(e);
  }
  // --- Drags (SPEC §9.1) ----------------------------------------------------
  /**
   * A press of a mouse's or a pen's primary button on an element with
   * `drag` in its `data-on` (the nearest, from the pressed one outward, and
   * only if it has an id) starts a drag. The surface then holds the pointer
   * until the release: captured by its root element, which no delta
   * replaces, so every move comes here wherever it is (over the cells,
   * another surface, or outside the page), and nothing else hears it.
   */
  onDragDown(e) {
    this.cancelDrag();
    if (this.detachedState || e.pointerType === "touch" || e.button !== 0 || !e.isPrimary) return;
    let start = null;
    for (const n of e.composedPath()) {
      if (isElement(n) && listens(n, "drag")) {
        start = n;
        break;
      }
    }
    const id = start?.getAttribute("id");
    if (!id) return;
    const [c, r] = this.cellOf(e);
    const mods = modsOf2(e);
    const d = { pointer: e.pointerId, start: id, steps: stepsOf(start), target: id, c, r, mods, step: {}, tstep: {}, clicks: 1 };
    d.step = this.stepIn(d, e.clientX, e.clientY);
    this.drag = d;
    this.emit("dragstart", id, { c, r, mods, ...d.step });
    try {
      this.doc.documentElement.setPointerCapture(e.pointerId);
    } catch {
    }
  }
  /** A move during a drag (`moveDrag`). */
  onDragMove(e) {
    const d = this.drag;
    if (!d || e.pointerId !== d.pointer) return;
    this.moveDrag(d, e.clientX, e.clientY, modsOf2(e));
  }
  /** The pointer or the finger of a drag moved to (x, y) of the document:
   * an event each time its target changes, while it has none each time its
   * cell does, and each time its step does (SPEC §9.1); one for a move that
   * changes more than one. */
  moveDrag(d, x, y, mods) {
    const [c, r] = this.cellAt(x, y);
    const t = this.dragTarget(this.elementAtPoint(x, y, c, r));
    const target = t?.getAttribute("id") ?? "";
    const step2 = this.stepIn(d, x, y);
    const tstep = stepInTarget(t, x, y);
    if (target !== d.target || target === "" && (c !== d.c || r !== d.r) || step2.x !== d.step.x || step2.y !== d.step.y || tstep.tx !== d.tstep.tx || tstep.ty !== d.tstep.ty) {
      this.emit("drag", target, { c, r, mods, ...step2, ...tstep });
    }
    Object.assign(d, { target, c, r, mods, step: step2, tstep });
  }
  /**
   * Where (x, y) of the document is in the element a drag started on, in
   * its steps (SPEC §9.1): measured against its border box where the user
   * sees it now, wherever the point is, and as they last were once the
   * element has left the document. None when it has no steps.
   */
  stepIn(d, x, y) {
    const s = d.steps;
    if (!s) return {};
    if (!s.el.isConnected) return d.step;
    const box = s.el.getBoundingClientRect();
    const step2 = {};
    if (s.x) step2.x = stepAt(x - box.left, box.width, s.x);
    if (s.y) step2.y = stepAt(y - box.top, box.height, s.y);
    return step2;
  }
  /** The release ends the drag, wherever it is. Where it began, it is the
   * click it would have been without the drag; anywhere else, no click. */
  onDragUp(e) {
    const d = this.drag;
    if (!d || e.pointerId !== d.pointer || e.button !== 0) return;
    this.drag = null;
    const [c, r] = this.cellOf(e);
    this.endDrag(d, e.clientX, e.clientY, c, r, this.elementAt(e, c, r), modsOf2(e));
  }
  /** A drag released at (x, y) of the document, cell (c, r), over `at`:
   * `dragend`, then the click it is where it began (SPEC §9.1): where the
   * nearest element that opts in to drags is the one it started on, whatever
   * its target, a drop-only element inside it included. */
  endDrag(d, x, y, c, r, at, mods) {
    const t = this.dragTarget(at);
    this.emit("dragend", t?.getAttribute("id") ?? "", { c, r, mods, ...this.stepIn(d, x, y), ...stepInTarget(t, x, y) });
    this.dragReleased = true;
    setTimeout(() => this.dragReleased = false, 0);
    if (at && this.draggedAt(at) === d.start) this.reportClick(ancestry(at), d.clicks, mods);
  }
  /** The browser took the pointer away before the release. */
  lostPointer(e) {
    if (this.drag && e.pointerId === this.drag.pointer) this.cancelDrag();
    if (e.pointerId === this.altPointer) this.altPointer = null;
  }
  /**
   * Ends a drag under way without a release (SPEC §9.1: its placement went
   * away, a new document came, or the pointer was lost): `dragend` with no
   * target, at the last cell and step the program heard of.
   */
  cancelDrag() {
    const d = this.drag;
    if (!d) return;
    this.dropDrag();
    this.emit("dragend", "", { c: d.c, r: d.r, mods: d.mods, ...d.step });
  }
  /** Ends a drag under way, silently (a detached or deleted surface). */
  dropDrag() {
    const d = this.drag;
    if (!d) return;
    this.drag = null;
    try {
      if (this.doc.documentElement.hasPointerCapture(d.pointer)) this.doc.documentElement.releasePointerCapture(d.pointer);
    } catch {
    }
  }
  /** The surface's cell under the pointer, from its top left (the frame's
   * origin), counting on past its edges. */
  cellOf(e) {
    return this.cellAt(e.clientX, e.clientY);
  }
  /** The element under the pointer, if it is in the window (SPEC §5.2). */
  elementAt(e, c, r) {
    return this.elementAtPoint(e.clientX, e.clientY, c, r);
  }
  /** A drag's target, t: the nearest element with an id and `drag` or
   * `drop` in its `data-on`, from `el` outward; null where there is none. */
  dragTarget(el) {
    for (let n = el; n; n = n.parentElement) {
      if (n.getAttribute("id") && (listens(n, "drag") || listens(n, "drop"))) return n;
    }
    return null;
  }
  /** What a drag would name at a cell's centre (SPEC §9.7), in this
   * document regardless of anything drawn above it. Changes no state. */
  hit(c, r) {
    if (!this.placed || this.box.style.display === "none") return {};
    const x = (c + 0.5) * this.cellW;
    const y = (r + 0.5) * this.cellH;
    const t = this.dragTarget(this.elementAtPoint(x, y, c, r));
    return t ? { t: t.getAttribute("id"), ...stepInTarget(t, x, y) } : {};
  }
  /** The id of the nearest element with an id and `drag` in its `data-on`,
   * from `el` outward: a drag that ends where this is the element it began
   * on is a click (SPEC §9.1). "" where there is none. */
  draggedAt(el) {
    for (let n = el; n; n = n.parentElement) {
      const id = n.getAttribute("id");
      if (id && listens(n, "drag")) return id;
    }
    return "";
  }
  // --- Touch drags (SPEC §9.1) ----------------------------------------------
  /**
   * Where a touch that has just gone past the slop, scrolling by dx, dy so
   * far, goes. A drag, where the touched element opts in and the
   * touch-action that counts for it allows no pan along the touch's first
   * move (the larger delta; a tie pans). Otherwise a pan, as §5.3 and §9
   * have it, whatever touch-action says: where the browser would pan the
   * document but the document's touch-action stops it, the surface scrolls
   * the box itself.
   */
  claimTouch(dx, dy, e, begin) {
    const el = begin.target && isElement(begin.target) ? begin.target : null;
    const pans = this.allowedPans(el);
    if (panBlocked(pans, dx, dy) && this.touchDragStart(e, begin)) return "drag";
    if (!this.axes) {
      this.touchRoute = "terminal";
      return true;
    }
    const to = this.gestureTarget(el, dx, dy);
    if (typeof to === "string") {
      this.touchRoute = to;
      return true;
    }
    if (pans.x && pans.y) {
      this.touchRoute = "doc";
      return false;
    }
    this.touchRoute = "self";
    this.touchBox = to;
    return true;
  }
  /** Whether the touch-action that counts at `target` allows every pan. */
  pansFreely(target) {
    const p = this.allowedPans(target && isElement(target) ? target : null);
    return p.x && p.y;
  }
  /**
   * The pans the document's touch-action allows a touch on `el`, as Pointer
   * Events determine them: its own, with its ancestors' up to the nearest
   * element that scrolls (SPEC §5.3), both included. The root's is the
   * document's only where it differs from the host's (`rootTouchAction`),
   * which is there for the browser's own panning.
   */
  allowedPans(el) {
    let x = true;
    let y = true;
    const root = this.doc.documentElement;
    const host = rootTouchAction(this.axes, this.host.pageScrolls);
    for (let n = el; n; n = n.parentElement) {
      const s = n.ownerDocument.defaultView.getComputedStyle(n);
      const value = n === root && s.touchAction === host ? "auto" : s.touchAction;
      const [px, py] = touchPans(value);
      x &&= px;
      y &&= py;
      if (n !== root && this.scrolls(s)) break;
    }
    return { x, y };
  }
  /** Whether an element with this style scrolls (SPEC §5.3): its overflow is
   * `auto` or `scroll` along an axis the document asked to scroll. */
  scrolls(s) {
    return !!(this.axes & 1) && /auto|scroll/.test(s.overflowY) || !!(this.axes & 2) && /auto|scroll/.test(s.overflowX);
  }
  /**
   * A touch becomes a drag (SPEC §9.1), if it can: one that began without
   * Alt and was not held still past a long press, on an element that opts
   * in and has an id, in a surface not detached. It presses at this moment
   * as a mouse does: `press`, then `dragstart`, both for where it began,
   * then what the press causes; then a `drag` if the finger is by now over
   * another element.
   */
  touchDragStart(e, begin) {
    if (this.detachedState || begin.altKey || e.timeStamp - begin.time >= LONG_PRESS_MS) return false;
    const el = begin.target && isElement(begin.target) ? begin.target : null;
    let start = null;
    for (let n = el; n; n = n.parentElement) {
      if (listens(n, "drag")) {
        start = n;
        break;
      }
    }
    const id = start?.getAttribute("id");
    if (!id) return false;
    this.cancelDrag();
    const [c, r] = this.cellAt(begin.x, begin.y);
    const mods = modsOf2(begin);
    this.reportPress(e);
    const d = { pointer: NaN, start: id, steps: stepsOf(start), target: id, c, r, mods, step: {}, tstep: {}, clicks: 1 };
    d.step = this.stepIn(d, begin.x, begin.y);
    this.drag = d;
    this.emit("dragstart", id, { c, r, mods, ...d.step });
    this.pressAt(e);
    const t = [...e.changedTouches][0];
    if (t) this.touchDragMove(t.clientX, t.clientY, e);
    return true;
  }
  /** A touch drag's finger moved to (x, y) of the document: as a pointer's
   * move (`moveDrag`). */
  touchDragMove(x, y, e) {
    const d = this.drag;
    if (d) this.moveDrag(d, x, y, modsOf2(e));
  }
  /** A touch drag's finger lifted at (x, y): as a pointer's release
   * (`onDragUp`), a click where it began. */
  touchDragEnd(x, y, e) {
    const d = this.drag;
    if (!d) return;
    this.drag = null;
    const [c, r] = this.cellAt(x, y);
    this.endDrag(d, x, y, c, r, this.elementAtPoint(x, y, c, r), modsOf2(e));
  }
  /** The surface's cell at (x, y) of the document, counting on past its
   * edges. */
  cellAt(x, y) {
    return [Math.floor(x / this.cellW), Math.floor(y / this.cellH)];
  }
  /** The element at (x, y) of the document, if cell (c, r) is in the
   * window (SPEC §5.2). */
  elementAtPoint(x, y, c, r) {
    const w = this.win;
    if (c < w.x || c >= w.x + w.w || r < w.y || r >= w.y + w.h) return null;
    return this.doc.elementFromPoint(x, y);
  }
  // --- Presses with Alt (SPEC §9.2) ---------------------------------------
  /**
   * A press of a mouse's or a pen's primary button with Alt held is the
   * program's: the addon replays it on the cells beneath, and the surface
   * hears nothing of it (no press, drag, click, focus or selection; its
   * mousedown is cancelled). It holds the pointer until the release, as a
   * drag does, so every move and the release go to the cells, wherever the
   * pointer is. Whether it is the program's is decided here, at the press:
   * Alt let go later changes nothing.
   */
  onAltDown(e) {
    if (e.pointerType === "touch" || e.button !== 0 && e.button !== 2 || !e.altKey || e.button === 0 && !e.isPrimary) return false;
    if (e.button === 2 && !this.host.secondary()) return false;
    this.cancelDrag();
    this.dropAlt();
    this.pointerPress = true;
    this.altPointer = e.pointerId;
    this.altButton = e.button;
    this.onLeave(e);
    try {
      this.doc.documentElement.setPointerCapture(e.pointerId);
    } catch {
    }
    const [x, y] = this.toPage(e);
    this.host.cells("down", e, x, y, this.altButton);
    return true;
  }
  onAltMove(e) {
    if (e.pointerId !== this.altPointer) return false;
    const [x, y] = this.toPage(e);
    this.host.cells("move", e, x, y, this.altButton);
    return true;
  }
  onAltUp(e) {
    if (e.pointerId !== this.altPointer || e.button !== this.altButton) return false;
    this.dropAlt();
    this.altReleased = true;
    setTimeout(() => this.altReleased = false, 0);
    const [x, y] = this.toPage(e);
    this.host.cells("up", e, x, y, this.altButton);
    return true;
  }
  /** Ends such a gesture without a release: the frame goes away. */
  dropAlt() {
    const p = this.altPointer;
    if (p === null) return;
    this.altPointer = null;
    try {
      if (this.doc.documentElement.hasPointerCapture(p)) this.doc.documentElement.releasePointerCapture(p);
    } catch {
    }
  }
  /** A point of the frame's document in the page. */
  toPage(e) {
    const r = this.frame.getBoundingClientRect();
    return [r.left + e.clientX, r.top + e.clientY];
  }
  // --- Secondary presses (SPEC §9) ----------------------------------------
  /** The nearest element on `path` with `secondary` in its `data-on`. */
  secondaryIn(path) {
    if (!this.host.secondary()) return null;
    for (const n of path) if (isElement(n) && listens(n, "secondary")) return n;
    return null;
  }
  /**
   * A mouse's secondary button or a pen's barrel button, pressed (SPEC
   * §9). Over an element that takes it, with neither Shift (the terminal's
   * menu, §9.2) nor Alt (the program's, `onAltDown`) held, it is a `press`
   * with button 2, and nothing else: the mousedown is cancelled (no focus,
   * selection or `:active`), the `contextmenu` after it too (`onContextMenu`),
   * and its moves, release and auxclick report nothing. Anywhere else it is
   * the browser's, and the terminal's: the embedder's menu or the browser's.
   */
  onSecondaryDown(e) {
    const context = mac && e.button === 0 && e.ctrlKey && e.pointerType === "mouse";
    if (e.pointerType === "touch" || e.button !== 2 && !context || e.shiftKey || e.altKey) return false;
    const el = this.secondaryIn(e.composedPath());
    if (!el) return false;
    this.cancelDrag();
    this.secondaryTaken = true;
    this.reportSecondary(e, el, context);
    e.preventDefault();
    return true;
  }
  /**
   * `contextmenu` (SPEC §9, HOST.md §3): the browser's menu opens over a
   * secondary press the surface took, or over one the program owns (Alt),
   * unless cancelled here. A `contextmenu` with no pointerdown of a
   * secondary button before it is a touch's long press, or a secondary
   * button pressed while another was down (no pointerdown comes of that).
   * Over an element that takes it, it is the same press. A keyboard's menu
   * is a key (§10.2), and Shift is the terminal's: both are left alone.
   */
  onContextMenu(e) {
    if (this.secondaryTaken) {
      this.secondaryTaken = false;
      e.preventDefault();
      return;
    }
    if (this.altPointer !== null) {
      e.preventDefault();
      return;
    }
    const down = this.lastDown;
    if (down === null) return;
    const pointer = e.pointerType;
    const touch = pointer === "touch" || down?.type === "touch" && !pointer;
    if (this.detachedState || e.shiftKey || e.altKey || !(touch || e.button === 2) || this.drag) return;
    const el = this.secondaryIn(e.composedPath());
    if (!el) return;
    e.preventDefault();
    if (touch) {
      if (this.longPress?.done) return;
      this.endLongPress();
      this.finger?.take();
      this.longPress = { id: -1, x: e.clientX, y: e.clientY, el, mods: void 0, timer: null, done: true };
    }
    this.reportSecondary(e, el);
  }
  /**
   * A touch held still over an element that takes a secondary press, past
   * the long-press time (SPEC §9.1), is one. The browser's own long press
   * is a `contextmenu` on mobile and nothing on a desktop's touch screen,
   * so the surface times it. The touch is then the surface's to its lift:
   * it scrolls nothing (`Touch.take`) and its lift is cancelled, so the
   * browser makes no click of it. Shift or Alt held at the touch is the
   * terminal's (§9.2).
   */
  longPressStart(e) {
    this.endLongPress();
    const t = e.touches.length === 1 ? e.touches[0] : void 0;
    if (!t || this.detachedState || e.shiftKey || e.altKey) return;
    const el = this.secondaryIn(e.composedPath());
    if (!el) return;
    const lp = { id: t.identifier, x: t.clientX, y: t.clientY, el, mods: modsOf2(e), timer: null, done: false };
    lp.timer = setTimeout(() => {
      lp.timer = null;
      if (this.longPress !== lp || this.detachedState || !el.isConnected) return;
      lp.done = true;
      this.finger?.take();
      const [c, r] = this.cellAt(lp.x, lp.y);
      this.emit("press", el.getAttribute("id") ?? "", { button: 2, area: this.areaOf(el), c, r, mods: lp.mods });
    }, LONG_PRESS_MS);
    this.longPress = lp;
  }
  longPressMove(e) {
    const lp = this.longPress;
    if (!lp) return;
    if (lp.done) {
      if (e.cancelable) e.preventDefault();
      return;
    }
    const t = [...e.touches].find((x) => x.identifier === lp.id);
    if (e.touches.length !== 1 || !t || Math.hypot(t.clientX - lp.x, t.clientY - lp.y) >= DRAG_SLOP) this.endLongPress();
  }
  /** The touch lifted or was lost: no long press after this; one that was
   * made cancels its lift. */
  longPressEnd(e) {
    const lp = this.longPress;
    if (!lp) return;
    if (lp.done && e.type === "touchend" && e.touches.length === 0 && e.cancelable) e.preventDefault();
    if (e.touches.length === 0 || e.type === "touchcancel") this.endLongPress();
  }
  endLongPress() {
    if (this.longPress?.timer) clearTimeout(this.longPress.timer);
    this.longPress = null;
  }
  /** `press` for a secondary press on `el` (SPEC §9): its id (absent when it
   * has none), button 2, its cells, the pointer's cell and the modifier keys
   * held; `ctrl` is left out of them where the platform makes it the
   * gesture (macOS). */
  reportSecondary(e, el, noCtrl = false) {
    const [c, r] = this.cellOf(e);
    const mods = modsOf2(noCtrl ? { shiftKey: e.shiftKey, ctrlKey: false, altKey: e.altKey, metaKey: e.metaKey } : e);
    this.emit("press", el.getAttribute("id") ?? "", { button: 2, area: this.areaOf(el), c, r, mods });
  }
  /** `press` (SPEC §9), if the surface asked for it (`p`, §5.2): `t` is the
   * nearest element with an id, and the detail its cells and the modifier
   * keys held. A press on a hyperlink is the terminal's. */
  reportPress(e) {
    if (!this.presses) return;
    const link = this.linkIn(e);
    if (link && this.isHyperlink(link)) return;
    for (const n of e.composedPath()) {
      const id = isElement(n) ? n.getAttribute("id") : null;
      if (id) {
        this.emit("press", id, { area: this.areaOf(n), mods: modsOf2(e) });
        return;
      }
    }
    this.emit("press", "", { mods: modsOf2(e) });
  }
  /**
   * The cells an element's border box covers as the user sees it, scrolled
   * included (SPEC §9: `area`), counted from the surface's top left cell:
   * whole, where it is clipped or scrolled away. An edge close enough to a
   * cell's (`slack`) is on it.
   */
  areaOf(el) {
    const b = el.getBoundingClientRect();
    const dpr = this.frame.ownerDocument.defaultView?.devicePixelRatio || 1;
    const c = Math.floor((b.left + slack(b.left, this.cellW, dpr)) / this.cellW);
    const r = Math.floor((b.top + slack(b.top, this.cellH, dpr)) / this.cellH);
    const right = Math.ceil((b.right - slack(b.right, this.cellW, dpr)) / this.cellW);
    const bottom = Math.ceil((b.bottom - slack(b.bottom, this.cellH, dpr)) / this.cellH);
    return { c, r, w: Math.max(0, right - c), h: Math.max(0, bottom - r) };
  }
  // --- Events for the program (PROTOCOL §8) --------------------------------
  listen() {
    const d = this.doc;
    const win = this.frame.contentWindow;
    d.addEventListener(
      "pointerdown",
      (e) => {
        this.secondaryTaken = false;
        this.lastDown = { type: e.pointerType };
        if (this.onSecondaryDown(e)) return;
        if (e.button === 0 && e.isPrimary) this.host.pressed(e.altKey ? null : this.name);
        if (this.onAltDown(e)) return;
        if (e.button === 0) this.maskField(e.target, false);
        this.pointerPress = e.pointerType !== "touch";
        if (this.pointerPress && e.button === 0 && e.isPrimary) this.reportPress(e);
        this.onDragDown(e);
        this.onPress(e);
      },
      true
    );
    d.addEventListener("pointermove", (e) => this.onAltMove(e) || this.onDragMove(e), true);
    d.addEventListener("pointerup", (e) => this.onAltUp(e) || this.onDragUp(e), true);
    d.addEventListener("pointercancel", (e) => this.lostPointer(e), true);
    d.addEventListener("lostpointercapture", (e) => this.lostPointer(e), true);
    d.addEventListener(
      "mousedown",
      (e) => {
        if (this.altPointer !== null || this.secondaryTaken) {
          e.preventDefault();
          e.stopImmediatePropagation();
          return;
        }
        if (!this.pointerPress && e.button === 0) this.reportPress(e);
        if (this.drag && e.button === 0 && e.detail > 0) this.drag.clicks = e.detail;
        this.pointerPress = false;
        this.onPress(e);
      },
      true
    );
    d.addEventListener("focusin", (e) => {
      const el = e.target;
      if (isElement(el)) this.dirty.delete(el);
      if (isElement(el) && this.isTextField(el)) this.committed.set(el, el.value);
      this.maskField(e.target, true);
      this.onFocusIn(e);
      this.noteCaret();
    }, true);
    d.addEventListener("focusout", (e) => {
      const el = e.target;
      if (isElement(el) && this.isTextField(el)) {
        this.kept.set(el, fieldSel(el));
        this.flushCaret(el);
        this.commitField(el);
      }
      if (e.target === this.masked) this.unmaskField();
    }, true);
    d.addEventListener("selectionchange", () => this.noteCaret(), true);
    d.addEventListener("pointerup", () => this.noteCaret(), true);
    d.addEventListener("mouseup", () => this.noteCaret(), true);
    d.addEventListener("keyup", () => this.noteCaret(), true);
    d.addEventListener("compositionstart", (e) => this.composing = e.target, true);
    d.addEventListener("compositionend", () => this.composing = null, true);
    d.addEventListener("click", (e) => this.onClick(e), true);
    d.addEventListener("auxclick", (e) => this.onAuxClick(e), true);
    d.addEventListener("contextmenu", (e) => this.onContextMenu(e), true);
    d.addEventListener("pointerdown", () => this.scrollDestinations.clear(), true);
    d.addEventListener("focusin", () => this.scrollDestinations.clear(), true);
    d.addEventListener("touchstart", (e) => {
      this.scrollDestinations.clear();
      this.longPressStart(e);
    }, { capture: true, passive: true });
    d.addEventListener("touchmove", (e) => this.longPressMove(e), { capture: true, passive: false });
    d.addEventListener("touchend", (e) => this.longPressEnd(e), { capture: true, passive: false });
    d.addEventListener("touchcancel", (e) => this.longPressEnd(e), { capture: true, passive: true });
    d.addEventListener("mouseover", (e) => this.onOver(e), true);
    d.addEventListener("mouseout", (e) => this.onOut(e), true);
    d.addEventListener("submit", (e) => this.onSubmit(e), true);
    d.addEventListener("change", (e) => this.onChange(e), true);
    d.addEventListener("beforeinput", (e) => this.onBeforeInput(e), true);
    d.addEventListener("input", (e) => this.onInput(e), true);
    d.addEventListener("keydown", (e) => {
      this.lastDown = null;
      this.onKey(e);
      this.noteCaret();
    }, true);
    d.addEventListener("keyup", (e) => this.onKeyUp(e), true);
    d.addEventListener("dragstart", (e) => e.preventDefault(), true);
    d.addEventListener("wheel", (e) => this.onWheel(e), { capture: true, passive: false });
    d.addEventListener("scroll", (e) => this.onScroll(e), true);
    d.addEventListener("scroll", (e) => this.viewScrolled(e), true);
    d.addEventListener("scrollend", (e) => this.scrollEnded(e), true);
    d.addEventListener("load", () => this.relayout(), true);
    d.addEventListener("error", () => this.relayout(), true);
    d.fonts.addEventListener("loadingdone", () => this.relayout());
    for (const kind of ["toggle", "transitionend", "animationend"]) d.addEventListener(kind, () => this.requestView(), true);
    const toPage = (x, y) => {
      const r = this.frame.getBoundingClientRect();
      return [r.left + x, r.top + y];
    };
    this.finger = new Touch(
      d,
      this.frame.ownerDocument.defaultView,
      {
        scroll: (dx, dy, x, y) => {
          if (this.touchRoute === "self") this.touchBox?.scrollBy(dx, dy);
          else if (this.touchRoute !== "terminal") return;
          else if (this.host.pageScrolls) this.host.scrollPage(dx, dy);
          else this.host.wheel(new WheelEvent("wheel", { deltaX: dx, deltaY: dy, deltaMode: 0, clientX: x, clientY: y }), x, y);
        },
        toPage,
        claim: (dx, dy, e, begin) => this.claimTouch(dx, dy, e, begin),
        dragMove: (x, y, e) => this.touchDragMove(x, y, e),
        dragEnd: (x, y, e) => this.touchDragEnd(x, y, e),
        dragCancel: () => this.cancelDrag()
      },
      false,
      // Where the page scrolls, the browser pans it from a touch the
      // surface leaves alone.
      this.host.pageScrolls ? (e) => !this.axes && this.inScroller(e.target) || !this.pansFreely(e.target) : void 0
    );
    win.addEventListener("focus", () => {
      if (!this.programFocus && !this.auxPress && !this.keyboard) this.settle();
    });
    win.addEventListener("blur", () => {
      this.prefix = false;
      this.commit();
      if (this.keyboard) {
        this.keyboard = false;
        this.told = null;
        this.emit("blur", "");
      }
    });
  }
  /**
   * Nothing in a surface scrolls, and a wheel over it is the terminal's
   * (SPEC §5.3, §9). Wheel events never leave an iframe, so without this
   * the terminal's scrollback (or, on the alternate screen, the program's
   * wheel input) would stall under the pointer. Ctrl and the wheel stay the
   * browser's: its zoom. Where the page scrolls (the `scroll` option), the
   * wheel is the browser's too: nothing in the frame scrolls, so it scrolls
   * the page, as over the cells; but over an element it would scroll
   * instead, the surface scrolls the page itself.
   */
  onWheel(e) {
    this.scrollDestinations.clear();
    if (e.ctrlKey) return;
    if (this.axes) {
      this.scrollWheel(e);
      return;
    }
    if (this.host.pageScrolls) {
      if (!this.inScroller(e.target)) return;
      e.preventDefault();
      const px = e.deltaMode === WheelEvent.DOM_DELTA_PIXEL ? 1 : e.deltaMode === WheelEvent.DOM_DELTA_LINE ? 20 : 400;
      this.host.scrollPage(e.deltaX * px, e.deltaY * px);
      return;
    }
    e.preventDefault();
    const r = this.frame.getBoundingClientRect();
    this.host.wheel(e, r.left + e.clientX, r.top + e.clientY);
  }
  /** Whether a wheel or a touch at target would scroll an element of the
   * document, overflowing with `overflow: auto` or `scroll`: the browser
   * gives the gesture to it before the page, and nothing in a surface
   * scrolls (SPEC §5.3), so the gesture would go nowhere. */
  inScroller(target) {
    for (let n = target && isElement(target) ? target : null; n; n = n.parentElement) {
      const s = n.ownerDocument.defaultView.getComputedStyle(n);
      const y = /auto|scroll/.test(s.overflowY) && n.scrollHeight > n.clientHeight;
      const x = /auto|scroll/.test(s.overflowX) && n.scrollWidth > n.clientWidth;
      if (x || y) return true;
    }
    return false;
  }
  /**
   * A wheel over a document that scrolls (SPEC §5.3). A gesture's first
   * wheel decides where all of it goes, as a browser latches scrolling to
   * what it began on: while an element under the pointer can still move
   * that way, the browser scrolls it; where none can, the gesture goes on
   * to the terminal, as over the cells beneath (§9), unless
   * `overscroll-behavior` stops it. A gesture the document took goes no
   * further when it reaches the end there, as in a browser. Where the page
   * scrolls (the `scroll` option), the page is the terminal, and the
   * browser chains to it from the document natively. A wheel the browser
   * does not let the page cancel is the browser's already.
   */
  scrollWheel(e) {
    if (this.host.pageScrolls || !e.cancelable) return;
    const route = this.host.wheelGesture(this.name, () => {
      const to = this.gestureTarget(e.target, e.deltaX, e.deltaY);
      this.wheelBox = typeof to === "string" ? null : to;
      return typeof to === "string" ? to : "doc";
    });
    const [axis, d] = mainAxis(e.deltaX, e.deltaY);
    if (route === "doc" && this.wheelBox && this.canMove(this.wheelBox, axis, d)) return;
    e.preventDefault();
    if (route !== "terminal") return;
    const r = this.frame.getBoundingClientRect();
    this.host.wheel(e, r.left + e.clientX, r.top + e.clientY);
  }
  /** Where a gesture that scrolls by dx, dy pixels at `target` goes, by its
   * main axis (SPEC §5.3). */
  gestureRoute(target, dx, dy) {
    const to = this.gestureTarget(target, dx, dy);
    return typeof to === "string" ? to : "doc";
  }
  gestureTarget(target, dx, dy) {
    const [axis, d] = mainAxis(dx, dy);
    return this.scroller(target && isElement(target) ? target : null, axis, d);
  }
  /** Whether a box that scrolls can still move along `axis`, `d`'s way. */
  canMove(el, axis, d) {
    if (!d || !(this.axes & (axis === "y" ? 1 : 2))) return false;
    const dest = this.scrollDestinations.get(el);
    const [at, span] = axis === "y" ? [dest?.top ?? el.scrollTop, el.scrollHeight - el.clientHeight] : [dest?.left ?? el.scrollLeft, el.scrollWidth - el.clientWidth];
    const rtl = axis === "x" && this.frame.contentWindow.getComputedStyle(el).direction === "rtl";
    const [min, max] = rtl ? [-span, 0] : [0, span];
    return d > 0 ? at < max - 0.5 : at > min + 0.5;
  }
  /**
   * What a scroll along `axis`, `d`'s way, from `from` (an element, or the
   * root for none) moves in a document that scrolls (SPEC §5.3): the
   * innermost box, from `from` outward, that the user scrolls along it
   * (`overflow` `auto` or `scroll`; the root's, unless `hidden` or `clip`),
   * that overflows there, and that can still move that way. At a box that
   * cannot, the scroll stops if its `overscroll-behavior` is `contain` or
   * `none`, and goes on outward otherwise. Past the root, and along an axis
   * the document did not ask for, it is the terminal's.
   */
  scroller(from, axis, d) {
    if (!d || !(this.axes & (axis === "y" ? 1 : 2))) return "terminal";
    const win = this.frame.contentWindow;
    const root = this.doc.documentElement;
    const body = this.doc.body;
    const rootStyle = win.getComputedStyle(root);
    const fromBody = rootStyle.overflowX === "visible" && rootStyle.overflowY === "visible";
    const prop = axis === "y" ? "overflowY" : "overflowX";
    for (let el = from ?? root; el; el = el.parentElement) {
      if (el === body && fromBody) continue;
      const s = el === root ? rootStyle : win.getComputedStyle(el);
      const scrolls2 = el === root ? !/hidden|clip/.test((fromBody ? win.getComputedStyle(body) : s)[prop]) : /auto|scroll/.test(s[prop]);
      const span = axis === "y" ? el.scrollHeight - el.clientHeight : el.scrollWidth - el.clientWidth;
      if (!scrolls2 || span < 1) continue;
      if (this.canMove(el, axis, d)) return el;
      const stops = axis === "y" ? s.overscrollBehaviorY : s.overscrollBehaviorX;
      if (stops === "contain" || stops === "none") return "stop";
    }
    return "terminal";
  }
  /** Whatever the browser scrolled along an axis the document did not ask
   * for (a focused element into view, say) goes back to zero. A text
   * field's own text follows its caret (SPEC §5.3). */
  onScroll(e) {
    if (this.axes === 3) return;
    const t = e.target;
    const el = t === this.doc ? this.doc.scrollingElement : t && isElement(t) ? t : null;
    if (!el || t !== this.doc && (el.localName === "input" || el.localName === "textarea")) return;
    const x = !(this.axes & 2) && el.scrollLeft !== 0;
    const y = !(this.axes & 1) && el.scrollTop !== 0;
    if (x || y) el.scrollTo(x ? 0 : el.scrollLeft, y ? 0 : el.scrollTop);
  }
  /** An event for the program (SPEC §9); a detached surface sends none (§5.5). */
  emit(kind, target, detail) {
    if (this.detachedState) return;
    this.host.event(this.name, kind, target, detail);
  }
  linkIn(e) {
    for (const n of e.composedPath()) {
      if (isElement(n) && n.localName === "a" && n.hasAttribute("href")) return n;
    }
    return null;
  }
  /** A link's `url` (SPEC §9): its href resolved against the document's
   * base, or "" under hotty.invalid or when it is no URL. */
  urlOf(link) {
    const href = this.resolver.get(link, "href") ?? "";
    let url = "";
    try {
      url = new URL(href, this.base).href;
    } catch {
    }
    return url.startsWith(NO_BASE) ? "" : url;
  }
  /** The link's box in the page (the iframe may be offset for a window). */
  pageBox(el) {
    const f = this.frame.getBoundingClientRect();
    const r = el.getBoundingClientRect();
    return new DOMRect(f.left + r.left, f.top + r.top, r.width, r.height);
  }
  /** A hyperlink (SPEC §9): a link with target="_blank" and a `url`. One
   * without a url is the program's, as any other link. */
  isHyperlink(link) {
    return link.getAttribute("target") === "_blank" && this.urlOf(link) !== "";
  }
  hyper = (link) => this.isHyperlink(link);
  /** A link's click (SPEC §9). A hyperlink is the terminal's, as an OSC 8
   * one is, and not reported; any other link is the program's: an event,
   * with or without an id. The host opens nothing itself. */
  onLink(link, e) {
    e.preventDefault();
    const url = this.urlOf(link);
    if (this.isHyperlink(link)) {
      if (e.button === 0) this.host.hyperlink("activate", e, url, this.pageBox(link));
      return;
    }
    const detail = { href: this.resolver.get(link, "href") ?? "" };
    if (url) detail.url = url;
    detail.area = this.areaOf(link);
    detail.mods = modsOf2(e);
    const n = this.host.clickCount(this.name, link, e.detail);
    if (n > 1) detail.count = n;
    this.emit("click", link.getAttribute("id") ?? "", detail);
  }
  /** Other buttons on a link do nothing: no navigation, no new tab. */
  onAuxClick(e) {
    if (this.linkIn(e)) e.preventDefault();
  }
  hovered = null;
  /** The pointer onto a hyperlink: the terminal hears it, as it does over
   * an OSC 8 one. */
  onOver(e) {
    if (this.altPointer !== null) return;
    const link = this.linkIn(e);
    const hyper = link && this.isHyperlink(link) ? link : null;
    if (hyper === this.hovered?.link) return;
    this.onLeave(e);
    const url = hyper ? this.urlOf(hyper) : "";
    if (hyper) {
      this.hovered = { link: hyper, url };
      this.host.hyperlink("hover", e, url, this.pageBox(hyper));
    }
  }
  onOut(e) {
    if (this.altPointer !== null) return;
    const to = e.relatedTarget;
    if (this.hovered && to && this.hovered.link.contains(to)) return;
    this.onLeave(e);
  }
  onLeave(e) {
    const h = this.hovered;
    if (!h) return;
    this.hovered = null;
    this.host.hyperlink("leave", e, h.url, this.pageBox(h.link));
  }
  onClick(e) {
    const link = this.linkIn(e);
    if (this.altReleased) {
      this.altReleased = false;
      e.preventDefault();
      e.stopImmediatePropagation();
      return;
    }
    if (this.dragReleased) {
      this.dragReleased = false;
      if (link) e.preventDefault();
      return;
    }
    if (link) {
      this.onLink(link, e);
      return;
    }
    this.reportClick(e.composedPath(), e.detail, modsOf2(e));
  }
  /** The click of the nearest element on `path` that reports one (SPEC §9),
   * the `count`th of its run as the platform counts clicks (`detail`; a
   * drag's release is the first), with the modifier keys held at its
   * release. */
  reportClick(path, count = 1, mods) {
    for (const n of path) {
      if (!isElement(n)) continue;
      if (n.localName === "a" && n.hasAttribute("href")) {
        if (this.isHyperlink(n)) return;
        const url = this.urlOf(n);
        const detail = { href: this.resolver.get(n, "href") ?? "" };
        if (url) detail.url = url;
        detail.area = this.areaOf(n);
        detail.mods = mods;
        const c = this.host.clickCount(this.name, n, count);
        if (c > 1) detail.count = c;
        this.emit("click", n.getAttribute("id") ?? "", detail);
        return;
      }
      const el = n;
      const tag = el.localName;
      const type = (el.getAttribute("type") ?? "").toLowerCase();
      const wants = (el.getAttribute("data-on") ?? "").split(/\s+/).includes("click");
      const reportable = tag === "button" || tag === "a" || tag === "summary" || tag === "input" && ["button", "submit", "reset"].includes(type) || wants;
      if (!reportable) continue;
      const id = el.getAttribute("id");
      if (id) {
        const detail = {};
        const href = this.resolver.get(el, "href");
        if (href !== null) detail.href = href;
        const value = this.resolver.get(el, "value");
        if (value !== null) detail.value = value;
        detail.area = this.areaOf(el);
        detail.mods = mods;
        const c = this.host.clickCount(this.name, el, count);
        if (c > 1) detail.count = c;
        this.emit("click", id, detail);
      }
      break;
    }
  }
  onSubmit(e) {
    e.preventDefault();
    const form = e.target;
    const id = form.getAttribute("id");
    if (!id) return;
    const fields = [];
    const data = new this.frame.contentWindow.FormData(form, e.submitter ?? void 0);
    for (const [k, v] of data) if (typeof v === "string") fields.push([k, v]);
    this.emit("submit", id, { fields });
  }
  /** A control's value committed (SPEC §9): a toggle, a check or a pick at
   * once, a text field's edits on Enter or as focus leaves it
   * (`commitField`). Each toggle and check is one more `rev`, and so is a
   * pick that no `input` counted (§6.4). */
  onChange(e) {
    if (this.applying) return;
    const el = e.target;
    if (!isElement(el) || !isControl(el)) return;
    const check = el.localName === "input" && (el.type === "checkbox" || el.type === "radio");
    if (check) this.bump(el);
    else if (this.isTextField(el)) return this.commitField(el);
    else if (!this.picked.delete(el)) this.bump(el);
    this.flushCaret(el);
    const id = el.getAttribute("id");
    if (!id) return;
    const rev = this.revOf(el);
    this.emit("change", id, check ? { checked: el.checked, value: el.value, rev } : { value: el.value, rev });
  }
  /** A text field's edits committed (SPEC §9 `change`): by Enter in an
   * `input`, whose `change` the browser fires, by the `submit` action
   * (§10.2), or by focus leaving it. Only the user's edits, and only a
   * value changed since the field last committed: a field that only
   * a=field changed sends none (§6.4), one committed already none again
   * when the browser's own `change` follows, and one edited back to what
   * it was none at all. */
  commitField(el) {
    if (!this.dirty.delete(el)) return;
    const value = el.value;
    if (this.committed.get(el) === value) return;
    this.committed.set(el, value);
    this.flushCaret(el);
    const id = el.getAttribute("id");
    if (id) this.emit("change", id, { value, rev: this.revOf(el) });
  }
  /** An edit the browser makes (typing, a paste): what the field was, for
   *  its history (SPEC §10.2, undo). A move of the caret before it is
   *  heard before it (`caret`). */
  onBeforeInput(e) {
    const el = e.target;
    if (this.applying || this.editing) return;
    if (isElement(el)) this.flushCaret(el);
    if (!isValueField(el) || e.inputType.startsWith("history")) return;
    this.pending = { el, before: stateOf(el), run: runOf(e.inputType) };
  }
  /** Puts an edit in its field's history: the one pending, else one from
   *  where the field last was. Undo's and redo's own are not edits. */
  record(e) {
    const el = e.target;
    const p = this.pending;
    this.pending = null;
    if (!isValueField(el) || e.inputType?.startsWith("history")) return;
    let h = this.histories.get(el);
    if (!h) this.histories.set(el, h = new History());
    const before = p?.el === el ? p.before : h.after;
    if (before) edited(h, before, stateOf(el), p?.el === el ? p.run : null);
  }
  /** Undoes a field's last edit, or with `redo` redoes the one last undone
   *  (SPEC §10.2), sending `input` as for an edit. An editing host has the
   *  browser's own history. */
  undo(el, redo) {
    this.goal = null;
    if (!isValueField(el)) {
      this.doc.execCommand(redo ? "redo" : "undo", false);
      return;
    }
    const h = this.histories.get(el);
    const s = h ? step(h, stateOf(el), redo) : null;
    if (!s) return;
    this.flushCaret(el);
    el.value = s.value;
    el.setSelectionRange(s.start, s.end, s.backward ? "backward" : "forward");
    const win = this.frame.contentWindow;
    el.dispatchEvent(new win.InputEvent("input", { bubbles: true, inputType: redo ? "historyRedo" : "historyUndo" }));
  }
  /** Cuts a field's selection to the clipboard (SPEC §10.2). */
  cut(el) {
    this.goal = null;
    if (!isValueField(el)) {
      const sel = this.doc.getSelection();
      if (!sel || sel.isCollapsed) return;
      this.host.writeClipboard(sel.toString());
      this.doc.execCommand("delete", false);
      return;
    }
    const [from, to] = [el.selectionStart ?? 0, el.selectionEnd ?? 0];
    this.host.writeClipboard(el.value.slice(from, to));
    this.replace(el, from, to, "", null);
  }
  /** Types the clipboard's text in place of a field's selection, as
   *  `paste` does where the browser does not (SPEC §10.2): an `input`
   *  drops line breaks. */
  pasteFrom(el, multiline) {
    void this.host.readClipboard().then((t) => {
      if (t === null || t === "" || this.doc.activeElement !== el) return;
      const text = multiline ? t.replace(/\r\n?/g, "\n") : t.replace(/[\r\n]/g, "");
      if (!isValueField(el)) {
        this.doc.execCommand("insertText", false, text);
        return;
      }
      this.replace(el, el.selectionStart ?? 0, el.selectionEnd ?? 0, text, null);
    });
  }
  /** An edit of a control's value, or a pick (SPEC §9): one more `rev`
   * (§6.4), and `input` with `data-on~=input`, its selection with it in a
   * text field with `data-on~=caret`; without `input`, that field's
   * `caret` says the edit. A toggle and a check count at their `change`;
   * a change of a=field's is none of these. */
  onInput(e) {
    if (this.applying) return;
    this.record(e);
    const el = e.target;
    if (!isElement(el) || !isControl(el)) return;
    if (el.localName === "input" && (el.type === "checkbox" || el.type === "radio")) return;
    this.bump(el);
    const text = this.isTextField(el);
    if (text) this.dirty.add(el);
    else this.picked.add(el);
    const id = el.getAttribute("id");
    if (!id || !listens(el, "input")) {
      this.noteCaret(el);
      return;
    }
    const rev = this.revOf(el);
    if (text && listens(el, "caret")) {
      const sel = wireSel(el.value, this.selectionOf(el));
      this.heard.set(el, { sel, rev });
      this.emit("input", id, { value: el.value, rev, sel });
    } else this.emit("input", id, { value: el.value, rev });
  }
  // --- Fields (SPEC §6.4) ---------------------------------------------------
  /**
   * `a=field`: changes a control as the program says, focused or not, if
   * the user has not changed it since `rev`, and sends nothing back. It is
   * checked whole before anything changes: the error is the first of
   * `EDETACHED`, `ENOTARGET` (no element `target`), `EINVAL` and `ESTALE`
   * (the user changed it since, or an input method composes in it) that
   * holds, as a `FieldError`. Positions come last, `EINVAL` past the end
   * or inside a code point only when the command is not stale: they count
   * bytes of the value the program heard.
   */
  field(target, body) {
    if (this.detachedState) throw new FieldError("EDETACHED", `surface ${this.name} is detached`);
    if (!target) throw new FieldError("EINVAL", "a=field needs t");
    const el = this.doc.getElementById(target);
    if (!el) throw new FieldError("ENOTARGET", target);
    const change = readField(body);
    const invalid2 = (why) => new FieldError("EINVAL", why);
    const kind = this.fieldKind(el);
    if (kind === null) throw invalid2(`${target} is not a control`);
    if (kind !== "text" && (change.text !== void 0 || change.sel !== void 0)) throw invalid2("text, from, to and sel are for text fields");
    if (kind === "check" && change.value !== void 0) throw invalid2("value is not for checkboxes and radio buttons");
    if (kind !== "check" && change.checked !== void 0) throw invalid2("checked is for checkboxes and radio buttons");
    const stale = () => {
      const rev = this.revOf(el);
      if (change.rev !== rev || this.composing === el) throw new FieldError("ESTALE", String(rev));
    };
    if (kind === "text") {
      this.fieldText(el, change, stale);
      return;
    }
    if (kind === "select") {
      const s = el;
      const i = Array.from(s.options).findIndex((o) => o.value === change.value);
      if (i < 0) throw invalid2(`${target} has no option ${JSON.stringify(change.value)}`);
      stale();
      s.selectedIndex = i;
      return;
    }
    stale();
    const c = el;
    if (kind === "check") c.checked = change.checked;
    else c.value = change.value;
  }
  /** `a=field` in a text field (SPEC §6.4): if it is not stale, positions
   * checked against its value, `sel` against the new one, then the edit
   * made as one of its history, and the selection set, or kept while it is
   * not focused. */
  fieldText(f, change, stale) {
    const invalid2 = (why) => new FieldError("EINVAL", why);
    stale();
    const now = f.value;
    const sel = this.selectionOf(f);
    let value = now;
    let edit = null;
    let after = sel;
    if (change.value !== void 0) {
      value = this.sanitized(f, change.value);
      edit = { from: 0, to: now.length, text: value };
      after = { start: value.length, end: value.length, backward: false };
    } else if (change.text !== void 0) {
      let [from, to] = [sel.start, sel.end];
      if (change.from !== void 0) {
        const a = unitAt(now, change.from);
        const b = unitAt(now, change.to);
        if (a === null || b === null) throw invalid2("from or to is negative, past the end of the value, or inside a code point");
        [from, to] = [a, b];
      }
      const text = f.localName === "textarea" ? change.text.replace(/\r\n?/g, "\n") : change.text.replace(/[\r\n]/g, "");
      value = now.slice(0, from) + text + now.slice(to);
      edit = { from, to, text };
      after = { start: from + text.length, end: from + text.length, backward: false };
    }
    if (change.sel !== void 0) {
      const [a, c] = change.sel.map((b) => unitAt(value, b));
      if (a == null || c == null) throw invalid2("sel is negative, past the end of the value, or inside a code point");
      const [anchor, caret] = [clusterStart(value, a), clusterStart(value, c)];
      after = { start: Math.min(anchor, caret), end: Math.max(anchor, caret), backward: anchor > caret };
    }
    const focused = this.doc.activeElement === f;
    this.applying = true;
    try {
      if (edit && focused) this.insert(f, edit.from, edit.to, edit.text, value);
      else if (edit) f.value = value;
      if (focused) f.setSelectionRange(after.start, after.end, after.backward ? "backward" : "forward");
      else this.kept.set(f, after);
    } finally {
      this.applying = false;
    }
    let h = this.histories.get(f);
    if (!h) this.histories.set(f, h = new History());
    edited(h, { value: now, ...sel }, { value, ...after }, null);
    if (this.pending?.el === f) this.pending = null;
    if (this.goal?.el === f) this.goal = null;
    if (this.caretField === f) this.caretField = null;
    if (listens(f, "caret")) this.heard.set(f, { sel: wireSel(value, after), rev: this.revOf(f) });
  }
  /** Puts `text` in place of code units `from` to `to` of a focused field
   * as a paste does, the editing command's events the field's own
   * (`applying`): the browser commits it with the user's edits as focus
   * leaves. Its value is then `value`, as the command left it or as set
   * where the command does not (a `maxlength`, a read-only field). */
  insert(f, from, to, text, value) {
    f.setSelectionRange(from, to);
    if (text) this.doc.execCommand("insertText", false, text);
    else if (from !== to) this.doc.execCommand("delete", false);
    if (f.value !== value) f.setRangeText(value, 0, f.value.length, "end");
  }
  /** `value` as `f` takes it from a script (SPEC §6.4): sanitized as its
   * type says, the program's for a field shown as `text` (`NO_CARET`). */
  sanitized(f, value) {
    if (f.localName === "textarea") return value.replace(/\r\n?/g, "\n");
    const probe = f.cloneNode(false);
    if (f === this.masked) probe.setAttribute("type", this.resolver.get(f, "type") ?? "text");
    probe.value = value;
    return probe.value;
  }
  /** What `a=field` takes `el` for (SPEC §6.4): a text field, a checkbox
   * or a radio button, a select, another control; null for what is none. */
  fieldKind(el) {
    if (el.localName === "textarea") return "text";
    if (el.localName === "select") return "select";
    if (el.localName !== "input") return null;
    const type = this.inputType(el);
    if (FIELD_TYPES.has(type)) return "text";
    if (type === "checkbox" || type === "radio") return "check";
    return NOT_CONTROLS.has(type) ? null : "other";
  }
  /** A text field of §10.2: an input of a text type, or a textarea. */
  isTextField(el) {
    return this.fieldKind(el) === "text";
  }
  /** A control's `rev` (SPEC §6.4); a radio button's is its group's. */
  revOf(el) {
    const group = radioGroup(el);
    return group ? Math.max(0, ...group.map((b) => this.revs.get(b) ?? 0)) : this.revs.get(el) ?? 0;
  }
  /** One more change of the user's to `el`: to its whole group, for a
   * radio button. */
  bump(el) {
    const rev = this.revOf(el) + 1;
    for (const b of radioGroup(el) ?? [el]) this.revs.set(b, rev);
  }
  /** A text field's selection: its own while focused, the one it keeps
   * otherwise (SPEC §10.2), the caret at the end of the value if none. */
  selectionOf(f) {
    return this.doc.activeElement === f ? fieldSel(f) : this.keptSel(f);
  }
  /** The selection a field keeps while it is not focused. */
  keptSel(f) {
    const n = f.value.length;
    const k = this.kept.get(f);
    return k ? { start: Math.min(k.start, n), end: Math.min(k.end, n), backward: k.backward } : { start: n, end: n, backward: false };
  }
  /** `a=focus` gives a field its selection back as it was (SPEC §10.1). */
  restoreSelection(f) {
    const s = this.keptSel(f);
    f.setSelectionRange(s.start, s.end, s.backward ? "backward" : "forward");
    if (this.caretField === f) this.caretField = null;
    if (listens(f, "caret")) this.heard.set(f, { sel: wireSel(f.value, s), rev: this.revOf(f) });
  }
  /** A delta set a field's value (SPEC §6.2): its caret goes to the end,
   * where the program knows it is. */
  valueSet(el) {
    if (!this.isTextField(el)) return;
    this.kept.delete(el);
    if (this.doc.activeElement === el) el.setSelectionRange(el.value.length, el.value.length);
    if (this.caretField === el) this.caretField = null;
    const end = byteAt(el.value, el.value.length);
    if (listens(el, "caret")) this.heard.set(el, { sel: [end, end], rev: this.revOf(el) });
  }
  /** The user may have moved the selection of the focused field, or `el`
   * (SPEC §9, `caret`): the next frame checks it, once however often it
   * moved meanwhile. Another field's check, waiting, is made first. */
  noteCaret(el = this.focusedControl()) {
    if (!el || this.detachedState || !this.isTextField(el) || !listens(el, "caret")) return;
    if (this.caretField && this.caretField !== el) this.flushCaret(this.caretField);
    this.caretField = el;
    if (this.caretFrame) return;
    this.caretFrame = this.frame.ownerDocument.defaultView.requestAnimationFrame(() => {
      this.caretFrame = 0;
      const f = this.caretField;
      this.caretField = null;
      if (f) this.checkCaret(f);
    });
  }
  /** Makes `el`'s check now, if one waits: before another event of the
   * field (SPEC §9). */
  flushCaret(el) {
    if (this.caretField !== el) return;
    this.caretField = null;
    this.checkCaret(el);
  }
  /** `caret`, if the field's selection or its `rev` is not what the program
   * last heard: the user moved it, or edited a field that sends no
   * `input`. */
  checkCaret(el) {
    if (!el.isConnected || this.detachedState || !this.isTextField(el) || !listens(el, "caret")) return;
    const sel = wireSel(el.value, this.selectionOf(el));
    const rev = this.revOf(el);
    const end = byteAt(el.value, el.value.length);
    const h = this.heard.get(el) ?? { sel: [end, end], rev: 0 };
    if (h.sel[0] === sel[0] && h.sel[1] === sel[1] && h.rev === rev) return;
    this.heard.set(el, { sel, rev });
    const id = el.getAttribute("id");
    if (id) this.emit("caret", id, { sel, rev });
  }
};
var NOT_CONTROLS = /* @__PURE__ */ new Set(["hidden", "button", "submit", "reset", "image", "file"]);
function isControl(el) {
  const n = el.localName;
  return n === "input" || n === "textarea" || n === "select";
}
function radioGroup(el) {
  const b = el;
  if (b.localName !== "input" || b.type !== "radio") return null;
  const name = b.getAttribute("name");
  if (!name || !b.isConnected) return null;
  return Array.from(b.ownerDocument.querySelectorAll("input")).filter((r) => r.type === "radio" && r.getAttribute("name") === name && r.form === b.form);
}
function fieldSel(f) {
  const n = f.value.length;
  return { start: f.selectionStart ?? n, end: f.selectionEnd ?? n, backward: f.selectionDirection === "backward" };
}
function isValueField(el) {
  const n = el?.localName;
  return n === "input" || n === "textarea";
}
function isElement(n) {
  return n.nodeType === 1;
}
function listens(el, what) {
  return (el.getAttribute("data-on") ?? "").split(/\s+/).includes(what);
}
function stepsOf(el) {
  const n = parseSteps(el.getAttribute("data-steps"));
  return n && { el, x: n[0], y: n[1] };
}
function stepInTarget(t, x, y) {
  const n = t && listens(t, "drop") && parseSteps(t.getAttribute("data-steps"));
  if (!t || !n) return {};
  const box = t.getBoundingClientRect();
  const step2 = {};
  if (n[0]) step2.tx = stepAt(x - box.left, box.width, n[0]);
  if (n[1]) step2.ty = stepAt(y - box.top, box.height, n[1]);
  return step2;
}
function ancestry(el) {
  const out = [];
  for (let n = el; n; n = n.parentElement) out.push(n);
  return out;
}
function modsOf2(e) {
  const bits = (e.shiftKey ? 1 : 0) | (e.altKey ? 2 : 0) | (e.ctrlKey ? 4 : 0) | (e.metaKey ? 8 : 0);
  return bits || void 0;
}
function takesFocus(el, hyper) {
  if (el.matches(":disabled")) return false;
  switch (el.localName) {
    case "button":
    case "select":
    case "textarea":
      return true;
    case "input":
      return el.type !== "hidden";
    case "a":
    case "area":
      if (el.hasAttribute("href")) return !hyper(el);
      break;
    case "summary": {
      const d = el.parentElement;
      if (d?.localName === "details" && d.querySelector(":scope > summary") === el) return true;
      break;
    }
  }
  const ce = el.getAttribute("contenteditable");
  if (ce !== null && ce.toLowerCase() !== "false" && el.isContentEditable) return true;
  return el.hasAttribute("tabindex") && el.tabIndex >= 0;
}
function focusTargetAt(e, hyper) {
  for (const n of e.composedPath()) {
    if (!isElement(n)) continue;
    if (n.localName === "a" && n.hasAttribute("href") && hyper(n)) return null;
    if (takesFocus(n, hyper)) return n;
    const control = n.localName === "label" ? n.control : null;
    if (control) return takesFocus(control, hyper) ? control : null;
  }
  return null;
}
function mainAxis(dx, dy) {
  return Math.abs(dy) >= Math.abs(dx) ? ["y", dy] : ["x", dx];
}
function keyScroll(e) {
  if (e.ctrlKey || e.altKey || e.metaKey) return null;
  if (e.shiftKey) return e.key === " " ? { axis: "y", sign: -1, by: "page" } : null;
  switch (e.key) {
    case "ArrowDown":
      return { axis: "y", sign: 1, by: "line" };
    case "ArrowUp":
      return { axis: "y", sign: -1, by: "line" };
    case "ArrowRight":
      return { axis: "x", sign: 1, by: "line" };
    case "ArrowLeft":
      return { axis: "x", sign: -1, by: "line" };
    case "PageDown":
    case " ":
      return { axis: "y", sign: 1, by: "page" };
    case "PageUp":
      return { axis: "y", sign: -1, by: "page" };
    case "End":
      return { axis: "y", sign: 1, by: "end" };
    case "Home":
      return { axis: "y", sign: -1, by: "end" };
  }
  return null;
}
function listOpen(el) {
  if (el?.localName !== "select") return false;
  try {
    return el.matches(":open");
  } catch {
    return false;
  }
}
function keysValues(el) {
  const values = [];
  for (let n = el; n; n = n.parentElement) {
    const v = n.getAttribute("data-keys");
    if (v !== null) values.unshift(v);
  }
  return values;
}
function typesNoText(e) {
  if (e.getModifierState?.("AltGraph")) return false;
  if ([...e.key].length === 1) return e.ctrlKey || e.metaKey;
  return !NOT_KEYS.has(e.key);
}
function typedKeys(e, data) {
  if (data === null) return [{ key: pressNames(e).at(-1) ?? null, data: "" }];
  const keys = inputKeys(data);
  return e.metaKey ? keys.map((k) => k.key !== null && !/(^|\+)super\+/.test(k.key) ? { ...k, key: null } : k) : keys;
}
function asked(keymap, e, data) {
  const named = typesNoText(e) ? pressNames(e).find((n) => keymap.binds(n)) : void 0;
  return named !== void 0 ? [named] : typedKeys(e, data).flatMap((k) => k.key === null ? [] : [k.key]);
}
function submitFrom(el) {
  const form = el.form ?? el.closest("form");
  if (!form) return;
  const controls = Array.from(form.elements);
  const button = controls.find((c) => c.localName === "button" && (c.type || "submit") === "submit" || c.localName === "input" && (c.type === "submit" || c.type === "image"));
  if (button) {
    if (!button.disabled) button.click();
    return;
  }
  const blocking = controls.filter((c) => c.localName === "input" && (FIELD_TYPES.has(c.type) || DATE_TYPES.has(c.type)));
  if (blocking.length <= 1) form.requestSubmit();
}

// src/version.ts
var VERSION = "0.8.0";
var HOST = "@neuroplastio/xterm-addon-hotty";

// src/wire.ts
var OSC = 7279;
var CHUNK = 4096;
var MAX_COMMAND = 16 << 20;
var encodedMax = (max) => Math.ceil((max + Math.floor(max / 256) + 64) / 3) * 4;
var Control = class _Control {
  pairs;
  constructor(pairs = []) {
    this.pairs = pairs;
  }
  static parse(text) {
    const pairs = [];
    if (text === "") return new _Control(pairs);
    for (const part of text.split(":")) {
      const eq = part.indexOf("=");
      if (eq <= 0) throw new Error(`bad control pair ${JSON.stringify(part)}`);
      pairs.push([part.slice(0, eq), part.slice(eq + 1)]);
    }
    return new _Control(pairs);
  }
  get(key) {
    for (const [k, v] of this.pairs) if (k === key) return v;
    return void 0;
  }
  set(key, value) {
    const i = this.pairs.findIndex(([k]) => k === key);
    if (i >= 0) this.pairs[i] = [key, value];
    else this.pairs.push([key, value]);
    return this;
  }
  delete(key) {
    const i = this.pairs.findIndex(([k]) => k === key);
    if (i >= 0) this.pairs.splice(i, 1);
    return this;
  }
  /** A continuation chunk carries nothing but `m` (SPEC §3.4). */
  onlyChunkKeys() {
    return this.pairs.every(([k]) => k === "m");
  }
  encode() {
    return this.pairs.map(([k, v]) => `${k}=${clean2(v)}`).join(":");
  }
};
function clean2(value) {
  return value.replace(/[^\x20-\x7e]|[:;=]/gu, "_");
}
var Assembler = class {
  partial = null;
  /** A chunked command was dropped as too large: its further chunks are dropped too, until its last. */
  dropping = false;
  /** A full reset ends the session, including a resource partway in. */
  reset() {
    this.partial = null;
    this.dropping = false;
  }
  /** Feeds the data of one OSC 7279 sequence. */
  feed(data) {
    const semi = data.indexOf(";");
    const ctlText = semi < 0 ? data : data.slice(0, semi);
    const b64 = semi < 0 ? "" : data.slice(semi + 1);
    let control;
    try {
      control = Control.parse(ctlText);
    } catch (e) {
      this.partial = null;
      return [{ kind: "invalid", reason: String(e.message) }];
    }
    const more = control.get("m") === "1";
    const dropping = this.dropping;
    this.dropping = false;
    if (dropping && !this.partial && control.get("m") !== void 0 && control.onlyChunkKeys()) {
      this.dropping = more;
      return [{ kind: "partial" }];
    }
    const partial = this.partial;
    this.partial = null;
    if (partial && control.onlyChunkKeys()) {
      partial.b64.push(b64);
      partial.size += b64.length;
      if (partial.size > encodedMax(MAX_COMMAND)) {
        this.dropping = more;
        return [{ kind: "invalid", reason: "command too large" }];
      }
      if (more) {
        this.partial = partial;
        return [{ kind: "partial" }];
      }
      return settle([], emit(partial.control, partial.b64.join("")));
    }
    const out = [];
    if (partial) out.push({ kind: "invalid", reason: "chunked command interrupted" });
    if (more) {
      this.partial = { control, b64: [b64], size: b64.length };
      out.push({ kind: "partial" });
      return out;
    }
    return settle(out, emit(control, b64));
  }
};
function settle(out, d) {
  if (d instanceof Promise) return d.then((x) => [...out, x]);
  out.push(d);
  return out;
}
function emit(control, b64) {
  const fail = (e) => ({ kind: "invalid", reason: `bad payload: ${e.message}` });
  let bytes;
  try {
    bytes = fromBase64(b64);
  } catch (e) {
    return fail(e);
  }
  const done = (payload) => {
    control.delete("m").delete("o");
    return { kind: "command", command: { control, payload } };
  };
  if (control.get("o") === "z") return inflate(bytes).then(done, fail);
  if (bytes.length > MAX_COMMAND) return { kind: "invalid", reason: "command too large" };
  return done(bytes);
}
function fromBase64(b64) {
  const s = b64.replace(/\s+/g, "");
  if (s === "") return new Uint8Array(0);
  const bin = atob(s);
  const out = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i++) out[i] = bin.charCodeAt(i);
  return out;
}
function toBase64(bytes) {
  let bin = "";
  for (let i = 0; i < bytes.length; i += 32768) {
    bin += String.fromCharCode(...bytes.subarray(i, i + 32768));
  }
  return btoa(bin);
}
async function inflate(bytes, max = MAX_COMMAND) {
  const ds = new DecompressionStream("deflate");
  const writer = ds.writable.getWriter();
  writer.write(bytes).catch(() => {
  });
  writer.close().catch(() => {
  });
  const reader = ds.readable.getReader();
  const parts = [];
  let size = 0;
  for (; ; ) {
    const { done, value } = await reader.read();
    if (done) break;
    size += value.length;
    if (size > max) {
      void reader.cancel().catch(() => {
      });
      throw new Error("command too large");
    }
    parts.push(value);
  }
  const out = new Uint8Array(size);
  let at = 0;
  for (const p of parts) {
    out.set(p, at);
    at += p.length;
  }
  return out;
}
var utf82 = new TextDecoder("utf-8", { fatal: false });
var utf8Encoder = new TextEncoder();
var strictUtf8 = new TextDecoder("utf-8", { fatal: true });
function strictText(bytes) {
  try {
    return strictUtf8.decode(bytes);
  } catch {
    return null;
  }
}
function encode(control, payload = new Uint8Array(0)) {
  const bytes = typeof payload === "string" ? utf8Encoder.encode(payload) : payload;
  const b64 = toBase64(bytes);
  const osc = (ctl, chunk) => `\x1B]${OSC};${ctl}${chunk === null ? "" : `;${chunk}`}\x1B\\`;
  if (b64.length <= CHUNK) return osc(control.encode(), b64 === "" ? null : b64);
  let out = "";
  for (let i = 0; i < b64.length; i += CHUNK) {
    const last = i + CHUNK >= b64.length;
    const ctl = i === 0 ? new Control([...control.pairs]).set("m", "1") : new Control([["m", last ? "0" : "1"]]);
    out += osc(ctl.encode(), b64.slice(i, i + CHUNK));
  }
  return out;
}
async function decodeAll(stream) {
  const a = new Assembler();
  const out = [];
  const re = /\x1b\]7279;([^\x07\x1b]*)(?:\x07|\x1b\\)/g;
  for (const m of stream.matchAll(re)) {
    for (const d of await a.feed(m[1])) if (d.kind === "command") out.push(d.command);
  }
  return out;
}

// src/addon.ts
var mac2 = typeof navigator !== "undefined" && /Mac|iPhone|iPad/.test(navigator.platform || navigator.userAgent);
function browserKeys(e) {
  if (e.key === "F5" || e.key === "F11" || e.key === "F12") return true;
  if (mac2 && e.metaKey) return true;
  const mod = e.ctrlKey || e.metaKey;
  const k = e.key.toLowerCase();
  if (mod && !e.altKey && k === "r") return true;
  if (mod && ["=", "+", "-", "0"].includes(e.key)) return true;
  if (e.ctrlKey && e.shiftKey && ["i", "j", "c"].includes(k)) return true;
  return false;
}
function fontsUrl(fonts) {
  if (!fonts) return "";
  try {
    const u = new URL(fonts, document.baseURI);
    if (!u.pathname.endsWith("/")) u.pathname += "/";
    u.search = u.hash = "";
    return u.href;
  } catch {
    return "";
  }
}
function copyKeys(e) {
  return (e.ctrlKey || e.metaKey) && !e.altKey && !e.shiftKey && e.key.toLowerCase() === "c";
}
var GESTURE_MS = 150;
var SPEAKS = ["0.8", "0.7"];
function answerIn(v) {
  const listed = (v ?? "").split(",");
  const shared = SPEAKS.find((s) => listed.includes(s));
  if (shared === void 0) throw new Failure("EVERSION", [...SPEAKS].reverse().join(","));
  return shared;
}
var LABEL_CHARS = 32;
var COLUMN_ITEMS = 10;
var Failure = class extends Error {
  code;
  constructor(code, detail) {
    super(detail);
    this.code = code;
  }
};
var FEATURES = ["count", "menu"];
var keysOf = (e) => ({ altKey: e.altKey, ctrlKey: e.ctrlKey, shiftKey: e.shiftKey, metaKey: e.metaKey });
var HottyAddon = class {
  term;
  opts;
  store;
  surfaces = /* @__PURE__ */ new Map();
  placements = /* @__PURE__ */ new Map();
  assembler = new Assembler();
  /** Replies' and events' bodies (SPEC §3.3), each sent as it is written. */
  writer = new Writer();
  disposables = [];
  layer = null;
  /** The terminal's element, whose presses drop the surfaces' selections,
   * and whose focus moves an `a=focus` that waits (`watch`). */
  pressed = null;
  dropSelections = () => {
    for (const s of this.surfaces.values()) s.clearSelection();
  };
  /** A press on the cells drops the surfaces' selections. A click (SPEC
   * §10.1: the primary button, without Alt) also ends an `a=focus` that
   * waits: the keyboard comes back to the terminal. */
  cellsPressed = (e) => {
    this.dropSelections();
    if (e.button === 0 && !e.altKey) for (const s of this.surfaces.values()) s.dropWait(true);
  };
  /** Focus comes into the terminal, and not by a click (Tab, or the page
   * focusing it): an `a=focus` that waited moves now. */
  focusIn = () => {
    for (const s of this.surfaces.values()) s.takeFocus();
  };
  /** Commands held while synchronized output is on (PROTOCOL §5). */
  held = [];
  /** How many times the alternate screen was entered: a placement held on
   * one that has gone since goes with it. */
  altScreens = 0;
  heldTimer = null;
  css = "";
  policy;
  /** The `fonts` option, absolute, ending in a slash; "" without it. */
  fonts;
  metrics = { cellW: 0, cellH: 0, key: "" };
  /** The capabilities the program last heard (SPEC §4), from its query or a
   * `caps` event; null while it has asked for none. */
  heard = null;
  /** The hotty version the session speaks (SPEC §4, HOST.md §5.1): the one
   *  the last query was answered in, until a full reset (RIS). Before a
   *  query it is the highest the addon speaks. Both versions have the
   *  secondary press (`Surface.secondary`). */
  version = SPEAKS[0];
  /** The font and the cell (CSS px) it heard them with: a new cell is owed
   * as `cellcss` when the font changed, or by more than a device pixel's
   * rounding (SPEC §5.3). */
  told = { cellW: 0, cellH: 0, font: "" };
  capsTimer = null;
  pixelWatch = null;
  /** A surface's key is being forwarded to the terminal (`key`), and the
   * terminal's own focus is held off meanwhile. A program in the page may
   * answer the key inside its dispatch, since xterm.js parses what follows
   * user input at once: a surface that gives the keyboard back then has
   * its focus carried out once the key is through. */
  forwarding = false;
  focusAfter = false;
  /** The wheel gesture under way: the surface its last wheel was over ("" for
   * the cells), where it goes, and when that wheel came. */
  gesture = null;
  /** How keys are read (SPEC §10.4): `mod` is `super` on a Mac, `ctrl` elsewhere. */
  keys;
  /** The edit commands the program takes on each screen (SPEC §10.6). */
  taken = { normal: /* @__PURE__ */ new Set(), alternate: /* @__PURE__ */ new Set() };
  /** Keys whose keydown was a command: their release sends nothing. */
  commandCodes = /* @__PURE__ */ new Set();
  /** A paste command waiting for the browser's paste, which brings the
   * clipboard's text: the key, and the fallback that reads it instead. */
  pasting = null;
  /** `encodeKey` runs a key through xterm.js: it is not typed. */
  encoding = false;
  /** The program's menu on each screen (SPEC §10.7). */
  menus = new Menus();
  /** The cells where a finger drags as the mouse, on each screen (SPEC
   * §9.6). */
  touchAreas = { normal: [], alternate: [] };
  /** The keys of the level shown (`a=menu:show=1`), null when none is, and
   * where it shows. */
  shownSeq = null;
  menuEl = null;
  /** The terminal's element, whose paste a `paste` command takes. */
  pasteOn = null;
  constructor(options = {}) {
    this.opts = { maxSurfaces: 64, resourceQuota: 64 << 20, ...options };
    const platform = { mod: mac2 ? "super" : "ctrl" };
    this.keys = { platform };
    this.policy = clean(options.network);
    this.fonts = fontsUrl(options.fonts);
    this.store = new Store(this.opts.resourceQuota);
    this.store.onChange = (names) => {
      for (const s of this.surfaces.values()) s.refresh(names);
    };
  }
  activate(term) {
    this.term = term;
    term.attachCustomKeyEventHandler((e) => {
      if (this.copyKey(e)) return false;
      if (!this.encoding) {
        if (e.type === "keyup" && this.commandCodes.delete(e.code)) return false;
        if (e.type === "keydown") {
          this.menuKey(e);
          if (this.command(e)) return false;
        }
      }
      return !this.browserKey(e);
    });
    this.disposables.push(
      term.onSelectionChange(() => {
        if (term.hasSelection()) this.dropSelections();
      })
    );
    const p = term.parser;
    this.disposables.push(
      p.registerOscHandler(OSC, (data) => this.onOsc(data)),
      // `CSI ? 2026 l` ends a synchronized batch: apply what was held, then
      // let xterm handle the mode as usual.
      p.registerCsiHandler({ prefix: "?", final: "l" }, (params) => {
        if (params.some((x) => x === 2026)) this.release();
        return false;
      }),
      // RIS: every surface goes (PROTOCOL §4).
      p.registerEscHandler({ final: "c" }, () => {
        this.reset();
        this.store.clear();
        this.assembler.reset();
        this.version = SPEAKS[0];
        this.taken.normal.clear();
        this.taken.alternate.clear();
        this.touchAreas.normal = [];
        this.touchAreas.alternate = [];
        this.menus.clear();
        this.shownSeq = null;
        this.drawMenu();
        return false;
      }),
      term.buffer.onBufferChange((b) => this.onBufferChange(b.type)),
      term.onRender(() => {
        this.bindTouch();
        this.bindPaste();
        this.checkMetrics();
      }),
      term.onResize(() => this.checkMetrics()),
      term.onScroll(() => this.reposition()),
      term.onWriteParsed(() => this.reposition())
    );
    this.watchPixels();
  }
  /** A zoom, or a display with another pixel density, changes
   * `--hotty-px` (SPEC §5.3, §8) even where nothing is drawn: the media
   * query for the current density stops matching when it changes. */
  watchPixels() {
    this.pixelWatch?.dispose();
    const query = window.matchMedia(`(resolution: ${window.devicePixelRatio}dppx)`);
    const changed = () => {
      this.checkMetrics();
      this.watchPixels();
    };
    query.addEventListener("change", changed, { once: true });
    this.pixelWatch = { dispose: () => query.removeEventListener("change", changed) };
  }
  dispose() {
    if (this.pasting) clearTimeout(this.pasting.timer);
    this.pasting = null;
    this.pasteOn?.removeEventListener("paste", this.onPaste, true);
    this.pasteOn = null;
    this.menuEl?.remove();
    this.menuEl = null;
    this.cellTouch?.dispose();
    this.cellTouch = null;
    this.reset();
    this.store.clear();
    this.assembler.reset();
    if (this.capsTimer) clearTimeout(this.capsTimer);
    this.capsTimer = null;
    this.pixelWatch?.dispose();
    this.pixelWatch = null;
    for (const d of this.disposables) d.dispose();
    this.disposables.length = 0;
    this.layer?.remove();
    this.layer = null;
    this.unwatch();
  }
  /**
   * An element of a surface as the program wrote it (tag, attributes, text,
   * child elements): the shape the shared conformance vectors check
   * (conformance/README.md). `null` if there is no such surface or element.
   */
  inspect(surface, id) {
    const s = this.surfaces.get(surface);
    const el = s?.doc.getElementById(id);
    if (!s || !el) return null;
    return {
      tag: el.localName,
      attrs: Object.fromEntries(s.resolver.attributes(el)),
      text: el.textContent ?? "",
      children: Array.from(el.children, (c) => [c.localName, c.getAttribute("id"), c.textContent ?? ""])
    };
  }
  // --- Wire -----------------------------------------------------------------
  onOsc(data) {
    const decoded = this.assembler.feed(data);
    if (decoded instanceof Promise) return decoded.then((d) => this.run(d)).then(() => true);
    const r = this.run(decoded);
    return r instanceof Promise ? r.then(() => true) : true;
  }
  run(decoded) {
    let chain;
    for (const d of decoded) {
      if (d.kind === "invalid") this.opts.onInvalid?.(d.reason);
      if (d.kind !== "command") continue;
      const cmd = d.command;
      chain = sequence(chain, () => this.submit(cmd));
    }
    return chain;
  }
  submit(cmd) {
    if (this.term.modes.synchronizedOutputMode && !["q", "hit"].includes(cmd.control.get("a") ?? "")) {
      const at = cmd.control.get("a") === "place" ? this.hold(cmd.control) : void 0;
      this.held.push({ cmd, at });
      this.heldTimer ??= setTimeout(() => this.release(), 1e3);
      return;
    }
    return this.apply([{ cmd }]);
  }
  /**
   * A placement held in a batch is drawn when the batch ends (SPEC §6.3),
   * but it goes where the cursor is as it comes, and unless `C=1` it moves
   * the cursor below it then (§5.2), so what the batch writes next is below
   * it. How far is its rows: for `r=auto` only the document can tell, so
   * what the batch held before it is applied first (and may be drawn
   * before the batch ends). A place that will fail moves nothing.
   */
  hold(c) {
    const at = this.cursorSpot();
    if (c.get("C") === "1") return at;
    const r = c.get("r");
    if (r === void 0 || r === "auto") this.release();
    try {
      const name = this.surfaceName(c);
      if (!this.surfaces.has(name) && !this.held.some((h) => h.cmd.control.get("a") === "doc" && h.cmd.control.get("s") === name)) return at;
      const { cellW, cellH } = this.cell();
      const { rows, win } = this.measure(c, (cols) => this.existing(c).contentRows(cols, cellW, cellH));
      at.rows = rows;
      this.moveCursorBelow(win.h);
      const p = this.placements.get(name);
      if (p) this.position(p);
    } catch {
    }
    return at;
  }
  release() {
    if (this.heldTimer) clearTimeout(this.heldTimer);
    this.heldTimer = null;
    const batch = this.held;
    this.held = [];
    if (batch.length) void this.apply(batch);
  }
  apply(batch) {
    const t0 = performance.now();
    let chain;
    for (const { cmd, at } of batch) chain = sequence(chain, () => this.handle(cmd, at));
    const done = () => this.opts.onFrame?.({ commands: batch.length, applyMs: performance.now() - t0 });
    if (chain) return chain.then(done);
    done();
  }
  handle(cmd, at) {
    const action = cmd.control.get("a") ?? "";
    const quiet = Number(cmd.control.get("q") ?? (action === "q" || action === "hit" || cmd.control.get("n") !== void 0 ? "0" : "1"));
    const ok = (r) => {
      if (quiet === 0) this.reply("ok", cmd, r ?? {});
    };
    const fail = (e) => {
      const code = e instanceof Failure || e instanceof DeltaError || e instanceof FieldError ? e.code : "EINVAL";
      const detail = e instanceof Error ? e.message : String(e);
      const extra = e instanceof DeltaError && e.i !== void 0 ? [["i", String(e.i)]] : [];
      if (quiet < 2) this.reply("err", cmd, { extra, body: { code, detail } });
    };
    try {
      const r = this.dispatch(action, cmd, at);
      if (r instanceof Promise) return r.then(ok, fail);
      ok(r);
    } catch (e) {
      fail(e);
    }
  }
  reply(kind, cmd, r) {
    const c = new Control().set("a", kind);
    for (const k of ["n", "s"]) {
      const v = cmd.control.get(k);
      if (v !== void 0) c.set(k, v);
    }
    c.set("re", cmd.control.get("a") ?? "");
    for (const [k, v] of r.extra ?? []) c.set(k, v);
    this.send(encode(c, r.body ? this.writer.body(r.body) : ""));
  }
  send(message) {
    this.term.input(message, false);
  }
  // --- Commands (PROTOCOL §3-§6) -------------------------------------------
  dispatch(action, cmd, at) {
    const c = cmd.control;
    switch (action) {
      case "q":
        return { body: this.hear(this.capabilities(answerIn(c.get("v")))) };
      case "hit": {
        const s = this.existing(c);
        if (s.detached) throw new Failure("EDETACHED", `surface ${s.name} is detached`);
        const cell = (key) => {
          const value = c.get(key) ?? "";
          const n = Number(value);
          if (!/^-?\d+$/.test(value) || !Number.isSafeInteger(n)) throw new Failure("EINVAL", `${key} is a whole decimal cell`);
          return n;
        };
        return { body: s.hit(cell("c"), cell("r")) };
      }
      case "doc": {
        const name = this.surfaceName(c);
        const doc = strictText(cmd.payload);
        if (doc === null) throw new Failure("EINVAL", "payload is not UTF-8");
        let s = this.surfaces.get(name);
        if (!s) {
          if (this.surfaces.size >= this.opts.maxSurfaces) throw new Failure("EQUOTA", `at most ${this.opts.maxSurfaces} surfaces`);
          s = new Surface(name, this.host(), this.hostCss(), this.scheme());
          this.surfaces.set(name, s);
        }
        if (c.get("d") !== "1") s.cancelDrag();
        const scroll = c.get("scroll");
        s.setDocument(doc, c.get("d") === "1", scroll === "1" || scroll === "2" || scroll === "3" ? Number(scroll) : 0);
        return;
      }
      case "place":
        try {
          return this.place(this.existing(c), c, at);
        } catch (e) {
          at?.marker?.dispose();
          throw e;
        }
      case "hide": {
        const s = this.existing(c);
        s.cancelDrag();
        if (s.hasKeyboard()) s.blur();
        this.unplace(s.name);
        return;
      }
      case "delta": {
        const s = this.existing(c);
        const failed = s.delta(readDelta(cmd.payload));
        if (failed) throw failed;
        return;
      }
      case "res": {
        const id = c.get("id");
        if (!id) throw new Failure("EINVAL", "res needs id=<name>");
        try {
          this.store.put(id, c.get("type") ?? "application/octet-stream", cmd.payload);
        } catch (e) {
          throw new Failure("EQUOTA", e.message);
        }
        return;
      }
      case "del": {
        const id = c.get("id");
        if (id !== void 0) {
          this.store.remove(id);
          return;
        }
        const name = c.get("s");
        if (name === void 0) {
          this.reset();
          return;
        }
        if (!this.surfaces.has(name)) throw new Failure("ENOENT", `no surface ${name}`);
        this.remove(name);
        return;
      }
      case "detach":
        this.existing(c).detach();
        return;
      case "focus": {
        const s = this.existing(c);
        if (s.detached) throw new Failure("EDETACHED", `surface ${s.name} is detached`);
        const now = this.focusInside();
        try {
          s.focus(c.get("t"), now);
        } catch (e) {
          throw new Failure("ENOTARGET", e.message);
        }
        for (const o of this.surfaces.values()) if (o !== s) o.dropWait(false);
        if (!now) this.watch();
        return;
      }
      case "blur":
        this.existing(c).blur();
        return;
      case "field":
        this.existing(c).field(c.get("t"), cmd.payload);
        return;
      case "edit": {
        const take = (c.get("take") ?? "").split(",").filter((n) => EDIT_ACTIONS.includes(n));
        this.taken[this.term.buffer.active.type] = new Set(take);
        return;
      }
      case "touch":
        this.touchAreas[this.term.buffer.active.type] = readTouchAreas(cmd.payload);
        return;
      case "menu": {
        const screen = this.term.buffer.active.type;
        const show = c.get("show");
        if (show !== void 0 && show !== "0" && show !== "1") throw new Failure("EINVAL", "show is 0 or 1");
        if (show === "1") {
          this.shownSeq = readSeq(cmd.payload);
        } else if (show === "0") {
          this.shownSeq = null;
        } else {
          this.menus.set(screen, cmd.payload.length ? readMenu(cmd.payload) : null);
        }
        this.drawMenu();
        return;
      }
      case "":
        throw new Failure("EINVAL", "missing a=<action>");
      default:
        throw new Failure("EINVAL", `unknown action a=${action}`);
    }
  }
  /** The capabilities (SPEC §4) in version `v`, each field as its type. */
  capabilities(v) {
    const { cellW, cellH } = this.cell();
    const scale = window.devicePixelRatio || 1;
    const limit = (n) => Number.isFinite(n) ? Math.min(Math.floor(n), Number.MAX_SAFE_INTEGER) : void 0;
    return {
      v,
      cellcss: { w: float(cellW), h: float(cellH) },
      scale: float(scale),
      scheme: this.scheme(),
      mod: this.keys.platform.mod,
      features: FEATURES,
      limits: { resources: limit(this.store.quota), surfaces: limit(this.opts.maxSurfaces), message: MAX_COMMAND },
      net: this.policy,
      edit: editOf(this.keys.platform),
      host: HOST,
      version: VERSION
    };
  }
  surfaceName(c) {
    const s = c.get("s");
    if (s === void 0) throw new Failure("EINVAL", "missing s=<surface>");
    if (!/^[A-Za-z0-9_-]{1,64}$/.test(s)) throw new Failure("EINVAL", `bad surface name ${JSON.stringify(s)}`);
    return s;
  }
  existing(c) {
    const name = this.surfaceName(c);
    const s = this.surfaces.get(name);
    if (!s) throw new Failure("ENOENT", `no surface ${name}`);
    return s;
  }
  /** `at`: where it was held in a batch (`hold`), not where the cursor is
   * now; the cursor moved then. */
  place(s, c, at) {
    const { cellW, cellH } = this.cell();
    const { cols, rows, auto, win, z } = this.measure(c, (cols2) => at?.rows ?? s.contentRows(cols2, cellW, cellH));
    s.autoRows = auto;
    s.setSize(cols, rows, cellW, cellH, win);
    s.setZ(z);
    this.unplace(s.name);
    const flag = (k) => c.get(k) === "1" ? true : c.get(k) === "0" ? false : void 0;
    s.presses = flag("p") ?? s.presses;
    s.fits = flag("f") ?? s.fits;
    s.setFit(s.fits ? rows : null);
    s.beginPlacement();
    const spot = at ?? this.cursorSpot();
    const p = { surface: s, kind: spot.kind, col: spot.col, row: spot.row, marker: spot.marker, disposables: [] };
    if (p.marker) p.disposables.push(p.marker.onDispose(() => this.unplace(s.name)));
    this.placements.set(s.name, p);
    const buf = this.term.buffer.active;
    if (p.marker?.isDisposed) this.unplace(s.name);
    else if (spot.alt !== void 0 && (buf.type !== "alternate" || spot.alt !== this.altScreens)) this.remove(s.name);
    else this.position(p);
    if (!at && c.get("C") !== "1") this.moveCursorBelow(win.h);
    return { extra: [["c", String(cols)], ["r", String(rows)]] };
  }
  /** A placement's size, window and z (SPEC §5.2), as its command gives
   * them; `contentRows` lays the document out for `r=auto`. */
  measure(c, contentRows) {
    const cols = Number(c.get("c"));
    if (c.get("c") === void 0 || !Number.isInteger(cols)) throw new Failure("EINVAL", "place needs c=<cols>");
    if (cols < 1 || cols > 1e3) throw new Failure("EINVAL", "c out of range");
    const r = c.get("r");
    const auto = r === void 0 || r === "auto";
    const rows = auto ? contentRows(cols) : Number(r);
    if (!Number.isInteger(rows) || rows < 1 || rows > 1e3) throw new Failure("EINVAL", "r out of range");
    const cell = (k, d) => {
      const v = c.get(k);
      if (v === void 0) return d;
      const n = Number(v);
      if (!Number.isInteger(n) || n < 0) throw new Failure("EINVAL", `bad ${k}`);
      return n;
    };
    const x = cell("x", 0);
    const y = cell("y", 0);
    const win = { x, y, w: cell("w", cols - x), h: cell("h", rows - y) };
    if (win.w < 1 || win.h < 1 || x + win.w > cols || y + win.h > rows) {
      throw new Failure("EINVAL", "the window is not inside the surface");
    }
    const z = c.get("z") === void 0 ? 0 : Number(c.get("z"));
    if (!Number.isInteger(z) || z < -1e3 || z > 1e3) throw new Failure("EINVAL", "z is an integer from -1000 to 1000");
    return { cols, rows, auto, win, z };
  }
  /** The cursor's cell now, where a placement goes. */
  cursorSpot() {
    const buf = this.term.buffer.active;
    const at = { kind: buf.type === "normal" ? "normal" : "alt", col: buf.cursorX, row: buf.cursorY };
    if (buf.type === "alternate") at.alt = this.altScreens;
    if (at.kind === "normal") {
      const marker = this.term.registerMarker(0);
      if (marker) at.marker = marker;
      else at.kind = "alt";
    }
    return at;
  }
  /** As the native host: index `rows` times, then carriage return. */
  moveCursorBelow(rows) {
    const core = this.term._core;
    const ih = core?._inputHandler;
    if (ih && core?.buffer) {
      for (let i = 0; i < rows; i++) ih.lineFeed();
      core.buffer.x = 0;
    }
  }
  /** Puts every placed surface where its cells are now. */
  reposition() {
    for (const p of this.placements.values()) this.position(p);
  }
  position(p) {
    const { cellW, cellH } = this.cell();
    const s = p.surface;
    let row = p.row;
    if (p.kind === "normal") {
      const buf = this.term.buffer.active;
      if (buf.type !== "normal" || !p.marker || p.marker.line < 0) {
        s.hide();
        return;
      }
      row = p.marker.line - buf.viewportY;
    }
    if (row + s.win.h <= 0 || row >= this.term.rows) {
      s.hide();
      return;
    }
    s.setSize(s.cols, s.rows, cellW, cellH, s.win);
    s.show(Math.round(p.col * cellW), row * cellH);
  }
  unplace(name) {
    const p = this.placements.get(name);
    if (!p) return;
    this.placements.delete(name);
    for (const d of p.disposables) d.dispose();
    p.marker?.dispose();
    p.surface.setFit(null);
    p.surface.endPlacement();
    p.surface.park();
  }
  remove(name) {
    this.unplace(name);
    this.surfaces.get(name)?.destroy();
    this.surfaces.delete(name);
  }
  reset() {
    for (const name of Array.from(this.surfaces.keys())) this.remove(name);
    for (const h of this.held) h.at?.marker?.dispose();
    this.held = [];
  }
  onBufferChange(type) {
    if (type === "alternate") this.altScreens++;
    for (const p of Array.from(this.placements.values())) {
      if (p.kind === "alt" && type === "normal") this.remove(p.surface.name);
    }
    if (type === "normal") {
      this.taken.alternate.clear();
      this.touchAreas.alternate = [];
    }
    this.menus.change(type);
    this.shownSeq = null;
    this.drawMenu();
    this.reposition();
  }
  // --- Geometry and look ---------------------------------------------------
  /** The run of clicks the last click was in (SurfaceHost.clickCount). */
  clickRun = null;
  host() {
    return {
      layer: this.ensureLayer(),
      store: this.store,
      secondary: () => this.version === "0.7" || this.version === "0.8",
      event: (surface, kind, target, detail) => {
        const c = new Control().set("a", "ev").set("s", surface).set("e", kind);
        if (target !== "") c.set("t", target);
        const fields = detail !== void 0 && Object.values(detail).some((v) => v != null);
        this.send(encode(c, fields ? this.writer.body(detail) : ""));
      },
      browserKey: (e) => this.browserKey(e),
      copyKey: (e) => this.copyKey(e),
      writeClipboard: (text) => void navigator.clipboard?.writeText(text).catch(() => {
      }),
      readClipboard: () => navigator.clipboard?.readText().catch(() => null) ?? Promise.resolve(null),
      pressed: (keep) => {
        if (keep !== null) this.term.clearSelection();
        for (const [name, s] of this.surfaces) if (name !== keep) s.clearSelection();
      },
      key: (e) => this.terminalKey(e),
      encodeKey: (e) => {
        const cs = this.term._core?.coreService;
        if (typeof cs?.triggerDataEvent !== "function") return null;
        let data = "";
        const own = Object.prototype.hasOwnProperty.call(cs, "triggerDataEvent");
        const prev = cs.triggerDataEvent;
        cs.triggerDataEvent = (d) => void (data += d);
        this.encoding = true;
        try {
          this.terminalKey(e);
        } finally {
          this.encoding = false;
          if (own) cs.triggerDataEvent = prev;
          else delete cs.triggerDataEvent;
        }
        return data;
      },
      keys: this.keys,
      command: (e) => this.command(e),
      typed: (e) => this.menuKey(e),
      sendKey: (data) => {
        const cs = this.term._core.coreService;
        this.throughTerminal(() => cs.triggerDataEvent(data, true));
      },
      focusTerminal: () => {
        if (this.forwarding) this.focusAfter = true;
        else this.term.focus();
      },
      clicked: (name) => {
        for (const [n, s] of this.surfaces) if (n !== name) s.dropWait(true);
      },
      policy: this.policy,
      fonts: this.fonts,
      pageScrolls: this.pageScrolls,
      scrollPage: (dx, dy) => this.scrollPage(dx, dy),
      wheelGesture: (surface, decide) => {
        const now = performance.now();
        const g = this.gesture;
        const same2 = g !== null && now - g.at < GESTURE_MS && (g.surface === surface || g.route === "terminal");
        const route = same2 ? g.route : decide();
        this.gesture = { surface, route, at: now };
        return route;
      },
      // A hyperlink (SPEC §9) goes where xterm.js sends an OSC 8 one: the
      // terminal's linkHandler, or its confirm-then-open default, and only
      // for http and https unless the handler allows other schemes.
      // The platform counts clicks (MouseEvent.detail), by the user's own
      // settings; a run is kept to one element of one surface (SPEC §9).
      clickCount: (surface, el, detail) => {
        if (detail < 1) {
          this.clickRun = null;
          return 1;
        }
        const r = this.clickRun;
        if (detail === 1 || !r || r.surface !== surface || r.el !== el) this.clickRun = { surface, el, base: detail };
        return detail - this.clickRun.base + 1;
      },
      hyperlink: (kind, e, url, box) => {
        const handler = this.term.options.linkHandler;
        if (!handler?.allowNonHttpProtocols) {
          try {
            if (!["http:", "https:"].includes(new URL(url).protocol)) return;
          } catch {
            return;
          }
        }
        const range2 = this.cellRange(box);
        if (kind === "activate") {
          if (handler) handler.activate(e, url, range2);
          else openAfterConfirm(url);
        } else if (kind === "hover") handler?.hover?.(e, url, range2);
        else handler?.leave?.(e, url, range2);
      },
      // Replayed on the terminal's screen, where xterm handles it as its own:
      // scrollback, or mouse reports and arrow keys on the alternate screen.
      wheel: (e, x, y) => this.forwardWheel(e, x, y),
      cells: (kind, e, x, y, button) => this.pressCells(kind, e, x, y, button)
    };
  }
  /**
   * A key through xterm.js's own keyboard handling, so it is encoded the
   * way the program asked: DECCKM, modifyOtherKeys, the kitty keyboard
   * protocol. A printable key xterm.js leaves to keypress (legacy input)
   * gets one.
   */
  terminalKey(e) {
    const ta = this.term.textarea;
    if (!ta) return;
    const init = {
      key: e.key,
      code: e.code,
      location: e.location,
      repeat: e.repeat,
      ctrlKey: e.ctrlKey,
      altKey: e.altKey,
      shiftKey: e.shiftKey,
      metaKey: e.metaKey,
      keyCode: e.keyCode,
      which: e.which,
      bubbles: true,
      cancelable: true
    };
    this.throughTerminal(() => {
      const ev = new KeyboardEvent(e.type, init);
      ta.dispatchEvent(ev);
      if (e.type === "keydown" && !ev.defaultPrevented && [...e.key].length === 1) {
        const c = e.key.codePointAt(0);
        ta.dispatchEvent(new KeyboardEvent("keypress", { ...init, charCode: c, keyCode: c, which: c }));
      }
    });
  }
  /** Something a surface sends through the terminal (a key, or what one
   * stands for). xterm.js focuses the terminal on every keyup; the surface
   * keeps the keyboard, so its focus does nothing meanwhile, and a surface
   * the program blurs meanwhile gives the keyboard back once it is
   * through. */
  throughTerminal(fn) {
    const ta = this.term.textarea;
    if (ta) Object.defineProperty(ta, "focus", { value: () => {
    }, configurable: true });
    this.forwarding = true;
    try {
      fn();
    } finally {
      if (ta) delete ta.focus;
      this.forwarding = false;
      if (this.focusAfter) {
        this.focusAfter = false;
        this.term.focus();
      }
    }
  }
  cellTouch = null;
  pageScrollBound = false;
  /** Whether the page scrolls, not the terminal (the `scroll` option). */
  get pageScrolls() {
    return this.opts.scroll === "page";
  }
  /**
   * Wheels and touches on the cells left to the browser, which scrolls the
   * page (`scroll: "page"`): they stop at the terminal's element, before
   * xterm.js, which would take them for its scrollback, or turn a drag into
   * arrow keys, and cancel the browser's scroll. Nothing here cancels it.
   */
  bindPageScroll(el) {
    if (this.pageScrollBound) return;
    this.pageScrollBound = true;
    const stop = (e) => e.stopPropagation();
    for (const type of ["wheel", "touchstart", "touchmove", "touchend", "touchcancel"]) {
      el.addEventListener(type, stop, { capture: true, passive: true });
      this.disposables.push({ dispose: () => el.removeEventListener(type, stop, { capture: true }) });
    }
  }
  /** Scrolls what scrolls the terminal's page: its nearest ancestor that
   * scrolls, or the document. */
  scrollPage(dx, dy) {
    let el = this.term.element?.parentElement ?? null;
    for (; el; el = el.parentElement) {
      const o = getComputedStyle(el).overflowY;
      if (/auto|scroll/.test(o) && el.scrollHeight > el.clientHeight) break;
    }
    (el ?? document.scrollingElement ?? document.documentElement).scrollBy(dx, dy);
  }
  cellWheelBound = false;
  /** A wheel over the cells begins or goes on with a gesture of the
   * terminal's: a document that scrolls under the pointer as the terminal
   * scrolls takes none of it (`wheelGesture`). */
  bindCellWheel(el) {
    if (this.cellWheelBound) return;
    this.cellWheelBound = true;
    const wheel = (e) => {
      if (e.isTrusted) this.gesture = { surface: "", route: "terminal", at: performance.now() };
    };
    el.addEventListener("wheel", wheel, { capture: true, passive: true });
    this.disposables.push({ dispose: () => el.removeEventListener("wheel", wheel, { capture: true }) });
  }
  /** Touch on the cells (the `touch` option), once the terminal is open. */
  bindTouch() {
    const el = this.term.element;
    if (el && this.pageScrolls) return this.bindPageScroll(el);
    if (el) this.bindCellWheel(el);
    if (this.cellTouch || !el || this.opts.touch === false) return;
    const screen = () => el.querySelector(".xterm-screen");
    this.cellTouch = new Touch(
      el,
      el.ownerDocument.defaultView,
      {
        scroll: (dx, dy, x, y) => this.forwardWheel(new WheelEvent("wheel", { deltaX: dx, deltaY: dy, deltaMode: 0 }), x, y),
        // A tap is the press and release a click would make: xterm.js
        // reports them to the program as it reports a mouse.
        tap: (x, y) => {
          const at = { clientX: x, clientY: y, button: 0, bubbles: true, cancelable: true, view: el.ownerDocument.defaultView };
          screen()?.dispatchEvent(new MouseEvent("mousedown", { ...at, buttons: 1 }));
          el.ownerDocument.dispatchEvent(new MouseEvent("mouseup", { ...at, buttons: 0 }));
        },
        toPage: (x, y) => [x, y],
        claim: (dx, dy, e, begin) => this.claimCells(dx, dy, e, begin),
        dragMove: (x, y, e) => this.fingerMouse("move", x, y, e),
        dragEnd: (x, y, e) => this.fingerMouse("up", x, y, e),
        dragCancel: () => {
          const f = this.finger;
          if (f) this.fingerMouse("up", f.x, f.y, f.keys);
        }
      },
      true
    );
  }
  /** Where the finger of a drag that is the program's mouse (SPEC §9.6)
   * last was, in the page, with the keys held then. */
  finger = null;
  /**
   * A touch on the cells that has just gone past the slop, by dx, dy so far
   * (SPEC §9.6). One that began in an area the program named, without Alt
   * and not held as a long press, whose first move goes a way the area
   * drags, is the mouse's primary button: pressed on the cell where it
   * began, then moved to the finger's cell if that is another. Every other
   * touch scrolls, as on any cell.
   */
  claimCells(dx, dy, e, begin) {
    if (begin.altKey || e.timeStamp - begin.time >= LONG_PRESS_MS) return true;
    const area = this.areaUnder(begin.x, begin.y);
    if (!area || !areaDrags(area, dx, dy)) return true;
    const [x, y] = [begin.x - dx, begin.y - dy];
    this.fingerMouse("down", begin.x, begin.y, { ...keysOf(e), altKey: false });
    const [c0, r0] = this.screenCell(begin.x, begin.y, true);
    const [c, r] = this.screenCell(x, y, true);
    if (c !== c0 || r !== r0) this.fingerMouse("move", x, y, e);
    return "drag";
  }
  /**
   * The area the program named (SPEC §9.6) under a point in the page, where
   * one applies: while the program has xterm.js report the mouse's buttons
   * (without Alt: `mouseEventsRequireAlt` leaves a finger nothing), and the
   * terminal shows the screen, not its scrollback.
   */
  areaUnder(x, y) {
    const t = this.term;
    const buf = t.buffer.active;
    if (t.modes.mouseTrackingMode === "none" || t.options.mouseEventsRequireAlt || buf.viewportY !== buf.baseY) return null;
    const [col, row] = this.screenCell(x, y, false);
    if (col < 0 || row < 0 || col >= t.cols || row >= t.rows) return null;
    return areaAt(this.touchAreas[buf.type], col, row);
  }
  /** The cell of the screen at a point in the page, from 0 at its top
   * left; `clamp`ed onto the screen as xterm.js reports a mouse past its
   * edge. */
  screenCell(x, y, clamp) {
    const screen = this.term.element?.querySelector(".xterm-screen")?.getBoundingClientRect();
    const { cellW, cellH } = this.cell();
    const col = Math.floor((x - (screen?.left ?? 0)) / cellW);
    const row = Math.floor((y - (screen?.top ?? 0)) / cellH);
    if (!clamp) return [col, row];
    return [Math.min(Math.max(col, 0), this.term.cols - 1), Math.min(Math.max(row, 0), this.term.rows - 1)];
  }
  /**
   * The finger of a drag in an area (SPEC §9.6) as the mouse's primary
   * button, pressed, moved or released at a point in the page: replayed on
   * the cells (`pressCells`), so that xterm.js reports it as a mouse's, in
   * the program's mode and encoding, with the keys held. Never with Shift,
   * which on a press makes xterm.js select instead of reporting (but on a
   * Mac), and nothing is selected.
   */
  fingerMouse(kind, x, y, keys) {
    const held2 = { altKey: keys.altKey, ctrlKey: keys.ctrlKey, metaKey: keys.metaKey, shiftKey: false };
    this.finger = kind === "up" ? null : { x, y, keys: held2 };
    this.pressCells(kind, { ...held2, screenX: 0, screenY: 0, buttons: kind === "up" ? 0 : 1, detail: kind === "move" ? 0 : 1 }, x, y);
  }
  /** A wheel, at a point in the page, replayed on the terminal's screen,
   * where xterm.js handles it as its own: scrollback, or mouse reports and
   * arrow keys on the alternate screen. */
  forwardWheel(e, x, y) {
    const ev = new WheelEvent("wheel", {
      deltaX: e.deltaX,
      deltaY: e.deltaY,
      deltaMode: e.deltaMode,
      clientX: x,
      clientY: y,
      shiftKey: e.shiftKey,
      altKey: e.altKey,
      metaKey: e.metaKey,
      bubbles: true,
      cancelable: true
    });
    legacyWheelDelta(ev, e);
    this.term.element?.querySelector(".xterm-screen")?.dispatchEvent(ev);
  }
  /**
   * A press with Alt held on a surface, a move of its gesture, or its
   * release (SPEC §9.2), or a finger's as the mouse (§9.6, `fingerMouse`),
   * replayed on the cells beneath, where xterm.js handles them as its own
   * mouse: a report to the program, with Alt, or its selection
   * (rectangular, with Alt). The press goes to the screen, where xterm.js
   * listens for presses; the moves and the release to its
   * document, where it listens while a button is held. The keyboard goes
   * back to the terminal first, as on a click on the cells (§10.1): a
   * surface that had it sends `blur`.
   */
  pressCells(kind, e, x, y, button = 0) {
    const el = this.term.element;
    if (!el) return;
    const init = {
      clientX: x,
      clientY: y,
      screenX: e.screenX,
      screenY: e.screenY,
      button,
      buttons: e.buttons,
      // xterm.js selects on a press whose detail is the click count.
      detail: kind === "down" ? e.detail || 1 : e.detail,
      altKey: e.altKey,
      ctrlKey: e.ctrlKey,
      shiftKey: e.shiftKey,
      metaKey: e.metaKey,
      bubbles: true,
      cancelable: true,
      view: el.ownerDocument.defaultView
    };
    if (kind === "down") {
      for (const s of this.surfaces.values()) if (s.hasKeyboard()) s.blur();
      el.querySelector(".xterm-screen")?.dispatchEvent(new MouseEvent("mousedown", init));
      return;
    }
    el.ownerDocument.dispatchEvent(new MouseEvent(kind === "move" ? "mousemove" : "mouseup", init));
  }
  ensureLayer() {
    if (this.layer?.isConnected) return this.layer;
    const screen = this.term.element?.querySelector(".xterm-screen");
    if (!screen) throw new Failure("EINVAL", "terminal is not open");
    const layer = document.createElement("div");
    layer.className = "hotty-layer";
    Object.assign(layer.style, { position: "absolute", inset: "0", overflow: "hidden", zIndex: "6", pointerEvents: "none" });
    screen.append(layer);
    this.layer = layer;
    this.watch();
    return layer;
  }
  /** Binds the terminal's element. A press on the cells drops a surface's
   * selection: there is one (SPEC §11). A press in a surface stays in its
   * frame. Focus coming into the element moves an `a=focus` that waits. */
  watch() {
    const el = this.term.element;
    if (!el || this.pressed === el) return;
    this.unwatch();
    el.addEventListener("mousedown", this.cellsPressed, true);
    el.addEventListener("focusin", this.focusIn, true);
    this.pressed = el;
  }
  unwatch() {
    this.pressed?.removeEventListener("mousedown", this.cellsPressed, true);
    this.pressed?.removeEventListener("focusin", this.focusIn, true);
    this.pressed = null;
  }
  /** Whether the page's focus is in the terminal, or nowhere: where
   * `a=focus` may move it. */
  focusInside() {
    const el = this.term.element;
    if (!el) return true;
    const doc = el.ownerDocument;
    let a = doc.activeElement;
    while (a?.shadowRoot?.activeElement) a = a.shadowRoot.activeElement;
    return !a || a === doc.body || a === doc.documentElement || el.contains(a);
  }
  // --- Commands in the cells and the menu (SPEC §10.6) -----------------
  /** The paste a `paste` command's key brings (SPEC §10.6), heard before
   * xterm.js's own on its textarea: its text goes with the command, and
   * xterm.js pastes nothing. */
  bindPaste() {
    const el = this.term.element;
    if (!el || this.pasteOn === el) return;
    this.pasteOn?.removeEventListener("paste", this.onPaste, true);
    el.addEventListener("paste", this.onPaste, true);
    this.pasteOn = el;
  }
  onPaste = (e) => {
    const p = this.pasting;
    if (!p) return;
    this.pasting = null;
    clearTimeout(p.timer);
    e.preventDefault();
    e.stopImmediatePropagation();
    this.sendCommand("paste", p.key, e.clipboardData?.getData("text/plain") ?? "");
  };
  /**
   * A key of an edit command the program on the active screen takes (SPEC
   * §10.6), named as pressed: the command goes to the program, in place of
   * the key's bytes. A `paste` brings the clipboard's text, from the
   * browser's paste where the key makes one, or else read.
   */
  command(e) {
    const taken = this.taken[this.term.buffer.active.type];
    if (taken.size === 0) return false;
    const names = pressNames(e);
    let name = null;
    for (const key of names) {
      name = editAction(this.keys.platform, key);
      if (name !== null) break;
    }
    const k = names[names.length - 1] ?? "";
    if (name === null || !taken.has(name)) return false;
    this.commandCodes.add(e.code);
    if (name !== "paste") {
      e.preventDefault();
      this.sendCommand(name, k);
      return true;
    }
    if (this.pasting) clearTimeout(this.pasting.timer);
    const read3 = () => {
      this.pasting = null;
      void (navigator.clipboard?.readText().catch(() => "") ?? Promise.resolve("")).then((text) => this.sendCommand("paste", k, text));
    };
    if (!e.isTrusted || this.forwarding) {
      e.preventDefault();
      read3();
    } else this.pasting = { key: k, timer: setTimeout(read3, 0) };
    return true;
  }
  /** The event `command` (SPEC §10.6): of no surface and no element, with
   * the key that made it where one did. A paste's text is left out where
   * the clipboard holds none. */
  sendCommand(name, key, text) {
    const c = new Control().set("a", "ev").set("e", "command");
    this.send(encode(c, this.writer.body({ name, key, text: text || void 0 })));
  }
  // --- The program's menu (SPEC §10.7) --------------------------------------
  /** A key's name as this host reads it (`mod` read), or null. */
  read = (k) => readKey(this.keys.platform, k);
  /** A key as the menu shows it (myelin key names §6): read, with arrows
   * and Tab as symbols. */
  show = (k) => displayKey(this.read(k) ?? k);
  /** The program's menu on the active screen. */
  menu() {
    return this.menus.get(this.term.buffer.active.type);
  }
  /** The level shown: the keyed items of the level `shownSeq` reaches. */
  level() {
    const m = this.menu();
    return m && this.shownSeq ? shownLevel(m, this.shownSeq, this.read) : null;
  }
  /**
   * The keyed items of the level shown, as the program sent them, or null
   * when none shows: what the conformance vectors' `menu: shown` checks.
   */
  shownMenu() {
    const l = this.level();
    return l ? l.items.map((i) => ({ key: i.key, label: i.label, group: isGroup(i), enabled: i.enabled })) : null;
  }
  /**
   * The items of the active screen's menu that have a name (SPEC §10.7), in
   * its order, for a page that lists them in a command palette of its own:
   * each under its groups' labels, with the keys that reach it, `mod` read
   * as the platform does, in myelin key names (a page shows them with
   * `displayKey`, §6). The addon draws no palette; `pick` runs an item.
   */
  menuItems() {
    const m = this.menu();
    return m ? entries(m, (k) => this.read(k) ?? k) : [];
  }
  /**
   * Picks the item the active screen's menu names `name` (SPEC §10.7), for
   * a page's own command palette: `command` goes to the program with just
   * the name. A `copy` while the terminal has text selected copies it
   * instead, and a `paste` brings the clipboard's text, as the terminal's
   * Edit menu does. Returns whether anything was picked: not a disabled
   * item, a name no item has, or no menu.
   */
  pick(name) {
    const m = this.menu();
    const item = m && find(m, name);
    if (!item?.enabled) return false;
    if (name === "copy" && this.term.hasSelection()) {
      navigator.clipboard?.writeText(this.term.getSelection()).catch(() => {
      });
    } else if (name === "paste") {
      void (navigator.clipboard?.readText().catch(() => "") ?? Promise.resolve("")).then((text) => this.sendCommand("paste", void 0, text));
    } else this.sendCommand(name);
    return true;
  }
  /** A key typed on the cells or in a surface while a level of the
   * program's menu shows (SPEC §10.7): Escape, and a key the level does not
   * list, hide it. The key still goes where it would without the menu. */
  menuKey(e) {
    const keys = pressNames(e);
    if (this.shownSeq === null || keys.length === 0) return;
    if (!hides(this.level()?.items ?? [], keys, this.read)) return;
    this.shownSeq = null;
    this.drawMenu();
  }
  /**
   * Draws the level of the program's menu shown (SPEC §10.7) at the bottom
   * right of the screen, on cells, over the surfaces too. A title row, the
   * keys typed and their group's label, then each key that can follow with
   * what it does, in columns of at most `COLUMN_ITEMS`. A group's label has
   * a `+`, and a disabled item is dimmed.
   */
  drawMenu() {
    const m = this.level();
    if (!m) {
      this.menuEl?.remove();
      this.menuEl = null;
      return;
    }
    let layer;
    try {
      layer = this.ensureLayer();
    } catch {
      return;
    }
    const { cellW, cellH } = this.cell();
    const o = this.term.options;
    const p = palette(o.theme);
    const accent = p.ansi[p.dark ? 12 : 4];
    const el = this.menuEl ??= document.createElement("div");
    el.className = "hotty-menu";
    el.replaceChildren();
    Object.assign(el.style, {
      position: "absolute",
      left: "0",
      top: "0",
      width: "",
      height: "",
      boxSizing: "border-box",
      padding: `0 ${cellW}px`,
      background: boxBackground(p),
      color: p.fg,
      fontFamily: o.fontFamily ?? "monospace",
      fontSize: `${o.fontSize ?? 15}px`,
      lineHeight: `${cellH}px`,
      whiteSpace: "pre",
      zIndex: "7",
      pointerEvents: "none"
    });
    const span = (text, style2) => {
      const s = document.createElement("span");
      s.textContent = text;
      Object.assign(s.style, style2);
      return s;
    };
    const title = document.createElement("div");
    title.append(span(this.shownSeq.map(this.show).join(" "), { color: accent }));
    if (m.label) title.append(" " + m.label);
    const cols = document.createElement("div");
    Object.assign(cols.style, { display: "flex", gap: `${2 * cellW}px` });
    const per = Math.max(1, Math.ceil(m.items.length / Math.max(1, Math.ceil(m.items.length / COLUMN_ITEMS))));
    for (let i = 0; i < m.items.length; i += per) {
      const col = document.createElement("div");
      for (const item of m.items.slice(i, i + per)) {
        const group = isGroup(item);
        let label = group && !item.label.startsWith("+") ? `+${item.label}` : item.label;
        const chars2 = [...label];
        if (chars2.length > LABEL_CHARS) label = chars2.slice(0, LABEL_CHARS - 1).join("") + "\u2026";
        const row = document.createElement("div");
        row.style.height = `${cellH}px`;
        if (!item.enabled) row.style.opacity = "0.5";
        else if (group) row.style.color = p.ansi[5];
        row.append(span(this.show(item.key), { color: accent, fontWeight: "bold" }), " " + label);
        col.append(row);
      }
      cols.append(col);
    }
    el.append(title, cols);
    if (el.parentElement !== layer) layer.append(el);
    const w = Math.min(this.term.cols, Math.ceil(el.offsetWidth / cellW));
    const h = Math.min(this.term.rows, Math.ceil(el.offsetHeight / cellH));
    Object.assign(el.style, {
      width: `${w * cellW}px`,
      height: `${h * cellH}px`,
      left: `${(this.term.cols - w) * cellW}px`,
      top: `${(this.term.rows - h) * cellH}px`,
      overflow: "hidden"
    });
  }
  browserKey(e) {
    return (this.opts.browserKeys ?? browserKeys)(e);
  }
  /** A copy key while something is selected (SPEC §11): the selection goes
   * to the clipboard, and the key no further, the browser's copy included.
   * A selection in a surface's document is copied by its keymap, which may
   * leave the key to the program; one on the cells by `copyKeys`. A focused
   * text field's selection is the field's keymap's (Surface.fieldKey). */
  copyKey(e) {
    const surfaces = [...this.surfaces.values()].sort((a, b) => Number(b.hasKeyboard()) - Number(a.hasKeyboard()));
    for (const s of surfaces) {
      const got = s.selectionCopy(e);
      if (got === null) continue;
      if (got === false) return false;
      return this.copied(e, got);
    }
    if (!(this.opts.copyKeys ?? copyKeys)(e) || !this.term.hasSelection()) return false;
    return this.copied(e, this.term.getSelection());
  }
  /** `text` goes to the clipboard, and the key `e` no further. */
  copied(e, text) {
    e.preventDefault();
    if (e.type === "keydown") navigator.clipboard?.writeText(text).catch(() => {
    });
    return true;
  }
  /** A box in the page as the buffer cells it covers (1-based, as xterm.js
   * gives an OSC 8 link's range). */
  cellRange(box) {
    const screen = this.term.element?.querySelector(".xterm-screen")?.getBoundingClientRect();
    const { cellW, cellH } = this.cell();
    const top = this.term.buffer.active.viewportY;
    const at = (x, y) => ({
      x: Math.floor((x - (screen?.left ?? 0)) / cellW) + 1,
      y: top + Math.floor((y - (screen?.top ?? 0)) / cellH) + 1
    });
    return { start: at(box.left, box.top), end: at(box.right - 1, box.bottom - 1) };
  }
  /** Cell size in CSS pixels (private in xterm.js), with a measured fallback. */
  cell() {
    const dims = this.term._core?._renderService?.dimensions?.css?.cell;
    if (dims && dims.width > 0 && dims.height > 0) return { cellW: dims.width, cellH: dims.height };
    const screen = this.term.element?.querySelector(".xterm-screen");
    if (screen && this.term.cols > 0) return { cellW: screen.clientWidth / this.term.cols, cellH: screen.clientHeight / this.term.rows };
    return { cellW: 9, cellH: 17 };
  }
  hostCss() {
    const { cellW, cellH } = this.cell();
    const o = this.term.options;
    this.css = hostCss(o.theme, { cellW, cellH, px: this.px(), fontFamily: o.fontFamily ?? "monospace", fontSize: o.fontSize ?? 15, fonts: this.fonts });
    return this.css;
  }
  /** One device pixel in CSS pixels (`--hotty-px`, SPEC §8), the opposite of
   * `scale`. */
  px() {
    return 1 / (window.devicePixelRatio || 1);
  }
  scheme() {
    return palette(this.term.options.theme).dark ? "dark" : "light";
  }
  /** On every render: did the cell size, font, theme or pixel density
   * change? A new pixel density (a zoom) changes `--hotty-px` and draws the
   * surfaces again, and tells the program nothing (SPEC §5.3). */
  checkMetrics() {
    const { cellW, cellH } = this.cell();
    const o = this.term.options;
    const key = `${cellW}x${cellH}|${this.px()}|${this.font()}|${JSON.stringify(o.theme ?? {})}`;
    if (key === this.metrics.key) return;
    const first = this.metrics.cellW === 0;
    this.metrics = { cellW, cellH, key };
    if (first) this.told = { cellW, cellH, font: this.font() };
    else this.settle();
    if (this.surfaces.size === 0) return;
    const css = this.hostCss();
    const scheme = this.scheme();
    for (const s of this.surfaces.values()) s.setHostCss(css, scheme);
    this.reposition();
  }
  /** What sets the cell's size, besides the pixel density. */
  font() {
    const o = this.term.options;
    return `${o.fontFamily}|${o.fontSize}|${o.lineHeight}|${o.letterSpacing}`;
  }
  /** The capabilities the program hears: it asked for them with its query,
   * and answers differ from them in `caps` (SPEC §4). */
  hear(caps) {
    const { cellW, cellH } = this.cell();
    this.heard = caps;
    this.version = String(caps.v);
    this.told = { cellW, cellH, font: this.font() };
    return caps;
  }
  /** Once the cell, font and theme settle, the program hears what changed.
   * xterm.js may take a font and its cell in two renders, so this waits for
   * both. */
  settle() {
    if (this.capsTimer) clearTimeout(this.capsTimer);
    this.capsTimer = setTimeout(() => {
      this.capsTimer = null;
      this.tellCaps();
    }, 100);
  }
  /**
   * Sends `caps` (SPEC §4, §9) with every field that differs from what the
   * program last heard, to a program that asked for the capabilities. A
   * zoom changes `scale` alone and sends nothing, and moves the cell only by
   * device pixel rounding, with the font kept: that is not a new `cellcss`.
   */
  tellCaps() {
    const { cellW, cellH } = this.cell();
    const font = this.font();
    const t = this.told;
    const owed = font !== t.font && (cellW !== t.cellW || cellH !== t.cellH) || Math.abs(cellW - t.cellW) > 1 || Math.abs(cellH - t.cellH) > 1;
    if (owed) this.told = { cellW, cellH, font };
    const heard = this.heard;
    if (!heard) return;
    const now = this.capabilities(String(heard.v));
    const same2 = (a, b) => JSON.stringify(a, plain) === JSON.stringify(b, plain);
    const changed = {};
    for (const [k, v] of Object.entries(now)) {
      if (v == null || k === "cellcss" && !owed || same2(v, heard[k])) continue;
      changed[k] = v;
    }
    if (Object.keys(changed).every((k) => k === "scale")) return;
    this.heard = { ...heard, ...changed };
    this.send(encode(new Control().set("a", "ev").set("e", "caps"), this.writer.body(changed)));
  }
};
var plain = (_, x) => x instanceof Float ? { float: x.value } : x;
function sequence(chain, f) {
  if (chain) return chain.then(f);
  const r = f();
  return r instanceof Promise ? r : void 0;
}
function openAfterConfirm(uri) {
  if (!confirm(`Do you want to navigate to ${uri}?

WARNING: This link could potentially be dangerous`)) return;
  const w = window.open();
  if (!w) {
    console.warn("Opening link blocked as opener could not be cleared");
    return;
  }
  try {
    w.opener = null;
  } catch {
  }
  w.location.href = uri;
}
function legacyWheelDelta(ev, from) {
  if (!("wheelDeltaY" in WheelEvent.prototype)) return;
  const legacy = from;
  const real = from.isTrusted;
  const pixels = from.deltaMode === WheelEvent.DOM_DELTA_PIXEL ? 1 : from.deltaMode === WheelEvent.DOM_DELTA_LINE ? 20 : 400;
  const x = real && legacy.wheelDeltaX || -from.deltaX * pixels * (120 / 50);
  const y = real && legacy.wheelDeltaY || -from.deltaY * pixels * (120 / 50);
  Object.defineProperty(ev, "wheelDeltaX", { value: x });
  Object.defineProperty(ev, "wheelDeltaY", { value: y });
}
export {
  Control,
  HOST,
  HottyAddon,
  OSC,
  VERSION,
  browserKeys,
  copyKeys,
  decodeAll,
  displayKey,
  encode
};
