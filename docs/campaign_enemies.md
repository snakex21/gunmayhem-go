# Campaign enemies, bosses and arena hazards (EXTRA campaign)

This note explains the non-player enemy framework (`internal/game/campaign_enemies*.go`),
the boss data (`extra_campaign_boss.go`), the hazards (`extra_campaign_hazards.go`),
and the Stage B big enemies: the mech walker and the gunship boss.

## Non-player enemies

`CampaignEnemy` is a campaign-only enemy, such as a drone or a shield-bearer. It is **not** an entry of
`Game.players`. The camera, the win checks, the HUD cards, the scoreboards and bot
targeting all loop over that slice and never see these enemies.

Each enemy owns a private `Body Player`, which is used only as a physics and hit receiver:

* `Game.hitTargets()` returns `g.players` plus every living body. `Game.Update` passes
  that list to `updateBullets`, `updateGrenadesWithAreas`, `updateFirePatches`,
  `updateAreaEffects` and the arena hazards, and wraps each step in the ability
  knockback windows. This gives enemies, through the same code as players:
  * knockback from every source,
  * `KnockbackScale` resistance (applied inside the windows),
  * kill credit through `LastHitBy` and `OwnerID`,
  * blindness, tear gas and burning.
* Bodies use IDs `campaignEnemyIDBase` (100) and up, on team `campaignEnemyTeam` (2).
  Enemy bullets therefore never hit enemy bots, and player bullets hit both.
* Enemies fire their real weapon through `fireCampaignEnemyWeapon` → `updatePlayerWeapon`.
  This handles ammo, reload, sound, credit, and aim disturbance from blindness and smoke.
* Target choice uses `aiCanDetect`, the same smoke and blindness rules as bots.
* Death:
  * A ring-out kills the enemy. Walkers use `Player.Kill` inside `UpdateWithFire`.
    Flyers use `campaignEnemyOutOfBounds`.
  * HP kills the enemy when `HP > 0`. `endCampaignEnemyHitPhase` turns hit counts and
    large impulses into damage.
  * `finishCampaignEnemy` credits the killer and adds the kill feed line and the FX.
* Win condition: `campaignEnemiesRemaining()` counts living enemies plus queued wave
  reinforcements. The EXTRA side check in `updateMatchInteractions` counts them as
  enemies still standing.
* Netcode: the snapshot carries `CampaignEnemies` and `HazardMarkers`, and both are
  registered in `netSnapshotListCaps`. The campaign is local-only today. With these
  fields an online co-op campaign would only need to start the mission on the host.
* Off-screen indicators reuse the player arrow (`drawCampaignEnemyArrows`).

### Adding a kind

1. Add a `campaignEnemyKind` constant and its defaults: `defaultName`, `defaultGun` and
   `defaultShotDelay`.
2. Set up the body in `spawnCampaignEnemy`: hitbox scale, `KnockbackScale` and the spawn point.
3. Write `updateX(e, arena)` and dispatch to it from `updateCampaignEnemies`. You can
   reuse `Player.UpdateWithFire` (walker) or integrate the motion yourself (flyer).
4. Draw it in `drawCampaignEnemies`.
5. Spawn it from mission data:

   ```go
   Units: []extraCampaignUnit{{Kind: ..., Count: N, MaxActive: M, HP: hp, Delay: d, Interval: i}}
   ```

## Bosses

A bot becomes a boss with `xe(...).asBoss(&extraCampaignBoss{...})`:

* `KnockbackScale` is the resistance. It is applied in the knockback windows, so it
  covers bullets, blasts, gas, fire and hazards.
* `Phases` are listed from first to last. Each phase starts when the boss has
  `Lives` lives left or fewer. A phase can:
  * swap the gun,
  * change the `ShotDelay` fire pace,
  * change move speed or resistance,
  * summon drones or shield-bearers,
  * show a banner.

  If the boss loses several lives at once, any phases it skipped still run, in order.
* The boss HUD bar (`drawBossBar`) shows name, title, remaining lives and phase ticks.

`bossPhaseFor(remaining, phases)` and `drawBossBar` take plain numbers, so they work
for an HP-based `CampaignEnemy` as well.

## Stage B: the mech and the gunship

Code: `campaign_enemies_big.go` (shared rules), `campaign_enemy_mech.go`,
`campaign_boss_heli.go`, `campaign_enemies_visuals_big.go` (procedural art and
boss bars). Tests: `campaign_stageb_test.go`. Missions 26-28 in `extra_campaign.go`.

