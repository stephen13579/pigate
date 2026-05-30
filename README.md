# PiGate

PiGate controls a physical gate from a Raspberry Pi. The current architecture keeps
the gate device private, uses a small cloud VM for shared infrastructure, and lets
trusted machines connect through Tailscale instead of public database or MQTT ports.

## High-Level Architecture

```text
                 Tailnet Maintenance Access
                 Tailscale private network

  Home/Admin PC  ------------------------------+
                                               |
  Windows Credential Server  ------------------+----  Cloud Control Plane Host
                                               |      Ubuntu droplet
  Raspberry Pi Gate Controller  ---------------+      - EMQX MQTT broker
                                                      - PostgreSQL database

  Local development
  - Docker/Podman Compose can run MQTT and Postgres locally for testing.
```

The droplet is not the gate controller. It is the private control plane host for
shared services. The Raspberry Pi remains the device that physically opens and
closes the gate.

## Services And Instances

### Cloud Control Plane Host

The cloud control plane host is a small Ubuntu VM, currently expected to run on a
provider such as DigitalOcean. It runs:

- PostgreSQL on port `5432`
- EMQX MQTT on port `1883`
- EMQX dashboard on port `18083`
- PiGate status page on the configured private HTTP address, for example `8090`
- Optional EMQX WebSocket MQTT on port `8083` for browser-based MQTT tools

These services should bind to the droplet's Tailscale IP, not its public internet
address. The intended deployment files live in [deploy/cloud](deploy/cloud).

PostgreSQL is the durable source of truth for credentials, access-time data, and
reported status.
EMQX is the message bus used for lightweight notifications and commands.
The PiGate status page subscribes to MQTT status topics, writes status events to
PostgreSQL, and publishes gate commands back to MQTT.

### Raspberry Pi Gate Controller

The Raspberry Pi runs the `gatecontroller` binary. It is responsible for:

- Reading keypad input.
- Validating credentials against a local SQLite database.
- Driving GPIO pins for the gate relay and status LED.
- Connecting to MQTT for gate commands and credential update notifications.
- Pulling credential and access-time data from PostgreSQL into local SQLite.

The Pi keeps a local SQLite cache so normal gate operation does not require a
database round trip for every keypad entry. On startup, every 24 hours, and when
it receives a credential update notification over MQTT, it syncs from Postgres.

Relevant config:

```text
pigate/configs/gatecontroller-config.toml
```

Important runtime environment variables:

```text
PIGATE_DB_PASSWORD
PIGATE_MQTT_PASSWORD
```

### Windows Credential Server

The Windows machine runs the `credentialserver` service. It is responsible for:

- Watching a configured folder for credential text file changes.
- Parsing credential data.
- Writing credential changes to PostgreSQL.
- Publishing an MQTT notification when updated credentials are ready.

The credential server keeps a long-lived MQTT connection while the service is
running. It does not connect and disconnect for each notification.

Relevant config:

```text
pigate/configs/credentialserver-config.toml
```

Important runtime environment variables:

```text
PIGATE_DB_PASSWORD
PIGATE_MQTT_PASSWORD
```

### Home/Admin Computer

The admin computer joins the same Tailscale network and can reach the droplet's
private Tailscale IP. It is used for:

- SSH maintenance.
- EMQX dashboard access.
- PiGate status page access.
- Database inspection.
- Manual MQTT testing with tools such as `mosquitto_pub`.

Example manual gate command:

```powershell
& "C:\Program Files\mosquitto\mosquitto_pub.exe" -h 100.65.247.9 -p 1883 -t "pigate-speedway-self-storage/pigate/command" -m "open" -q 1
```

## Data Flow

### Credential Update Flow

1. A credential file changes on the Windows credential server.
2. `credentialserver` parses the file.
3. `credentialserver` writes credentials to PostgreSQL on the control plane host.
4. `credentialserver` publishes:

   ```text
   Topic: <location-id>/credentials/status
   Payload: update_available
   ```

5. `gatecontroller` receives the MQTT notification.
6. `gatecontroller` pulls updated credentials and access times from PostgreSQL.
7. `gatecontroller` stores the data in local SQLite.

### Gate Operation Flow

For keypad access:

