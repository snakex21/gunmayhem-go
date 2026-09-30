# Legacy source-font compatibility

This narrowly bounded runtime compatibility path restores use of the **five exact checked-in legacy font versions** in `assets/fonts`. It is not a repair facility for arbitrary fonts, a font conversion tool, or an expansion of the supplied glyph subsets.

## Scope and byte preservation

`loadSourceFontCandidates` tries ordinary `opentype.Parse` in the existing candidate order. An ordinary native/full font always wins over a normalized legacy subset, even if the subset path came first. Only when no candidate parses normally is the first valid normalized copy used. A copy can be made only after ordinary parsing fails and the full input SHA256 matches one of the five hashes in `source_font_compat.go`. Unknown or modified font bytes receive no speculative normalization. The hashes identify repository asset versions; they are not an authenticity or security guarantee.

The compatibility path verifies the expected TrueType directory, table identities, alignment, non-overlapping bounds, head magic/version, and hhea/maxp lengths, versions and metric counts using widened arithmetic. It then changes only:

- The declared `head` length, from 56 to 54, after checking its two trailing bytes are zero
- The declared `hmtx` length, removing exactly four zero tail bytes beyond the count-derived payload
- Directory table checksums and `head.checkSumAdjustment`, using standard SFNT calculation

The original buffers and files are unchanged. All glyph outlines, mapping, metrics, naming/licensing data and padding bytes are retained. The only changed table payload bytes are the required four-byte `head.checkSumAdjustment` metadata field. The normalized copy must still pass the existing ordinary parser; no additional content defects are repaired.

The same two-phase candidate loader is used by both `sourceMenuFace` and `sourceHUDFace`. The five bundled roles and HUD now load on Linux without Windows fonts. Native/full fonts that worked before retain priority.

At the rendering boundary, a small whole-label face selector checks actual glyph availability. It keeps the selected source/native face for fully covered strings. For a missing face or missing glyph, it selects the already-present `golang.org/x/image/font/gofont/goregular` font only if that font covers the entire label. Menu, HUD, centered HUD, power-up X2 and player-arrow distance call sites choose this same face before measurement and drawing. There is no per-glyph mixing or layout rewrite. Go Regular also supplies the otherwise unavailable Arial/Arial Bold label roles on systems without those native fonts.

Go Regular is embedded from the existing dependency, without new downloads or changes to original font files. Its copyright and BSD-style license are retained verbatim in `assets/fonts/Go-Regular-LICENSE.txt`, which the existing portable packager includes when copying the assets tree.

## Checksum evidence

All 55 original table checksums and five whole-font adjustments were found to match a nonstandard signed-byte accumulation with repeating shifts `[0, 24, 16, 8]`. This establishes a consistent metadata defect, without claiming to identify its original implementation or version. The runtime does **not** apply this observed algorithm to recognize arbitrary new files: only the exact SHA256 list can enter normalization.

For the copy, checksums are unsigned big-endian 32-bit word sums with zero padding and modulo-2^32 arithmetic. Head's directory checksum uses a zero adjustment, then the whole-font adjustment is set to make the complete checksum `0xb1b0afba`. Tests independently check every normalized table checksum, whole-font checksum, and full-file SHA256 goldens generated with Python word sums.

References: [SFNT directory and checksum specification](https://learn.microsoft.com/en-us/typography/opentype/spec/otff#calculating-checksums), [head](https://learn.microsoft.com/en-us/typography/opentype/spec/head), [hmtx metric-count formula](https://learn.microsoft.com/en-us/typography/opentype/spec/hmtx).

## Actual glyph and layout coverage

Tests use the actual five shipped fonts, never a Windows replacement. Every normalized glyph outline is loaded; exact printable-ASCII subsets are pinned; selected labels are measured and rasterized into pixel images with clipping checks.

- Extra Bold: 118 glyphs; all printable ASCII; `KNOCKBACK RESISTANCE` measures 188 + 11/64 px at size 20, 72 DPI, without hinting
- Condensed: 74 glyphs; uppercase Z and some punctuation are absent
- Tw Cen MT: 58 glyphs; digits 0, 3, 5–9 and uppercase J/Q/X/Z are absent
- Showcard Gothic: 55 glyphs; uppercase/digit coverage with limited lowercase and punctuation
- Century Gothic: 17 glyphs; numeric/punctuation subset only

These are preserved source limitations. `JOIN SESSION`, `PLAYER 3` and `PAGE 10/20` exercise whole-label Go Regular fallback and are tested for complete coverage, consistent measurement and unclipped rasterization. Covered source/native labels retain their original face. Go Regular is not universal Unicode coverage; if neither face covers a label, existing missing-glyph behavior is retained. This change does not claim full-screen layout validation or coverage of arbitrary user text.

## Verification

On Go 1.26.2 / Linux with temporary Xvfb:

- Focused font tests: pass, including all five actual assets, normal reparsing, checksum goldens, input/disk/payload preservation, 31 malformed-directory/bounds cases, unknown inputs, unrelated invalid cmap, and menu/HUD loading
- Focused font tests with `-race -count=3`: pass, including ordinary-candidate priority, first-known fallback order, missing-font versus missing-glyph fallback, covered-face identity, unsupported Unicode, and fractional-size caching
- Visual sample montage of all five actual normalized fonts: rendered and inspected, legible without clipping
- Full `go test ./... -count=1`: not clean; failed on two pre-existing AMR2 missing-frame tests, GM2 weapon-card hit detection, sandbox-denied LAN netlink access, and a timing-dependent gateway-close test
- Baseline recheck using an overlay of both original `56f9082` loader files reproduces the first four failures; the gateway test passes that recheck. No unrelated production fixes are included here
- Full suite excluding the five named baseline/environment/timing failures: pass on the final runtime/tests (game package 55.047s)
- Windows/amd64 cross-build with `-buildvcs=false`: pass; execution on Windows is not claimed
- `go vet ./...`: reports the existing unreachable code at `internal/game/menu.go:262`

Independent code review and focused race tests also passed. The normal full suite was not represented as passing. This branch starts at network milestone `56f9082`; knockback and later network changes are kept separate.
