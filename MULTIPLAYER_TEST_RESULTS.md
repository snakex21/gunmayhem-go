# Multiplayer experiment verification

Base: `ded993a` (v0.16.0), branch `experiment/cloudflare-netplay-reliability`.
Verification date: 2026-09-30. Go toolchain: 1.26.2, Linux amd64.

## Environment

Ebitengine requires a graphical runtime even for the game package tests. The
cloud environment initially lacked X11/ALSA headers and a display. Official
Debian packages were downloaded and extracted into temporary test-only storage;
no system packages, public tunnels, accounts, credentials or permissions were
changed. Tests ran with a temporary Xvfb display. Network fixtures are local.
Windows execution and a real Cloudflare Internet session were not performed.

Normal developer equivalents:

```sh
go test ./...
go test -race ./internal/game -run 'Test(Netplay|CloudflareGateway|CloudflareLongestInvite|TrackedWebSocket|ExperimentalInternet|InternetInvite|InternetHostPolicy|HostSetup|LANOpenHost|LANInviteHost)' -count=1
go vet ./...
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o GunMayhem-experimental.exe .
```

On Linux, install the normal Ebitengine build dependencies and run the game tests
under a display/Xvfb. Test-only environment setup is not a product dependency.

## Command outcomes

- Final focused race-enabled network suite: **passed**, 33.111 seconds
- Final complete `go test ./...`: **failed** with the same four baseline failures
  listed below; game package completed in 46.822 seconds
- Windows amd64 cross-build: **passed**; binary was not executed on Windows or
  distributed with original assets
- `go vet ./...`: **failed** at existing `internal/game/menu.go:262` unreachable
  code. That untouched legacy menu function calls the replacement menu and
  returns before its retained reference implementation
- `git diff --check`: **passed**

## Verified behavior

- Actual local TLS + WebSocket upgrade, private challenge-response, player
  assignment, client input, host snapshot, and connection closure
- Wrong secret, wrong/missing subprotocol, foreign/missing Origin and oversized
  application frame rejection
- Host close during unauthenticated handshake, upgraded WebSocket shutdown,
  pending writer cancellation and no stale slot on repeated rejoin
- Neutral input after 500 ms without fresh gameplay input or immediately after
  a peer closes; established message identity bound to its assigned slot
- Independent heartbeat survival through an idle period longer than 10 seconds
  and eviction of a client that never sends another complete application frame
- Latest-input/state coalescing under synthetic per-write delays of 0, 40 and
  100 ms plus 0–2 ms deterministic jitter. This is not a measured Internet RTT
- A stalled receiver cannot block snapshot delivery to a healthy peer and is
  evicted by its independent write deadline
- Internet secret retention, opt-in/private-only shared entrypoints, rejected
  Internet policy preserving an existing LAN host, and complete longest invite
  entry without lifting the bounded input limit

## Full-suite baseline limitations

Both the unchanged base and the modified tree report these same four failures
in this environment:

1. `TestAMR2BoltHandUsesOneRequestedPlayerColor`: AMR-2 frame 3 missing
2. `TestAMR2BoltHandDetectionUsesPixelsInsteadOfTimelineGuess`: looped hand frame missing
3. `TestGM2WeaponLibraryExposesAllTwentyAdditions`: GM2 weapon card 67 hit=0
4. `TestLANDiscoveryBuildsDirectedBroadcastForPrivateSubnet`: sandbox denies
   network-interface route enumeration (`netlinkrib: operation not permitted`)

These were reproduced before code edits. The focused network checks pass, but
`go test ./...` is **not a clean pass**. The original asset/UI failures and
restricted LAN-interface enumeration have not been hidden or relabeled as fixed.

## Independent review

An independent read-only review exercised the focused local suite and the race
detector. It identified legacy hosting entrypoints bypassing the UI opt-in and
an 80-character input limit truncating valid long invitations. Both were fixed
with regressions before the final handoff. This review is scoped to this network
experiment, not a full security audit or a remote-play certification.

## Same-process rejoin follow-up

Two additional regression tests demonstrated that a Game instance retained its
“configuration already sent” flag and old sound sequence after disconnecting.
Both failed before the follow-up change. `CloseNetplay` now clears only these
session caches, so an unchanged character is resent on the next join and the
new host's sequence starting at one is accepted. Character/setup preferences are
preserved. This still does not implement automatic reconnect or identity resume.

Independent focused race regressions passed three times; combined network/perk
race checks passed (31.319 seconds), and Windows cross-build passed. A subsequent
10-repeat race selection of rejoin and gateway shutdown passed (11.149 seconds).
The gateway shutdown assertion now waits for its separate HTTP handler's cleanup
rather than incorrectly assuming the handshake goroutine orders that defer.

## Cumulative experimental branch verification

The combined branch contains the network, perk, same-process rejoin, and bounded
font compatibility changes. Its exact-asset copy normalization and licensed
whole-label fallback were independently reviewed before integration.

- Integrated network/perk/font/rejoin race suite: passed (31.950 seconds)
- Integrated unfiltered `go test ./...`: completed with exactly the same four
  baseline failures listed above (47.418 seconds); no extra failure was observed
- Windows amd64 cross-build: passed; Windows execution remains untested
- `go vet ./...`: same existing unreachable legacy menu code at line 262
- Four runtime render fixtures (Custom Game perk picker, online perk picker,
  host perk blocklist, and gameplay HUD) passed with the font override unset.
  Actual shipped fonts/production fallback rendered the new perk labels and
  HUD names/lives/ammo legibly, without new clipping in those screens
- Independent review also checked the production-font previews. Existing
  dark-on-dark heading contrast is outside these changes
- No previously tracked `assets/` or `source/` file was modified or removed.
  The sole asset-tree addition is the verbatim upstream Go Regular license notice

These remain local synthetic, render and cross-build checks. No live Cloudflare
endpoint, remote-player match, Windows execution, or full security audit was
performed. The existing GitHub workflow runs on release tags, not this branch;
local checks are not represented as remote CI results.
