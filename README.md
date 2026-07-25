# woodhouse-zigbee

A Woodhouse bridge for Zigbee devices via [zigbee2mqtt](https://www.zigbee2mqtt.io/), exposing
each paired device to Woodhouse and forwarding actions back through zigbee2mqtt. Device network
data can be sourced either over zigbee2mqtt's websocket API (default, more reliable/complete) or
over MQTT (`--use-mqtt`). A separate `zigbee-tool` CLI is included for poking at the zigbee2mqtt
websocket API directly.

## Usage

```sh
woodhouse-zigbee --store <path> --addr <woodhouse-server-addr> --web-addr <external-web-addr>
```

Key flags:

- `--store` (required) — path to local storage for pairing/config state
- `--addr` (required) — Woodhouse server address
- `--web-addr` (required) — external web server address (used to build device web URLs)
- `--id` — bridge ID (default `zigbee`)
- `--ws-addr` — zigbee2mqtt websocket server address (default `localhost:8080`)
- `--use-mqtt` — use the MQTT connection instead of websockets
- `--mqtt-server` — MQTT server address (default `mqtt://localhost:1883`)
- `--mqtt-topic` — MQTT root topic (default `zigbee2mqtt`)
- `--debug` / `-v` — enable debug logging

## Taskfile tasks

- `task build` — build the bridge binary into `build-<os>-<arch>/`
- `task run` — build and run the bridge (extra args after `--` are passed through)
- `task build-tool` — build the `zigbee-tool` CLI into `build-<os>-<arch>/`
- `task run-tool` — build and run the `zigbee-tool` CLI
- `task init-deploy` — create/populate `.env` with the deploy variables (`DEPLOY_HOST`,
  `DEPLOY_USER`, `DEPLOY_DIR`, `DEPLOY_GOOS`, `DEPLOY_GOARCH`)
- `task deploy` — build the bridge for the configured deploy target, rsync it to the remote
  host, and restart its systemd service

`build-component` and `build-and-deploy-component` are internal helpers shared by the tasks
above; they aren't meant to be run directly.
