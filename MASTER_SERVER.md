# Gun Mayhem public master + relay

This document describes the retained v1 design. The experimental branch now
defaults the master command to the separate **v2 directory-only** mode, on
loopback with no gameplay relay. See [PUBLIC_ROOMS.md](PUBLIC_ROOMS.md) for the
active public-room experiment. The game does not automatically enable the v1
relay, DHT or private-code lookup flow described below. No shared directory has
been deployed.

## Architecture

Gun Mayhem keeps the player-hosted model: the host computer runs the authoritative match simulation. The public infrastructure has two separate jobs:

1. **Master directory** - keeps the public server list.
2. **Relay fallback** - forwards the already validated game stream only when a direct connection cannot be established.

When a player selects an Internet HOST GAME, the game:

- listens on TCP port `7777` by default,
- keeps LAN discovery completely independent from the Internet infrastructure,
- treats direct TCP/QUIC and automatic port mapping only as latency optimizations,
- registers the session with the public master every 10 seconds when a master URL is configured,
- registers OPEN sessions in the public list and INVITE sessions privately under their locally generated six-character code,
- opens an authenticated **outbound** relay tunnel when the master advertises a relay,
- keeps the direct connection as the preferred path,
- uses the relay automatically for NAT/CGNAT/firewall/VPN cases. The player host does not need a public inbound port.

Without a configured public master/relay the game still attempts decentralized DHT/QUIC traversal, but that path is best-effort and cannot reliably cross every CGNAT or symmetric NAT.

The master removes entries that have not refreshed for 35 seconds.

## Security model

The internet-facing code is deliberately restricted:

- public master registration uses a cryptographically random server ID and secret,
- the secret is sent to a public master only over HTTPS; plain HTTP public master URLs are rejected by the game,
- the relay never receives the raw host secret; it authenticates the host with a one-time HMAC challenge-response,
- server deletion and updates require the server secret,
- master registration/list/delete requests are rate-limited per source IP,
- the master limits the number of active server registrations globally and per IP,
- relay pending handshakes and active relay clients per server are bounded,
- game packets use size-prefixed CBOR with strict decode limits,
- clients may send only hello/input/player-config messages,
- remote player configuration is range-checked and names are bounded,
- oversized, unknown, malformed or excessive messages are rejected,
- the network protocol contains no file-upload, filesystem-path, shell-command or executable-launch message.

This substantially reduces the class of bugs that made some old network games vulnerable to malformed-packet RCE. It is not a claim that any network program can be mathematically guaranteed exploit-free.

The current gameplay stream is not end-to-end encrypted. The parser and command surface are hardened, but gameplay traffic should not be treated as confidential.

## Run a public master and relay

Recommended single-endpoint deployment (HTTPS and WSS on TCP 443):

```text
GunMayhemMaster.exe -addr :443 -relay-addr "" -public-relay-address wss://master.example.com/v1/relay -tls-cert fullchain.pem -tls-key privkey.pem
```

Linux:

```text
./GunMayhemMaster -addr :443 -relay-addr "" -public-relay-address wss://master.example.com/v1/relay -tls-cert fullchain.pem -tls-key privkey.pem
```

Recommended production layout:

- expose the master API and `/v1/relay` WebSocket through HTTPS/WSS on TCP 443,
- use WSS for both host and client outbound connections so VPNs do not need to allow a custom public TCP port,
- pass the public relay endpoint explicitly with `-public-relay-address`,
- enable `-trust-proxy` only when the HTTP master is behind a trusted reverse proxy that sets the real client IP.

Example behind a local reverse proxy:

```text
GunMayhemMaster.exe -addr 127.0.0.1:7780 -relay-addr "" -public-relay-address wss://master.example.com/v1/relay -trust-proxy
```

Do not enable `-trust-proxy` when the HTTP master is directly exposed to the internet.

## Point the game at the public master

Create `master_server.txt` next to `GunMayhem.exe` containing exactly one URL, for example:

```text
https://master.example.com
```

For development the same value can be supplied with:

```text
GUNMAYHEM_MASTER_URL=https://master.example.com
```

For safety, public `http://` master URLs are rejected. Plain HTTP is accepted only for localhost/private LAN addresses. Development can explicitly override this with `GUNMAYHEM_ALLOW_INSECURE_MASTER=1`.

## Network ports

- public master API: TCP `7780` directly, or preferably HTTPS/443 through a reverse proxy,
- relay: WSS `/v1/relay` on TCP `443` (recommended); raw TCP `7781` remains a development/legacy option,
- player-hosted direct game: TCP `7777` by default.

Only the public master and relay machine must be globally reachable for the relay fallback to work. A player behind CGNAT can still host because the host creates an outbound relay tunnel.

## Connection order

OPEN browser sessions:

1. LAN discovery is returned first and never waits for Internet discovery.
2. The public master directory is queried in parallel with DHT.
3. A master result can therefore appear immediately without waiting for a slow DHT traversal.
4. The client first tries the direct player-hosted address with a short timeout when a relay is available.
5. If direct connection or handshake fails, it automatically uses the relay.
6. DHT/QUIC remains an additional decentralized direct path.

INVITE sessions:

1. The client first resolves the code on LAN.
2. If a public master is configured, the same private code is resolved there without exposing the session in the public browser.
3. The client tries direct TCP and then the authenticated relay.
4. DHT/QUIC remains the final decentralized fallback.

Direct Join by manually entering an `IP:PORT` remains available, but it does not have a public-master relay record to fall back to.
