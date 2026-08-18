# Multiplayer (main branch)

The developed `main` branch includes an experimental host-authoritative two-player network mode.

## How it works

- The host runs the real game simulation at 30 TPS.
- Player 1 is controlled locally on the host.
- The remote client controls Player 2.
- The client sends only its current movement/shoot/grenade input.
- The host decides movement, bullets, hits, crates, powerups, deaths and win state, then streams authoritative snapshots back to the client.
- The client renders those snapshots instead of running a second competing combat simulation.

This architecture is intentional: random crates/powerups and Flash-derived edge cases stay authoritative on one machine rather than requiring two machines to produce exactly the same random simulation.

## Host

The normal UI now has a **MULTIPLAYER** button on the main menu. Choose **HOST GAME** to listen on TCP port `7777`, then use the normal Custom Game setup to choose mode, map, lives and players. Player 1 is local and Player 2 is reserved for the remote client.

The command line remains available for development or a custom port:

```text
GunMayhem.exe --host
GunMayhem.exe --host --port 9000
```

## Join

On the second computer, open **MULTIPLAYER**, enter the host as `IP:port` (for example `192.168.1.20:7777`) and press **JOIN GAME**. The client shows a waiting screen until the host starts gameplay.

The command-line equivalent is:

```text
GunMayhem.exe --join 192.168.1.20:7777
```

If only a hostname/IP is supplied by the internal API, port `7777` is assumed.

The client uses its local **Player 1** key bindings to control the host's **Player 2**. This lets each computer use a normal single-player keyboard layout instead of requiring Player 2's local-host key layout.

While the host is in menus, the client shows a waiting screen. Once the host enters gameplay, the client switches to the authoritative game view automatically.

## Internet play

The current transport is direct TCP. For play outside the same LAN, the host must currently be reachable on the selected TCP port (for example by router port forwarding or a VPN/overlay network). A relay/lobby service can be added later without changing the host-authoritative game model.

## Current scope

This is the first working network layer, intended to stabilize simulation and synchronization before Gun Mayhem 2/custom mission content is added. Host/Join is available directly from the game menu; command-line flags remain as a development convenience.
