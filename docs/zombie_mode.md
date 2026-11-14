# ZOMBIE mode

Co-op survival against waves of zombies, local (1-4 players and AI bots on one
screen) or online (host-authoritative, every guest on its own screen). Step 1
is the single-arena mode; step 2 adds maps made of several zones linked by
doors that players buy with points ("Zones and doors" below).

Code: `internal/game/zombie_*.go`

| File | Contents |
| --- | --- |
| `zombie_defs.go` | All tunable data: zombie kinds, wave scaling, points, player rules, arenas (zones, spawn points, box spots, doors) |
| `zombie_mode.go` | Match flow (phases, waves), players (HP, down, revive), hit phase (damage, points), records |
| `zombie_ai.go` | Spawning, platform navigation graph, steering, melee, leaper / exploder / boss specials |
| `zombie_shop.go` | Shop catalog and purchase rule, bot shopping, mystery box |
| `zombie_shop_ui.go` | Shop panels (Mayhem UI kit), per-player keys |
| `zombie_net.go` | Online: snapshot lists, guest shop requests, validation |
| `zombie_zones.go` | Step 2: zone worlds, the per-zone tick, doors (purchase rule, gather, passing), zombie door routing, spawn zones, per-guest snapshot views |
| `zombie_zones_visuals.go` | Doors (barricade, gate, price sign, opening animation), gather prompt, zone name |
| `zombie_bots.go` | AI teammates walk to a downed teammate and revive them |
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

**Spawning**: zombies come out of the spawn points of a zone. On a multi-zone
map the zone is picked by weight: 6 per living player in an unlocked zone, 2
for an unlocked zone next to (one open door away from) a zone with players,
0 otherwise; the boss comes in the zone with the most players. Spawn points
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
  moves to another spot after 3 s. On a multi-zone map the new spot can be in
  any unlocked zone; only players in the box's zone can use it.

### AI teammates

A slot set to BOT plays as a normal AI bot. In a zombie match,
`aiTargetPool()` also gives the bots the zombies' bodies as targets. At the
start of a break a bot buys the best gun it can afford that beats its current
one, then a medkit and ammo, and it is READY at once. Bots also count as
revivers.

**Bots revive** (`zombie_bots.go`): a bot with at least 30 HP walks to the
nearest downed teammate of its zone (not one that gets up by itself) and
stands on the marker until the revive is done. It keeps aiming and shooting;
only its movement keys are taken over, along the zombie nav graph
(`navSteer`). It does not start when 3 or more zombies are within 150 px of
itself or of the marker, and gives up an ongoing revive at 5. Bots do not use
doors: on a shared screen they move with the team, online they follow a
human who leaves their zone.

## Arenas

| Arena | Map | Notes |
| --- | --- | --- |
| MIDNIGHT WOOD | GM1 map 9 | single arena; nine ledges; spawns at the outer ledges and the treetops |
| DESERT DESTRUCTION | GM1 map 7 | single arena; long floors, side decks, a middle bridge |
| OUTBREAK | GM1 maps 11, 12, 10, 6 | four zones: CITY (start) - SAVANNA - TUNDRA - PEAK |

Every arena and zone is tested so that every platform can be reached from
every spawn point by both mobility classes (`TestZombieNavReachesEveryPlatform`,
`TestZombieZonesDataAndNavigation`). The setup screen shows one tile per
arena; a multi-zone map shows a 2x2 mosaic of its zones and a "4 ZONES"
badge.

## AI and navigation

`buildZombieNav` turns the platforms into a graph and precomputes all-pairs
next hops with Floyd-Warshall. A map has fewer than 20 platforms. The links
are:

| Link | When | How |
| --- | --- | --- |
| walk | same height, touching | walk |
| hop | a gap at the same height or slightly higher, or a lower platform too far for a plain fall | jump at the edge, double jump if still below the target |
| climb | higher and overlapping | stand under the overlap and jump up through the one-way platform, double jump at the top of the arc, stay over the overlap |
| drop | lower and overlapping | press DOWN (only above `LowestY`) as soon as the fall, at the current speed, ends well inside the platform below; a fast body brakes first |
| fall | lower, past an edge, within the drift of the fall | walk off and steer onto it |
| door | (step 2) an open door, when the zombie's zone has no living player | walk to the door spot, come out at the other side |

There are two mobility classes, each with its own graph. Heavy zombies
(brute, boss) cannot hop gaps over 60 px and drift less when they fall.

Fast bodies (runners at 8 px/frame) need speed-aware steering, added in
step 2 for the new maps' tighter geometry: they brake before a climb launch
point near an edge, steer in the air toward the landing spot with their
braking distance in mind, turn back when a hop to a platform at their own
height overshoots it, and an edge guard brakes a grounded body that would
slide off where the path does not leave the platform.

