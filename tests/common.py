#!/usr/bin/python
# Copyright 2017 Northern.tech AS
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
import pytest
import json
import requests
import random
from base64 import b64encode

from Crypto.PublicKey import RSA
from Crypto.Signature import PKCS1_v1_5
from Crypto.Hash import SHA256
from tenantadm import fake_tenantadm

from pymongo import MongoClient

from client import CliClient, InternalClientSimple, ManagementClientSimple
import tenantadm
import deviceauth

tenantIds = ['tenant1', 'tenant2']

def get_keypair():
    private = RSA.generate(1024)
    public = private.publickey()
    return private.exportKey().decode(), public.exportKey().decode()

def sign_data(data, privateKey):
    rsakey = RSA.importKey(privateKey)
    signer = PKCS1_v1_5.new(rsakey)
    digest = SHA256.new()
    if type(data) is str:
        data = data.encode()
    digest.update(data)
    sign = signer.sign(digest)
    return b64encode(sign)

def make_id_jwt(sub, tenant=None):
    """
        Prepare an almost-valid JWT token, suitable for consumption by our identity middleware (needs sub and optionally mender.tenant claims).

        The token contains valid base64-encoded payload, but the header/signature are bogus.
        This is enough for the identity middleware to interpret the identity
        and select the correct db; note that there is no gateway in the test setup, so the signature
        is never verified. It's also enough to provide a tenant_token for deviceauth.

        If 'tenant' is specified, the 'mender.tenant' claim is added.
    """
    payload = {"sub": sub}
    if tenant is not None:
        payload["mender.tenant"] = tenant
    payload = json.dumps(payload)
    payloadb64 = b64encode(payload.encode("utf-8"))
    return "bogus_header." + payloadb64.decode() + ".bogus_sign"


# Create devices using the Device Authentication microservice
# Assumes single-tenant, default db.
@pytest.fixture(scope="class")
def create_devices(api_client_mgmt):
    count = pytest.config.getoption("devices")
    do_create_devices(None, count, api_client_mgmt)

# Create devices using the Device Authentication microservice
# Assumes multiple tenants (predefined).
@pytest.fixture(scope="class")
def create_devices_mt(api_client_mgmt):
    count = pytest.config.getoption("devices")

    with fake_tenantadm():
        for tid in tenantIds:
            do_create_devices(tid, count, api_client_mgmt)

def do_create_devices(tenant_id, count, api_client_mgmt):
    for i in range(int(count)):
        id = "".join(["{}".format(random.randint(0,9)) for i in range(6)])
        device_id = "".join(["{}".format(random.randint(0,9)) for i in range(6)])

        mac = ":".join(["{:02x}".format(random.randint(0x00, 0xFF), 'x') for i in range(6)])

        identity_data = json.dumps({"mac": mac})

        privateKey, publicKey = get_keypair()

        mc = ManagementClientSimple()

        auth=None
        if tenant_id is not None:
             auth = mc.make_user_auth("user", tenant_id)

        mc.put_device(id, device_id, publicKey, identity_data, auth)

@pytest.fixture(scope="session")
def mongo():
    return MongoClient('mender-mongo:27017')


def mongo_cleanup(mongo):
    dbs = mongo.database_names()
    dbs = [d for d in dbs if d not in ['local', 'admin']]
    for d in dbs:
        mongo.drop_database(d)


@pytest.yield_fixture(scope='function')
def clean_db(mongo):
    mongo_cleanup(mongo)
    yield mongo
    mongo_cleanup(mongo)

@pytest.fixture(scope="session")
def cli():
    return CliClient()

@pytest.fixture(scope="session")
def api_client_int():
    return InternalClientSimple()

@pytest.fixture(scope="session")
def api_client_mgmt():
    return ManagementClientSimple()

# these fixtures are similar to create_devices..., but are 'new style',
# i.e. function-scoped, with proper conventions wrt to db cleaning (no data sharing between tests)
# also: init authsets have various states, including 'preauthorized'
@pytest.fixture(scope="function")
def init_authsets(clean_db, api_client_mgmt):
    """
        Create a couple auth sets in various states, including 'preauthorized'.
        The fixture is specific to testing internal PUT /devices/{id}/status.
        Some common funcs are reused, but existing common fixtures don't fit the bill.
    """
    return do_init_authsets(api_client_mgmt)

TENANTS = ['tenant1', 'tenant2']
@pytest.fixture(scope="function")
def init_authsets_mt(clean_db, api_client_mgmt):
    """
        Create a couple auth sets in various states, including 'preauthorized', in a MT context (2 tenants).
        The fixture is specific to testing internal PUT /devices/{id}/status.
    """
    tenant_authsets = {}
    with tenantadm.fake_tenantadm():
        for t in TENANTS:
            tenant_authsets[t] = do_init_authsets(api_client_mgmt, t)

    return tenant_authsets

def do_init_authsets(api_client_mgmt, tenant_id=None):
    auth=None
    if tenant_id is not None:
        auth = api_client_mgmt.make_user_auth("user", tenant_id)

    # create 5 auth sets in 'pending' state
    count = 5
    do_create_devices(tenant_id, count, api_client_mgmt)
    devs = api_client_mgmt.get_devices(auth=auth)
    assert len(devs) == count

    # using deviceadm's api, change up some statuses
    with deviceauth.run_fake_update_authset_status(devs[0].device_id, devs[0].id, 'accepted', 204):
        api_client_mgmt.change_status(devs[0].id, 'accepted', auth)

    with deviceauth.run_fake_update_authset_status(devs[3].device_id, devs[3].id, 'rejected', 204):
        api_client_mgmt.change_status(devs[3].id, 'rejected', auth)

    # add a preauthorized device
    identity = json.dumps({"mac": "preauth-mac"})

    with deviceauth.run_fake_preauth(identity, 'preauth-key', 201):
        api_client_mgmt.preauthorize(identity, 'preauth-key', auth)

    devs = api_client_mgmt.get_devices(auth=auth)
    assert len(devs) == count + 1
    return devs
