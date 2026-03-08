# linear-cli

CLI for interacting with Linear's GraphQL API. Designed for both human and AI agent use.

## Setup

1. Set `LINEAR_API_KEY` in `~/.env` or the repo's `.env` file
2. Build and install:
   ```
   cd tools/linear-cli
   go build -o linear .
   cp linear /usr/local/bin/
   ```

## Commands

| Command | Description |
|---|---|
| `linear me` | Show current user info |
| `linear teams` | List all teams |
| `linear issues list` | List issues (with filters) |
| `linear issues view <ID>` | View issue details |
| `linear issues create` | Create a new issue |
| `linear issues update <ID>` | Update an existing issue |

## Common Flags

- `-o json` — JSON output (all commands)
- `--team ENG` — Filter by team key
- `--status "In Progress"` — Filter by status name
- `--assignee me` — Filter to your issues
- `--limit N` — Number of results (default 25)

## Man Pages

```
man linear
man linear-issues-list
```

Regenerate with: `linear gendocs --dir ./man`
