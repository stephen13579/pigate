# PiGate Project

The PiGate project is designed to control a gate using a Raspberry Pi, with access managed via keypad and credentials stored locally. The system updates credentials through MQTT communication.

## Project Structure

- `cmd/`: Main applications.
  - `gatecontroller/`: Runs on the Raspberry Pi to control the gate.
  - `credentialserver/`: Server to receive and parse credential files.

- `pkg/`: Reusable packages.
  - `gate/`: Gate control logic.
  - `keypad/`: Keypad handling using Wiegand protocol.
  - `mqttclient/`: MQTT client for gate controller.
  - `database/`: Local database management.
  - `updater/`: Handles credential updates on the gate controller.
  - `server/`: Server functionalities including HTTP server and MQTT publisher.
  - `config/`: Configuration management.

- `internal/`: Internal packages not intended for external use.
  - `parser/`: Parses new credential files.
  - `utils/`: Utility functions.

- `resources/`: Static files like `credentials.json`.

## Setup Instructions

### Prerequisites

- Go 1.17 or higher installed.
- A Raspberry Pi with GPIO access.
- An MQTT broker (e.g., emqx).
- SQLite3 installed.

### Build the Applications

```bash
# Build the gate controller
cd cmd/gatecontroller
go build -o gatecontroller

# Build the credential server
cd ../credentialserver
go build -o credentialserver
