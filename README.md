[![Build Status](https://travis-ci.org/redBorder/dynamic-sensors-watcher.svg?branch=master)](https://travis-ci.org/redBorder/dynamic-sensors-watcher)
[![Coverage Status](https://coveralls.io/repos/github/redBorder/dynamic-sensors-watcher/badge.svg?branch=master)](https://coveralls.io/github/redBorder/dynamic-sensors-watcher?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/redBorder/dynamic-sensors-watcher)](https://goreportcard.com/report/github.com/redBorder/dynamic-sensors-watcher)

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
| 0.4      | Chef updater        | Done      |
| 0.5      | Instrumentation     | _Pending_ |
