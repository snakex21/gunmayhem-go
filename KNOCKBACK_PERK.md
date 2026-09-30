# Custom Game knockback-resistance perk

Separate gameplay milestone on top of the private-network experiment.

## Behavior

Custom Game now offers perk **10: KNOCKBACK -30%**. It multiplies each newly
received combat impulse by 0.70, leaving existing velocity untouched. It uses the
ordinary single-perk slot, so selecting it forgoes the other perks. Thirty percent
is initial tuning, not a claim of competitive balance.

Covered impulses: ordinary bullets/shotguns/minigun, grenade and rocket blasts,
ignition and periodic burning, fire-contact pulses, Molotov impact, thrown reload
parts, and unpaired portal-weapon pulses. Instagib's impulse is reduced but its
existing delayed death still occurs. Existing shield modifiers apply first;
resistance multiplies the resulting impulse.

Movement, jump height, self-recoil, paired portal movement, damage attribution,
kill/hit counts and burn duration are unchanged. Environmental forces such as
Chaos wind, quake and shock pulses are intentionally excluded. Campaign perk
selection stays at the original 1–9, preserving its existing balance.

## UI and authoritative networking

The Custom Game and online character pickers include a code-native text tile.
The host can disable this perk through the existing perk blocklist or disable
all perks. The blocklist uses the actual click dispatch, including repeated
clicks to block and unblock. No new graphics, downloaded art or original assets
are added or changed.

The host/local game applies the modifier. Remote clients send only their existing
character configuration and input; snapshots carry the resulting player state.
The expanded perk-blocklist array bumps the wire protocol to **20**. Everyone in
a match must use the same build; network-only protocol-19 clients are rejected.
The separate network-only milestone remains available at commit `56f9082`.

## Verification

- Deterministic comparisons across 11 received-combat impulse paths, including
  shield interaction and preservation of hit attribution/burn duration
- Pre-existing velocity untouched; original perks unchanged
- Instagib still triggers delayed death; paired portal position/velocity unchanged
- Unchanged movement/jumping, self-recoil and environmental wind
- Actual picker hit/selection, unchanged parsed source cache and campaign exclusion
- Host blocklist hit→activation→toggle→untoggle, perks-off and invalid-config rejection
- Real local transport roundtrip for perk configuration, authoritative velocities
  and blocklist, plus protocol-19 rejection
- Independent scoped review caught and verified fixes for the unpaired portal
  pulse and blocklist click-dispatch range

The optional `perkpreview` build-tagged test renders the three affected screens.
Layout was inspected using an explicitly test-only DejaVu font override because
the original bundled TTF metadata is rejected by the current Go font parser.
Those screenshots are **layout fixtures, not actual production-font appearance**.
The separate font compatibility investigation is not part of this perk commit.

```sh
go test -race ./internal/game -run '^TestKnockbackResistance' -count=1
# Optional standalone graphics fixture, run under a display:
GUNMAYHEM_PREVIEW_DIR=/tmp/perk-preview \
GUNMAYHEM_PREVIEW_FONT=/path/to/test-only-font.ttf \
go test -tags perkpreview ./internal/game -run '^TestKnockbackPreviewScreenshots$' -count=1
```

The four full-suite baseline failures and existing vet warning are recorded in
`MULTIPLAYER_TEST_RESULTS.md`. Neither milestone claims a clean full-suite pass,
Windows gameplay execution or a real Cloudflare cross-network match.

Final milestone checks: focused perk race checks passed (1.776 seconds); combined
network/perk race checks passed (32.641 seconds); the complete suite finished
with the same four baseline failures (49.926 seconds). Windows amd64 cross-build
and diff whitespace checks passed. The fallback-font layout fixture was rerun
after the final UI label changes. The new numeric perk labels deliberately use
the existing extra-bold face, whose subset includes all of their characters.

### Integrated production-font check

After the separate font compatibility commit, the three setup screens and a
runtime HUD fixture were rerendered with `GUNMAYHEM_PREVIEW_FONT` unset. The actual
source fonts and production whole-label fallback rendered the labels and HUD
numbers without new clipping in these four screens. The earlier DejaVu images
remain only historical layout fixtures. This is representative render QA, not
Windows execution or an Internet gameplay test.