1. A user enters a code on the keypad.
2. `gatecontroller` validates the code against local SQLite.
3. If valid and allowed by access-time rules, the Pi triggers the GPIO relay.

For MQTT command access:

1. A trusted MQTT client publishes to:

   ```text
   Topic: <location-id>/pigate/command
   Payload: open | close | hold_open
   ```

2. `gatecontroller` receives the command.
3. The Pi triggers the matching gate action.

## MQTT Topics

The current topic namespace is based on `LOCATION_ID`.

```text
<location-id>/credentials/status
<location-id>/pigate/command
<location-id>/pigate/status
```

Known payloads:

```text
credentials/status: update_available
pigate/command:     open, close, hold_open
pigate/status:      opened, locked_open, closed
```

## PiGate Status Page

The status page is served by:

```text
pigate/cmd/statusserver
```

It is intended to run on the Control Plane Host and bind to the host's Tailscale
IP, for example:

```toml
HTTP_ADDR = "100.x.y.z:8090"
```

The page has no username/password yet and should only be reachable through
Tailnet Maintenance Access. It shows current MQTT/Postgres reachability, the
latest gate status, credential update status, and the last gate command. The
Open, Lock Open, and Close buttons publish to:

```text
<location-id>/pigate/command
```

The status server creates these PostgreSQL tables when it starts:

```text
pigate_status_events
pigate_status_latest
```

## Security Model

The baseline security model is:

- Do not expose PostgreSQL or MQTT publicly.
- Bind cloud services to the droplet's Tailscale IP.
- Use Tailscale for maintenance and client connectivity.
- Use EMQX MQTT authentication for application clients.
- Keep the unauthenticated PiGate status page bound to the Tailscale IP.
- Keep database and MQTT passwords in environment variables, not source control.
- Keep the Raspberry Pi outbound-only from a public internet perspective.

The EMQX dashboard password is separate from MQTT client credentials. Dashboard
login working does not prove that MQTT client authentication is configured.

## GitHub Actions Deployment

The recommended deployment path is one self-hosted GitHub Actions runner per
runtime host:

```text
Control Plane Host          self-hosted, Linux, pigate-control-plane
Windows Credential Server   self-hosted, Windows, pigate-credentialserver
Raspberry Pi Gate Controller self-hosted, Linux, pigate-gatecontroller
```

GitHub environment secrets hold passwords, while the checked-in TOML files keep
non-secret defaults. The deploy workflows write service runtime values such as
`PIGATE_DB_PASSWORD`, `PIGATE_MQTT_PASSWORD`, `PIGATE_DB_HOST`, and
`PIGATE_MQTT_BROKER` before restarting each service.

Deployment details, runner prerequisites, and the required GitHub environment
variables are documented in [deploy/github-actions.md](deploy/github-actions.md).

## Local Development

Local development can still use Compose to run MQTT and PostgreSQL on a developer
machine. The older local examples live in:

```text
mqtt_broker/docker-compose.yml
postgres_broker/docker-compose.yml
```

The cloud-oriented Compose stack lives in:

```text
deploy/cloud/docker-compose.yml
```

Use local broker/database addresses for development configs and the droplet's
Tailscale IP or MagicDNS name for deployed configs.

## Build

From the Go module directory:

```bash
cd pigate
go build ./cmd/gatecontroller
go build ./cmd/credentialserver
go build ./cmd/statusserver
```

For Raspberry Pi deployment, build the gate controller for Linux on the Pi or
cross-compile for the Pi's architecture.

## Project Layout

```text
pigate/cmd/gatecontroller       Raspberry Pi gate controller entrypoint
pigate/cmd/credentialserver     Windows credential server entrypoint
pigate/cmd/statusserver         PiGate status page and command API
pigate/pkg/gate                 Gate logic, keypad, and GPIO integration
pigate/pkg/database             SQLite and PostgreSQL repositories
pigate/pkg/credentialparser     Credential file parsing and file watching
pigate/pkg/messenger            MQTT client, topics, commands, and status
pigate/configs                  Example application config files
deploy/cloud                    Cloud control plane Compose stack
deploy/systemd                  Linux service templates used by GitHub Actions
deploy/windows                  Windows service helper scripts
```
