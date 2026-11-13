# Multiplayer / Custom Game

Gun Mayhem Go uses the host player's PC as the authoritative game server. The current build supports **LAN** and an experimental **Internet** path based on Cloudflare Quick Tunnels.

Internet multiplayer is intentionally experimental. Quick Tunnels are useful for testing with friends, but they are temporary public endpoints and are not a production-grade, low-latency matchmaking service.

## Supported multiplayer modes

Open **CUSTOM GAME -> ONLINE -> CREATE SESSION**.

The available network reach modes are:

- **LAN ONLY** - players connected to the same local network.
- **INTERNET + LAN** - the host also starts a temporary Cloudflare Quick Tunnel. **INVITE CODE** is recommended.

Internet hosting is shown as an experimental option in the session setup screen. It requires the bundled `cloudflared.exe` (or a binary selected with `GUNMAYHEM_CLOUDFLARED`).

LAN sessions can use either access policy:

- **OPEN** - the session appears automatically in the LAN server browser.
- **INVITE CODE** - the session stays hidden and a joining player must enter the generated `GM-XXXXXX` code.

Internet sessions also support both access policies:

- **OPEN** - the temporary tunnel is published through the experimental DHT directory; this is less reliable than an invite.
- **INVITE CODE** - the host displays a `GMT1-...` code containing the temporary tunnel hostname and session secret. The recipient pastes that code into the normal JOIN CODE field.

Returning from the hosted lobby or match setup to the server browser closes the host session immediately while keeping the Custom Game configuration in memory until the game is closed.

## Joining a LAN session

A player can:

1. open the **LAN SERVER BROWSER**, press **REFRESH**, choose an OPEN session and join it,
2. enter the host's `GM-XXXXXX` invite code for an INVITE CODE session,
3. enter a direct LAN `IP:PORT` address as an advanced fallback.

LAN discovery uses UDP port `7778`. The gameplay listener uses TCP port `7777` by default.

If Windows Firewall asks for permission, allow `GunMayhem.exe` on private networks. On the first launch after moving the portable folder, the game asks for administrator approval once and creates path-specific rules for the current `GunMayhem.exe` and `cloudflared.exe`; moving the folder again causes the path check to recreate them.

## OPEN LAN sessions

OPEN sessions advertise themselves locally and are shown in the LAN browser together with player count and available match preview information.

No join code is required. The host remains authoritative for gameplay and connected clients send only their input and character configuration.

## INVITE CODE LAN sessions

INVITE CODE sessions do not appear in the normal open LAN list. The host displays a short `GM-XXXXXX` code.

The code is used for private LAN discovery and authentication. The client proves knowledge of the session key before receiving a player slot.

## Joining an Internet session

For the most reliable experimental Internet flow, the host chooses **INTERNET + INVITE CODE** and sends the displayed `GMT1-...` code to the other player. The player enters it in the same JOIN CODE field used for LAN codes. The host must keep the game running; restarting it creates a different tunnel and invalidates the old code.

The Quick Tunnel uses outbound TCP/UDP `7844` for the tunnel and HTTPS/TCP `443` for the public health check and WSS connection. No router port-forwarding is required. Creating the session no longer waits on the game screen for Cloudflare readiness: the UI returns immediately and publishes an OPEN session only after the public endpoint becomes healthy. The endpoint is temporary and must not be treated as a permanent server address. Internet + OPEN relies on the experimental DHT directory and may fail when UDP/DHT discovery is blocked; use an invite code when possible.

Quick Tunnel traffic passes through Cloudflare. Do not share an invite code publicly: possession of the code grants access to the session.

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

Internet multiplayer is available as an **experimental** Quick Tunnel path in this build. It is best for temporary tests with a known player, not for public competitive release. The repository also contains experimental DHT, QUIC and optional relay/master-server infrastructure for future work.
