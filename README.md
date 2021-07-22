Gotify-MQTT
===========

A plugin for using [Gotify](https://gotify.net/) to ingest messages from MQTT.

Usage
-----

Download the plugin from the releases page, or build it from source (using the Makefile).

Configure the settings and add your MQTT Brokers.

Example config:

```yaml
servers:
- address: 127.0.0.1:1883
  username: ""
  password: ""
  subscribe:
  - 'gotify/mqtt'

```
