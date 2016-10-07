#!/bin/bash
docker-compose run --name mender-device-auth mender-device-auth &
docker-compose run --name mender-device-adm mender-device-adm &
docker-compose run --name mender-inventory mender-inventory &
