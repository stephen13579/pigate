# PiGate Project

The PiGate project is designed to control a gate using a Raspberry Pi, with access managed via keypad and credentials stored locally. The system uses MQTT for remote gate control and for messages between the credential server and the Pi. 

MQTT service is useful for a device which has the ability to disconnect from the network but we still want messages to be delivered when it reconnects. Its also lightweight and for most of our message payloads it will suffice. 

Database tables:
```
credentials:
------------------------------------------------
string   | string   | int          | bool
------------------------------------------------
code     | username | access_group | locked_out
------------------------------------------------

access_times:
----------------------------------------
string       | int     | int          
----------------------------------------
access_group | start_time | end_time
----------------------------------------
```
Note: start and end time are minutes from start of day

Note: we probably want to secure gate codes more securely, right now they are plain text in a sqlite database!

## Project Structure

## Setup Instructions

### Prerequisites

- Go 1.19 or higher installed.
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
