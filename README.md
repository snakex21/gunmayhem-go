# Gun Mayhem Go

A source-faithful reimplementation of **Gun Mayhem** in Go using [Ebitengine](https://ebitengine.org/).

The goal is not to reproduce the original Flash code structure. The goal is to reproduce **how the game actually behaves**: movement, weapons, AI, campaign rules, HUD timing, Flash timelines, map geometry, effects and all the small edge cases that made the original game feel the way it did.

> **Status:** work in progress. The game is playable, but this repository is still being compared against the original Flash source and corrected mechanic by mechanic.

## Goals

- Match the original Gun Mayhem gameplay as closely as possible.
- Treat the original ActionScript/XFL as the behavioral specification.
- Keep the Go implementation cleaner and more efficient than the original Flash architecture.
- Preserve original timing, hit behavior, weapon animation logic and campaign-specific quirks instead of replacing them with approximations.
- Avoid unnecessary Flash-era duplication where the same result can be expressed cleanly in Go.

## Current implementation

The port already includes substantial parts of the original game, including:

- player movement, jumping, platform interaction and knockback,
- original weapon roster and weapon-specific timelines,
- bullets, shotguns, Instagib, minigun, grenades and dropped weapon parts,
- crates and power-ups,
- AI players and campaign-specific AI variants,
- campaign missions and progression,
- Pylon Man's independently controlled double/helper behavior,
- 1 Hit 1 Kill's delayed Instagib death timeline,
- HUD, lives, Last Life / Game Over / Level Up states,
- original map geometry reconstructed from XFL,
- source menu/HUD fonts,
- original audio playback,
- Flash-style visual timelines and transforms,
- Custom Game and post-game/statistics work in progress.

## Why this port is different

The original game is heavily tied to Flash concepts such as `MovieClip`, `attachMovie`, frame scripts, `_root`, instance names and `onEnterFrame`. Recreating that architecture directly would preserve a lot of complexity without providing any benefit.

This project instead uses a **behavioral port** approach:

1. trace the complete behavior in the original ActionScript/XFL,
2. determine the final gameplay rule and exact timing,
3. implement that behavior explicitly in Go,
4. verify it against the original game.

For example, the original `playerAI_double` spawns a normal `playerAI` instance named `double`. Several unrelated-looking source conditions then alter its spawn, lives, transparency and death behavior. In this port those rules can be represented explicitly while still producing the same result.

## Requirements

- Go **1.26.2** or newer compatible with the current module configuration
- Windows, Linux or macOS supported by Ebitengine

A fresh clone contains the runtime assets required by the Go port in `assets/`, including the font files used by the source-faithful HUD/menu rendering. No separate local asset pack is required.

## Build

```bash
go mod download
go build -o GunMayhem.exe .
```

Run tests with:

```bash
go test ./...
```

## Developer / cheat mode

The normal executable starts with all porting/debug shortcuts disabled. To deliberately enable the built-in developer tools, launch the same executable with:

```text
GunMayhem.exe --dev
```

`--debug` is accepted as an alias. In this mode the porting shortcuts are available again: F1 debug overlays, F2/F3 map switching, F4 AI toggle, R arena reset, and F10 campaign unlock. This keeps one executable useful both as the clean Original build and as a debugging/verification build without exposing developer behavior during normal play.

## Project structure

```text
.
├── assets/                    # all runtime assets used by the Go port
│   ├── fla/
│   ├── fonts/
│   ├── sounds/
│   └── sprites/
├── source/                    # tracked original/reference Flash/FFDec material
├── cmd/
├── internal/game/             # gameplay, source parsers and rendering
├── ASSETS.md                  # asset separation and runtime dependencies
├── go.mod
├── go.sum
└── main.go
```

See [ASSETS.md](ASSETS.md) for the runtime/reference split. The Go runtime reads only `assets/`; `source/` is kept separate for 1:1 reverse-engineering work.

## Development rule

When behavior differs from the original, the original Flash source wins.

A fix should normally be based on the full source path involved in the mechanic rather than on visual guesswork. That may mean following behavior across a root frame script, a player class, a nested MovieClip timeline and a projectile script before changing the Go implementation.

## Legal / attribution

This is an independent technical reimplementation and is not affiliated with the original Gun Mayhem developers or publishers.

Gun Mayhem, its name, artwork, audio and other original game assets belong to their respective rights holders.
