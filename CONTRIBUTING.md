# Contributing to frogly

Thanks for your interest in contributing. This guide explains how to set up
the repository, how to propose a change and what we look for in a pull
request.

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

`mise install` installs [Go](https://go.dev/),
[golangci-lint](https://golangci-lint.run/), [just](https://just.systems/),
[lefthook](https://lefthook.dev/), [reuse](https://reuse.software/),
[process-compose](https://f1bonacc1.github.io/process-compose/) and
[Hurl](https://hurl.dev/) in the versions that `mise.toml` names. `just setup` installs the git hooks and
runs every check.

Go fetches the two remaining tools on its own:
[arch-go](https://github.com/arch-go/arch-go) checks the package
dependency rules in `arch-go.yml`, and
[govulncheck](https://go.dev/doc/security/vuln/) checks for known
vulnerabilities. Both are pinned in `hack/tools/go.mod`, so their
dependencies stay out of frogly's. The one exception is
[Ginkgo](https://onsi.github.io/ginkgo/), which runs the integration tests.
Its command line tool is pinned in frogly's own `go.mod`, because it has to
match the Ginkgo library the tests use.

Run `just` to list all recipes. `just ci` runs every check. Its
integration tests need Keyline, so it starts Keyline in the background if
it isn't running yet. That needs Docker or Podman (see
[Running the services](#running-the-services)).

## Running the services

frogly needs Postgres and [Keyline](https://github.com/The127/Keyline), the
service that logs people in, during development. Start both with:

```sh
just dev
```

This runs [process-compose](https://f1bonacc1.github.io/process-compose/)
with `process-compose.yml`, which starts everything in containers. Only your
own machine can reach them:

| Service | Address | Notes |
|---|---|---|
| Postgres | `localhost:5432` | User, password and database are all `frogly`. |
| Keyline | `http://localhost:8082` | Set up by `dev/keyline/config.yaml`. |
| Keyline's login pages | `http://localhost:5177` | Log in as `karo` with the password `frogly`. |

The data stays in the `frogly-postgres`, `frogly-keyline-data` and
`frogly-keyline-keys` volumes between runs.

You need Docker or Podman for this. If you use Podman, set
`CONTAINER_CLI=podman`. Keyline only publishes amd64 images. On a Mac with
Apple silicon, install Rosetta once with
`softwareupdate --install-rosetta --agree-to-license`.

## Making a change

1. Fork the repository and create a branch for your change.
2. Make your change in small commits. Sign off every commit (see
   [Developer Certificate of Origin](#developer-certificate-of-origin)).
3. Run `just ci`.
4. Open a pull request against `main`.
5. A maintainer reviews the pull request and may ask for changes.
6. Once it is approved, a maintainer squash merges it.

## Writing tests

Tests live in an external test package, like `package config_test` for
`config`, so they use a package the way its callers do. `just test` runs
them with the race detector.

Each test has three parts, marked with a comment:

```go
func TestVariablesOverrideDefaults(t *testing.T) {
	// arrange
	environ := []string{"FROGLY_LOG__LEVEL=debug"}

	// act
	loaded, err := config.Load(environ)

	// assert
	require.NoError(t, err)
	assert.Equal(t, slog.LevelDebug, loaded.Log.Level)
}
```

Leave out `// arrange` when there is nothing to set up. Checks use
[testify](https://github.com/stretchr/testify): `require` when the test
can't go on after a failure, `assert` otherwise. Checks on an error always
use `require`, and the linter enforces that.

### Integration tests

Integration tests live in `test/integration`. They build frogly, run it as
a binary against Keyline, and use it from the outside, the way a person or
an agent does. Each story gets a test there that shows it works. Run them
with `just integration`, which starts Keyline first if it isn't running.

They use [Ginkgo](https://onsi.github.io/ginkgo/) and
[Gomega](https://onsi.github.io/gomega/). `Describe`, `Context` and `It`
say what the test is about, so they need no arrange, act and assert
comments:

```go
var _ = Describe("seeing your name", func() {
	Context("when you are logged in", func() {
		It("greets you by name", func(ctx SpecContext) {
			// ...
		})
	})
})
```

A test logs in as the service user `frogly-tests`, with the key in
`dev/keyline/test-service-user.pem`. The files carry the `integration`
build tag, so `go test ./...` leaves them out.

### HTTP tests

Requests to the API and the answers they get are described in
[Hurl](https://hurl.dev/) files in `test/http`, one file per endpoint:

```hurl
GET {{frogly}}/api/me
Authorization: Bearer {{token}}
HTTP 200
[Asserts]
jsonpath "$.name" == "frogly-tests"
```

The integration tests run them against the frogly they start, so `frogly`,
`issuer` and the service user's `token` are set for you.

## Commit messages and pull request titles

Commit messages follow [Conventional Commits](https://www.conventionalcommits.org/)
without a scope:

```
<type>: <description>
```

For example:

- `feat: add sprints`
- `fix: keep the order of the backlog after a move`
- `docs: explain how to run the tests`

Allowed types are `build`, `chore`, `ci`, `docs`, `feat`, `fix`, `perf`,
`refactor`, `revert` and `test`. For a breaking change, add `!` after the
type, for example `feat!: remove the v1 API`.

The git hooks reject a commit message that doesn't follow this form.
Pull requests are squash merged, so the pull request title becomes the
commit message on `main`. Give it the same form. A check on every pull
request validates the title.

## Developer Certificate of Origin

All contributions must be signed off under the
[Developer Certificate of Origin](https://developercertificate.org/) (DCO).
By signing off, you confirm that you wrote the change or have the right to
submit it under the project's license.

To sign off a commit, add `-s` to `git commit`:

```sh
git commit -s -m "fix: keep the order of the backlog after a move"
```

This adds a line like this to the commit message:

```
Signed-off-by: Jane Doe <jane@example.com>
```

The email in this line must match the commit author. The git hooks reject a
commit that isn't signed off, and a check on every pull request makes sure
all its commits are signed off.

If you forgot to sign off, you can add the sign-off to all commits on your
branch and update the pull request:

```sh
git rebase --signoff main
git push --force-with-lease
```

## Using AI tools

You may use AI tools to write code and text for frogly. A change made with
an AI tool follows the same rules as any other change.

An AI tool can't certify the
[Developer Certificate of Origin](#developer-certificate-of-origin), so it
must not add a `Signed-off-by` line. Before you sign off, review the change,
make sure its license fits the project and take full responsibility for it.

If an AI tool helped write a commit, add an `Assisted-by` line to the commit
message:

```
Assisted-by: AGENT_NAME:MODEL_VERSION [TOOL1] [TOOL2]
```

`AGENT_NAME` is the AI tool and `MODEL_VERSION` is the exact model it used.
After them you may list analysis tools that helped, like a linter or a
static analyzer. Don't list basic tools like git, the compiler or your
editor. For example:

```
Assisted-by: Claude:claude-opus-5-5
```

Don't add a `Co-Authored-By` line for an AI, and don't link to an AI
session.

The squash merge keeps the commit messages from your branch, so this line
ends up on `main`.

## License

By contributing, you agree that your contributions are licensed under the
[GNU Affero General Public License v3.0 or later](LICENSE).
