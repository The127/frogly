# list available recipes
default:
    @just --list

# check that no em dash is in the repository
prose:
    bash hack/check-prose.sh

# everything that must pass before a push
ci: prose
