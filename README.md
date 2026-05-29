# MelodyGuard

A Discord bot for server verification and role management.

## Features

- **Auto-role** — new members get `Unverified` role on join
- **Button Verify** — click a button to verify, no slash commands needed
- **Welcome message** — automatic welcome embed in system channel with verify button
- **Auto-cleanup** — automatically kicks unverified members after a configurable timeout

## Requirements

- Go 1.24+
- Redis
- Discord Bot Token with **Server Members Intent** enabled

## Setup

1. **Create a Discord application** at https://discord.com/developers/applications
2. Go to **Bot** → enable **Server Members Intent** under Privileged Gateway Intents
3. Copy the bot token
4. Invite the bot to your server with these scopes and permissions:

**Scopes:** `bot` `applications.commands`

**Permissions:** `Manage Roles` `Kick Members` `Send Messages` `Manage Messages` `Read Message History`

5. Clone the repo and copy the env file:

```sh
cp .env.sample .env
```

6. Edit `.env` with your Discord token and Redis address

7. Run the bot:

```sh
go run ./cmd
```

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `DISCORD_TOKEN` | — | Discord bot token (required) |
| `REDIS_ADDRESS` | `localhost:6379` | Redis server address |
| `REDIS_PASSWORD` | — | Redis password |
| `REDIS_DB` | `0` | Redis database number |
| `VERIFIED_ROLE_NAME` | `Verified` | Name of the verified role |
| `UNVERIFIED_ROLE_NAME` | `Unverified` | Name of the unverified role |
| `CLEANUP_ENABLED` | `true` | Enable auto-kick of unverified users |
| `CLEANUP_INTERVAL_MINUTES` | `30` | How often to check for expired users |
| `CLEANUP_MAX_AGE_HOURS` | `48` | Max hours before unverified users are kicked |

## Commands

- `/verify` — Shows a verify button (ephemeral)
- `/help` — Shows available commands
