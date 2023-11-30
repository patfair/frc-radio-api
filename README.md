# frc-radio-api [![Build Status](https://github.com/patfair/frc-radio-api/actions/workflows/test.yml/badge.svg)](https://github.com/patfair/frc-radio-api/actions)
Minimal API server that lives on the FRC access point and robot radios to facilitate configuration.

## Installation
### Using a Pre-Built Release
Download the desired release from the [releases page](https://github.com/patfair/frc-radio-api/releases) and unzip it.
If on a system with a UNIX shell (e.g. MacOS or Linux), run the `install-access-point` or `install-robot-radio` script
and follow the prompts. Otherwise, follow the instructions in the manual installation section below.

### From Source
Install [Go](https://go.dev/dl/) version 1.20 or later and clone the repository. Then, run
`install-access-point --build` or `install-robot-radio --build` and follow the prompts.

### Manually
The installation script is a convenience wrapper around the following steps:
1. Stop the API service if it is already running on the target device (with `/etc/init.d/frc-radio-api stop`).
1. Copy the `frc-radio-api` binary to `/usr/bin/` on the target device. Ensure that it is executable using
`chmod +x /usr/bin/frc-radio-api`.
1. For the access point only, copy the `wireless-boot-linksys` or `wireless-boot-vh` baseline configuration file
(depending on radio model) to `/etc/config/wireless-boot` on the target device.
1. Copy the `access-point.init` or `robot-radio.init` init script to `/etc/init.d/frc-radio-api` on the target device.
Ensure that it is executable using `chmod +x /etc/init.d/frc-radio-api`.
1. Create a symbolic link from `/etc/rc.d/S11frc-radio-api` to `/etc/init.d/frc-radio-api` on the target device.
1. For the access point only, comment out the `wifi detect` line in `/etc/init.d/boot` on the target device; it isn't
needed and just makes it take longer for the Ethernet interface to come up on boot.
1. Start the API service on the target device (with `/etc/init.d/frc-radio-api start`).

## Access Point API
The access point API is a simple REST API that allows for the configuration of the access point. It runs on both the
Linksys and Vivid-Hosting access points and abstracts away the differences between the two so that the field management
system doesn't need to know which type is being used.

The installation of the API includes a baseline no-team Wi-Fi configuration file that is copied to overwrite the last
configuration on every boot. This ensures that the access point will always come up in a known good state when
power-cycled.

### Authentication
The API is optionally protected by token authentication. The installation script prompts for an optional password, and
if one is provided, the API will require that password to be provided in a `Authorization: Bearer [password]` header.

### /admin Endpoint
The `/health` GET endpoint returns a successful response if the API is running. For example:
```
$ curl http://10.0.100.2:8081/health
OK
```

### /status Endpoint
The `/status` GET endpoint returns the current status of the access point. It returns a JSON object like this:
```
$ curl http://10.0.100.2:8081/status
{
  "channel": 93,
  "status": "ACTIVE",
  "stationStatuses": {
    "blue1": null,
    "blue2": {
      "ssid": "5555",
      "hashedWpaKey": "2d0d7870bef68c589212a2bc47b650091585005cdd9404842dc9e3d27809b6c2",
      "wpaKeySalt": "Tj5DuBrAYhfFvNMZ",
      "isRobotRadioLinked": false,
      "rxRateMbps": 0,
      "txRateMbps": 0,
      "signalNoiseRatio": 0,
      "bandwidthUsedMbps": 0
    },
    "blue3": null,
    "red1": {
      "ssid": "1111",
      "hashedWpaKey": "e418de38d25cd254d0faf73f3206631b9eed8fdd8094004da655749cf536af7a",
      "wpaKeySalt": "B4Vx1KSX1TPzErKA",
      "isRobotRadioLinked": false,
      "rxRateMbps": 0,
      "txRateMbps": 0,
      "signalNoiseRatio": 0,
      "bandwidthUsedMbps": 0
    },
    "red2": null,
    "red3": null
  }
}
```
A null value for a team station indicates that no team is assigned.

WPA keys are not exposed directly to prevent unauthorized users from learning their value. However, a user who already
knows a WPA key can verify that it is correct by concatenating it with the `wpaKeySalt` and hashing the result using
SHA-256; the result should match the `hashedWpaKey`.

### /configuration Endpoint
The `/configuration` POST endpoint allows the access point to be configured. It accepts a JSON object like this:
```
$ curl http://10.0.100.2:8081/configuration -XPOST -d '{
  "channel": 93,
  "stationConfigurations": {
    "red1": {"ssid": "1111", "wpaKey": "11111111"},
    "blue2": {"ssid": "5555", "wpaKey": "55555555"}
  }
}'
New configuration received and will be applied asynchronously.
```

The `/status` endpoint can then be polled to check whether the configuration has been applied. For example:
```
$ curl http://10.0.100.2:8081/status
{
  "channel": 93,
  "status": "CONFIGURING",
  "stationStatuses": {
    "blue1": null,
    "blue2": null,
    "blue3": null,
    "red1": null,
    "red2": null,
    "red3": null
  }
}
```

## Robot Radio API
The robot radio API is a simple REST API that allows for the configuration of the robot radio for a given team. It runs
on the Vivid-Hosting robot radio.

### Authentication
Same as the access point API.

### /admin Endpoint
Same as the access point API.

### /status Endpoint
The `/status` GET endpoint returns the current status of the robot radio. It returns a JSON object like this:
```
$ curl http://10.12.34.1:8081/status
{
  "teamNumber": 1234,
  "ssid": "1234",
  "hashedWpaKey": "d40e29b90743ddf71c75bfaedab1333e23bf43eb29f5c8c1ba55756e96e99d84",
  "wpaKeySalt": "DzCKbEIu53vCmf0p",
  "status": "ACTIVE"
}
```
See the access point API documentation regarding the `hashedWpaKey` and `wpaKeySalt` fields.

### /configuration Endpoint
The `/configuration` POST endpoint allows the robot radio to be configured for a different team. It accepts a JSON
object like this:
```
$ curl -XPOST http://10.12.34.1:8081/configuration -d '{"teamNumber":5678,"wpaKey":"12345678"}'
New configuration received and will be applied asynchronously.
```

Reconfiguring the radio will cause its IP address to change, so the user should renew their DHCP or reconfigure their
static IP and then check the status of the radio at its new IP address:
```
$ curl http://10.56.78.1:8081/status
{
  "teamNumber": 5678,
  "ssid": "5678",
  "hashedWpaKey": "63b7edb8b5c6b832dd495220e67d65414238165b92ef1feb52d6f39c052ac693",
  "wpaKeySalt": "6BjRXMUm3kExcAiR",
  "status": "ACTIVE"
}
```
