# list available recipes
default:
    @just --list

# check that no em dash is in the repository
prose:
    bash hack/check-prose.sh

# install the git hooks
hooks:
    lefthook install

# one-time setup after cloning: install the git hooks, then run every check
setup: hooks ci

# everything that must pass before a push
ci: prose