### Shared rules for big enemies

* **Damage = knockback.** A big enemy loses 1 HP for every 20 of *raw* push, which is
  the push before its own `KnockbackScale` (`bigEnemyPushPerHP`). Some examples:
  a starter pistol round is 1 HP, a rifle round 1.1, a sniper round 3, a bat swing
  1.3, and a point-blank rocket about 6. Fractions carry over between hits. Leaf
  blower wind is not damage.
* The de-materializer cannot erase a big enemy. Its beam burns 10 HP instead.
* Co-op: each extra human adds 60% of the base HP (`campaignCoopHP`).
* Hit-phase hooks (`Game.Update`, before `updateBullets`):
  `updateCampaignBossBullets` stops gunship rounds on platforms (cover works). It
  also turns a batted or blown gunship rocket into a homing return shot.
* Death sequences: a mech melts down for 40 ticks and a gunship crashes for up to
  80 ticks. During that time `hittable()` is false, but the enemy still counts in
  `campaignEnemiesRemaining`, so the win waits for the explosion. The kill goes to
  the player whose hit brought the HP to zero. A mech kill scores 400 and a gunship
  kill scores 1000. The wreck debris is drawn for 60 ticks.
* Marked strikes (`queueCampaignStrike`): a `HazardMarker` appears at once and a
  HIMARS-type strike falls when the warning runs out. The shell lands on the floor
  under the target. It starts just below the next platform above that floor, so on
  a tiered map it does not hit the roof.
* `Player.hitW` / `hitH` (unexported, host only) give a big body its own hitbox.

### MECH (`campaignEnemyMech`)

| Property | Value |
|---|---|
| Hitbox | 70 x 120 px. The art is drawn at 1.25x and fills the box. |
| Resistance | `KnockbackScale` 0.12: a pistol round slides it about 34 px |
| Movement | Moves at 0.34 of normal speed and jumps at 0.92. It uses the heavy zombie platform graph (`zombieNavHeavy`): hops of at most 60 px, and it climbs and drops. |
| Weak spot | The exposed reactor on its back. A hit from an attacker standing behind it (or, with no known attacker, a push along its facing) is multiplied by 2.5, and so is the damage, with sparks. A grenade thrown from the front that lands behind it does not count, and neither does a harpoon reel. The mech turns in 26 ticks, which gives you a window to flank it. |
| Ring-out | It can still be rung out (normal walker bounds). |
| Twin MG | A red laser shows at chest height for 26 ticks, then a burst of 6 SMG rounds (gun 21, firepower x0.85) alternates between two barrels. Jump over the burst. The rounds stop on platforms; a jammed gun (reload, blindness) leaves the fire state after the burst time. |
| Stomp | The target must be within 140 px. A leg rises and an orange ring grows for 30 ticks. Then everyone standing on that surface within 175 px is pushed away: 12.5 to 6.25 push plus 5.5 lift (x0.4 with BLAST SHIELD or the shield powerup). Anyone in the air is safe. Recovery takes 26 ticks. |
| Mortar | `Rockets` units only. The pod glows for 32 ticks, then 2 shells are marked at target x ±46 and land 50 ticks later. Cooldown 210 ticks. |
| Harpoon | A fresh hook jams a leg and cancels the current wind-up. The pull itself is x0.12. |
| Stuck | It turns toward where its route goes (a way up often starts away from the target), toward the target only on the same platform or with no route step. If it walks without moving for 75 ticks, it hops and re-plans its path. |

### GUNSHIP (`campaignEnemyHelicopter`)

It flies its own motion: no gravity, no platforms and no ring-out. It stays inside
the arena x range ±60 and cannot go below the bottom platform.

* **Hitbox** 128 x 54 px. `KnockbackScale` 0.15.
* **Hover**: the hull bottom stays 175 px above the target's feet. A double-jump shot
  reaches it. It changes sides of the target between attacks.
