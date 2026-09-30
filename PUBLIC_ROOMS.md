# Public Internet rooms experiment

This is an opt-in experiment. A directory implementation and local tests do not
mean a shared Internet directory has been deployed. The player hosting the room
continues to run the authoritative game simulation; this service only lists
public room routes and match previews. No account, live tunnel, public listener
or public deployment was created for these checks.

## Directory server

Build the existing `./cmd/master-server` command. Its default mode is now
`public-rooms`, bound to `127.0.0.1:7780`, with the legacy gameplay relay disabled.
The new handler exposes only `/v2/rooms` and read-only `/healthz`. It never fetches
listed endpoints, relays gameplay, stores private invitations or exposes the old
private-code lookup API.

For a local development directory:

```sh
go run ./cmd/master-server -mode public-rooms -addr 127.0.0.1:7780
```

This example is loopback-only. A usable shared directory separately requires an
approved hosting environment and a stable, reachable HTTPS endpoint configured
identically for hosts and guests. Place the directory behind a TLS reverse proxy
or supply both `-tls-cert` and `-tls-key`. Listening on a public interface is an
explicit operator action; it is not enabled by default. Keep the backend private
when using a reverse proxy, overwrite forwarded-IP headers at that proxy, and
only then enable `-trust-proxy`. Otherwise anyone able to reach the backend could
forge the IP used for quotas.

`-mode legacy` explicitly selects the retained, separately experimental v1
master/relay. Its previous security limitations have not been repaired or
certified by this work. Relay flags in the default public-rooms mode are rejected.
The v2 directory does not need a gameplay relay or router port mapping.

### Public API and identity

- `POST /v2/rooms` accepts one complete `masterapi.PublicRoom` JSON object and
  creates or refreshes that room
- `GET /v2/rooms?protocol=20` returns only active rooms for that protocol, bounded
  to 128 rooms. Change the query version when the game protocol changes
- `DELETE /v2/rooms?id=<room-id>` delists and tombstones the owned room
- POST and DELETE require one `X-GunMayhem-Room-Owner` header containing a
  separately generated 256-bit ownership secret as 64 lowercase hexadecimal
  characters. The directory keeps only a SHA-256 hash of this value
- Public room IDs contain 128 random bits as 32 lowercase hexadecimal characters.
  They are public routing identifiers, not authentication secrets
- A room endpoint must exactly match
  `wss://<single-label>.trycloudflare.com/ws/room/<room-id>`. Credentials, ports,
  alternative paths, escaped paths, query strings, fragments and noncanonical
  casing are rejected. No arbitrary TCP, UDP, IP or relay address is published
- Responses contain no ownership header, invitation code, private session key or
  client public IP. Operators must also exclude ownership headers and request
  bodies from proxy/APM logs. Never place secrets in names or rule descriptions

Registration requires `Content-Type: application/json`, every DTO field, an
exact 20-element integer map array and no duplicate, case-aliased or unknown
fields, null values, trailing JSON or invalid UTF-8. Names and summaries have
byte/rune limits and cannot contain control characters, bidi/format controls or
Unicode line/paragraph separators. Capacity is limited to four players.

Public entries are advertisements from unverified hosts. Directory ownership
protects changes to a registry ID; it does not establish a human host's identity,
verify tunnel ownership, guarantee reachability or certify gameplay safety.
Clients must keep endpoint/protocol validation and bounded gameplay decoding.

### Lifetime, limits and errors

The default active TTL is 35 seconds; hosts should refresh every ten seconds.
`-ttl` can choose a positive TTL up to two minutes. Invalid or larger values use
the 35-second default. State is memory-only; restarting the directory drops all
listings and tombstones, and it has no multi-replica consistency mechanism.

Deletion and active expiry create two-minute tombstones. These also cover a
DELETE arriving before its canceled initial registration. Same-owner repeated
DELETE succeeds without extending retention; a late same-ID heartbeat receives
409 while the tombstone exists. Hosts must cancel/wait for bounded in-flight
requests before deletion and must not automatically reopen a closed room.
After the retention window or a directory restart, old IDs can be registered
again; this is bounded stale-request protection, not permanent revocation.

