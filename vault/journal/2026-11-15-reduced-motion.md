# Reduced motion in hottyterm 0.8 (V8-01)

The kit's existing pulse rule works in hottyterm 0.8, on its released
hotty-blitz v0.8.0 (22ee077). No kit stylesheet or runtime change was
needed.

A temporary app ran `surfrun.Run` with a Compact surface containing
`surfui.Dot().Pulse()`. Only its size and colour were enlarged for pixel
measurements: a 180 × 80 CSS px green rectangle, no change to its
animation or media rules. Its full stylesheet travelled through the
real runtime and hotty-go, including normal resource chunking.

The app ran on a private headless Sway display under CAP, with CPU
affinity retained for the display, an isolated D-Bus session and its own
GTK config. A test-only preload changed that hottyterm process's
`gtk-enable-animations` setting, as hottyterm's own reduced-motion check
does; it changed no desktop setting. The probe sent no document or
delta when the setting changed.

| Setting | Screenshots | Green pixels per shot | Mean green channel |
| --- | ---: | ---: | --- |
| Reduced motion at startup (`gtk-enable-animations=false`) | 5 | 57,600 | 255.0 in every shot |
| Enable animations in the running process | 9 | 57,600 | 121.99–253.99, changing through the pulse |
| Restore reduced motion in the running process | 5 | 57,600 | 255.0 in every shot |

The headless check exited **0**, and hottyterm logged no hotty warning
or error. This verifies the real pulse's media rule and live setting
propagation in both directions. The list-marker transition rule was not
separately measured. The ordinary fidelity gate draws animations still,
so it cannot establish this check by itself.

The temporary probe, logs and screenshots are local cache artifacts,
under the migration's `hotty-surf/motion` directory. No probe code is
added to the kit.
