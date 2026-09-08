# Terraria (Wii U) NEX server

A NEX server replacement for Terraria's Wii U online multiplayer, written
from scratch: no reference server implementation for this title exists
anywhere (unlike most other Wii U titles Pretendo has full or partial
servers for).

`game_server_id` `10198f00` (270110464 decimal), title ID
`0005000010198f00`. Access key and build/branch strings come from
[kinnay.github.io](https://kinnay.github.io/)'s public Wii U NEX game
database.

## What this implements

Static analysis of the game's own binary (RPX) shows it links Nintendo's
official `nn::nex` SDK wrapper using only the **standard, generic**
protocol set - `MatchMakingClient`, `MatchmakeExtensionClient`,
`UtilityClient`, `JobNNIDLogin`/`JobCafeLogin` - with no bespoke
per-title RMC protocol anywhere in the binary. This server therefore
registers only the generic protocol set real traffic exercises:
`SecureConnection`, `NATTraversal`, `MatchMaking` (including the hand-written
`GetDetailedParticipants` handler - see `nex/get_detailed_participants.go`,
not auto-wired by `nex-protocols-common-go`), `MatchMakingExt`, and
`MatchmakeExtension`. `DataStore` and `Ranking` are intentionally left
unregistered: nothing in the game's linked symbols or strings points at
either being used by this title.

## Account model

There is no local account database and no account gRPC service to run.
`globals/password_from_pid.go` derives each PID's NEX password directly
from a shared HMAC secret (`PN_TERRARIA_NEX_PASSWORD_SECRET`) - whatever
issues this title's NEX token needs to derive the password the exact same
way, so both sides agree without ever storing or exchanging it.

## Configuration

All configuration is environment variables (a local `.env` file works too,
see `.env.example`):

| Variable | Purpose |
|---|---|
| `PN_TERRARIA_KERBEROS_PASSWORD` | Password for the two system Kerberos accounts (defaults to `password` if unset - fine for testing, change it for anything real) |
| `PN_TERRARIA_NEX_PASSWORD_SECRET` | Hex-encoded, at least 32 bytes. Must match whatever issues this title's NEX token |
| `PN_TERRARIA_AUTHENTICATION_SERVER_PORT` | UDP port for the authentication (ticket-granting) server |
| `PN_TERRARIA_SECURE_SERVER_HOST` | Hostname/IP advertised to clients for the secure server |
| `PN_TERRARIA_SECURE_SERVER_PORT` | UDP port for the secure server |
| `PN_TERRARIA_POSTGRES_URI` | Postgres connection string (matchmaking state) |

## Running

```sh
go build -o terraria-nex-server .
./terraria-nex-server
```

Needs a Postgres database reachable at `PN_TERRARIA_POSTGRES_URI` -
`nex-protocols-common-go`'s `MatchmakingManager` creates its own schema on
first connect, nothing to migrate by hand.