Limits are enforced in memory:

- 128 active rooms globally
- 1,024 total room records, including tombstones
- 16 room records per source IP, including tombstones; moves between IPs are
  checked against the destination quota
- 120 GET requests and 120 POST requests per minute per IP; 60 DELETE requests
  per minute per IP
- 4,096 tracked method/IP rate keys with a hard cap and expiry
- Four-KiB registration bodies, bounded query length, and an eight-KiB HTTP
  header limit in directory-only server mode

Status codes are 400 for malformed input, 401 for missing/incorrect ownership on
deletion or ownership conflicts, 409 for a closed/expired owned ID, and 429 for
request/room limits. A malformed or missing ownership header on POST is 400.
Responses are `Cache-Control: no-store`; the service does not emit credential or
request-body logs. Rate-limited responses include `Retry-After: 60`.

Tombstones and rate caps intentionally fail closed under saturation. These
application limits do not replace hosting-provider connection, bandwidth or
DDoS protection. Users sharing an IP also share its limits.

### Local verification

```sh
go test ./internal/masterapi ./internal/publicrooms ./cmd/master-server
go test -race ./internal/publicrooms ./cmd/master-server
```

The directory tests use local in-process handlers and a local TLS fixture. They
cover registry/list/delete, secret separation, ownership conflicts, deletion
before initial registration, concurrent late heartbeats, active/tombstone expiry,
quotas including IP moves, bounded rate maps/list responses, strict JSON and
endpoints, unsupported protocol filtering and exclusion of legacy/private/relay
routes. They do not establish cross-network Internet play or production service
availability.

### Tunnel hosting limits

The existing Quick Tunnel experiment requires official `cloudflared` on the
host. Cloudflare currently describes Quick Tunnels as testing/development only:
no account/domain is needed, but hostnames change, there is no uptime guarantee,
and the service has a 200 in-flight request cap. Stable production Cloudflare
Tunnel setup requires an account, domain and a server/VM. A production tunnel
would also require an explicitly designed endpoint policy beyond this
experiment's narrow `trycloudflare.com` allowlist.

