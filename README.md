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
  "channelBandwidth": "HT40",
  "redVlans": "40_50_60",
  "blueVlans": "10_20_30",
  "status": "ACTIVE",
  "stationStatuses": {
    "blue1": null,
    "blue2": {
      "ssid": "5555",
      "hashedWpaKey": "2d0d7870bef68c589212a2bc47b650091585005cdd9404842dc9e3d27809b6c2",
      "wpaKeySalt": "Tj5DuBrAYhfFvNMZ",
      "isLinked": false,
      "macAddress": "",
      "signalDbm": 0,
      "noiseDbm": 0,
      "signalNoiseRatio": 0,
      "rxRateMbps": 0,
      "rxPackets": 0,
      "rxBytes": 0,
      "txRateMbps": 0,
      "txPackets": 0,
      "txBytes": 0,
      "bandwidthUsedMbps": 0
    },
    "blue3": null,
    "red1": {
      "ssid": "1111",
      "hashedWpaKey": "e418de38d25cd254d0faf73f3206631b9eed8fdd8094004da655749cf536af7a",
      "wpaKeySalt": "B4Vx1KSX1TPzErKA",
      "isLinked": true,
      "macAddress": "48:DA:35:B0:01:CF",
      "signalDbm": -53,
      "noiseDbm": -93,
      "signalNoiseRatio": 40,
      "rxRateMbps": 860.3,
      "rxPackets": 4095,
      "rxBytes": 5177,
      "txRateMbps": 6,
      "txPackets": 5246,
      "txBytes": 11830,
      "bandwidthUsedMbps": 4.102
    },
    "red2": null,
    "red3": null
  },
  "version": "1.2.3"
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
  "channelBandwidth": "HT20",
  "redVlans": "40_50_60",
  "blueVlans": "70_80_90",
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
  "channelBandwidth": "HT20",
  "redVlans": "40_50_60",
  "blueVlans": "70_80_90",
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
  "networkStatus24": {
    "ssid": "FRC-1234",
    "hashedWpaKey": "5147695f755c47cda0c60ec59b6a278cc3a6b217e78ad4a4480f9d027a139c40",
    "wpaKeySalt": "n5OZJgKdhjWQgRXL",
    "isLinked": false,
    "macAddress": "",
    "signalDbm": 0,
    "noiseDbm": 0,
    "signalNoiseRatio": 0,
    "rxRateMbps": 0,
    "rxPackets": 0,
    "rxBytes": 0,
    "txRateMbps": 0,
    "txPackets": 0,
    "txBytes": 0,
    "bandwidthUsedMbps": 0
  },
  "networkStatus6": {
    "ssid": "1234",
    "hashedWpaKey": "4430f81c11c7bad4d36a886be2ca3b34deb5fd6c8a71ccaf244a22c44ce062e8",
    "wpaKeySalt": "darLGfhgtJazer9C",
    "isLinked": true,
    "macAddress": "4A:DA:35:B0:3A:27",
    "signalDbm": -56,
    "noiseDbm": -93,
    "signalNoiseRatio": 37,
    "rxRateMbps": 7.3,
    "rxPackets": 4095,
    "rxBytes": 344,
    "txRateMbps": 516.2,
    "txPackets": 0,
    "txBytes": 52765,
    "bandwidthUsedMbps": 0.002
  },
  "status": "ACTIVE",
  "version": "1.2.3"
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
  "networkStatus24": [...],
  "networkStatus6": [...],
  "status": "ACTIVE"
}
```

## Updating Firmware Via the API
Both the Access Point and Robot Radio APIs support updating the firmware of the device via the `/firmware` endpoint. The
endpoint uses the same authentication scheme as described above.

The endpoint can be configured to only accept firmware files that are encrypted with an asymmetric key pair via the
[age](https://github.com/FiloSottile/age) tool. The party creating new firmware builds generates a key pair and
distributes the secret (decryption) key with the API server. They can then distribute new firmware builds by encrypting
them with their public key (which they keep secret) and making them available for download (along with the
checksum of the unencrypted firmware). Then, when it receives a new firmware file, the API server on
the radio decrypts it and verifies its checksum before flashing the radio.

The `/firmware` endpoint accepts a multipart/form-data request with a `file` parameter containing the firmware file to
be flashed, and a `checksum` parameter containing the expected SHA-256 checksum of the decrypted firmware file.

### Setting Up the API Server
To set up the API server to accept encrypted firmware files, first install [age](https://github.com/FiloSottile/age)
then generate a key pair using `age-keygen`:
```
$ age-keygen
# created: 2023-12-29T09:52:33-08:00
# public key: age1r9x7t8rzy7l3yccvtd8q3thlt5kvy5fmd58t4s0nqdkyvp9ama9q3swxt6
AGE-SECRET-KEY-12D95QEN2T7VZAEG6KKS2YFE7K26YZRZH48Y32YMJF6YAKJFFTM4QQDCW7F
```

Copy the secret key to `/root/frc-radio-api-firmware-key.txt` on the radio (or use the install scripts as above
which will prompt for the key). The API server will automatically detect the presence of the key file and enable
decryption of firmware files. If the key file is blank, the API server will accept unencrypted firmware files.

### Encrypting Firmware Files
To encrypt a firmware file, use the `age` tool:
```
$ age --encrypt -o firmware-encrypted.bin -r age1r9x7t8rzy7l3yccvtd8q3thlt5kvy5fmd58t4s0nqdkyvp9ama9q3swxt6 firmware-unencrypted.tar
```

### Uploading Firmware to the API
An encrypted firmware file can be uploaded to the API server as follows:
```
$ curl -v -XPOST http://10.0.100.2:8081/firmware -F 'file=@firmware-encrypted.bin' -F 'checksum=84fbed65950291a4f0bb252387c651dc0937df32108e952c81bf689ff7c52665'
New firmware received and will be applied now.
```
