[![Build Status](https://travis-ci.org/redBorder/dswatcher.svg?branch=master)](https://travis-ci.org/redBorder/dswatcher)
[![Coverage Status](https://coveralls.io/repos/github/redBorder/dswatcher/badge.svg?branch=master)](https://coveralls.io/github/redBorder/dswatcher?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/redBorder/dswatcher)](https://goreportcard.com/report/github.com/redBorder/dswatcher)

# dswatcher (Dynamic Sensors Watcher)

* [Overview](#overview)
* [Installing](#installing)
* [Usage](#usage)
* [Configuration](#configuration)

## Overview

Service for dynamically add and remove Teldat sensors on the Netflow collector
by updating the information on the Chef node.

- When a new sensor starts to send data to the Netflow collector, the data will
be discarded to a Kafka topic.
- `dswatcher` will analyze the discarded Netflow data looking for
a specific *Option Template* that carries a *Serial Number*.
- `dswatcher` will look up on the Chef sensor nodes for a
node with the *Serial number*. If this sensor exists, the IP address for the
sensor and the Observation ID will be updated with the IP address and Observation
ID of the Netflow sender.
- `dswatcher` will listen for alerts about sensors that reached
their limits. The sensor will be marked as blocked on the Chef node.
- `dswatcher` will listen for alerts about counters resets. When this message
is received all the sensors block status will be set to **false**.

## Installing

To install this application ensure you have the
[GOPATH](https://golang.org/doc/code.html#GOPATH) environment variable set and
**[glide](https://glide.sh/)** installed.

```bash
curl https://glide.sh/get | sh
```

And then:

1. Clone this repo and cd to the project:

    ```bash
    git clone https://github.com/redBorder/dswatcher.git && cd dswatcher
    ```
2. Install dependencies and compile:

    ```bash
    make
    ```
3. Install on desired directory:

    ```bash
    prefix=/opt/dynamic-sensors-watcher/ make install
    ```

## Usage

Usage of dswatcher:

```
--version
    Show version info
--config string
    Config file
--debug
    Print debug info
```

## Configuration

```yaml
broker:
  address: kafka:9092        # Kafka host
  consumer_group: dswatcher  # Kafka consumer group ID
  netflow_topics:
    - rb_flow_discard  # Topic to look up for the Option Template where the serial number is
  limits_topics:
    - rb_limits        # Topic listen for notification about sensors limits

decoder:
  element_id: 300          # Netflow element id of the serial number
  option_template_id: 258  # ID of the Option Template where the serial number is

updater:
  chef_server_url: <chef_server_url>            # URL of the Chef server
  node_name: <node_name>                        # Node name on Chef
  client_key: key.pem                           # Path to the key used for Chef authorization
  serial_number_path: org/serial_number         # Path to the serial number of the sensor on Chef
  sensor_uuid_path: org/sensor_uuid             # Path to the UUID of the sensor on Chef
  ipaddress_path: ipaddress                     # Path to the IP address of the sensor to update
  observation_id_path: org/observation_id       # Path to the Observation Domain ID to update
  fetch_interval_s: 60                          # Time between updates of the internal sensors database
  blocked_status_path: org/blocked              # Path to the block status
  update_interval_s: 30                         # Time between updates of the Chef node
```
