# Multiplayer / Custom Game

Gun Mayhem Go uses the host player's PC as the authoritative game server. In the current public release, multiplayer is supported **only inside a local network (LAN)**.

Internet multiplayer experiments using DHT, QUIC, relay/master-server and Cloudflare Quick Tunnels remain in the source tree for future work, but they are not exposed as a supported gameplay path in this release because they are not reliable enough yet.

## Supported multiplayer modes

Open **CUSTOM GAME -> LAN -> CREATE SESSION**.

The supported network reach is:

- **LAN ONLY** - players connected to the same local network.

Internet hosting is shown as disabled in the session setup screen.

LAN sessions can use either access policy:

- **OPEN** - the session appears automatically in the LAN server browser.
- **INVITE CODE** - the session stays hidden and a joining player must enter the generated `GM-XXXXXX` code.

Returning from the hosted lobby or match setup to the server browser closes the host session immediately while keeping the Custom Game configuration in memory until the game is closed.

## Joining a LAN session

A player can:

1. open the **LAN SERVER BROWSER**, press **REFRESH**, choose an OPEN session and join it,
2. enter the host's `GM-XXXXXX` invite code for an INVITE CODE session,
3. enter a direct LAN `IP:PORT` address as an advanced fallback.

LAN discovery uses UDP port `7778`. The gameplay listener uses TCP port `7777` by default.

If Windows Firewall asks for permission, allow `GunMayhem.exe` on private networks.

## OPEN LAN sessions

OPEN sessions advertise themselves locally and are shown in the LAN browser together with player count and available match preview information.

No join code is required. The host remains authoritative for gameplay and connected clients send only their input and character configuration.

## INVITE CODE LAN sessions

INVITE CODE sessions do not appear in the normal open LAN list. The host displays a short `GM-XXXXXX` code.

The code is used for private LAN discovery and authentication. The client proves knowledge of the session key before receiving a player slot.

## Custom Game series

Custom Game can run a configurable multi-map series in local and LAN multiplayer. Each map gets its own Post Game Summary and the series keeps cumulative statistics. In network play, connected human players can mark themselves ready between maps before the host continues.

## Security model

The host is authoritative for gameplay. Remote clients do not run authoritative combat simulation.

Network hardening includes:

- bounded CBOR frames,
- strict message-type validation,
- protocol-version checks,
- player/configuration range validation,
- handshake timeout before assigning a player slot,
- per-client message-rate limits,
- pending-handshake limits,
- bounded host snapshots,
- no multiplayer messages for files, filesystem paths, shell commands or executable code,
- HMAC challenge-response for INVITE CODE sessions before a remote player receives a slot.

## Internet multiplayer status

Internet multiplayer is **not supported in this release**.

The repository still contains experimental work for direct P2P, DHT discovery, QUIC, optional master/relay infrastructure and Cloudflare WebSocket tunnels. These implementations are kept for continued development and for a future attempt with a stronger networking design, but the released game intentionally exposes only the working LAN path.