Sources: [Cloudflare Quick Tunnels](https://developers.cloudflare.com/tunnel/get-started/quick-tunnels/)
and [Cloudflare Tunnel setup](https://developers.cloudflare.com/tunnel/get-started/).

## Game client and host setup

Everyone runs this experimental protocol-20 build. The gameplay wire format is
unchanged by public discovery; there are still at most four players. The host's
PC runs the authoritative game and requires official `cloudflared`. Guests do
not install cloudflared. No separate paid gameplay server is required, but the
shared directory and Cloudflare transport are still infrastructure dependencies.

After a directory operator has an approved, reachable HTTPS endpoint, each
computer explicitly enables the experiment and configures that **same** base
URL using `GUNMAYHEM_MASTER_URL` or the existing `master_server.txt` beside the
executable. A private-invite trial requires no directory. There is no shipped
public-directory default, account creation or automatic deployment.

An experimental portable package can include that approved `master_server.txt`
and an explicit **Start-Internet.bat** launcher. Players then launch that file
and browse the shared room list without typing a URL or setting environment
variables. The launcher opts in only for that game process; launching the plain
executable still defaults to LAN. `GUNMAYHEM_MASTER_URL` remains an advanced
override. The packager includes the config and launcher only when a bounded,
valid HTTPS config file exists; missing configuration creates neither a
placeholder endpoint nor an Internet launcher. Syntax validation does not prove
that an endpoint is approved, deployed or reachable. Native Windows launcher
execution and cross-network play still need acceptance testing.

In a development build with Internet explicitly enabled but no catalog, the
browser reports **CATALOG NOT CONFIGURED**. A failed catalog request reports
**CATALOG UNAVAILABLE - RETRY**, rather than claiming there are no rooms. A
successful empty catalog reports **NO OPEN ROOMS FOUND**; LAN results remain
available during catalog failures.

For operator/developer configuration without the packaged launcher:

```powershell
$env:GUNMAYHEM_EXPERIMENTAL_INTERNET = "1"
$env:GUNMAYHEM_MASTER_URL = "https://YOUR-APPROVED-DIRECTORY"
# Host only, if cloudflared is not already beside the game or in PATH:
$env:GUNMAYHEM_CLOUDFLARED = "C:\Tools\cloudflared.exe"
.\GunMayhem.exe
```

The URL above is a placeholder, not a live service. Directory URLs must use
HTTPS and contain no embedded credentials, query or fragment. The legacy
`GUNMAYHEM_ALLOW_INSECURE_MASTER` flag does not enable HTTP for public rooms.
The loopback HTTP server example earlier only exercises server development;
game-side TLS fixtures use explicitly trusted local test certificates.

1. Host: **CUSTOM GAME → ONLINE → CREATE SESSION → INTERNET**. The initial
   access choice is still **INVITE CODE**. Explicitly choose **OPEN**, then
   **CREATE SESSION**. The UI discloses public room publication
2. Configure **MODE → MAPS → SLOTS**, open **ONLINE PLAYER** slots, and wait for
   the footer to say **PUBLIC ROOM: LISTED**. Directory failure is shown there;
   a running tunnel alone does not mean the room is listed
3. Guests: **CUSTOM GAME → ONLINE → REFRESH**, select the **INTERNET** room and
   click **JOIN SESSION**. No invitation or link is required for that OPEN room
4. Host starts **START ONLINE MATCH**. Existing character configuration, READY,
   kick and manual rejoin rules apply. A kick closes that slot; it is not a
   durable identity ban, and an anonymous player can use another open slot
5. **CLOSE ROOM** on the host's SLOTS page ends the entire hosted session,
   stops admission, disconnects peers and delists. Leaving/restarting the host
   does the same. There is no host migration or automatic guest reconnect

Closing cancels and waits for heartbeat work, then sends authenticated DELETE.
The independently generated ID/ownership pair exists before the first POST,
so an initial request whose response is lost can still be tombstoned. If DELETE
fails, the directory's TTL removes the listing. Tunnel failure also stops new
admission and attempts delisting. Existing peers are not forcibly closed merely
because the directory is offline; if the directory expires and rejects a room
with 409, the host must create a new session rather than silently reopen it.

Private sessions never start the advertiser and have no public-room ID. Their
128-bit invitation authentication stays unchanged. Directory ownership uses a
different random 256-bit value, never a private game secret. Public guests need
no such secret: the public 128-bit ID in `/ws/room/<id>` identifies the intended
room, and a different ID/private route is rejected. Public listings are
unverified advertisements, not a host trust or identity certification.

Directory HTTP clients refuse all redirects, including 307/308, and bound
responses. The browser preserves fast LAN results during public-directory
failure. The new Internet route accepts only the exact validated WSS endpoint;
it cannot fall back to arbitrary TCP, QUIC or legacy relay addresses. Native
Origin gating is spoofable; OPEN is intentionally public admission, with the
existing handshake/frame/input/slot bounds still enforced.

## Integrated local test scope

The game tests run the real directory handler over local TLS and the real WSS
game handler behind another local TLS fixture. A dependency-injected dialer
maps a validated synthetic Cloudflare route to that fixture; it does not contact
Cloudflare or disable certificate verification. Coverage includes discovery,
host plus three guests, a full-room rejection, input and authoritative snapshots,
occupancy updates, kick/manual rejoin, wrong public ID/private path rejection,
private-session exclusion, configuration gates, entropy failure, redirects,
closing before a delayed initial registration, dead-tunnel delisting, expiry,
failed DELETE and preservation of LAN results during a directory outage.

These checks cannot establish Windows execution, real Internet latency,
Cloudflare availability, production anti-abuse capacity or competitive balance.
The separate manual two-computer, then four-player cross-network acceptance
gate remains required. No public endpoint has been deployed by these tests.
