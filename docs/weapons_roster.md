# Weapons roster: second added arsenal and near-duplicate review

Code: `internal/game/arsenal2_defs.go` (numbers, rules), `arsenal2_combat.go`
(gameplay), `arsenal2_visuals.go` (procedural art), `arsenal2_ai.go` (bots),
`developed_weapon_identity.go` (near-duplicate tweaks). Tests:
`arsenal2_test.go`, `developed_weapon_identity_test.go`.

Units: the game runs at 30 ticks per second. Damage is knockback: one impulse
`F` slides a target about `F / (1 - 0.93) = 14.3 F` px. "Push per tick" is
the sustained push of a held weapon (a starter pistol sustains about 1.0-1.6,
an assault rifle about 5.5; walking against a push gives 0.64).

## The package

| Item | Kind | Numbers | Notes |
|---|---|---|---|
| HARPOON GUN (179) | crate weapon, two-handed | 3 hooks, fire delay 40, direct push 3, recoil 2.5, bolt speed 22 (flat) | A hit **pulls** the victim toward the shooter: impulse `(distance - 70) * 0.07`, clamped 7..23 (it stops about 70 px from the shooter; 23 is about 330 px), plus a 4.5 hop (more when the shooter stands higher, up to +6). The rope is drawn in code and is cut when the shooter dies. The reel scales with the hook's firepower (DOUBLE KNOCKBACK, campaign firepower). The bolt stops on platforms, so it never hooks through level geometry. ANCHOR blocks the pull, HEAVY / LIGHT scale it, the one-hit SHIELD absorbs it, the shield powerup bubble leaves 1/3. It hooks a BALLISTIC SHIELD (and the campaign shield-bearer's plate) and reels the bearer in: the harpoon is the shield counter. Replaces the old plain-bolt harpoon (1 round, push 78). |
| LEAF BLOWER (232) | crate weapon, two-handed | 150 ticks of fuel (5 s), cone 200 px long, about 23 degrees half-angle | Hold: per-tick push 1.25 at the nozzle, falling to 0.45 at the end of the cone, plus 0.10 lift. Wind is not a "hit" (no hit counter, below the one-hit SHIELD threshold) but it credits a ring-out. Throwables in the cone get +1.1 (x 0.4..1) horizontal per tick and float; launcher grenades too; incoming rockets (RPG, PIAT, Javelin, ...) are **turned around** and become the blower's. Zombie mode: it pushes zombies but deals no damage (crowd control); campaign enemies are pushed without a hit flash or HP loss. The campaign shield-bearer's plate halves it. Never through a platform (line of sight from the nozzle). |
| BASEBALL BAT (233) | starter option and crate pickup, one-handed melee | push 26 plus a 4.5 pop-up, one swing every 15 ticks (2/s), reach 66 px in front, 80 px tall | DOUBLE KNOCKBACK doubles it (also the blower and the harpoon reel). Hits everyone in reach once per swing (1.7 push per tick sustained: the top of the starter band, but only up close). A starter bat never breaks; a crate bat lasts 20 swings. Bats grenades back (speed 15, lift 7, the grenade becomes the batter's) and reflects rockets in reach. Into a raised BALLISTIC SHIELD: 50%; into the campaign shield-bearer's plate: 50% and no hit. Never through a platform. Zombie mode: +16 bonus damage per hit (a wave-1 walker, 40 HP, drops in one swing). No muzzle flash. |
| BALLISTIC SHIELD | gear (ability 13) and crate powerup (kind 8) | front bullets 22%, run x0.90, jump x0.93 | Always raised, no key. In front with one-handed weapons (pistols, the bat), **on the back** with every two-handed weapon and during AKIMBO; it comes back to the front when the player is on a pistol or the bat again. A covered hit is mostly horizontal (steeper than about 50 degrees from above is not covered) and comes at the shield's side. Bullets, pellets, bolts and flares (no ignition) are covered; the de-materializer's instagib kill, explosions, fire, gas, the blower and the harpoon are not. Zombie claws into the shield: half damage, 40% push. Drawn in code like the campaign shield-bearer's plate; it does not replace the hat. The crate pickup lasts until death. |
| AKIMBO | crate powerup (kind 7) | 360 ticks (12 s) | Dual handguns: the fire delay is halved (rounded up), shots alternate between the two guns, and the off-hand gun has its own magazine (so the reload count stays the same). A source handgun can fire its second gun while the first one's shot timeline plays (never during a reload); that shot ejects its own casing. A new gun always starts on the main hand. With any non-pistol (a two-handed pickup or the bat) the effect is **suspended while the timer keeps running**; back on a pistol it resumes if time is left. Both hands are busy, so a BALLISTIC SHIELD goes on the back. |
| THERMAL GOGGLES vs invisibility | existing gear | bots: 420 px (normally 135) | A thermal viewer (online: each guest for itself; local: when there is one human) sees invisible enemies as a faint orange heat **outline**, not a silhouette. On a shared local screen with several humans, a thermal wearer marks invisible enemies with a small heat ripple in the wearer's colour (like the smoke chevrons). Bots with thermal goggles detect invisible players at 420 px. |

The new powerups only spawn when the developed arsenal is in the match's crate
pool (Custom Game, EXTRA campaign, zombie mode): after the source roll, 2 in 9
spawns become AKIMBO or BALLISTIC SHIELD. The GM1 campaign keeps the exact
source roll.

### Bots

* Bat: approach to about 30 px (edge limit 90 px instead of 200), swing only
  when the target is inside the bat's reach, turn around on a target right
  behind.
* Blower: blow when a target is in the cone, and also at incoming enemy
  grenades and rockets even without a target.
* Harpoon: keep about 230 px, never hook under 120 px; the last hook only
  across a gap (a likely ring-out).
* A bot with a raised shield closes in to about 100 px and turns to face a
  projectile about to hit its unprotected back.
* Against a raised shield, a bot with a bullet weapon jumps over the bearer
  (only when there is ground behind it) and scores throwables higher ("throw
  over it").
* AKIMBO and shield powerups are picked up like any powerup.

### Zombie shop

| Item | Price | Tab |
|---|---|---|
| Baseball Bat | 450 | GUNS |
| Leaf Blower | 700 | GUNS (bots never buy it: no damage) |
| BALLISTIC SHIELD | 1100 | GEAR |

The harpoon is box-only (reeling zombies in is rarely wanted). AKIMBO is not
sold: a timed powerup would run out during the break.

### Online (protocol 24)

New synced `Player` fields: `AkimboTime`, `AkimboHand`, `ShieldGearPickup`,
`ShieldGearSpark`, `MeleeSwing`, `BlowerTime`, `HarpoonTether`,
`HarpoonTetherID`; host-only (`net:"-"`): `MeleeHitPending`, `BlowerPulse`.
`Bullet.ShooterID` (the rope anchor). New enum values: projectile kinds
`Harpoon` / `Gust`, ability 13, powerup kinds 7-8, weapons 232-233 (so the
`OnlineBlockedWeapons` / `OnlineBlockedAbilities` arrays grew). No new
snapshot list. `validateNetSnapshot` bounds every new field, the powerup kinds
and the projectile kinds. The rope, the swing and the gust are drawn from
synced state. The harpoon itself is an ordinary bullet, so render
interpolation covers it (a new harpoon snaps, then slides), and the reel is a
velocity change, so a reeled player keeps being interpolated.

## Weapon handedness (BALLISTIC SHIELD rule)

Source 8 (COOL PISTOL INFINITY) is one-handed, 9 (DE-MATERIALIZER) is
two-handed.

| Class | One-handed (shield stays in front) |
|---|---|
| Pistol | 1 Colt M1911A1, 2 Desert Eagle, 3 Glock 17, 4 Taurus Raging Bull M444, 5 FN Forty-Nine, 6 Colt Python, 98 FN Five-seveN, 99 Glock 18C, 100 Beretta 92FS, 101 SIG Sauer P226, 102 CZ 75, 103 H&K USP .45, 104 H&K Mark 23, 105 Walther P99, 106 Browning Hi-Power, 107 Tokarev TT-33, 108 Makarov PM, 109 Luger P08, 110 Mauser C96, 111 S&W Model 29, 112 Mateba 6 Unica, 190 Lancaster Howdah Pistol, 206 Colt Model 1903 Hammerless, 207 Frommer Stop, 216 Nambu Type 14, 219 AF2011-A1 Double Barrel, 231 Peacekeeper |
| Special | 181 Flare Gun |
| Melee | 233 Baseball Bat |

| Class | Two-handed (shield goes on the back) |
|---|---|
| Rifle | 10, 17, 20, 56, 58, 59, 60, 61, 62, 63, 72, 73, 74, 75, 76, 91, 92, 93, 124, 125, 126, 127, 128, 129, 130, 131, 132, 133, 134, 135, 136, 137, 138, 139, 154, 208, 209, 210, 211, 213, 218, 222, 224, 225, 226 |
| Sniper | 11, 14, 16, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 77, 78, 79, 80, 81, 150, 151, 152, 153, 155, 156, 157, 158, 159, 160, 161, 186, 191, 192, 193, 205, 215, 227, 228 |
| SMG | 12, 19, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 67, 68, 69, 70, 71, 96, 97, 113, 114, 115, 116, 117, 118, 119, 120, 121, 122, 123, 194, 195, 196, 197, 202, 212, 214, 217, 223, 230 |
| Shotgun | 13, 15, 18, 44, 45, 46, 47, 48, 49, 50, 51, 52, 53, 54, 82, 83, 84, 85, 86, 162, 163, 164, 165, 166, 167, 183, 204, 229 |
| LMG | 55, 57, 64, 65, 66, 94, 95, 140, 141, 142, 143, 144, 145, 146, 147, 148, 149, 198, 199, 200, 201, 203, 220, 221 |
| Launcher | 87, 88, 89, 90, 168, 169, 170, 171, 172, 173, 174, 175, 176, 184 |
| Special | 177, 178, 179, 180, 182, 185, 187, 188, 189, 232 |

## Near-duplicate review

Method: every pair of weapons in the same class with the same projectile and
pellet count, the same fire delay, push within 10%, magazine within 20% and
recoil within 25% was listed (265 pairs). The strict subset that plays the
same (push within 1, magazine within 2, recoil within 0.1, and the same
weight, accuracy and velocity) and involves an added weapon was fixed below.
`TestAddedWeaponsHaveNoNearIdenticalTwin` keeps it that way.

Fire delays of added real firearms follow their real cyclic rate
(`developed_fire_rates.go`) and were not changed. Every tweak stays inside the
class caps (sustained push, self-recoil, LMG magazine 60, starter pistol
band). Weight is the run-speed multiplier while the gun is held; spread is the
vertical scatter in px; speed is the projectile speed in px per tick.

### Original weapons (report only, not changed)

The source games themselves contain these near-identical groups (fire delay /
push / recoil / magazine):

| Class | Weapons (ROF / push / recoil / mag) |
|---|---|
| Rifle | 10 AK-47 (4 / 22 / 0.8 / 30); 56 H&K G36C (4 / 20 / 0.8 / 30); 60 Colt M4A1 (4 / 20 / 0.8 / 30) |
| Sniper | 11 H&K PSG-1 (31 / 60 / 10 / 5); 33 Accuracy International AWM-F (31 / 63 / 10 / 5); 35 Barrett M95 (31 / 63 / 10 / 5); 36 DSR-1 (31 / 60 / 10 / 5) |
| SMG | 12 H&K MP5K-PDW (4 / 17 / 0.8 / 30); 22 Thompson SMG (4 / 20 / 0.8 / 25); 23 H&K UMP45 (4 / 21 / 0.8 / 25); 24 Steyr AUG A3 Para (4 / 20 / 0.8 / 30); 25 H&K MP5A3 (4 / 18 / 0.8 / 30); 27 Beretta M12 (4 / 15 / 0.8 / 32); 29 AKS-74U (4 / 22 / 1 / 25); 70 Daewoo K7 (4 / 20 / 0.6 / 30); 71 B&T APC (4 / 21 / 0.6 / 30) |
| Shotgun | 13 Winchester Model 1200 Sawed-Off (25 / 8 / 5 / 7); 48 PM-5-350 (PM-12) (25 / 7 / 5 / 10); 49 Remington 870 Marine Magnum (25 / 8 / 5 / 7); 53 Fallout Combat Shotgun (SPAS-97) (25 / 9 / 5 / 6); 84 Benelli M3 Super 90 (25 / 8 / 5 / 5) |
| Shotgun | 15 USAS-12 (6 / 5 / 0.8 / 12); 47 Pancor Jackhammer MK3A2 (6 / 8 / 0.8 / 10); 82 Origin-12 (6 / 7 / 0.8 / 10) |
| Rifle | 17 Ruger Mini-14 (5 / 30 / 0.8 / 15); 72 CETME Rifle (5 / 28 / 1 / 20) |
| Rifle | 20 Bushmaster M17S (4 / 22 / 0.5 / 30); 62 FN F2000 (4 / 25 / 0.2 / 30); 74 OTs-14 Groza (4 / 24 / 0.4 / 30) |
| SMG | 28 MAS-38 (3 / 15 / 0.8 / 32); 32 MAS-48 (3 / 15 / 0.8 / 32) |
| SMG | 30 KRISS Vector (3 / 16 / 0.2 / 28); 31 B&T MP9 (3 / 15 / 0.5 / 30); 67 CZ Scorpion EVO 3 A1 (3 / 15 / 0.4 / 30); 69 ST Kinetics CPW (3 / 16 / 0.5 / 30) |
| Sniper | 34 PGM Hecate II (31 / 70 / 10 / 5); 79 ArmaLite AR-50 (31 / 70 / 10 / 1) |
| Sniper | 37 Steyr SSG 69 (25 / 60 / 10 / 5); 38 Blaser R93 LRS2 (25 / 60 / 10 / 5) |
| Shotgun | 52 Saiga-20 (12 / 6 / 2 / 7); 86 Browning Auto-5 (12 / 8 / 2 / 4) |
| LMG | 57 RPK-74 (3 / 20 / 0.8 / 40); 66 Steyr AUG HBAR (3 / 18 / 0.8 / 42) |
| Rifle | 58 SIG SG 552 (5 / 22 / 0.8 / 30); 59 IMI Galil ARM (5 / 25 / 0.8 / 35); 73 StG 44 (5 / 28 / 1 / 30) |
| Rifle | 61 H&K HK33 (7 / 32 / 2 / 20); 63 FN SCAR-H (7 / 34 / 2 / 20) |

The closest cases: PSG-1 / DSR-1 and AWM-F / M95 are exact twins
(31 / 60-63 / 10 / 5), SSG 69 / LRS2 are exact twins, MAS-38 / MAS-48 are
exact twins, and the Model 1200 / KSG-12 / 870 pump shotguns differ by one
frame at most.

### Added weapons (changed)

Values: fire delay / push / recoil / magazine / weight / spread / speed.

| # | Weapon | Class | Before (ROF / push / recoil / mag / weight / spread / speed) | After | Why |
|---|---|---|---|---|---|
| 94 | PKP Pecheneg | LMG | 3 / 20 / 0.8 / 60 / 0.7 / 4 / 26 | 3 / 20 / 0.8 / 60 / 0.7 / 2.5 / 26 | PKP: forced-air barrel, accurate |
| 95 | RPD | LMG | 3 / 20 / 0.75 / 60 / 0.7 / 4 / 26 | 3 / 19 / 0.75 / 50 / 0.78 / 4 / 26 | RPD: light, drum of 50 |
| 101 | SIG Sauer P226 | Pistol | 8 / 19 / 0.8 / 15 / 0.95 / 2.5 / 25 | 8 / 19 / 0.85 / 15 / 0.95 / 1.8 / 28 | P226: tighter and faster than the 92FS |
| 102 | CZ 75 | Pistol | 8 / 18 / 0.75 / 16 / 0.95 / 2.5 / 25 | 8 / 18 / 0.6 / 16 / 0.95 / 2 / 25 | CZ 75: soft-shooting |
| 109 | Luger P08 | Pistol | 10 / 21 / 0.9 / 8 / 0.95 / 2.5 / 25 | 10 / 21 / 0.7 / 8 / 0.95 / 1 / 29 | Luger: precise, fast round (vs Colt M1911A1) |
| 114 | Uzi | SMG | 5 / 19 / 0.8 / 32 / 0.9 / 3.5 / 25 | 5 / 18 / 0.8 / 32 / 0.95 / 4.2 / 25 | Uzi: light, sprays |
| 115 | Micro Uzi | SMG | 2 / 12 / 0.5 / 32 / 0.9 / 3.5 / 25 | 2 / 12 / 0.5 / 25 / 0.9 / 3 / 25 | Micro Uzi: shorter magazine, tighter |
| 116 | MAC-11 | SMG | 2 / 11 / 0.5 / 32 / 0.9 / 3.5 / 25 | 2 / 11 / 0.5 / 32 / 0.9 / 4.8 / 25 | MAC-11: pure spray |
| 119 | SIG MPX | SMG | 3 / 17 / 0.45 / 30 / 0.9 / 3.5 / 25 | 3 / 17 / 0.4 / 30 / 0.9 / 2.4 / 25 | SIG MPX: accurate (vs ST Kinetics CPW) |
| 121 | Sterling L2A3 | SMG | 5 / 19 / 0.7 / 34 / 0.9 / 3.5 / 25 | 5 / 19 / 0.7 / 34 / 0.9 / 2.2 / 25 | Sterling: accurate |
| 122 | Sten Mk II | SMG | 5 / 19 / 0.8 / 32 / 0.9 / 3.5 / 25 | 5 / 18 / 0.8 / 32 / 0.9 / 4.8 / 25 | Sten: crude |
| 123 | PPS-43 | SMG | 5 / 19 / 0.7 / 35 / 0.9 / 3.5 / 25 | 5 / 20 / 0.7 / 35 / 0.9 / 3.5 / 25 | PPS-43: hardest-hitting of the WWII SMGs |
| 127 | FN FAL | Rifle | 7 / 34 / 2 / 20 / 0.8 / 2.5 / 27 | 7 / 35 / 2 / 20 / 0.74 / 2 / 27 | FAL: heavier and harder than the SCAR-H |
| 129 | AEK-971 | Rifle | 3 / 18 / 0.6 / 30 / 0.8 / 2.5 / 27 | 3 / 18 / 0.4 / 30 / 0.8 / 3 / 27 | AEK-971: balanced recoil |
| 130 | AK-12 | Rifle | 4 / 23 / 0.7 / 30 / 0.8 / 2.5 / 27 | 4 / 24 / 0.9 / 30 / 0.8 / 3.2 / 27 | AK-12: hits harder, kicks and wanders |
| 131 | AK-74 | Rifle | 5 / 22 / 0.8 / 30 / 0.8 / 2.5 / 27 | 5 / 22 / 0.6 / 30 / 0.8 / 3.2 / 27 | AK-74: soft recoil, loose (vs SIG SG 552) |
| 134 | IWI Tavor TAR-21 | Rifle | 3 / 18 / 0.55 / 30 / 0.8 / 2.5 / 27 | 3 / 17 / 0.55 / 30 / 0.86 / 2 / 27 | Tavor: compact bullpup |
| 136 | L85A2 | Rifle | 4 / 23 / 0.65 / 30 / 0.8 / 2.5 / 27 | 4 / 22 / 0.65 / 30 / 0.74 / 1.4 / 27 | L85A2: very accurate, heavy |
| 137 | SIG SG 550 | Rifle | 4 / 24 / 0.65 / 30 / 0.8 / 2.5 / 27 | 4 / 24 / 0.65 / 20 / 0.8 / 1.6 / 30 | SG 550: precise, 20-round magazine |
| 138 | H&K XM8 | Rifle | 4 / 22 / 0.55 / 30 / 0.8 / 2.5 / 27 | 4 / 21 / 0.45 / 30 / 0.86 / 2.5 / 27 | XM8: light carbine |
| 140 | PKM | LMG | 3 / 20 / 0.8 / 60 / 0.7 / 4 / 26 | 3 / 19 / 0.7 / 60 / 0.7 / 4.5 / 26 | PKM: loose spray |
| 141 | MG42 | LMG | 2 / 13 / 0.35 / 50 / 0.7 / 4 / 26 | 2 / 13 / 0.35 / 50 / 0.7 / 4.5 / 26 | MG42: wilder |
| 142 | MG3 | LMG | 2 / 13 / 0.45 / 50 / 0.7 / 4 / 26 | 2 / 13 / 0.45 / 60 / 0.7 / 3 / 26 | MG3: belt of 60, tighter than the MG42 |
| 144 | M240B | LMG | 3 / 20 / 0.8 / 60 / 0.7 / 4 / 26 | 3 / 20 / 0.6 / 60 / 0.64 / 3 / 26 | M240B: heavy, steady |
| 148 | Lewis Gun | LMG | 4 / 24 / 1 / 47 / 0.7 / 4 / 26 | 4 / 24 / 1 / 47 / 0.66 / 3 / 26 | Lewis: cooling shroud, heavy |
| 151 | Mosin-Nagant M91/30 | Sniper | 25 / 56 / 7 / 5 / 0.72 / 1.2 / 32 | 25 / 57 / 8 / 5 / 0.7 / 1.6 / 32 | Mosin: kicks, heavier |
| 156 | M40A5 | Sniper | 27 / 62 / 8 / 5 / 0.72 / 1.2 / 32 | 27 / 64 / 8.5 / 5 / 0.72 / 1.2 / 34 | M40A5: hotter round than the M24 |
| 164 | Winchester Model 1887 | Shotgun | 18 / 8 / 5 / 5 / 0.78 / 8 / 25 | 18 / 8 / 5 / 5 / 0.78 / 8 / 25 (pellet fan 1.4 deg) | full-length 1887: tighter pattern than the sawed-off (18) |
| 195 | MP 28 | SMG | 5 / 19 / 0.75 / 32 / 0.9 / 3.5 / 25 | 5 / 19 / 0.75 / 50 / 0.86 / 3.5 / 25 | MP 28: 50-round magazine, heavier |
| 196 | Erma EMP | SMG | 5 / 19 / 0.7 / 32 / 0.9 / 3.5 / 25 | 5 / 20 / 0.6 / 32 / 0.9 / 3 / 25 | Erma EMP: steady |
| 199 | Maxim MG 08/15 | LMG | 4 / 24 / 1 / 60 / 0.7 / 4 / 26 | 4 / 24 / 0.8 / 60 / 0.6 / 2.5 / 26 | Maxim: water-cooled, steady, very heavy |
| 200 | MG 15 n.A. | LMG | 4 / 24 / 1 / 60 / 0.7 / 4 / 26 | 4 / 23 / 1 / 50 / 0.76 / 4.5 / 26 | MG 15: light aircraft gun |
| 204 | Sjögren Inertial | Shotgun | 12 / 6.8 / 2 / 5 / 0.78 / 8 / 25 | 12 / 6.8 / 2 / 5 / 0.78 / 8 / 25 (pellet fan 3 deg) | Sjogren: wide pattern (vs Saiga-20) |
| 205 | Springfield M1903 | Sniper | 25 / 58 / 7 / 5 / 0.72 / 1.2 / 32 | 25 / 58 / 7.5 / 5 / 0.72 / 1 / 34 | Springfield: flat and precise |
| 206 | Colt Model 1903 Hammerless | Pistol | 9 / 18 / 0.7 / 8 / 0.95 / 2.5 / 25 | 9 / 17 / 0.7 / 8 / 1 / 2.5 / 25 | Colt 1903: pocket pistol, no slowdown |
| 212 | H&K UMP9 | SMG | 4 / 18 / 0.65 / 30 / 0.9 / 3.5 / 25 | 4 / 18 / 0.65 / 30 / 0.88 / 2.5 / 25 | UMP9: accurate, heavier than the Vityaz |
| 214 | M3 Grease Gun | SMG | 6 / 23 / 0.8 / 30 / 0.9 / 3.5 / 25 | 6 / 23 / 0.6 / 30 / 0.9 / 4 / 25 | Grease Gun: slow, soft, loose |
| 215 | Type 99 Arisaka | Sniper | 25 / 58 / 7 / 5 / 0.72 / 1.2 / 32 | 25 / 56 / 6 / 5 / 0.76 / 1.2 / 32 | Arisaka: mild 6.5/7.7 mm, lighter |
| 216 | Nambu Type 14 | Pistol | 9 / 16 / 0.65 / 8 / 0.95 / 2.5 / 25 | 9 / 16 / 0.65 / 8 / 0.95 / 3.5 / 22 | Nambu: slow, wandering round |
| 218 | M1941 Johnson Rifle | Rifle | 9 / 31 / 1.2 / 10 / 0.8 / 2.5 / 27 | 9 / 30 / 1.2 / 10 / 0.8 / 1.6 / 27 | Johnson: accurate |
| 225 | Mondragón M1908 | Rifle | 9 / 31 / 1.3 / 10 / 0.8 / 2.5 / 27 | 9 / 32 / 1.6 / 8 / 0.8 / 3 / 27 | Mondragon: 8 heavy, kicking rounds |
| 231 | Peacekeeper | Pistol | 22 / 36 / 5 / 6 / 0.95 / 2.5 / 25 | 22 / 34 / 4.5 / 6 / 0.95 / 1.2 / 30 | Peacekeeper: long-barrel accuracy, less push than the Python |
