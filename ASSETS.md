# Asset layout

This project is a source-faithful Go reimplementation of Gun Mayhem. During development, the original Flash/XFL export is used as a behavioral and visual reference.

## Runtime assets

The Go version currently reads these paths at runtime:

- `assets/flash_source/fla/DOMDocument.xml` — original media/linkage metadata, including the five 11.025 kHz fallback sounds.
- `assets/flash_source/fla/bin/` — only five tiny raw PCM fallbacks (`drop1`, `drop2`, `drop3`, `explosion1`, `explosion2`) that the Go MP3 decoder cannot reliably decode from the original MPEG-2.5 exports.
- `assets/flash_source/fla/LIBRARY/` — XFL symbol definitions used for maps, transforms, timelines, hit geometry and vector reconstruction.
- `assets/flash_source/fonts/` — optional locally supplied source fonts used by HUD/menu reconstruction; `.ttf` binaries are not redistributed through Git and the runtime uses supported system-font fallbacks where available.
- `assets/flash_source/scripts/` — selected original ActionScript, currently read for weapon timeline/animation data.
- `assets/flash_source/sounds/` — compact FFDec MP3 exports used by the runtime audio engine for music and normal sound effects.
- `assets/flash_source/sprites/` — FFDec raster exports used where a source symbol is rendered as a raster or as a fallback.

`assets/flash_source/fla/LIBRARY/*.wav` is intentionally ignored by Git because those files duplicate the audio exports above.

## Reference-only archive

`assets/source_reference_only/` contains exports that are useful when checking the original game but are not required by the current Go runtime:

- `buttons/`
- `frames/`
- `images/`
- `shapes/`
- `texts/`
- the full unused portion of the original `fla/bin/` raw-audio archive
- `gunmayhem.swf`
- `gunmayhem_meta.sqlite`
- XFL publishing metadata such as `PublishSettings.xml` and `fla.xfl`

The reference-only archive is deliberately excluded from Git. Nothing was deleted when the asset tree was reorganized; the files remain available locally for reverse-engineering.

## Philosophy

The goal is **behavioral fidelity, not architectural fidelity**. The original ActionScript and XFL are treated as the specification for gameplay timing, transforms and edge cases, while the Go implementation is free to use a cleaner and more efficient architecture.
