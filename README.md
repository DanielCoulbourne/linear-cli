# linear-cli

A lightweight CLI for Linear's GraphQL API. Designed for both humans and AI agents.

## Install

Requires [Go](https://go.dev/dl/) 1.21+.

```bash
# Clone and build
git clone https://github.com/DanielCoulbourne/linear-cli.git
cd linear-cli
go build -o linear .

# Install binary and man pages
cp linear /usr/local/bin/
mkdir -p /usr/local/share/man/man1
cp man/*.1 /usr/local/share/man/man1/
```

## Authentication

Create a [Linear API key](https://linear.app/settings/api) and set it as an environment variable:

```bash
export LINEAR_API_KEY=lin_api_xxxxx
```

Or add it to a `.env` file in your working directory (the CLI walks up the directory tree to find one).

## Commands

| Command | Description |
|---|---|
| `linear me` | Show current user info |
| `linear teams` | List all teams |
| `linear projects` | List all projects |
| `linear issues list` | List issues (with filters) |
| `linear issues view <ID>` | View issue details |
| `linear issues create` | Create a new issue |
| `linear issues update <ID>` | Update an existing issue |
| `linear initiatives list` | List initiatives |
| `linear initiatives view <name>` | View initiative details |
| `linear initiatives create` | Create an initiative |
| `linear initiatives update <name>` | Update an initiative |
| `linear initiatives link-project` | Link a project to an initiative |
| `linear initiatives unlink-project` | Unlink a project from an initiative |

## Common Flags

- `-o json` — JSON output (all commands)
- `--team ENG` — Filter by team key
- `--status "In Progress"` — Filter by status name
- `--assignee me` — Filter to your issues
- `--project "My Project"` — Set project (create/update)
- `--limit N` — Number of results (default 25)

## Examples

```bash
# List your in-progress issues
linear issues list --assignee me --status "In Progress"

# Create an issue
linear issues create --title "Fix login bug" --team ENG --priority 2 --project "Q1 Sprint"

# Update an issue's status
linear issues update ENG-123 --status "Done"

# List active initiatives
linear initiatives list --status Active

# Create an initiative and link a project
linear initiatives create --name "Q2 Platform Rebuild" --status Active
linear initiatives link-project --initiative "Q2 Platform Rebuild" --project "Auth Rewrite"

# JSON output for scripting / AI agents
linear issues list --team ENG -o json
```

## Man Pages

```bash
man linear
man linear-issues-list
man linear-issues-create
```

Regenerate with: `linear gendocs --dir ./man`

## AI Agent Usage

This CLI is designed to be called from AI agents via bash. Use `-o json` for machine-readable output. The CLI loads `.env` files automatically, so agents running from a project directory with a `.env` file don't need any extra configuration.
