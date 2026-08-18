# Asset layout

This project is a source-faithful Go reimplementation of Gun Mayhem. The repository intentionally keeps the runnable Go port separate from the original/reference Flash export.

## `assets/` — runtime only

The Go game reads only from `assets/`. Files that are useful only for reverse-engineering belong in `source/`, not here.

Current runtime groups:

- `assets/fla/DOMDocument.xml` — media/linkage metadata required by the audio loader.
- `assets/fla/bin/` — five tiny 11.025 kHz raw PCM fallbacks (`drop1`, `drop2`, `drop3`, `explosion1`, `explosion2`) that the Go MP3 decoder cannot reliably decode from the original MPEG-2.5 exports.
- `assets/fla/LIBRARY/` — XFL symbol definitions and bitmap data still consumed by the Go render/timeline/map code.
- `assets/fonts/` — only the bundled fonts actually requested by the current HUD/menu loaders.
- `assets/sounds/` — compact audio files actually addressable by the current runtime.
- `assets/sprites/` — raster exports used by the Go renderer or as source-faithful fallbacks.

The runtime no longer reads decompiled ActionScript files for weapon animation constants; those values are transcribed into Go data and the original scripts live under `source/scripts/`.

The large XFL `LIBRARY/*.wav` exports live under `source/fla/LIBRARY/`, not `assets/`; runtime audio uses the compact files in `assets/sounds/` plus the five raw PCM fallbacks in `assets/fla/bin/`.

## `source/` — original/reference material

`source/` is tracked in Git on purpose. It contains the original/reference Flash/FFDec material used to verify behavior and continue the 1:1 port, including the SWF, decompiled scripts, authoring/export data, raw reference audio and other exported material.

The Go runtime never falls back to `source/`. A clean clone can build and run from the Go code plus `assets/`, while `source/` remains available beside it for comparison and future porting work.

Some material may intentionally exist in both `source/` and `assets/`: one copy is reference/source material, the other is the runtime representation used by the Go port.

## Philosophy

The goal is **behavioral fidelity, not architectural fidelity**. The original ActionScript/XFL is the specification for gameplay timing, transforms and edge cases, while the Go implementation uses a cleaner and more efficient architecture.