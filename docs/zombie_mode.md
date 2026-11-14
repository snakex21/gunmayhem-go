# ZOMBIE mode (step 1)

Co-op survival against waves of zombies, local (1-4 players and AI bots on one
screen) or online (host-authoritative, every guest on its own screen).

Code: `internal/game/zombie_*.go`

| File | Contents |
| --- | --- |
| `zombie_defs.go` | All tunable data: zombie kinds, wave scaling, points, player rules, arenas (zones, spawn points, box spots, doors) |
| `zombie_mode.go` | Match flow (phases, waves), players (HP, down, revive), hit phase (damage, points), records |
| `zombie_ai.go` | Spawning, platform navigation graph, steering, melee, leaper / exploder / boss specials |
| `zombie_shop.go` | Shop catalog and purchase rule, bot shopping, mystery box |
| `zombie_shop_ui.go` | Shop panels (Mayhem UI kit), per-player keys |
| `zombie_net.go` | Online: snapshot lists, guest shop requests, validation |
| `zombie_hud.go`, `zombie_visuals.go` | HUD, procedural zombie art, box, down markers, popups |
| `zombie_menu.go` | Setup and results screens, online lobby settings, records |

## Rules

### Modes

* **SURVIVAL**: 20 waves. Clearing wave 20 wins the match.
* **ENDLESS**: the waves never end. The best wave and the best team score per
  arena are saved in `progress.json` (`zombie_records`, key `<arena id>/<mode>`).

A wave starts with a 3 s banner. The wave ends when every zombie of the wave
has spawned and died. The break between waves lasts 25 s. During the break the
shop is open. When every player is READY, the next wave starts after 3 s.

### Zombies

Zombies are not entries of `Game.players`. Each zombie has a private `Player`
body, like the campaign enemies (`docs/campaign_enemies.md`). The body is used
for platform physics and as a hit receiver. `hitTargets()` passes the living
bodies to the shared code for bullets, blasts, gas, flashbangs and fire. This
gives zombies the following without a separate code path:

* every weapon and throwable works on them;
* knockback is scaled by `KnockbackScale` inside the ability knockback windows;
* falling off the map kills a zombie (`Player.Kill`).

**Damage**: `endZombieHitPhase` takes the velocity change of the frame and
divides it by the zombie's knockback share. A brute that barely moves still
takes the full damage. A bullet therefore does damage equal to its firepower,
and a blast does damage equal to its push. A bullet that stops in the top 40 %
of the hitbox is a **headshot** and does +50 % damage. Burning does 1 HP every
6 frames.

Wave 1 values. HP is multiplied by the wave's HP multiplier and speed by its
speed multiplier.

| Kind | Colour | HP | Move | Knockback taken | Size | Hit (HP / push) | Kill pts | From wave |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| WALKER | green | 40 | 0.42 | 16 % | 1.0 | 12 / 7 | 60 | 1 |
| RUNNER | yellow, red shirt | 22 | 0.95 | 26 % | 0.9 | 7 / 5 | 70 | 2 |
| EXPLODER | orange, bloated | 30 | 0.55 | 20 % | 1.08 | blast | 80 | 3 |
| BRUTE | purple, wide | 170 | 0.32 | 6 % | 1.38 | 26 / 19 | 160 | 4 |
| LEAPER | blue, crouched, claws | 34 | 0.60 | 20 % | 0.92 | 12 / 9 (pounce) | 90 | 6 |
| ABOMINATION (boss) | red, horns | 900 x level | 0.36 | 2.5 % | 2.2 | 30 / 22 + slam | 1000 | every 5th |

* **Exploder**: starts a 22-frame fuse within 46 px of a player. The blast has
  a 105 px radius. Players take up to 30 HP plus knockback, and zombies take up
  to 90 HP. Shooting an exploder dead detonates it at once. An exploder caught
  in a blast goes off 6 frames later, so they chain. The kills are credited to
  the player who started the chain. An exploder that falls off the map does
  not explode.
