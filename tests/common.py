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
import pytest
import json
import requests
from Crypto.PublicKey import RSA
from Crypto.Signature import PKCS1_v1_5
from base64 import b64encode
from Crypto.Hash import SHA256
import random

apiURL = "http://%s/api/%s/auth_requests" % (pytest.config.getoption("devauth_host"), pytest.config.getoption("api"))


def get_keypair():
    private = RSA.generate(1024)
    public = private.publickey()
    return str(private.exportKey()), str(public.exportKey())

def sign_data(data, privateKey):
    rsakey = RSA.importKey(privateKey)
    signer = PKCS1_v1_5.new(rsakey)
    digest = SHA256.new()
    digest.update(data)
    sign = signer.sign(digest)
    return b64encode(sign)


# Create devices using the Device Authentication microservice
@pytest.fixture(scope="class")
def create_devices(tenantToken="dummy"):
    count = pytest.config.getoption("devices")
    for i in range(int(count)):
        privateKey, publicKey = get_keypair()
        mac = ":".join(["{:02x}".format(random.randint(0x00, 0xFF), 'x') for i in range(6)])

        tenantToken = "dummy"
        macJSON = json.dumps({"mac": mac})
        authRequestPayload = json.dumps({"id_data": macJSON, "tenant_token": tenantToken, "pubkey": publicKey, "seq_no": 1})

        XMENSignature = sign_data(authRequestPayload, privateKey)
        headers = {"Content-type": "application/json", "X-MEN-Signature": XMENSignature}

        r = requests.post(apiURL, headers=headers, data=authRequestPayload, verify=False)

        assert r.status_code == 401
