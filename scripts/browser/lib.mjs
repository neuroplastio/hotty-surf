// What the browser checks share.

// until polls cond until it holds, or throws after ms.
export async function until(cond, what, ms = 15000) {
  const end = Date.now() + ms;
  for (;;) {
    try {
      if (await cond()) return;
    } catch {
      // a frame being replaced; try again
    }
    if (Date.now() > end) throw new Error("never: " + what);
    await new Promise((r) => setTimeout(r, 50));
  }
}

// surface is the frame of the hotty surface (xterm-addon-hotty draws each
// in an iframe) whose text holds text, once it does.
export async function surface(page, text, ms = 30000) {
  let found = null;
  await until(async () => {
    for (const f of page.frames()) {
      if (f === page.mainFrame()) continue;
      const body = await f.evaluate(() => document.body?.innerText ?? "").catch(() => "");
      if (body.includes(text)) {
        found = f;
        return true;
      }
    }
    return false;
  }, `a surface with ${JSON.stringify(text)}`, ms);
  return found;
}
