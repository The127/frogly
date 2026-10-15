# Services

`just dev` runs [process-compose](https://f1bonacc1.github.io/process-compose/)
with `process-compose.yml`. It starts frogly from the source, the frontend
with hot reload, and the services frogly needs:
[Keyline](https://github.com/The127/Keyline), which logs people in, and
Postgres. Only your own machine can reach them:

| Service | Address | Notes |
|---|---|---|
| frogly's frontend | `http://localhost:5173` | Reloads when you change a file in `web/`. Sends API requests on to frogly. |
| frogly | `http://localhost:8080` | The API, and the frontend as `just web` last built it. |
| Keyline | `http://localhost:8082` | Set up by `dev/keyline/config.yaml`. |
| Keyline's login pages | `http://localhost:5177` | |
| Postgres | `localhost:5432` | User, password and database are all `frogly`. |

frogly and the frontend run from the source. Restart frogly in
process-compose after you change Go code.

## The frontend in the binary

frogly's frontend lives in `web/`, and the frogly binary carries its build.
`just web` builds it, and `just build` runs that first. A binary built
without it serves the API but no frontend.

## Containers

Keyline and Postgres run in containers. Docker is the default. If you use
Podman, set `CONTAINER_CLI=podman`.

Keyline only publishes amd64 images. On a Mac with Apple silicon, install
Rosetta once with `softwareupdate --install-rosetta --agree-to-license`.

`just integration` and `just ci` need Keyline and Postgres too. They start
them in the background if they aren't running yet.

## Data

The data stays in the `frogly-postgres`, `frogly-keyline-data` and
`frogly-keyline-keys` volumes between runs. Keyline only reads
`dev/keyline/config.yaml` on its first start. After a change there, stop
`just dev`, remove its volumes with
`docker volume rm frogly-keyline-data frogly-keyline-keys`, and start
`just dev` again. A volume Keyline still uses can't be removed.
