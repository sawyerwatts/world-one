# world-one

This repo is a rebuild/iteration on [sf0.org](sf0.org) that a friend and I
discussed and briefly attempted a couple years ago. My friend plays `sf0`, which
is essentially an ARG (not the virtual reality kind) mixed with a social media.
I'm revisiting this project for funsies.

Check out the wiki!

## Developer Setup

- `make stub-.env` will auto-generated a stubbed `.env` file. This file is in
the `.gitignore` and is to contain credentials, secrets, etc. As such, run that
recipe and initilize the variables.
- If you need to run database migrations, `source ./.env` will produce a
`migrate` alias and a `W1_PGURL` environment variable.
- When running locally, `http://localhost:8080/v1` is the default webpage to the
  Scalar UI.

## TODO

- Breakout features

### GitHub

- Create ci/cd pipeline and have it pad PRs

### Azure

- Identity
   - [Clerk?](https://clerk.com/?utm_source=fireship&utm_medium=youtube&utm_campaign=libsql)
   - Add to the following built endpoints:
      - N/A
- Compute
- DB

### Application

- Eras
- Factions
- Players + Characters
    - Start ERD now that there's related tables
- Etc

## Installation

### Database

1. The application is written to assume that the database's local timezone
   (which is used to auto-convert stored UTC datetimes into local timezone
   appropriate values) is configured to UTC. To do this at a database, level,
   use `# show config_file` to get the absolute path to `postgresql.conf`, and
   then update all timezones to `'GMT'`.
2. Apply the migrations.

### Web API

- The following environment variables should be set when deployed:
   - `W1_PGURL`: the URL of DB.
   - `GIN_MODE=release`: instruct Gin to operate in release mode.

### Azure Responsibilities

Some responsibilities are delegated to the hypothetical Azure deployment.

- Azure Monitor
   - Redirect logs from STDOUT to a persistent location
   - Alert when a 500 occurs
- Principle
   - Don't use basic auth in DB connection string! Health checks may leak it

