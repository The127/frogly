# Testing

frogly has four kinds of tests: unit tests in Go, integration tests that
run frogly as a binary, HTTP tests that describe the API, and frontend
tests. `just ci` runs all of them.

## Unit tests

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

## Integration tests

Integration tests live in `test/integration`. They build frogly, run it as
a binary against [Keyline](https://github.com/The127/Keyline), the service
that logs people in, and use it from the outside, the way a person or
an agent does. Each story gets a test there that shows it works. Run them
with `just integration`, which starts Keyline and Postgres first if they
aren't running.

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

A test logs in as one of two service users, `frogly-tests` and
`frogly-tests-2`, with the keys in `dev/keyline/test-service-user.pem` and
`dev/keyline/test-service-user-2.pem`. There are two, so a test can check
what one person can't see of another's. A third, `frogly-tests-no-role`
with `dev/keyline/test-service-user-no-role.pem`, has no frogly role in
Keyline, so frogly turns it away.

Keyline reads `dev/keyline/config.yaml` only on its first start. If you
change the config, delete Keyline's data once with
`docker volume rm frogly-keyline-data`, while it doesn't run. The files carry the `integration`
build tag, so `go test ./...` leaves them out.

Tests in `internal/` that need a running service, like the Postgres
adapter's, carry the same tag, and `just integration` runs them too. They
use the Postgres from `just dev` unless `TEST_POSTGRES_URL` names another
one. It doesn't start with `FROGLY_`, since frogly refuses variables of its
own prefix that it doesn't know.

## HTTP tests

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

## Frontend tests

The frontend's tests sit next to the code they test, like `web/src/api.ts`
and `web/src/api.test.ts`. They use [Vitest](https://vitest.dev/) and the
same three parts as the Go tests. Keep logic that needs a test out of the
Svelte components, in plain TypeScript modules. `just web-check` runs the
tests and checks the types.

## Tools

Most tools come from `mise.toml`. Go fetches the rest on its own:
[arch-go](https://github.com/arch-go/arch-go) checks the package
dependency rules in `arch-go.yml`, and
[govulncheck](https://go.dev/doc/security/vuln/) checks for known
vulnerabilities. Both are pinned in `hack/tools/go.mod`, so their
dependencies stay out of frogly's.

The one exception is Ginkgo. Its command line tool is pinned in frogly's
own `go.mod`, because it has to match the Ginkgo library the tests use.