* **Leaper**: pounces (vx 9.5, vy -10.5) when the target is on the same
  platform within 230 px. The cooldown is 2.5 s.
* **Boss**: comes every 5th wave and is level wave/5. Its HP is
  900 x (1 + 0.6 (level-1)) x (1 + 0.5 (players-1)). Its moves:
  * Ground slam every 8 s. It jumps, and on landing hits players within
    230 px for up to 22 HP plus knockback.
  * Summons 3 adds every 12 s.
  * Below 50 % HP (phase 2), it slams every 4.8 s and summons every 9 s, and
    the adds include an exploder.

  The boss has its own HP bar at the top. A boss wave has half the usual adds.

**Spawning**: zombies come out of the spawn points of the zone. Spawn points
within 200 px of a player are skipped. Each spawn shows a glowing crack and a
chevron for 0.8 s, then the zombie climbs out for 0.6 s and cannot be hit
yet. The boss uses the spawn point farthest from the players and takes twice
as long.

### Wave scaling (`zombieWavePlanFor`)

| | Formula |
| --- | --- |
| count | (6 + 3 (w-1) + 1.5 max(0, w-10)) x (1 + 0.45 (players-1)), max 120, halved on boss waves |
| HP multiplier | (1 + 0.13 (w-1)) x (1 + 0.04 max(0, w-10)) x (1 + 0.15 (players-1)) |
| speed multiplier | min(1 + 0.02 (w-1), 1.35) |
| alive at once | min(14 + 4 players + w, 48) |
| spawn interval | max(10, 40 - 2w) frames |
| type weights | walker 100; runner 20+6/wave (max 70); exploder 10+3 (35); brute 8+3 (30); leaper 10+3 (35) |

Examples for one player: wave 1 has 6 zombies, wave 5 has 9 plus the boss, wave
10 has 33, and wave 20 has 78 zombies with 4.9x HP.

### Players

* 100 HP. A zombie hit does damage and knockback through the players'
  knockback window, so HEAVY, ANCHOR, KEVLAR and SHIELD work. The SHIELD
  ability also blocks the damage of the hit it absorbs. The shield powerup
  takes 1/5 of the damage.
* **DOWN** at 0 HP or after a ring-out. The down marker is placed on the
  platform under the player, or after a ring-out at the last place the player
  stood. A teammate (human or bot) standing within 58 x 64 px of the marker
  for **3 s** revives the player with 50 HP, 2 s of invulnerability and the
  weapons they had. The reviver gets +150 points. Leaving the marker resets
  the progress.
* **Bleed-out**: a player who is not revived within 30 s is DEAD (gravestone)
  until the wave ends.
* When the wave ends, downed players get up and dead players drop back in.
* **Solo**: a solo player starts with **1 self-revive**. A REVIVE TOKEN from
  the shop adds one (max 3). A player with a token gets up by themselves after
  3 s. Tokens also work in co-op.
* **GAME OVER** when no player is standing and nobody can self-revive.

Rationale for the solo rule: Gun Mayhem is about knockback and ring-outs, so
deaths happen more often than in a shooter. A full respawn every wave would
make solo play meaningless. One token, plus tokens you can buy, keeps solo
fair without removing the stakes.

### Points (per player)

| Event | Points |
| --- | --- |
| start | 500 |
| bullet / blast hit on a zombie | +10 per hit |
| kill | kind's kill points (60-160, boss 1000) |
| headshot kill | +50 |
| ring-out kill (pushed off the map within 5 s of your hit) | +40 |
| revive a teammate | +150 |
| wave cleared (standing players) | +100 |

Points are also added to `Player.Score`. The results screen shows the points
each player earned, kills, headshots, downs and revives.

### Shop (between waves)

The shop opens automatically at the start of the break, with one panel per
local human. The panels are laid out as 1 big, 2 side by side, or a 2x2 grid.
Keys:

