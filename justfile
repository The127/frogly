# list available recipes
default:
    @just --list

# build the binary into bin/, with the frontend in it
build: web
    CGO_ENABLED=0 go build -o bin/frogly ./cmd/frogly

# build the frontend into web/dist, which the binary embeds
web:
    pnpm --dir web install --frozen-lockfile
    pnpm --dir web build

# check the frontend's types and run its tests
web-check:
    pnpm --dir web install --frozen-lockfile
    pnpm --dir web check
    pnpm --dir web test

# test
test:
    go test -race ./...

# test with a coverage profile
cover:
    go test -race -coverprofile=coverage.out -covermode=atomic ./...

# run the integration tests, starting Keyline in the background first if
# it isn't running
integration: keyline web
    go tool ginkgo -tags integration -race ./test/integration

# start Keyline in the background unless it's running already, `just dev`
# starts it too
keyline:
    #!/usr/bin/env bash
    set -euo pipefail
    if ! curl -sf http://localhost:8082/oidc/frogly/.well-known/openid-configuration > /dev/null; then
        process-compose up keyline --detached
        process-compose project is-ready --wait
    fi

# lint
lint:
    golangci-lint run ./...

# format
fmt:
    golangci-lint fmt ./...

# check the package dependency rules in arch-go.yml
arch:
    go tool -modfile=hack/tools/go.mod arch-go

# describe the package dependency rules in prose
arch-describe:
    go tool -modfile=hack/tools/go.mod arch-go describe

# check for known vulnerabilities in reachable code
vuln:
    go tool -modfile=hack/tools/go.mod govulncheck ./...

# check that every file has license and copyright information
reuse:
    reuse lint

# check that no em dash is in the repository
prose:
    bash hack/check-prose.sh

# start Postgres, Keyline and frogly for local development, see
# process-compose.yml
dev:
    # `just integration` may have left Keyline running in the background,
    # and two process-composes can't share a port
    -process-compose down 2> /dev/null
    process-compose up

# install the git hooks
hooks:
    lefthook install

# one-time setup after cloning: install the git hooks, then run every check
setup: hooks ci

# everything that must pass before a push
ci: web lint arch reuse prose build web-check cover integration vuln
