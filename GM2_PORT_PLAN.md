# Gun Mayhem 2 source audit / port plan

This document records facts extracted from the unpacked Gun Mayhem 2 Flash/XFL source before integrating GM2 content into the Go port. The rule stays the same as for GM1: observable behavior comes from source, not guesswork.

## Content identity

GM2 must not be treated as a blind replacement of GM1 data.

- Flash/XFL symbol numbers are renumbered in GM2, so `Symbol N` from GM1 and `Symbol N` from GM2 can represent different objects.
- Map numbers are also renumbered between games.
- GM2 campaign is a separate 16-mission campaign, not GM1's 10 missions plus six appended missions.
- Weapon numbers 1..66 remain compatible; GM2 appends weapons 67..86. This is the one major content ID range that can safely extend the existing catalog.

Runtime assets therefore need a GM1/GM2 namespace. Never merge the two `fla/LIBRARY` trees into one flat directory.

## GM2 weapons

GM2 keeps weapons 1..66 and adds 20 weapons. The values below come directly from each GM2 weapon `frame_1/DoAction.as`.

| # | Source symbol | Source name | ROF | Firepower | Recoil | Ammo | Shotgun | Weight | Blowback | Pushback | Idle rotate | Shell X | Flash X |
|---:|---|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| 67 | `DefineSprite_573_gun_A1` | GREY SMG | 3 | 15 | 0.4 | 30 | 0 | 0.9 | 10 | 3 | 30 | 20 | 55 |
| 68 | `DefineSprite_575_gun_A2` | CHEAP SMG | 2 | 12 | 0.3 | 32 | 0 | 0.9 | 10 | 3 | 30 | 20 | 55 |
| 69 | `DefineSprite_577_gun_A3` | MINI SMG | 3 | 16 | 0.5 | 30 | 0 | 0.95 | 10 | 3 | 40 | 17 | 45 |
| 70 | `DefineSprite_579_gun_A4` | CHROME SMG | 4 | 20 | 0.6 | 30 | 0 | 0.8 | 7 | 10 | -70 | 21 | 64 |
| 71 | `DefineSprite_581_gun_A5` | FANCY SMG | 4 | 21 | 0.6 | 30 | 0 | 0.9 | 10 | 3 | 30 | 25 | 50 |
| 72 | `DefineSprite_583_gun_B1` | RELIABLE RIFLE | 5 | 28 | 1 | 20 | 0 | 0.75 | 7 | 10 | -60 | 17 | 65 |
| 73 | `DefineSprite_585_gun_B2` | ANTIQUE RIFLE | 5 | 28 | 1 | 30 | 0 | 0.8 | 7 | 10 | -60 | 20 | 72 |
| 74 | `DefineSprite_587_gun_B3` | BULLPUP STEALTH RIFLE | 4 | 24 | 0.4 | 30 | 0 | 0.8 | 10 | 10 | 30 | -3 | 58 |
| 75 | `DefineSprite_589_gun_B4` | PRECISION CARBINE | 5 | 26 | 0.4 | 20 | 0 | 0.85 | 7 | 10 | -70 | 21 | 62 |
| 76 | `DefineSprite_591_gun_B5` | OLD ASSAULT RIFLE | 3 | 19 | 0.6 | 40 | 0 | 0.8 | 7 | 10 | -70 | 21 | 64 |
| 77 | `DefineSprite_593_gun_C1` | .50 SNIPER | 18 | 62 | 1.5 | 5 | 0 | 0.7 | 8 | 15 | -60 | 20 | 79 |
| 78 | `DefineSprite_595_gun_C2` | STEADY SNIPER | 12 | 48 | 1.2 | 10 | 0 | 0.8 | 8 | 15 | 30 | -10 | 77 |
| 79 | `DefineSprite_597_gun_C3` | SINGLE SHOT SNIPER | 31 | 70 | 10 | 1 | 0 | 0.7 | 10 | 30 | -70 | 25 | 75 |
| 80 | `DefineSprite_599_gun_C4` | TACTICAL STEALTH SNIPER | 25 | 55 | 5 | 5 | 0 | 0.75 | 10 | 30 | 30 | -5 | 75 |
| 81 | `DefineSprite_601_gun_C5` | LIGHT SNIPER | 31 | 50 | 7 | 5 | 0 | 0.8 | 10 | 30 | -60 | 20 | 70 |
| 82 | `DefineSprite_603_gun_D1` | AUTOMATIC SHOTGUN | 6 | 7 | 0.8 | 10 | 7 | 0.8 | 10 | 20 | -60 | 20 | 60 |
| 83 | `DefineSprite_605_gun_D2` | HI-CAP SHOTGUN | 12 | 7 | 0.6 | 12 | 7 | 0.75 | 8 | 15 | 30 | 20 | 60 |
| 84 | `DefineSprite_609_gun_D3` | STANDARD SHOTGUN | 25 | 8 | 5 | 5 | 7 | 0.8 | 20 | 40 | -80 | 20 | 58 |
| 85 | `DefineSprite_611_gun_D4` | STREET SWEEPER | 10 | 7 | 0.5 | 12 | 7 | 0.85 | 8 | 15 | 30 | 20 | 60 |
| 86 | `DefineSprite_613_gun_D5` | 4 ROUND SHOTGUN | 12 | 8 | 2 | 4 | 7 | 0.85 | 10 | 20 | 25 | 20 | 63 |