* **How players hurt it** (all of this is in the briefing and on the HUD):
  1. During a strafing run it is only 112 px above the platform, so a single jump
     puts your muzzle on the hull.
  2. Its rockets can be sent back. The BASEBALL BAT and the LEAF BLOWER reflect
     rockets. A reflected gunship rocket homes on it at 13 px/tick and hits it for
     7 HP.
  3. The HARPOON drags it down. For 90 ticks it is held beside the shooter (150 px
     to the side, hull 30 px above the shooter's feet), within reach of guns and
     bats. The hook breaks off any attack: rockets of a salvo that have not left
     the pods are called off and their markers removed (rockets already in the
     air keep theirs). After the drag it ignores hooks for 120 ticks.
  4. Explosives that hit the hull count their full push.
* **Phases** (boss bar ticks at 2/3 and 1/3 HP, with a banner at each phase):
  1. **Strafing runs.** A red band, dashes and chevrons cover 260 px each side of the
     target's platform for 50 ticks. Then it makes a pass at 7 px/tick, firing a
     round down along the band every 3 ticks: push 7, along the run. The rounds
     stop on platforms.
  2. **Adds rocket salvos.** 3 rockets. Every impact point is marked the moment the
     salvo is planned, 40 ticks before the first rocket leaves the pods. Rockets fly
     at 7 px/tick in a straight line (radius 120, power 26). The impact point is the
     first platform on that line.
  3. **Drops troops.** It drops a shield-bearer, then a drone, in turn. At most 3 of
     its adds are alive at once and 6 in total. It gives up a drop point it
     cannot reach within 150 ticks. Salvos grow to 4 rockets and
     strafing runs speed up to 8.5 px/tick.
* At 0 HP, the "GOING DOWN!" banner shows. It spins down trailing smoke and small
  explosions, then explodes when it hits a platform or after 80 ticks.

### Missions (EXTRA 26-28)

| # | Title | Map | Enemies |
|---|---|---|---|
| 26 | Iron Stride | 1 NO NAME | Elite mech "Strider" (HP 70, boss bar), a Spotter bot (3 lives), 2 drones later |
| 27 | Steel Rain | 7 DESERT DESTRUCTION | 2 mortar mechs "Howitzer" (HP 55, one at a time), 3 shield-bearers |
| 28 | Gunship | 5 GREAT WALL BRAWL | Gunship "Black Kite" (HP 90). The crates are the anti-air kit (`CratePool`): bats, blower, harpoons, RPG-7, M72 LAW, sniper, AK, P90. |

Old saves unlock mission 26 once mission 25 is completed (`normalizeExtraCampaignLevels`).
The EXTRA campaign has no checkpoints.

Co-op: two local humans are a team game. Before Stage B, the generic "all active
players on one team" check ended a bot-less EXTRA mission at once. The EXTRA check
by side now decides alone (`screen_flow.go`). There are no AI teammates in the
campaign. Mission bots (team 2) never target enemies, and the enemies' bullets
and stomps skip team 2. Explosions (rockets, mortar) push everyone, including the
gunship's own drones.

### Online (protocol 25)

The campaign is still local-only. The snapshot carries everything a client needs
to draw it:

* New `CampaignEnemy` fields: `Phase`, `Elite`, `Title`, `AimX0/AimX1/AimY`
  (telegraphs), `Hooked` and `Mortar`.
* The snapshot also carries `BossBanner` and `BossBannerFrame`.
* Zombies carry `HarpoonTether` and `HarpoonTetherID`, so the rope is drawn on
  clients.
* Host-only data stays unexported: damage carry, cooldowns, the salvo plan,
  the hooker, the nav state and `Player.hitW/hitH`.
* No new snapshot list. Mechs, the gunship and its adds all use `CampaignEnemies`
  (cap 32). A gunship can have at most 3 adds alive, so it stays far below the cap.
* `validateCampaignSnapshot` bounds the list length (64), the kind, HP, phase and the state per kind, the hook, the
  names, finite positions, the body's arsenal2 fields, and the banner.
* Render interpolation already covers every `CampaignEnemy` body, matched by ID. A
  new enemy snaps (tested).

Performance: the finale runs through `Game.Update`. One tick averages about 0.2 ms
(`TestStageBFinalePerformance`).

## Hazards

`extraCampaignMission.Hazard` and `HazardInterval` (optional period override) are
data-driven:

* `DynamiteRain` shows a landing marker 60 frames before each drop.
* `ShockPulse` respects Blast Shield and the shield powerup (1/5 push).
* `Wind` alternates gusts, with a 1.5 s warning in the HUD.

Hazard pushes keep the kill credit of the last real attacker. Hazard grenades use
`OwnerID = extraHazardOwnerID` (-1), and `explodeGrenade` leaves `LastHitBy` untouched
for a negative owner.
