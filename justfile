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

# run the integration tests, starting Keyline and Postgres first if needed
integration: containers services web
    go test -race -tags integration ./internal/...
    go tool ginkgo -tags integration -race ./test/integration

# start Keyline and Postgres in the background unless they run already
services:
    #!/usr/bin/env bash
    set -euo pipefail
    cli="${CONTAINER_CLI:-docker}"
    keyline() { curl -sf --max-time 5 http://localhost:8082/oidc/frogly/.well-known/openid-configuration > /dev/null; }
    # over TCP, since Postgres's first start runs a temporary server that
    # only listens on a socket
    postgres() { "$cli" exec frogly-postgres pg_isready -h 127.0.0.1 -U frogly -d frogly > /dev/null 2>&1; }
    missing=()
    keyline || missing+=(keyline)
    postgres || missing+=(postgres)
    ((${#missing[@]} == 0)) && exit 0
    # a second process-compose can't take the port of one that already
    # runs, so the missing services start inside the running one. Starting
    # a running service again would clash with its container.
    if process-compose process list > /dev/null 2>&1; then
        for service in "${missing[@]}"; do
            process-compose process start "$service"
        done
    else
        process-compose up "${missing[@]}" --detached
    fi
    # pulling the images on a first run takes a while, but a service that
    # never gets ready mustn't hold the tests forever
    deadline=$((SECONDS + 300))
    until keyline && postgres; do
        if ((SECONDS >= deadline)); then
            echo "Keyline or Postgres didn't get ready within five minutes, see: process-compose process logs keyline (or postgres)" >&2
            exit 1
        fi
        sleep 5
    done

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
dev: containers
    @# `just integration` may have left Keyline running in the background,
    @# and two process-composes can't share a port
    -@process-compose down > /dev/null 2>&1
    process-compose up

# check that Docker or Podman answers, the services run in containers
[private]
containers:
    #!/usr/bin/env bash
    set -euo pipefail
    cli="${CONTAINER_CLI:-docker}"
    if ! "$cli" info > /dev/null 2>&1; then
        echo "$cli doesn't answer. Is it running, and does DOCKER_HOST point to it?" >&2
        "$cli" info 2>&1 | tail -3 >&2 || true
        exit 1
    fi

# install the git hooks
hooks:
    lefthook install

# one-time setup after cloning: install the git hooks, then run every check
setup: hooks ci

# everything that must pass before a push
ci: lint arch reuse prose build web-check cover integration vuln
