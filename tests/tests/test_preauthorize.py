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
from common import api_client_mgmt, \
                   clean_db, clean_db_devauth, \
                   mongo, mongo_devauth, \
                   init_authsets, init_authsets_mt
import bravado
import json
import pytest

class TestMgmtApiPostDevicesBase:
    def _test_ok(self, api_client_mgmt, init_authsets, auth=None):
        identity = json.dumps({"mac": "new-preauth-mac"})
        api_client_mgmt.preauthorize(identity, 'new-preauth-key', auth)

        asets = api_client_mgmt.get_all_devices(auth=auth)
        assert len(asets) == len(init_authsets) + 1

        preauth = [a for a in asets if a.status == 'preauthorized' and a.device_identity==identity]
        assert len(preauth) == 1

    def _test_bad_req_iddata(self, api_client_mgmt, init_authsets, auth=None):
        try:
            api_client_mgmt.preauthorize('not-valid-json', 'new-preauth-key', auth)
        except bravado.exception.HTTPError as e:
            assert e.response.status_code == 400

        asets = api_client_mgmt.get_all_devices(auth=auth)
        assert len(asets) == len(init_authsets)

    def _test_conflict(self, api_client_mgmt, init_authsets, auth=None):
        for aset in init_authsets:
            try:
                identity = aset.device_identity
                api_client_mgmt.preauthorize(identity, 'new-preauth-key', auth)
            except bravado.exception.HTTPError as e:
                assert e.response.status_code == 409

        asets = api_client_mgmt.get_all_devices(auth=auth)
        assert len(asets) == len(init_authsets)


class TestMgmtApiPostDevices(TestMgmtApiPostDevicesBase):
    def test_ok(self, api_client_mgmt, init_authsets):
        self._test_ok(api_client_mgmt, init_authsets)

    def test_bad_req_iddata(self, api_client_mgmt, init_authsets):
        self._test_bad_req_iddata(api_client_mgmt, init_authsets)

    def test_conflict(self, api_client_mgmt, init_authsets):
        self._test_conflict(api_client_mgmt, init_authsets)


class TestMgmtApiPostDevicesMultitenant(TestMgmtApiPostDevicesBase):
    @pytest.mark.parametrize("tenant_id", ["tenant1", "tenant2"])
    def test_ok(self, api_client_mgmt, init_authsets_mt, tenant_id):
        auth = api_client_mgmt.make_user_auth("user", tenant_id)
        self._test_ok(api_client_mgmt, init_authsets_mt[tenant_id], auth)

    @pytest.mark.parametrize("tenant_id", ["tenant1", "tenant2"])
    def test_bad_req_iddata(self, api_client_mgmt, init_authsets_mt, tenant_id):
        auth = api_client_mgmt.make_user_auth("user", tenant_id)
        self._test_bad_req_iddata(api_client_mgmt, init_authsets_mt[tenant_id], auth)

    @pytest.mark.parametrize("tenant_id", ["tenant1", "tenant2"])
    def test_conflict(self, api_client_mgmt, init_authsets_mt, tenant_id):
        auth = api_client_mgmt.make_user_auth("user", tenant_id)
        self._test_conflict(api_client_mgmt, init_authsets_mt[tenant_id], auth)
