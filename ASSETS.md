# Asset layout

This project is a source-faithful Go reimplementation of Gun Mayhem. During development, the original Flash/XFL export is used as a behavioral and visual reference.

## Runtime assets

The Go version currently reads these paths at runtime:

- `assets/fla/DOMDocument.xml` — media/linkage metadata needed by the Go runtime, including the five 11.025 kHz fallback sounds.
- `assets/fla/bin/` — only five tiny raw PCM fallbacks (`drop1`, `drop2`, `drop3`, `explosion1`, `explosion2`) that the Go MP3 decoder cannot reliably decode from the original MPEG-2.5 exports.
- `assets/fla/LIBRARY/` — XFL symbol definitions actually used by the port for maps, transforms, timelines, hit geometry and vector reconstruction.
- `assets/fonts/` — optional locally supplied fonts used by HUD/menu reconstruction; `.ttf` binaries are not redistributed through Git and the runtime uses supported system-font fallbacks where available.
- `assets/scripts/` — selected ActionScript files still consumed as runtime data for weapon timeline/animation behavior.
- `assets/sounds/` — compact FFDec MP3 exports used by the runtime audio engine for music and normal sound effects.
- `assets/sprites/` — FFDec raster exports used where a symbol is rendered as a raster or as a fallback.

`assets/fla/LIBRARY/*.wav` is intentionally ignored by Git because those files duplicate the audio exports above.

## Reference-only archive

`source/` is the separate local reference tree for material from the original Flash export that is useful while checking the 1:1 port but is not required by the current Go runtime:

- `buttons/`
- `frames/`
- `images/`
- `shapes/`
- `texts/`
- the full unused portion of the original `fla/bin/` raw-audio archive
- `gunmayhem.swf`
- `gunmayhem_meta.sqlite`
- XFL publishing metadata such as `PublishSettings.xml` and `fla.xfl`

The `source/` tree is deliberately excluded from Git. Nothing was deleted when the asset tree was reorganized; the files remain available locally for reverse-engineering. Runtime code resolves files only from `assets/` and does not fall back to `source/`.

## Philosophy

The goal is **behavioral fidelity, not architectural fidelity**. The original ActionScript and XFL are treated as the specification for gameplay timing, transforms and edge cases, while the Go implementation is free to use a cleaner and more efficient architecture.
