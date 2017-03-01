# dynamic-sensors-watcher

## Overview

Service for allowing new sensors to send flow based on a serial number.

The serial number is sent on an Netflow **option template**.

## Installing

To install this application ensure you have the
[GOPATH](https://golang.org/doc/code.html#GOPATH) environment variable set and
**[glide](https://glide.sh/)** installed.

```bash
curl https://glide.sh/get | sh
```

And then:

1. Clone this repo and cd to the project

    ```bash
    git clone https://github.com/redBorder/dynamic-sensors-watcher.git && cd dynamic-sensors-watcher
    ```
2. Install dependencies and compile

    ```bash
    make
    ```
3. Install on desired directory

    ```bash
    prefix=/opt/dynamic-sensors-watcher/ make install
    ```

## Usage

Usage of dynamic-sensors-watcher:

```
--version
    Show version info
--config string
    Config file
--debug
    Print debug info
```

## Roadmap

| Version  | Feature             | Status    |
|----------|---------------------|-----------|
| 0.1      | Kafka consumer      | Done      |
| 0.2      | Netflow decoder     | Done      |
| 0.3      | Chef checker        | _Pending_ |
| 0.4      | Chef updater        | _Pending_ |
| 0.5      | Instrumentation     | _Pending_ |