Zombies press the same AI keys as bots, and `Player.UpdateWithFire` moves
them. Knockback, burning panic, tear gas, flashbang blindness and smoke
therefore work like for players.

Anti-stuck works on two levels:

1. A zombie that stays inside a 22 px box for 2 s while it wants to move
   jumps and turns around.
2. A zombie whose path distance to its target has not improved for 20 s climbs
   out again at a spawn point. `zombieMatch.stuckRespawns` counts these
   re-spawns, and the tests require 0.

## Online co-op (protocol 22, zones: 23)

* The host runs everything. A snapshot carries the following:

  | Field | Contents | Cap |
  | --- | --- | --- |
  | `Zombies` | compact; the physics body and AI state are unexported and never sent | 64, dying zombies listed first so they are trimmed first |
  | `ZombiePops` | points popups (with their zone) | 32 |
  | `ZombieHUD` | waves, per-player HP, points, shop state and zone, the box, doors, the gather, the view zone | one value |
  | `OnlineZombie`, `ZombieSetupMode`, `ZombieSetupArena` | lobby | |

  A zombie costs under 64 raw bytes before DEFLATE.
* Guests buy through `netMessageZombie` (`ZombieReqs`). Every message repeats
  all requests the host has not acknowledged yet. The ack is
  `ZombiePlayerState.ReqAck`. A lost message never loses a purchase, and a
  repeated request is never applied twice.
* `validateNetZombieRequests` checks the structure on the reader goroutine
  (`netwire_secure.go` style). The host applies the requests on the game
  thread with the same `zombieCheckBuy` as local purchases.
* Doors and the mystery box are used with DOWN, so a guest's door purchase
  reaches the host as the guest's input: the host runs `zombieCheckDoor` on
  the guest's player (in reach of the door in its own zone, alive, points,
  wave lock). The guest's prediction does not drop through the platform at a
  door or at the box (`netPredictEnv.noDrop`).
* While a guest's shop is open, the guest sends empty input. The host masks
  that input too.
* Players can join a running match. They start with 500 points.
* The host can start a zombie session alone. Use HOST ONLINE in the zombie
  setup.
* The MATCH and MAPS tabs of a zombie lobby show the zombie settings. Guests
  see them read-only.
* The server browser rules line says "ZOMBIE CO-OP - mode - arena", and the
  browser shows a green ZOMBIE badge with a zombie head on such sessions
  (detected from that rules line, so the discovery formats did not change).
* Render interpolation covers zombies (`render_interp.go`). Zombies are
  matched by ID. Rising and dead zombies, and teleports (anti-stuck
  re-spawns), snap.
* Multi-zone maps: see "Online views" below.

## Performance

48 living zombies cost about 0.7 ms per tick on the dev machine
(`TestZombieHordePerformance`; the budget at 30 TPS is 33 ms). Spread over
the four zones of OUTBREAK with two players, the multi-zone tick takes about
0.3 ms (`TestZombieMultiZoneTickPerformance`). Every zombie
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

## Zones and doors (step 2)

### Rules

* A multi-zone map (`zombieArenaDef.Zones` with more than one zone) starts in
  `StartZone`. Every other zone is locked behind a door.
* A door (`zombieDoorDef`) joins a spot of one zone (`Zone`, `At`) to a spot
  of another (`ToZone`, `To`). It is bought once, by one player, for `Cost`
  points, and then works both ways for players and zombies. `OpenFrom` can
  lock a door until a wave. Prices grow with depth: OUTBREAK's doors cost
  750, 1250 and 2000.
* Press DOWN at a door (within 46 x 30 px of its spot). DOWN at a door never
  drops you through the platform. `zombieCheckDoor` is the one rule: alive,
  standing at the door in your zone, not already open, wave lock, points.
  Doors can be bought during waves and breaks.
* Opening a door unlocks the zone behind it: its spawn points join the wave
  spawns and its box spots join the mystery box spots.
* **Shared local screen** (one camera): DOWN at an open door starts a gather.
  The prompt "ALL PLAYERS TO THE DOOR k/n" shows how many living humans
  stand at the door; when all of them have stood there for 1.5 s, the whole
  team moves to the other side and the camera with them. Downed and dead
  players and the bots come along (their markers move to the door). The
  gather ends when nobody is at the door or after 20 s.
* **Online**: each player has an own view and passes alone, at once. A bot
  left in a zone without humans follows the human who left. The HUD card of
  a teammate in another zone shows "IN <ZONE>". The host's screen shows the
  host player's zone.
