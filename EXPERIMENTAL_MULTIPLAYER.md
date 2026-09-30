# Private Internet multiplayer experiment

This branch extends the existing **native Go/Ebitengine game**, not a browser
client. The host player's game remains the authoritative server. It preserves
LAN play and adds an explicitly enabled, invite-only Internet path using the
existing Cloudflare WebSocket gateway. The later [public-room extension](PUBLIC_ROOMS.md)
adds explicitly selected OPEN rooms only when a shared HTTPS directory is configured.
No directory or live tunnel was deployed during this work.

## What changed

- Fixed the first-WebSocket-client crash: the custom upgrade handshake did not
  initialize `websocket.Config.Origin`, so recording the peer address could
  dereference a nil URL
- The native-client Origin and application subprotocol are checked before
  attaching a session. Origin is spoofable by nonbrowser software; the private
  invite's HMAC challenge-response is the authentication boundary
- Internet invites retain the randomly generated 128-bit session key. They no
  longer derive authentication from the short six-character LAN code
- Host authentication and allowed slots are initialized before accepting peers
- Idle transport heartbeats run every second. A peer that sends no complete
  application frame for 10 seconds is disconnected, including partial-frame
  stalls. Stale gameplay input becomes neutral after 500 ms
- Established messages must name their assigned player slot. Clients cannot
  submit authoritative combat state
- Closing a session cancels pending handshakes. Closing the gateway also closes
  upgraded WebSockets, which `http.Server.Shutdown` does not own
- Slow readers have independent writers and bounded latest-state queues; a
  stalled write expires after five seconds without blocking other players
- Network games continue running when the host switches windows. Local games
  still pause on focus loss, and unfocused local controls are neutral
- The shared address field accepts the complete maximum-length 95-character
  Cloudflare invite, with a 128-character input bound
- The CLI `--host` and legacy HOST button are LAN-only. Internet hosting requires
  explicit opt-in in the shared API and tunnel launcher. INVITE remains the
  default; OPEN additionally requires the configured HTTPS directory and a
  separate public routing ID. DHT, QUIC and legacy relay publication remain off

The wire protocol on the gameplay-extension branch is **20** (the network-only
milestone uses **19**). All players must use this branch's build; the
released protocol-18 build is intentionally rejected rather than misinterpreted.

## Run a private trial

1. Build the same branch on the host and guests. Keep the normal runtime assets
   beside the executable, as described in the README
2. The host obtains `cloudflared` from Cloudflare's official installation page:
   <https://developers.cloudflare.com/cloudflare-one/networks/connectors/cloudflare-tunnel/downloads/>
   This branch does not download, install or update it automatically
3. On the host, set `GUNMAYHEM_EXPERIMENTAL_INTERNET=1` and optionally set
   `GUNMAYHEM_CLOUDFLARED` to the official executable's absolute path. Launch the
   game from that environment. For example, in PowerShell:

   ```powershell
   $env:GUNMAYHEM_EXPERIMENTAL_INTERNET = "1"
   $env:GUNMAYHEM_CLOUDFLARED = "C:\Tools\cloudflared.exe"
   .\GunMayhem.exe
   ```

4. Open **CUSTOM GAME → ONLINE → CREATE SESSION → INTERNET**. With the opt-in
   unset the network browser is still labeled LAN. This experimental
   selection defaults to **INVITE CODE**; retain it for a private trial. Clicking CREATE starts a public temporary
   Cloudflare tunnel; the UI discloses this before creation. Startup can take up
   to 20 seconds and is currently synchronous
5. Configure the match and mark the intended guest slots as network humans.
   Share the complete `GMT1-…` invitation only with the intended players. It
   contains the session secret; treat it like a password. On Windows the host
   can explicitly click **COPY INVITE** in SLOTS. Guests run the same native
   build, click the join field, press **Ctrl+V**, check it and click **JOIN**.
   Paste replaces the full field (including its default loopback address).
   Invalid, multiline or overlong pasted values leave the old field unchanged;
   surrounding whitespace is trimmed. Clipboard access occurs only on that
   shortcut/click and never logs its contents. This implementation uses Windows
   clipboard APIs; other platforms show an explicit unsupported message and
   still accept typed codes. A busy/unavailable clipboard reports failure
6. The host must keep both the game and its child `cloudflared` process running.
   Leaving the session shuts the gateway and tunnel down. A new hosted session
   uses a new secret and generally a new tunnel hostname

Unset the opt-in variable to restore the release's LAN-only host UI. Regular
`--host` remains LAN-only even when this variable is set.

The Internet path uses HTTPS/WSS through Cloudflare to a loopback HTTP gateway.
Guests do not need `cloudflared`. This is not a raw UDP tunnel, browser port, or
peer-host migration system. No Cloudflare account, API token or named tunnel is
created by this branch.

## Reconnection and current limits

- Disconnect/rejoin is tested. Once a disconnected slot is free, a new client
  can join it without inheriting held input or READY state
- There is **no transparent reconnect/resume, reserved identity, or match-state
  restoration**. Rejoin uses normal join-in-progress rules. If the host returns
  to setup after the last participant leaves, it may need to reopen a human slot
- A changed Quick Tunnel hostname requires the host's new invitation. Host
  shutdown ends the session; there is no host migration
- The four-player cap, existing maps and campaign content are unchanged.
  This branch separately adds the Custom Game-only perk described in
  [KNOCKBACK_PERK.md](KNOCKBACK_PERK.md). Larger player counts require removing fixed four-player arrays,
  revisiting camera/HUD/spawn geometry, then load/bandwidth tests
- Snapshot transport is reliable/ordered and sends render-complete state.
  Internet latency can remain visible; this work does not add local prediction,
  reconciliation, interpolation, delta compression or an Internet latency SLA
- Quick Tunnels are for development/testing, have no uptime guarantee, allow
  at most 200 in-flight requests, and do not support SSE. Cloudflare supports
  WebSockets, but restarts and network changes can drop long-lived connections

Official references checked 2026-09-30:
- <https://developers.cloudflare.com/cloudflare-one/faq/cloudflare-tunnels-faq/>
- <https://developers.cloudflare.com/network/websockets/>
- <https://developers.cloudflare.com/cloudflare-one/networks/connectors/cloudflare-tunnel/do-more-with-tunnels/trycloudflare/>

## Verification and next acceptance gate

See `MULTIPLAYER_TEST_RESULTS.md` for actual commands and outcomes. The added
fixtures bind to loopback and do not start `cloudflared`, register a domain,
announce a public game, or use real player accounts. The WSS fixture uses a real
local TLS handshake with an explicitly trusted test certificate, not disabled
certificate validation.

Before recommending Internet play as supported, run an approved real
cross-network trial on the target Windows builds: two and four players,
normal/high latency, host alt-tab, a slow/stalled guest, gateway restart,
manual rejoin, between-round READY, and a full multi-map match. Record RTT,
input-to-render delay, bandwidth, snapshot size and disconnect reasons. Local
synthetic delay is not evidence of real Cloudflare performance.

Once that gate passes, add smoke/flash status replication and line-of-sight
rules, then helmet flash resistance/thermal rendering and clearly bounded perk
balance. Larger maps/player counts and new campaign enemies belong in separate
changes with rendering, AI and multiplayer coverage. This experiment deliberately
establishes the transport/lifecycle baseline first.

## Asset/licensing note

The inspected base commit (`ded993a`) has no LICENSE file. Its README attributes
original names, artwork and audio to their rights holders. No license or rights
to those assets are inferred here; the review handoff contains code patches and
notes, not a redistributed game/asset bundle.
