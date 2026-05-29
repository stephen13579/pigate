# PiGate

PiGate controls a physical gate from a Raspberry Pi while allowing remote administration without exposing the device directly to the public internet.

## Language

**Device**:
The Raspberry Pi gate controller installed at the physical gate.
_Avoid_: server, box, remote machine

**Device ID**:
The stable identifier for a specific Raspberry Pi controller instance.
_Avoid_: location ID, gate ID

**Location**:
The physical gate or site controlled by PiGate.
_Avoid_: device when referring to the place rather than the hardware

**Control Plane**:
The remote administrative system that receives device status and coordinates commands, credential updates, and software updates.
_Avoid_: dashboard, cloud, backend when the boundary is ambiguous

**Control Plane Store**:
The durable Postgres data store used by the Control Plane for device records, version targets, and reported status.
_Avoid_: MQTT state, retained messages as source of truth

**Control Plane Host**:
The remote VM that runs private Control Plane infrastructure such as MQTT and Postgres.
_Avoid_: public database server, cloud box when the network boundary is ambiguous

**Outbound-only Remote Access**:
A remote access rule where the Device initiates all network connections needed for administration and updates.
_Avoid_: public device endpoint, exposed Pi

**Tailnet Maintenance Access**:
The private Tailscale network path used by trusted operators and Devices to reach Control Plane services without exposing those services publicly.
_Avoid_: public Postgres, public MQTT, open admin ports, manually managed WireGuard peers unless Tailscale is replaced

**Update Agent**:
The software running on the Device that checks for, verifies, and applies software updates.
_Avoid_: updater when it could mean the remote service

**Release Source**:
The remote location from which a Device can download approved software artifacts.
_Avoid_: repo, server, bucket when the release boundary is ambiguous

**GitHub Release Feed**:
The GitHub Releases metadata and artifacts used as PiGate's Release Source.
_Avoid_: git pull, branch, source checkout

**Desired Version**:
The software version that the Control Plane has approved for a specific Device to run.
_Avoid_: latest, target version when it is not device-specific

**Current Version**:
The software version currently running on a Device.
_Avoid_: installed build, app version when it is not reported by the Device

**Device-pulled Update**:
A software update flow where the Update Agent discovers and downloads approved releases through outbound network connections.
_Avoid_: push update, SSH deploy

**Safe Update State**:
A Device state in which the gate is closed and no gate operation is in progress, allowing the Update Agent to apply a software update.
_Avoid_: maintenance window when referring only to gate safety

**Status Page**:
The remote user interface that shows Device health, gate state, and installed software version.
_Avoid_: device page when it implies direct connection to the Device

## Relationships

- A **Device** reports status to the **Control Plane** through **Outbound-only Remote Access**.
- A **Device** has exactly one **Device ID**.
- A **Location** may have one active **Device**.
- A **Control Plane Host** runs private Control Plane infrastructure for one or more **Devices**.
- The **Control Plane** persists Device records in the **Control Plane Store**.
- The **Control Plane Store** is reachable through **Tailnet Maintenance Access**, not the public internet.
- MQTT and Postgres on a **Control Plane Host** bind to the host's Tailscale IP, not its public IP.
- MQTT clients authenticate to the **Control Plane Host**; anonymous MQTT is not allowed for gate commands.
- A **Status Page** reads Device status from the **Control Plane**.
- An **Update Agent** runs on exactly one **Device**.
- An **Update Agent** applies a **Device-pulled Update** from a **Release Source**.
- An **Update Agent** applies updates only when its **Device** is in a **Safe Update State**.
- The **GitHub Release Feed** is PiGate's first **Release Source**.
- The **Control Plane** assigns a **Desired Version** for each **Device**.
- A **Device** reports its **Current Version** to the **Control Plane**.

## Example Dialogue

> **Dev:** "Should the **Status Page** call the **Device** directly?"
> **Domain expert:** "No. The **Device** should use **Outbound-only Remote Access** and report status to the **Control Plane**."
>
> **Dev:** "Can we update the gate by connecting over VPN and running deploy commands?"
> **Domain expert:** "Only for maintenance. Normal software updates should be **Device-pulled Updates** from a **Release Source**."
>
> **Dev:** "Should a **Device** install every new GitHub release automatically?"
> **Domain expert:** "No. The **Control Plane** sets the **Desired Version** for that **Device**, and the **Device** reports its **Current Version**."
>
> **Dev:** "If we replace the Raspberry Pi at the front gate, is it the same **Device**?"
> **Domain expert:** "No. It is a new **Device** at the same **Location**."
>
> **Dev:** "Can the **Update Agent** restart the software while the gate is open?"
> **Domain expert:** "No. It must wait for a **Safe Update State**."

## Flagged Ambiguities

- "Remote access" can mean either an inbound public service on the Device or an outbound trusted connection from the Device. Resolved: PiGate uses **Outbound-only Remote Access** as the baseline.
- "Update the device" can mean manual commands over VPN or a repeatable pull-based release flow. Resolved: normal updates are **Device-pulled Updates**; VPN access is for maintenance and recovery.
- "Latest version" is not the same as the version a Device should install. Resolved: the **Control Plane** chooses a per-Device **Desired Version**.
- "Device status" can mean transient messages or durable administrative state. Resolved: the **Control Plane Store** is the source of truth; MQTT may notify or transport status but does not own it.
- "Location ID" was used as the MQTT topic identity in existing code, but update targeting needs a distinct **Device ID**. Resolved: **Location** and **Device** are separate concepts.
- "Safe to update" means the gate is closed and no gate operation is in progress. Resolved: updates are deferred while the Device is open, locked open, or actively handling gate movement.
- "Cloud instance" can mean a public application host or a private infrastructure host. Resolved: PiGate's first cloud VM is a **Control Plane Host** reached through **Tailnet Maintenance Access**; MQTT and Postgres are not public services.
- "VPN" can mean manually managed WireGuard or managed WireGuard through Tailscale. Resolved: PiGate will start with **Tailnet Maintenance Access** because it is easier to administer for a small home deployment.