GM2 `gunsound()` also contains explicit sound mappings for 67..86.

## GM2 maps

The GM2 custom-menu text export maps frames 1..21 to:

1. Safari Showdown
2. Polar Pwn4ge
3. Midnight Wood
4. Hovering Houses
5. Desert Destruction
6. Great Wall Brawl
7. Solar Shootout
8. Underwater Slaughter
9. Dessert Duel
10. No Name
11. Grim City
12. Mushroom Mountain
13. Jungle
14. Ski Lift
15. Space Station
16. Avalon
17. Venice
18. Alien Planet
19. Sub Base
20. Highway
21. Castle

Special selector frames:

- 22 = Random (new maps), source resolves this to random map 1..9.
- 23 = Random (all maps), source resolves this to random map 1..21.

GM2 also contains `mapfx_rain`, in addition to snow/wall map effects.

## GM2 campaign

GM2 stores `campaign[16]`. Fresh defaults are:

- mission 1 = available
- mission 2 = available
- missions 3..16 = locked

Mission -> map mapping from `DefineSprite_1301`:

`[5, 9, 19, 2, 7, 4, 6, 14, 1, 20, 12, 3, 21, 18, 10, 16]`

This campaign is independent from GM1. Source-specific opponents include Easy AI, Triple Jumper, Moderate AI, The Ghost, Gangster, Tiny Guy, Pistol Man, Too Fast, Terminator and multi-AI encounters. Several missions have dedicated code in player, AI, bullet or weapon scripts, so each mission must be ported from its full source path rather than represented only as a table of opponent cosmetics.

## Challenges

GM2 stores seven challenges as `challenge[7][2]`:

- `[i][0]` = medal/result state 0..3
- `[i][1]` = best elapsed time from Flash `getTimer()` in milliseconds

Challenge medal thresholds from source:

| Challenge | Bronze | Silver | Gold |
|---:|---:|---:|---:|
| 1 | 37.0 s | 30.0 s | 23.0 s |
| 2 | 34.0 s | 27.0 s | 20.0 s |
| 3 | 36.0 s | 29.0 s | 22.0 s |
| 4 | 35.0 s | 28.0 s | 21.0 s |
| 5 | 34.0 s | 27.0 s | 20.0 s |
| 6 | 27.0 s | 20.0 s | 13.0 s |
| 7 | 34.0 s | 29.0 s | 24.0 s |

Challenge maps from root frame 10 are 7, 1, 9, 2, 6, 8 and 5 respectively.

## Modes

Source-confirmed menu modes include the GM1 set plus at least:

- 10 = Gun Game Reversed
- 11 = Jetpacks

The export also contains mode buttons 6, 12 and 13. Mode 6's exported display text is the authoring placeholder `Asdf Qwer`; 12/13 need additional source tracing before assigning user-facing names. Do not invent names for them.

## GM2-specific defaults / differences

- GM2 P1 default controls are arrows + Z/X (`90`/`88`) rather than GM1's arrows + `[`/`]` (`219`/`221`). Keep per-game defaults separate if a GM2-faithful preset is exposed.
- GM2 player setup stores an `eyes` selection separately and has expanded cosmetics.
- GM2 root source still uses gravity 0.88, friction 0.93, air friction 0.88, speed 0.7 and jump power 13.5, so the core movement constants are shared.
- GM2 crate timer starts at 200 and uses a 350 threshold in root frame 10; do not assume GM1's runtime crate cadence for GM2 modes.
- GM2 powerup timer threshold is 550 in root frame 10.

## Current implementation status

Completed on `main`:

- GM1 remains isolated by the existing source-parity tests and `original`/legacy branch.
- Save format v2 has separate GM2 campaign (16 slots) and seven challenge records.
- GM2 XFL/raster lookup has its own namespace and cache keys, so equal `Symbol N` names cannot collide with GM1.
- Weapons 67..86 are in the shared Go weapon catalog with source-exact ROF, firepower, recoil, ammo, shotgun count, weight, shell/flash positions, hand/shoot pose, blowback/pushback/idle rotation and GM2 `gunsound()` mappings.
- Local source tests verify all twenty definitions against the separately unpacked GM2 ActionScript and verify their first-frame PNG, XFL bounds and timeline.
- During import development `assets/gm2` is preferred, with the separately unpacked sibling `gun mayhem 2` tree accepted as a read-only fallback. Custom Game only enables the 10..86 GM2 crate pool when those render assets are available; otherwise it safely falls back to the GM1 pool.

## Integration order

1. ~~Keep GM1 behavior frozen/covered by existing source tests.~~
2. ~~Add GM2 save state (16 campaign missions + seven challenge records) without changing GM1 progress.~~
3. ~~Add versioned GM2 asset resolver; never flatten GM1 and GM2 XFL symbol namespaces.~~
4. **In progress:** weapons 67..86 are behaviorally integrated; curate/package their runtime files under `assets/gm2` so a fresh clone no longer needs the local unpacked-source fallback.
5. Add GM2 map namespace and maps 1..21; preserve GM1 numeric map IDs inside the GM1 namespace.
6. Add GM2-only modes, starting with Gun Game Reversed and Jetpacks after full source tracing.
7. Port the GM2 campaign as a separate campaign selector/set.
8. Port seven challenges and their timing/medal UI.
9. Only after parity, decide how the combined user-facing menu presents GM1, GM2 and future custom missions.
