# PiGate Cloud Control Plane Host

This Compose stack runs the first cloud **Control Plane Host** on an Ubuntu droplet.
It is intended to run after the droplet has joined the PiGate tailnet.

## Network Model

Postgres and EMQX are published only on the droplet's Tailscale IP. They should not
listen on the public DigitalOcean address.

Use this on the droplet to find the bind address:

```bash
tailscale ip -4
```

## First Run

Compose needs three runtime values: the droplet Tailscale IP, the Postgres
password, and the EMQX dashboard password. The simplest one-host setup is a local
`.env` file next to this Compose file. Do not commit the real `.env`.

Create it from the example:

```bash
cp .env.example .env
nano .env
```

Set `TAILSCALE_IP` to the droplet's `100.x.y.z` Tailscale address and replace both
passwords with long random values.

If you prefer not to create `.env`, export the values in the shell before running
Compose:

```bash
export TAILSCALE_IP=100.x.y.z
export POSTGRES_PASSWORD='replace-with-long-random-value'
export EMQX_DASHBOARD_PASSWORD='replace-with-long-random-value'
```

Start the services:

```bash
docker compose up -d
docker compose ps
```

From a device on the same tailnet, test Postgres:

```bash
psql "host=100.x.y.z port=5432 dbname=pigate_db user=pigate_user sslmode=disable"
```

Open the EMQX dashboard from a tailnet device:

```text
http://100.x.y.z:18083
```

The EMQX default dashboard user is `admin`; use the password from `.env`.

## Application Config

For the Windows credential service and Raspberry Pi gate controller, point config at
the droplet's Tailscale IP or MagicDNS name:

```toml
MQTT_BROKER = "tcp://100.x.y.z:1883"
MQTT_USERNAME = "pigate_gatecontroller"
MQTT_PASSWORD_ENV = "PIGATE_MQTT_PASSWORD"
DB_HOST = "100.x.y.z"
DB_PORT = "5432"
DB_NAME = "pigate_db"
DB_USER = "pigate_user"
DB_PASSWORD_ENV = "PIGATE_DB_PASSWORD"
```

Set the matching MQTT password in the runtime environment for each app:

```bash
export PIGATE_MQTT_PASSWORD='replace-with-client-password'
```

## MQTT Client Authentication

EMQX allows anonymous MQTT clients until an authenticator is enabled. To reject
anonymous clients:

1. Log in to the EMQX dashboard at `http://100.x.y.z:18083`.
2. Go to **Access Control** / **Authentication**.
3. Create a password-based authenticator using the built-in database.
4. Add MQTT users for the PiGate applications, for example:
   - `pigate_gatecontroller`
   - `pigate_credentialserver`
5. Use the same passwords in each app's `PIGATE_MQTT_PASSWORD` environment
   variable.

After the authenticator is enabled, clients that do not send valid MQTT
credentials are rejected.

## Security Notes

This stack keeps network exposure narrow by binding services to the Tailscale IP.
MQTT should also require client authentication before the cloud broker is used for
gate commands.
