# blog-aggregator

A CLI RSS aggregator ("gator") built in Go as a guided project on boot.dev. It fetches posts from feeds you follow and stores them in Postgres so you can browse them later.

## Requirements

- [Go](https://go.dev/doc/install) 1.24+
- [PostgreSQL](https://www.postgresql.org/download/)

## Installation

```
go install github.com/Zallu35/blog-aggregator@latest
```

This builds a `blog-aggregator` binary and installs it to your `$GOPATH/bin` (make sure that directory is on your `PATH`).

## Setup

1. Create a Postgres database, e.g. `gator`.
2. Run the migrations in `sql/schema` against it using [goose](https://github.com/pressly/goose):

   ```
   goose -dir sql/schema postgres "postgres://user:password@localhost:5432/gator?sslmode=disable" up
   ```

3. Create a config file at `~/.gatorconfig.json`:

   ```json
   {
     "db_url": "postgres://user:password@localhost:5432/gator?sslmode=disable",
     "current_user_name": ""
   }
   ```

## Usage

Run commands as `blog-aggregator <command> [args]` (or `go run . <command> [args]` from inside the repo).

- `register <name>` — create a new user and log in as them
- `login <name>` — switch the current user
- `users` — list all users, marking the current one
- `addfeed <name> <url>` — add a feed and automatically follow it
- `feeds` — list all feeds and who added each one
- `follow <url>` — follow an existing feed
- `unfollow <url>` — stop following a feed
- `following` — list the feeds you're following
- `agg <duration>` — continuously fetch and store posts from feeds (e.g. `agg 1m`); leave this running in its own terminal
- `browse [limit]` — print your most recent posts (default limit: 2)
- `reset` — delete all users (cascades to their feeds and follows)
