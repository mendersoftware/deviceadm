#!/usr/bin/python3
# Copyright 2018 Northern.tech AS
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
import os
import json
import logging
import mockserver

from contextlib import contextmanager

def get_fake_deviceauth_addr():
    return os.environ.get('FAKE_DEVICEAUTH_ADDR', '0.0.0.0:9997')

def handler_update_authset_status(devid, authid, status, ret_status):
    log = logging.getLogger('deviceauth.handler_update_authset_status')
    def update_authset_status(request, did, aid):
        log.info('fake update authset status, devid: %s, aid: %s', devid, aid)

        assert did==devid
        assert aid==authid
        s = json.loads(request.body.decode('utf-8'))
        assert s['status'] == status
        return (ret_status, {}, '')

    return update_authset_status

def handler_preauth(id_data, pubkey, ret_status):
    log = logging.getLogger('deviceauth.handler_preauth')
    def preauth(request):
        log.info('fake preauth')
        req = json.loads(request.body.decode('utf-8'))
        assert req['device_id'] is not None
        assert req['auth_set_id'] is not None
        assert req['id_data'] == id_data
        assert req['pubkey'] == pubkey

        return (ret_status, {}, '')

    return preauth

def handler_delete_device(devid, authid, ret_status):
    log = logging.getLogger('deviceauth.handler_delete_device')
    def delete_device(request, did, aid):
        log.info('fake delete device, did: %s, aid: %s', did, aid)

        assert did==devid
        assert aid==authid
        return (ret_status, {}, '')

    return delete_device

@contextmanager
def run_fake_update_authset_status(devid, aid, status, ret_status):
    handlers = [
        ('PUT', '/api/management/v1/devauth/devices/(.*)/auth/(.*)/status', handler_update_authset_status(devid, aid, status, ret_status)),
    ]
    with mockserver.run_fake(get_fake_deviceauth_addr(),
                             handlers=handlers) as server:
        yield server

@contextmanager
def run_fake_preauth(id_data, pubkey, ret_status):
    handlers = [
        ('POST', '/api/management/v1/devauth/devices', handler_preauth(id_data, pubkey, ret_status))
        ]

    with mockserver.run_fake(get_fake_deviceauth_addr(),
                             handlers=handlers) as server:
        yield server

@contextmanager
def run_fake_delete_device(devid, aid, ret_status):
    handlers = [
        ('DELETE', '/api/management/v1/devauth/devices/(.*)/auth/(.*)', handler_delete_device(devid, aid, ret_status)),
    ]
    with mockserver.run_fake(get_fake_deviceauth_addr(),
                             handlers=handlers) as server:
        yield server
