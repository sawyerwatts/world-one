# world-one

This repo is a rebuild/iteration on [sf0.org](sf0.org) that a friend and I
discussed and briefly attempted a couple years ago. My friend plays `sf0`, which
is essentially an ARG (not the virtual reality kind) mixed with a social media.
I'm revisiting this project for funsies.

## Developer Setup

- `make stub-.env` will auto-generated a stubbed `.env` file. This file is in
the `.gitignore` and is to contain credentials, secrets, etc. As such, run that
recipe and initilize the variables.
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

