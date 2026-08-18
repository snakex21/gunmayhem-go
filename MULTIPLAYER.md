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

Start the game with:

```text
GunMayhem.exe --host
```

The default TCP port is `7777`. A different port can be selected with:

```text
GunMayhem.exe --host --port 9000
```

Then use the normal game UI on the host and start a Custom Game (or a co-op Campaign setup) with Player 2 enabled. Until a client connects, the network-controlled Player 2 receives no input.

## Join

On the second computer:

```text
GunMayhem.exe --join 192.168.1.20:7777
```

Replace `192.168.1.20` with the host's IP address. If only a hostname/IP is supplied by the internal API, port `7777` is assumed.

The client uses its local **Player 1** key bindings to control the host's **Player 2**. This lets each computer use a normal single-player keyboard layout instead of requiring Player 2's local-host key layout.

While the host is in menus, the client shows a waiting screen. Once the host enters gameplay, the client switches to the authoritative game view automatically.

## Internet play

The current transport is direct TCP. For play outside the same LAN, the host must currently be reachable on the selected TCP port (for example by router port forwarding or a VPN/overlay network). A relay/lobby service can be added later without changing the host-authoritative game model.

## Current scope

This is the first working network layer, intended to stabilize simulation and synchronization before Gun Mayhem 2/custom mission content is added. The next UI step is exposing Host/Join directly in the game menus instead of requiring command-line flags.
