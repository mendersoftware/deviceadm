#!/usr/bin/python
# Copyright 2016 Mender Software AS
#
#    Licensed under the Apache License, Version 2.0 (the "License");
#    you may not use this file except in compliance with the License.
#    You may obtain a copy of the License at
#
#        https://www.apache.org/licenses/LICENSE-2.0
#
#    Unless required by applicable law or agreed to in writing, software
#    distributed under the License is distributed on an "AS IS" BASIS,
#    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#    See the License for the specific language governing permissions and
#    limitations under the License.
from common import api_client_int, api_client_mgmt, \
                   clean_db, clean_db_devauth, \
                   mongo, mongo_devauth, \
                   init_authsets, init_authsets_mt

import bravado
import pytest
import json
import tenantadm

class TestMgmtApiDeleteDevice:
    def test_ok(self, api_client_mgmt, init_authsets):
        id = init_authsets[0].id

        rsp = api_client_mgmt.delete_device_mgmt(id)
        assert rsp.status_code == 204

        devs = api_client_mgmt.get_all_devices()
        assert len(devs) == len(init_authsets) - 1

        deleted = [a for a in devs if a.id == id]
        assert len(deleted) == 0

    def test_ok_nonexistent(self, api_client_mgmt, init_authsets):
        id = "nonexistent"

        rsp = api_client_mgmt.delete_device_mgmt(id)
        assert rsp.status_code == 204

        devs = api_client_mgmt.get_all_devices()
        assert len(devs) == len(init_authsets)
