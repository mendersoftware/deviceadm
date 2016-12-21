#!/bin/bash
docker-compose run -d --name mender-device-auth mender-device-auth
docker-compose run -d --name mender-inventory mender-inventory
docker-compose run -d --name mongo-device-adm mender-mongo-device-adm