* Arriving players get 1 s of invulnerability.

Design choice: zones are separate arenas (each an existing GM1 map with its
own art, platforms and death bounds) linked by door transitions, not one big
stitched map. The source physics, camera and death bounds assume one
900 x 600 arena, so stitching maps would have meant rewriting them; separate
worlds keep every existing code path unchanged.

### Zone worlds

The simulation keeps one world per zone (`zombieZoneWorld`): the map, the
camera and every entity list of `Game` (bullets, grenades, fire, gas, blasts,
effects, crates, pickups). The `Game` fields hold the resident zone; the other
zones' lists wait in `zombieMatch.worlds`. `zombieLoadZone` swaps them.

A multi-zone tick (`runZombieZoneTick`) does this:

1. Count the living players per zone and route every zone without players
   to the door toward the nearest player (`refreshZoneInfo`, a breadth-first
   search over the open doors).
2. Spawn the wave (once per tick, into the picked zone).
3. Bucket the players and zombies by their zone, then for every unlocked
   zone: load the zone, put its players into `Game.players` and its zombies
   into `zombieMatch.zombies`, and run the ordinary gameplay step
   (`simulateArenaTick`, extracted from `Game.Update`) and the player rules
   (HP, down, revive). The zone shown on this screen runs last as the primary
   pass; HUD timers, the kill feed and map FX advance only there. Sounds of
   other zones are not played locally (`Game.sfxLocalMute`) but are sent to
   the guests in that zone.
4. Put the lists back together and load the view zone again.

So every weapon, throwable, AI, knockback window and hit test works inside a
zone without a zone check, and nothing crosses a zone border except through a
door. A body that goes through a door during one zone's pass is not run again
in the same tick. Between ticks the view zone is resident, so `Draw`, render
interpolation and the snapshot code see exactly one zone. Drawing hides the
players, zombies, popups and markers of other zones.

Single-zone arenas never use any of this: their tick is exactly the step-1
tick.

### Zombies and doors

A zombie in a zone without a living player walks to the door that leads
toward the nearest player and comes out at the other side, spread by up to
26 px (`zombieThroughDoor`). With players in its zone it hunts them as
before. Closed doors block both zombies and players. Zombies are never lost
or duplicated at a door (`TestZombieZonesDoorPathing` checks this every
frame).

### Online views

The host builds one snapshot per zone that a guest is in
(`zombieNetViews`): the zone's map, camera, entity lists, its zombies and
popups, and every player (at most four; a guest draws only the players of its
zone). `netHostEncoder.encodeViews` encodes each view once per tick and each
peer's writer sends the view of its own zone (`netHostFrame.peerView`). The
map goes along only when it changes for that peer (a small cache of the zone
maps). SFX events carry their zone (host only, `net:"-"`) and reach only the
guests of that zone. So a snapshot is bounded by one zone's lists and the
same `netSnapshotListCaps`, however many zones a map has.

`ZombieHUD.View` tells the guest which zone the snapshot shows. The guest's
playout never blends two snapshots of different zones, and the render
interpolation snaps when the view zone changes (`renderInterpState.zone`),
when a zombie came through a door (`Zombie.DoorPasses`) and when a player
changed zone.

### How to add a multi-zone map

1. Add a `zombieArenaDef` with several `zombieZoneDef` (IDs 0..n-1, at most
   8, each with its own GM1 map number: render interpolation and the guest's
   map cache key on it), spawn points and box spots on platform tops.
2. Add `zombieDoorDef` values (at most 8) with spots on platform tops, ideally
   on the floor. Make deeper doors dearer (`TestZombieZonesDataAndNavigation`).
3. Run the zone tests: every spawn must reach every platform and every door
   spot in both nav classes, and zombies of every zone must reach players in
   every other zone.

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
| Zones (`zombie_zones_test.go`) | zone data, prices by depth, nav in every zone; door purchase rules (local, guest validated by the host, wave lock, other zones); shared-screen gather and team move with dead players and bots; gather cancel; independent zones online (zombies, bullets, per-zone snapshot views through the wire); per-zone SFX; no stuck or fallen zombies inside every zone; door pathing across all zones in both directions without losses or duplicates; closed doors; spawn weighting; box across unlocked zones; interpolation snaps; draw hides other zones; bots revive (and not into a crowd); browser badge; multi-zone tick time |
| Rendering | render interpolation of zombies |
| Flow | full two-wave flow through `Game.Update`; survival victory and record save / load; no last-survivor win and no crates in zombie mode |
| Menus | setup, results, shop panel kit IDs and keys, online lobby settings |
| Performance | 48-zombie tick time |
