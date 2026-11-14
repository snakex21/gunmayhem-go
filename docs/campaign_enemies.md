# Campaign enemies, bosses and arena hazards (EXTRA campaign)

This note explains the non-player enemy framework (`internal/game/campaign_enemies*.go`),
the boss data (`extra_campaign_boss.go`) and the hazards (`extra_campaign_hazards.go`),
and how to add the next big boss (a hovering helicopter) on top of them.

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

## Next stage: a hovering helicopter boss (sketch)

1. Add the kind `campaignEnemyHelicopter`. Give its body a large hitbox (`PlayerScale`
   about 2.2) and `KnockbackScale` about 0.15, so it barely drifts. Do not apply
   gravity. Give it `HP` of about 60. Ring-out is impossible because the update clamps
   it to a flight lane, for example y = -40…60 just above the camera's top clamp, or
   beyond one arena edge.
2. Give `CampaignEnemy` an optional `Boss *extraCampaignBoss`. Phases then read `e.HP`
   instead of bot lives. Draw the bar with
   `drawBossBar(name, title, HP/MaxHP, ticks, phase, alive, y)`.
3. Rocket barrage: for each target point, append a `HazardMarker`, which is the
   dynamite-rain warning. After about 60 frames, append a real
   `Bullet{DevelopedProjectile: DevelopedProjectileStrike, OwnerID: body.ID, Team: 2}`
   above the marker. The existing strike projectile explodes on the ground.
4. Drops: use `spawnCampaignEnemy(extraCampaignUnit{Kind: campaignEnemyDrone or
   campaignEnemyShieldBearer}, side, arena)` and set the body position under the
   helicopter. Summoned units count toward the win automatically.
5. Off-map position: `drawCampaignEnemyArrows` already points at it. Add a laser or
   telegraph line in its draw function before each barrage.

## Hazards

`extraCampaignMission.Hazard` and `HazardInterval` (optional period override) are
data-driven:

* `DynamiteRain` shows a landing marker 60 frames before each drop.
* `ShockPulse` respects Blast Shield and the shield powerup (1/5 push).
* `Wind` alternates gusts, with a 1.5 s warning in the HUD.

Hazard pushes keep the kill credit of the last real attacker. Hazard grenades use
`OwnerID = extraHazardOwnerID` (-1), and `explodeGrenade` leaves `LastHitBy` untouched
for a negative owner.
