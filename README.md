[![Build Status](https://travis-ci.org/mendersoftware/deviceadm.svg?branch=master)](https://travis-ci.org/mendersoftware/deviceadm)
[![codecov](https://codecov.io/gh/mendersoftware/deviceadm/branch/master/graph/badge.svg)](https://codecov.io/gh/mendersoftware/deviceadm)
[![Go Report Card](https://goreportcard.com/badge/github.com/mendersoftware/deviceadm)](https://goreportcard.com/report/github.com/mendersoftware/deviceadm)
[![Docker pulls](https://img.shields.io/docker/pulls/mendersoftware/deviceadm.svg?maxAge=3600)](https://hub.docker.com/r/mendersoftware/deviceadm/)

# This repository is no longer maintained.

New versions of mender backend will not contain Device Admission Service.
[Device Authentication Service](https://github.com/mendersoftware/deviceauth) has assumed Device Admission Service responsibilities.
Issue reports and pull requests will not be attended.

Mender: Device Admission Service
==============================================

Mender is an open source over-the-air (OTA) software updater for embedded Linux
devices. Mender comprises a client running at the embedded device, as well as
a server that manages deployments across many devices.

This repository contains the Mender Device Admission service, which is part of the
Mender server. The Mender server is designed as a microservices architecture
and comprises several repositories.

Device Admission allows the user to review the identities of devices requesting an authentication token,
and either grant or deny them access to the system. This decision is based on vendor-specific device identity
attributes, which are:
- reported by the device during bootstrap request as an opaque, encrypted blob
- decrypted and presented to the user by Device Admission

This service can be considered a reference implementation, and could be replaced
with a vendor-specific, proprietary module, implementing any desired identity encoding
schema for maximum confidentiality. As such, it serves as a potential
integration point with custom 3rd party identity stores.


![Mender logo](https://mender.io/user/pages/04.resources/_logos/logoS.png)


## Getting started

To start using Mender, we recommend that you begin with the Getting started
section in [the Mender documentation](https://docs.mender.io/).


## Building from source

As the Mender server is designed as microservices architecture, it requires several
repositories to be built to be fully functional. If you are testing the Mender server it
is therefore easier to follow the getting started section above as it integrates these
services.

If you would like to build the Device Audmission service independently, you can follow
these steps:

```
git clone https://github.com/mendersoftware/deviceadm.git
cd deviceadm
go build
```

## Configuration

The service can be configured by:
* providing configuration file (supports JSON, TOML, YAML and HCL formatting).
The default configuration file is provided to be downloaded from [config.yaml](https://github.com/mendersoftware/deviceadm/blob/master/config.yaml).
* setting environment variables. The service will check for a environment variable
with a name matching the key uppercased and prefixed with "DEVICEADM_".
Eg. for "listen" the variable name is "DEVICEADM_LISTEN".

## Contributing

We welcome and ask for your contribution. If you would like to contribute to Mender, please read our guide on how to best get started [contributing code or
documentation](https://github.com/mendersoftware/mender/blob/master/CONTRIBUTING.md).

## License

Mender is licensed under the Apache License, Version 2.0. See
[LICENSE](https://github.com/mendersoftware/deviceadm/blob/master/LICENSE) for the
full license text.

## Security disclosure

We take security very seriously. If you come across any issue regarding
security, please disclose the information by sending an email to
[security@mender.io](security@mender.io). Please do not create a new public
issue. We thank you in advance for your cooperation.

## Connect with us

* Join our [Google
  group](https://groups.google.com/a/lists.mender.io/forum/#!forum/mender)
* Follow us on [Twitter](https://twitter.com/mender_io?target=_blank). Please
  feel free to tweet us questions.
* Fork us on [Github](https://github.com/mendersoftware)
* Email us at [contact@mender.io](mailto:contact@mender.io)