| Key | Action |
| --- | --- |
| UP / DOWN | pick an item |
| LEFT / RIGHT | switch the tab |
| SHOOT | buy |
| GRENADE | close the shop (READY) |
| ABILITY (during the break) | reopen the shop |
| mouse | click a tab, an item or READY |
| ESC (online guests) | close the shop |

While a shop is open, that player's game input is masked.

**Guns.** Each gun comes with spare magazines: ceil(90 / magazine), between 2
and 8. An empty magazine reloads from the spares. The source rule is that an
empty gun falls back to the starter pistol; a spare magazine re-equips the gun.
The rest of the arsenal is only in the mystery box.

| Gun | Price |
| --- | --- |
| FIFTY EIGHT | 350 |
| SNAKE | 500 |
| SHORT SMG | 800 |
| SAWED OFF SHOTGUN | 900 |
| MAFIA SMG | 1000 |
| COOL SNIPER | 1000 |
| TACTICAL SMG | 1100 |
| COMPACT RIFLE | 1100 |
| CLASSIC ASSAULT RIFLE | 1200 |
| STREET SWEEPER | 1400 |
| GUERILLA ASSAULT RIFLE | 1500 |
| HI-POWER RIFLE | 1600 |
| MILITARY SNIPER | 1800 |
| CLASSIC MACHINE GUN | 2000 |
| MILITARY MACHINE GUN | 2400 |
| grenade launcher (developed) | 2500 |
| MINIGUN | 3000 |

**Throwables.** Each purchase adds 3, up to 9. Buying another type replaces
yours.

| Throwable | Price |
| --- | --- |
| SMOKE | 200 |
| DYNAMITE | 300 |
| US MK 2 | 300 |
| RGD-5 | 300 |
| FLASHBANG | 300 |
| SOVIET F-1 | 350 |
| SHOCK | 350 |
| TEAR GAS | 350 |
| GERMAN M24 | 400 |
| MOLOTOV | 450 |

**Gear.** These are the existing abilities; a new one replaces yours.

| Gear | Price |
| --- | --- |
| FLASH GOGGLES | 400 |
| THERMAL GOGGLES | 400 |
| LIGHT | 500 |
| GAS MASK | 500 |
| DASH | 750 |
| TRIPLE JUMP | 750 |
| GRENADIER | 800 |
| ANCHOR | 900 |
| HEAVY | 1000 |
| KEVLAR | 1000 |
| SHIELD | 1250 |

**Supplies.**

| Item | Price | Effect |
| --- | --- | --- |
| MEDKIT | 300 | full HP |
| AMMO REFILL | half the gun's price (min 150) | refills all spare magazines and the current one |
| REVIVE TOKEN | 1500 | self-revive, max 3 |

Price guide: wave 1 earns about 500 points per player, and waves 5-8 earn about
1000-1500. A good SMG or rifle is affordable after 2-3 waves, the machine guns
around wave 6-8, and the minigun later. All purchase rules are in one function,
`zombieCheckBuy`:

* only during the break;
* only while your shop is open;
* not while you are downed;
* not without enough points;
* not past a cap (FULL / OWNED).

### Mystery box

* Press DOWN next to the box. Near the box, DOWN does not drop you through the
  platform.
* A roll costs 950. It rolls for 2.5 s, cycling weapon icons, then offers a
  random weapon for 6 s.
* Only the player who rolled can take the weapon (DOWN again). The weapon
  comes with spare magazines.
* The pool is every non-pistol weapon of the arsenal, including the developed
  ones, but not the Portal Gun.
* After 4-8 rolls, a roll comes up "BYE BYE". The 950 is refunded, and the box
  moves to another spot of the zone after 3 s.

### AI teammates

A slot set to BOT plays as a normal AI bot. In a zombie match,
`aiTargetPool()` also gives the bots the zombies' bodies as targets. At the
start of a break a bot buys the best gun it can afford that beats its current
one, then a medkit and ammo, and it is READY at once. Bots also count as
revivers.

