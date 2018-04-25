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
                   clean_db, \
                   mongo, \
                   init_authsets, init_authsets_mt

import bravado
import pytest
import json
import tenantadm
import deviceauth


class TestMgmtApiDeleteDeviceBase:
    def _test_ok(self, api_client_mgmt, init_authsets, auth=None):
        id = init_authsets[0].id
        device_id = init_authsets[0].device_id

        with deviceauth.run_fake_delete_device(device_id, id, 204):
            rsp = api_client_mgmt.delete_device_mgmt(id, auth)
            assert rsp.status_code == 204

        devs = api_client_mgmt.get_devices(auth=auth)
        assert len(devs) == len(init_authsets) - 1

        deleted = [a for a in devs if a.id == id]
        assert len(deleted) == 0

    def _test_ok_nonexistent(self, api_client_mgmt, init_authsets, auth=None):
        id = "nonexistent"

        rsp = api_client_mgmt.delete_device_mgmt(id, auth)
        assert rsp.status_code == 204

        devs = api_client_mgmt.get_devices(auth=auth)
        assert len(devs) == len(init_authsets)


class TestMgmtApiDeleteDevice(TestMgmtApiDeleteDeviceBase):
    def test_ok(self, api_client_mgmt, init_authsets):
        self._test_ok(api_client_mgmt, init_authsets)

    def test_ok_nonexistent(self, api_client_mgmt, init_authsets):
        self._test_ok_nonexistent(api_client_mgmt, init_authsets)


class TestMgmtApiDeleteDeviceMultitenant(TestMgmtApiDeleteDeviceBase):
    @pytest.mark.parametrize("tenant_id", ["tenant1", "tenant2"])
    def test_ok(self, api_client_mgmt, init_authsets_mt, tenant_id):
        auth = api_client_mgmt.make_user_auth("user", tenant_id)
        self._test_ok(api_client_mgmt, init_authsets_mt[tenant_id], auth)

    @pytest.mark.parametrize("tenant_id", ["tenant1", "tenant2"])
    def test_ok_nonexistent(self, api_client_mgmt, init_authsets_mt, tenant_id):
        auth = api_client_mgmt.make_user_auth("user", tenant_id)
        self._test_ok_nonexistent(api_client_mgmt, init_authsets_mt[tenant_id], auth)
