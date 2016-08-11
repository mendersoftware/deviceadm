## Collaborators

* ```devauth```
* ```Mender user``` (via web UI or other client)

## Overview

The device admission service plays a crucial part in the system's auth pipeline, effectively deciding whether authentication should be granted or denied for a device accessing the system.

```devadm``` interfaces with ```devauth``` on one side, and with the system's user on the other:
* ```devauth``` will submit all new device auth requests for admission by the user

* the user will review, grant or reject authentication using the ```devadm``` API; this decision will be reflected both in the ```devadm``` and ```devauth``` data stores, and used in subsequent device auth/admission requests

Details of various scenarios are described in the 'Use Cases' section.

### Device Identity Data

Each admission request carries a ```device_identity``` data string, uniquely identifying the device. It is important to note that:

* it is an opaque data structure, which can contain anything from a UUID, to a JSON-encoded set of device attribute data, possibly encrypted, etc.

* ```devauth``` will pass this data verbatim to ```devadm```

* it is ```devadm```'s responsibility to parse this data and present it to the user in a human-readable form

In this (reference) implementation ```devadm```, the identity data is a set of
JSON-encoded device attributes, encrypted with the user-provided public key (TODO - plaintext json currently).


### Device ID
The device identity data is used to derive a system-wide unique ID of the device, for use in API requests between services.

The unique ID is computed by ```devauth``` as a SHA256 over the identity data blob, and passed to ```devadm``` in the admission request.

###Integration

In production environment, the reference implementation of the service can be swapped out and substituted with the customer's own implementation, integrating e.g. with a custom inventory systems and implementing a custom identification scheme.

# Use Cases
The diagram below summarizes all ```devadm``` 's use cases (including its collaborators); a description of each use case follows.

```
             +------------------+         +----------------+            +-----------------+           +-------------------+
             |                  |         |                |            |                 |           |                   |
             |      device      |         |     devauth    |            |    devadm       |           |    user (web UI)  |
             |                  |         |                |            |                 |           |                   |
             +--------+---------+         +-------+--------+            +-------+---------+           +---------+---------+
                      |                           |                             |                               |
                      |    POST /auth_request     |                             |                               |
                      +--------------------------->       POST /devices         |                               |
                      |                           +-----------------------------> ----\                         |
                      |                           |                             |     |  device stored          |
                      |                           |                             |     |  for user review        |
 1. New device        |                           |                             | ----/  (status: pending)      |
    requests auth     |                           |                             |                               |

                      |                           |                             |        GET /devices           |
                      |                           |                             |        GET /devices/{id}      |
 2. User reviews      |                           |                             <-------------------------------+
    the device        |                           |                             |                               |

                      |                           |                             |    POST /devices/{id}/status  |
                      |                           |                             <-------------------------------+
                      |                           +                             | ----\                         |
                      |                           |                             |     |  device status:         |
                      |                           |                             | ----/   accepted              |
                      |                           |                             |                               |
                      |                           |   POST /devices/{id}/status |                               |
                      |                           <-----------------------------+                               |
                      |     POST /auth_request    |                             |                               |
                      +--------------------------->                             |                               |
                      <---------------------------+                             |                               |
 2a. User accepts     |   HTTP 200 (token issued) |                             |                               |
     the device       |                           |                             |                               |

                      |                           |                             |                               |
                      |                           |                             |    POST /devices/{id}/status  |
                      |                           |                             <-------------------------------+
                      |                           |                             | ----\                         |
                      |                           |                             |     |  device status:         |
                      |                           |                             | ----/   rejected              |
                      |                           |   POST /devices/{id}/status |                               |
                      |                           <-----------------------------+                               |
                      |     POST /auth_request    |                             |                               |
                      +--------------------------->                             |                               |
                      <---------------------------+                             |                               |
 2a. User rejects     |   HTTP 401 (unauthorized) |                             |                               |
     the device       |                           |                             |                               |

```

## 1. New device requests authentication
Endpoints:

* ```POST /devices```

Workflow:

* ```devauth``` gets a request from a new, unrecognized device (not present in its db)
* ```devauth``` adds the device to ```devadm``` for admission via ```POST /devices```
** it also computes the device's unique ID (SHA256 over device identity data)
* ```devadm``` decodes the identity data to a human readable form, and stores it in its db for later user review (device status: pending)

## 2. User reviews the device
Endpoints:

* ```GET /devices```
* ```GET /devices/{id}```

Workflow:

* user can review a list of devices, or individual devices, optionally filtered by their admission status
* the decoded, human-readable identity data are the basis for device acceptance or rejection

### 2a. User accepts the device
Endpoints:

* ```POST /devices/{id}/status```

Workflow:
* the user issues an acceptance request
* ```devadm``` internally modifies the device's state to ```accepted```
* ```devadm``` propagates this information to ```devauth```
* upon the next auth request, the device will be granted authentication

### 2b. User rejects the device

Endpoints:

* ```POST /devices/{id}/status```

Workflow:
* the user issues a rejection request
* ```devadm``` internally modifies the device's state to ```rejected```
* ```devadm``` propagates this information to ```devauth```
* upon the next auth request, the device will *not* be granted authentication