## Arenas

| Arena | Map | Notes |
| --- | --- | --- |
| MIDNIGHT WOOD | GM1 map 9 | nine ledges; spawns at the outer ledges and the treetops |
| DESERT DESTRUCTION | GM1 map 7 | long floors, side decks, a middle bridge |

Both arenas are tested so that every platform can be reached from every spawn
point by both mobility classes (`TestZombieNavReachesEveryPlatform`).

## AI and navigation

`buildZombieNav` turns the platforms into a graph and precomputes all-pairs
next hops with Floyd-Warshall. A map has fewer than 20 platforms. The links
are:

| Link | When | How |
| --- | --- | --- |
| walk | same height, touching | walk |
| hop | a gap at the same height or slightly higher, or a lower platform too far for a plain fall | jump at the edge, double jump if still below the target |
| climb | higher and overlapping | stand under the overlap and jump up through the one-way platform, double jump at the top of the arc, stay over the overlap |
| drop | lower and overlapping | press DOWN (only above `LowestY`) |
| fall | lower, past an edge, within the drift of the fall | walk off and steer onto it |

There are two mobility classes, each with its own graph. Heavy zombies
(brute, boss) cannot hop gaps over 60 px and drift less when they fall.

Zombies press the same AI keys as bots, and `Player.UpdateWithFire` moves
them. Knockback, burning panic, tear gas, flashbang blindness and smoke
therefore work like for players.

Anti-stuck works on two levels:

1. A zombie that stays inside a 22 px box for 2 s while it wants to move
   jumps and turns around.
2. A zombie whose path distance to its target has not improved for 20 s climbs
   out again at a spawn point. `zombieMatch.stuckRespawns` counts these
   re-spawns, and the tests require 0.

## Online co-op (protocol 22)

* The host runs everything. A snapshot carries the following:

  | Field | Contents | Cap |
  | --- | --- | --- |
  | `Zombies` | compact; the physics body and AI state are unexported and never sent | 64, dying zombies listed first so they are trimmed first |
  | `ZombiePops` | points popups | 32 |
  | `ZombieHUD` | waves, per-player HP, points, shop state, the box | one value |
  | `OnlineZombie`, `ZombieSetupMode`, `ZombieSetupArena` | lobby | |

  A zombie costs under 64 raw bytes before DEFLATE.
* Guests buy through `netMessageZombie` (`ZombieReqs`). Every message repeats
  all requests the host has not acknowledged yet. The ack is
  `ZombiePlayerState.ReqAck`. A lost message never loses a purchase, and a
  repeated request is never applied twice.
* `validateNetZombieRequests` checks the structure on the reader goroutine
  (`netwire_secure.go` style). The host applies the requests on the game
  thread with the same `zombieCheckBuy` as local purchases.
* While a guest's shop is open, the guest sends empty input. The host masks
  that input too.
* Players can join a running match. They start with 500 points.
* The host can start a zombie session alone. Use HOST ONLINE in the zombie
  setup.
* The MATCH and MAPS tabs of a zombie lobby show the zombie settings. Guests
  see them read-only.
* The server browser rules line says "ZOMBIE CO-OP - mode - arena".
* Render interpolation covers zombies (`render_interp.go`). Zombies are
  matched by ID. Rising and dead zombies, and teleports (anti-stuck
  re-spawns), snap.

## Performance

48 living zombies cost about 0.7 ms per tick on the dev machine
(`TestZombieHordePerformance`; the budget at 30 TPS is 33 ms). Every zombie
pose is drawn once into a cached image (`zombieSpriteFor`), so drawing a horde
costs one image draw per zombie.

## How to add a zombie kind

1. Add a constant before `zombieKindCount` in `zombie_defs.go`. Only append:
   the number is sent in snapshots.
