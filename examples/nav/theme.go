package main

// themeCSS is the nav's dark palette, as Basecoat's theme variables (the
// sidebar's, and the plain ones the nav reads) plus the few it needs of its
// own (--busy, --busy-tint, --badge). It loads after the kit's CSS, so it
// wins over the kit's dark theme. The templ uses them through Tailwind's
// theme colours: bg-sidebar, text-muted-foreground, border-sidebar-border,
// and bg-busy-tint for its own.
const themeCSS = `
.dark {
  --sidebar: #191919;             /* the nav's background */
  --sidebar-foreground: #b4b4b4;  /* names and status lines */
  --muted-foreground: #9b9b9b;    /* icons, the workspace, the thumbnails' strokes */
  --sidebar-accent: #1e272d;      /* the active session's row, and a hovered one */
  --accent: #414141;              /* the selected row */
  --sidebar-primary: #617698;     /* the selection's bar, the active tool */
  --sidebar-border: #9b9b9b;      /* the thumbnails' box and splits */
  --sidebar-ring: #617698;        /* the keyboard focus outline */
  --busy: #6fb7c9;                /* a working agent's arc */
  --busy-tint: rgb(110 165 195 / 0.11); /* a working pane's fill */
  --badge: #1e1f21;               /* the disc the arc turns on */
}
`
