# world-one

This repo is a rebuild/iteration on [sf0.org](sf0.org) that a friend and I
discussed and briefly attempted a couple years ago. My friend plays `sf0`, which
is essentially an ARG (not the virtual reality kind) mixed with a social media.
I'm revisiting this project for funsies.

## Developer Setup

- `.env` should contain environment variables for secrets and credentials since
   it is not checked in. Here's the list of variables to `export`:

   - `W1_PGHOST`
   - `W1_PGUSER`
   - `W1_PGPASSWORD`

- If you need to run database migrations, `source ./env` will produce a
`migrate` alias and a `W1_PGURL` environment variable.

## TODO

### GitHub

- Create ci/cd pipeline and have it pad PRs

### Application

- Eras
- Factions
- Players + Characters
    - B2C auth
- Etc

