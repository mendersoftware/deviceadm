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

class TestMgmtApiPostDevices:
    def test_ok(self, api_client_mgmt, init_authsets):
        identity = json.dumps({"mac": "new-preauth-mac"})
        api_client_mgmt.preauthorize(identity, 'new-preauth-key')

        asets = api_client_mgmt.get_all_devices()
        assert len(asets) == len(init_authsets) + 1

        preauth = [a for a in asets if a.status == 'preauthorized' and a.device_identity==identity]
        assert len(preauth) == 1

    def test_bad_req_iddata(self, api_client_mgmt, init_authsets):
        try:
            api_client_mgmt.preauthorize('not-valid-json', 'new-preauth-key')
        except bravado.exception.HTTPError as e:
            assert e.response.status_code == 400

        asets = api_client_mgmt.get_all_devices()
        assert len(asets) == len(init_authsets)

    def test_conflict(self, api_client_mgmt, init_authsets):
        for aset in init_authsets:
            try:
                identity = aset.device_identity
                api_client_mgmt.preauthorize(identity, 'new-preauth-key')
            except bravado.exception.HTTPError as e:
                assert e.response.status_code == 409

        asets = api_client_mgmt.get_all_devices()
        assert len(asets) == len(init_authsets)
