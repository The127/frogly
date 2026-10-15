# list available recipes
default:
    @just --list

# build the binary into bin/
build:
    CGO_ENABLED=0 go build -o bin/frogly ./cmd/frogly

# test
test:
    go test -race ./...

# test with a coverage profile
cover:
    go test -race -coverprofile=coverage.out -covermode=atomic ./...

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

# start the services for local development, see process-compose.yml
dev:
    process-compose up

# install the git hooks
hooks:
    lefthook install

# one-time setup after cloning: install the git hooks, then run every check
setup: hooks ci

# everything that must pass before a push
ci: lint arch reuse prose build cover vuln
