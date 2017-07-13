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


apiURL = "http://%s/api/devices/v1/authentication/auth_requests" % \
         pytest.config.getoption("devauth_host")

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
def create_devices():
    count = pytest.config.getoption("devices")
    do_create_devices(None, count)

# Create devices using the Device Authentication microservice
# Assumes multiple tenants (predefined).
@pytest.fixture(scope="class")
def create_devices_mt():
    count = pytest.config.getoption("devices")

    with fake_tenantadm():
        for tid in tenantIds:
            do_create_devices(tid, count)

def do_create_devices(tenant_id, count):
    for i in range(int(count)):
        privateKey, publicKey = get_keypair()
        mac = ":".join(["{:02x}".format(random.randint(0x00, 0xFF), 'x') for i in range(6)])

        macJSON = json.dumps({"mac": mac})
        authReq = {"id_data": macJSON, "pubkey": publicKey, "seq_no": 1}

        if tenant_id is not None:
            authReq["tenant_token"] = make_id_jwt("user", tenant_id)

        authReqJson = json.dumps(authReq)

        XMENSignature = sign_data(authReqJson, privateKey)
        headers = {"Content-type": "application/json", "X-MEN-Signature": XMENSignature}

        r = requests.post(apiURL, headers=headers, data=authReqJson, verify=False)

        assert r.status_code == 401
