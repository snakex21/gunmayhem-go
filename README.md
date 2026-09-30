# Gun Mayhem Go

A source-faithful reimplementation of **Gun Mayhem** in Go using [Ebitengine](https://ebitengine.org/).

## Experimental branch: Internet multiplayer

This branch is a reviewable experiment on top of the LAN release. It adds:

- explicit opt-in, private-invite Cloudflare/WSS hosting with authoritative
  simulation, heartbeat/timeouts, input expiry and tested disconnect/rejoin
- opt-in OPEN Internet rooms through a separately configured HTTPS directory,
  with code-free discovery/join; **no shared directory has been deployed**
- a Custom Game-only **KNOCKBACK -30%** perk with host-controlled restrictions
- narrowly bounded compatibility for the five existing exported font files,
  plus a licensed built-in fallback for labels those font subsets cannot cover

All players must run the same **protocol-20** build. LAN remains the default;
Internet hosting requires explicit opt-in and defaults to private invitations.
OPEN additionally needs the same approved directory URL on every computer. There has been no
real Cloudflare cross-network playtest or Windows gameplay run, so this is not a
supported Internet release. The complete test suite still has the documented
baseline failures. Do not infer automatic reconnection or more than four players.

Start with [the private Internet instructions](EXPERIMENTAL_MULTIPLAYER.md),
[public room setup and directory limits](PUBLIC_ROOMS.md),
[perk behavior and tests](KNOCKBACK_PERK.md),
[font compatibility scope](SOURCE_FONT_COMPATIBILITY.md), and
[verification limits](MULTIPLAYER_TEST_RESULTS.md). No original art/audio/font
asset files are modified by these changes.

The goal is not to reproduce the original Flash code structure. The goal is to reproduce **how the game actually behaves**: movement, weapons, AI, campaign rules, HUD timing, Flash timelines, map geometry, effects and all the small edge cases that made the original game feel the way it did.

> **Status:** work in progress. The game is playable, but this repository is still being compared against the original Flash source and corrected mechanic by mechanic. Multiplayer in the current public release is **LAN-only**; Internet multiplayer remains experimental and is disabled in the released UI.

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
- Custom Game uses a rebuilt MATCH / PLAYLIST / PLAYERS interface. A 1-20 round series can assign a different GM1/GM2 map to every round, with per-map post-game statistics, a cumulative Series Summary and a manually configurable Chaos Mode event pool.

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

The current Go toolchain requires Windows 10 / Windows Server 2016 or newer
([official requirements](https://go.dev/wiki/MinimumRequirements)). The checked
Windows artifact targets x64; this game build is not a Windows 2000 binary.

A fresh clone contains the runtime assets required by the Go port in `assets/`, including the font files used by the source-faithful HUD/menu rendering. No separate local asset pack is required.

## Build

On Windows, use `build.bat`. It runs the tests with project-local Go scratch/cache directories and builds `GunMayhem.exe` as a Windows GUI application, so launching the game does not open a console window.

Manual Windows build:

```bash
go mod download
go build -ldflags "-H=windowsgui" -o GunMayhem.exe .
```

Run tests with:

```bash
go test ./...
```

## Persistent settings and saves

The developed `main` branch keeps machine settings separate from campaign progress.

- `config.json` stores window size/position, fullscreen state, music, sound, render quality and all four players' control bindings.
- `save.json` stores campaign completion/unlocks.
- F11 toggles fullscreen and the selected fullscreen state is remembered.
- Resizing/moving the desktop window is remembered on normal exit. If the game exits while fullscreen, the last windowed size/position is preserved rather than being replaced by the monitor resolution.

Files are stored in the portable `data/` directory beside the executable. Moving the whole game folder therefore moves its settings and saves too. For development/tests, `GUNMAYHEM_DATA_DIR` can explicitly override the directory.

## Automatic updates

Tagged Windows release builds check `snakex21/gunmayhem-go` GitHub Releases asynchronously at startup. When a newer version exists, the main menu shows `UPDATE AVAILABLE ... - DOWNLOAD & UPDATE` with download progress. The update package is downloaded only after the user clicks the notice.

The updater uses the release ZIP `GunMayhem-Windows-x64.zip` (the older `GunMayhem-Multiplayer-Portable.zip` name is also accepted), installs files after the running game exits, preserves the local `data/` directory, restarts the new executable and removes the downloaded ZIP/helper afterward. Update work files stay under the game's own `data/update/` directory; the system TEMP directory is not used.

`build.bat` injects the exact Git tag as the application version when HEAD is on a tag. The current LAN-only release line defaults to `v0.16.0`; `GUNMAYHEM_VERSION=vX.Y.Z` can override it explicitly. For a release: build from the release tag (or set that environment variable), run `go run ./scripts/package_portable.go`, then attach `build/GunMayhem-Windows-x64.zip` to the GitHub Release with the same/newer version tag.

## Project structure

```text
.
├── assets/                    # all runtime assets used by the Go port
│   ├── fla/
│   ├── fonts/
│   ├── scripts/
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

The developed `main` branch unifies local and LAN network play inside **Custom Game**. The shared player setup supports local humans, LAN human slots, AI bots and empty slots (including mixed matches such as 2 humans vs 2 bots), while LAN play keeps authoritative hosting, synchronized setup/rules, server discovery and join-in-progress. Internet networking experiments remain in the source tree but are disabled in the public release. Custom Game can also run a configurable 1-20 round playlist where every round stores its own map, with per-map READY flow in network play and cumulative per-nickname standings. Chaos Mode can rotate a host-selected pool of weapon, movement, environmental and hazard events during a match, including grenade/dynamite/Molotov rain and crate/powerup storms. See [MULTIPLAYER.md](MULTIPLAYER.md) for details.

## Development rule

When behavior differs from the original, the original Flash source wins.

A fix should normally be based on the full source path involved in the mechanic rather than on visual guesswork. That may mean following behavior across a root frame script, a player class, a nested MovieClip timeline and a projectile script before changing the Go implementation.

## Legal / attribution

This is an independent technical reimplementation and is not affiliated with the original Gun Mayhem developers or publishers.

Gun Mayhem, its name, artwork, audio and other original game assets belong to their respective rights holders.
