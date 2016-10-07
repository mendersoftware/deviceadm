#!/bin/bash
docker-compose run -T --name mender-device-auth mender-device-auth &
docker-compose run -T --name mender-inventory mender-inventory &
