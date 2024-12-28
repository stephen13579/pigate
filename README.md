# PiGate Project

The PiGate project is designed to control a gate using a Raspberry Pi, with access managed via keypad and credentials stored locally. The system uses MQTT for remote gate control and for messages between the credential server and the Pi. 

MQTT service is useful for a device which has the ability to disconnect from the network but we still want messages to be delivered when it reconnects. Its also lightweight and for most of our message payloads it will suffice. 

There is also a credential server that parses a 3rd party file and creates a JSON file which is uploaded to an S3 bucket. MQTT can be used to notify the gate controller which can retrieve this file from the S3 bucket and update the local database accordingly. This will require configuring the S3 configuration in AWS, configuration file for the project, and using environment variables to hold the secret credentials that allow access to the S3 bucket. 

Database tables:
```
credentials:
------------------------------------------------
string   | string   | int          | bool
------------------------------------------------
code     | username | access_group | locked_out
------------------------------------------------

gate_request_log:
----------------------------------------
string       | string     | string          
----------------------------------------
code         | time       | status
----------------------------------------

access_times:
----------------------------------------
string       | int     | int          
----------------------------------------
access_group | start_time | end_time
----------------------------------------
```
Note: start and end time are minutes from start of day

Note: we might want to look into security for sqlite which stores the valid gate codes locally, and the S3 file also has plain text gate codes in JSON, its encrypted on AWS, but not when we pull it or create it initially with the credential server. #TODO

## Project Structure

## Setup Instructions

### Prerequisites

- Go 1.21 or higher installed.
- A Raspberry Pi with GPIO access.
- An MQTT broker (e.g., emqx).
- SQLite3 installed.

### Build the Applications

```bash
# Build the gate controller
cd cmd/gatecontroller
go build -o bin/gatecontroller ./cmd/gatecontroller

# Build the credential server
go build -o bin/credentialserver ./cmd/credentialserver
