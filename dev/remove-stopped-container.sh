#!/usr/bin/env bash
# Removes the container named $1 when an earlier run left it behind, so a
# new one can take its name. A running one belongs to another `just dev`
# and stays. It asks for the container by its exact name, since some Docker
# APIs, like socktainer's, ignore `docker ps --filter`.
set -uo pipefail

cli="${CONTAINER_CLI:-docker}"

status=$("$cli" inspect --format '{{.State.Status}}' "$1" 2> /dev/null) || exit 0

if [ "$status" != "running" ]; then
    "$cli" rm "$1" > /dev/null
fi