2. Add its `zombieStats` entry: HP, move, jump, knockback, scale, melee, kill
   points, and the `MinWave` / weight curve. A kind with weight 0 never spawns
   by itself.
3. Add a colour palette (`zombiePalettes`) and, if it needs one, a silhouette
   branch in `drawZombieFigure`. Poses are cached per kind automatically.
4. Special behaviour: add a branch in `updateZombie` (state switch) like
   `zombieTryLeap` or `zombieBossSpecial`. New states go after
   `zombieStateSlam`. If the kind is slow or heavy, return `zombieNavHeavy`
   from `zombieNavClass`.
5. If the kind's art does not match the 0.8-scale player hitbox, look at
   `zombieHeadFraction`.
6. Add the kind to `TestZombieAIReachesPlayersOnEveryArena`.

## Step 2: several zones connected by doors (design)

The data structures are already in place:

* `zombieArenaDef.Zones` lists the zones of an arena. Each zone has its own
  `MapNumber`, `Spawns` and `BoxSpots`. `StartZone` is the zone the match
  starts in.
* `zombieArenaDef.Doors` lists `zombieDoorDef` values. Each door has a
  position in its zone, a target zone and position, a cost, and an optional
  `OpenFrom` wave.
* Every runtime entity carries its zone: `Zombie.Zone`,
  `ZombiePlayerState.Zone` and `ZombieBoxState.Zone`.
* Navigation graphs are cached per zone and class: `zombieMatch.navFor(zone,
  arena, class)`.

Plan for step 2:

1. **Door state**: add `Doors []ZombieDoorState` (open, bought by) to
   `ZombieHUD`. Buy a door with DOWN at the door, like the box, through the
   same `zombieUseBox`-style interaction and a `netZombieReq` action for
   guests.
2. **Several maps at once**: the simulation needs one `Map` per zone.
   `Game.arena` stays the zone shown on the shared camera. Zombie and player
   physics use `zoneMap(zone)` instead of `g.arena`, and every `hitTargets` /
   bullet loop filters by zone. Bullets and grenades get a `Zone` field.
3. **Players passing a door**:
   * Local shared screen: all living players must stand at the door, which
     shows a "2/3 at the door" counter. Then everybody moves to the target
     zone and `Game.arena` switches. Downed or dead players move with them.
   * Online: each player passes alone. Each guest renders its own zone. The
     snapshot already carries per-entity zones, and the guest camera centres
     on its own player. The host runs every occupied zone.
4. **Zombies**: an open door becomes a nav link between zones: walk to the
   door, then teleport to the target position. The target platform for
   pathing is the door when the target player is in another zone. Spawning
   picks spawn points only in zones with players, plus their neighbours.
5. **The box** moves between zones when it relocates (`ZombieBoxState.Zone`).

## Tests

All tests are in `zombie_test.go`.

| Area | Tests |
| --- | --- |
| Waves | wave scaling: counts, HP, speed, caps, boss levels, type mix, co-op |
| AI | nav reachability per arena and class; every kind from every spawn point reaches a player on every platform of both arenas without anti-stuck re-spawns; boss slam and summons |
| Damage | damage and points; headshots; brute knockback resistance; ring-out bonus; exploder chain and fuse; melee |
| Players | down, revive, revive interruption, bleed-out, game over; ring-out down position; solo self-revive; wave-end revive and respawn |
| Shop | rules and caps; catalog validity; spare magazines; mystery box including relocation; DOWN at the box without a drop-through; bots shop and target zombies |
| Online | request validation and wire round trip; host-side purchase validation (points, phase, replays, other slots); guest request queue bound; snapshot caps, trim order, encode / stream / decode / validate / apply, size per zombie |
| Rendering | render interpolation of zombies |
| Flow | full two-wave flow through `Game.Update`; survival victory and record save / load; no last-survivor win and no crates in zombie mode |
| Menus | setup, results, shop panel kit IDs and keys, online lobby settings |
| Performance | 48-zombie tick time |
