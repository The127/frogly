# Contributing to frogly

Thanks for your interest in contributing. This guide gets you from cloning
the repository to an open pull request. The details of frogly's internals
are in [docs/development](#learning-more).

## Before you start

For bug fixes and small changes, open a pull request directly. For larger
changes or new features, open an issue first and describe what you want to
do. This way we can agree on the approach before you spend time on it.

## Setting up the repository

frogly pins its development tools in `mise.toml`. First, install
[mise](https://mise.jdx.dev/) and
[activate it in your shell](https://mise.jdx.dev/getting-started.html#activate-mise).
Then run this once after you clone:

```sh
mise install
just setup
```

`mise install` installs Go, just, pnpm and the other tools in the versions
that `mise.toml` names. `just setup` installs the git hooks and runs every
check. Run `just` to list all recipes.

You also need Docker or Podman, since frogly's services run in containers.
If you use Podman, set `CONTAINER_CLI=podman`.

## Running frogly

Start frogly with everything it needs:

```sh
just dev
```

Then open <http://localhost:5173> and log in as `karo` with the password
`frogly`. The frontend reloads when you change a file in `web/`. Restart
frogly in process-compose after you change Go code.

[Services](docs/development/services.md) says what `just dev` starts and
how to reset its data.

## Making a change

1. Fork the repository and create a branch for your change.
2. Make your change in small commits, each signed off.
3. Run `just ci`. It runs every check, including the integration tests.
4. Open a pull request against `main`.
5. A maintainer reviews the pull request and may ask for changes.
6. Once it is approved, a maintainer squash merges it.

[Testing](docs/development/testing.md) says how to write the tests.

## Commit messages and pull request titles

Commit messages follow [Conventional Commits](https://www.conventionalcommits.org/)
without a scope, like `fix: keep the order of the backlog after a move`.
The types are `build`, `chore`, `ci`, `docs`, `feat`, `fix`, `perf`,
`refactor`, `revert` and `test`. For a breaking change, add `!` after the
type, for example `feat!: remove the v1 API`.

Pull requests are squash merged, so the pull request title becomes the
commit message on `main`. Give it the same form. The git hooks check your
commit messages, and a check on every pull request checks its title.

## Signing off

All contributions must be signed off under the
[Developer Certificate of Origin](https://developercertificate.org/) (DCO).
By signing off, you confirm that you wrote the change or have the right to
submit it under the project's license. Add `-s` to `git commit`:

```sh
git commit -s -m "fix: keep the order of the backlog after a move"
```

This adds a `Signed-off-by` line with your name and email, which must match
the commit author. The git hooks and a check on every pull request reject
commits without it. If you forgot, sign off all commits on your branch and
update the pull request:

```sh
git rebase --signoff main
git push --force-with-lease
```

## Using AI tools

You may use AI tools to write code and text for frogly. A change made with
an AI tool follows the same rules as any other change.

An AI tool can't certify the DCO, so it must not add a `Signed-off-by`
line. Before you sign off, review the change, make sure its license fits
the project and take full responsibility for it.

If an AI tool helped write a commit, add an `Assisted-by` line with the
tool and the exact model. After them you may list analysis tools that
helped, like a linter, but not basic ones like git or your editor:

```
Assisted-by: Claude:claude-opus-5-5
```

Don't add a `Co-Authored-By` line for an AI, and don't link to an AI
session. The squash merge keeps the commit messages from your branch, so
the line ends up on `main`.

## Learning more

When you work on frogly's internals, these pages explain how they fit
together:

- [Services](docs/development/services.md): what `just dev` starts, the
  addresses, Keyline's setup and its data.
- [Testing](docs/development/testing.md): unit, integration, HTTP and
  frontend tests, and the tools behind them.
- [The data layer](docs/development/data-layer.md): how frogly stores its
  data and how to add a model.

## License

By contributing, you agree that your contributions are licensed under the
[GNU Affero General Public License v3.0 or later](LICENSE).
