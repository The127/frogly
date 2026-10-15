# list available recipes
default:
    @just --list

# build the binary into bin/, with the frontend in it
build: web
    CGO_ENABLED=0 go build -o bin/frogly ./cmd/frogly

# install the frontend's packages in the versions pnpm-lock.yaml names
web-deps:
    pnpm --dir web install --frozen-lockfile

# build the frontend into web/dist, which the binary embeds
web: web-deps
    pnpm --dir web build

# check the frontend's types and run its tests
web-check: web-deps
    pnpm --dir web check
    pnpm --dir web test

# test
test:
    go test -race ./...

# test with a coverage profile
cover:
    go test -race -coverprofile=coverage.out -covermode=atomic ./...

# run the integration tests, starting Keyline first if it isn't running
integration: keyline web
    go tool ginkgo -tags integration -race ./test/integration

# start Keyline in the background unless it's running already
keyline:
    #!/usr/bin/env bash
    set -euo pipefail
    ready() { curl -sf --max-time 5 http://localhost:8082/oidc/frogly/.well-known/openid-configuration > /dev/null; }
    ready && exit 0
    process-compose up keyline --detached
    # pulling the image on a first run takes a while, but a Keyline that
    # never gets ready mustn't hold the tests forever
    for _ in $(seq 60); do
        ready && exit 0
        sleep 5
    done
    echo "Keyline didn't get ready within five minutes, see: process-compose process logs keyline" >&2
    exit 1

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

# start frogly and everything it needs, see process-compose.yml
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
ci: lint arch reuse prose build web-check cover integration vuln
