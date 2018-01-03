import logging
import os
import subprocess

import pytest
import json
import base64
import common
from bravado.swagger_model import load_file
from bravado.client import SwaggerClient, RequestsClient
from requests.utils import parse_header_links
from urllib import parse as urlparse
import requests

class SwaggerApiClient:
    config = {
        'also_return_response': True,
        'validate_responses': True,
        'validate_requests': False,
        'validate_swagger_spec': False,
        'use_models': True,
    }

    log = logging.getLogger('client.SwaggerApiClient')
    spec_option = 'spec'

    def setup_swagger(self):
        self.http_client = RequestsClient()
        self.http_client.session.verify = False

        spec = pytest.config.getoption(self.spec_option)
        self.client = SwaggerClient.from_spec(load_file(spec),
                                              config=self.config,
                                              http_client=self.http_client)

        self.client.swagger_spec.api_url = "http://{}/api/{}/v1/admission".format(pytest.config.getoption("host"), self.api_type)

    def make_api_url(self, path):
        return os.path.join(self.client.swagger_spec.api_url,
                            path if not path.startswith("/") else path[1:])

class InternalClient(SwaggerApiClient):
    log = logging.getLogger('client.InternalClient')

    spec_option = 'internal_spec'
    api_type = "internal"

    uauth = {"Authorization": "Bearer foobarbaz"}

    def setup(self):
        self.setup_swagger()

    def delete_device(self, id, auth=None):
        return requests.delete(self.make_api_url('/devices?device_id={}'.format(id)), headers=auth)

    def make_id_auth(self, user, tenant):
        jwt = common.make_id_jwt(user, tenant)
        return {"Authorization" : "Bearer " + jwt}

    def create_tenant(self, tenant_id):
        return self.client.tenants.post_tenants(tenant={
                    "tenant_id": tenant_id}).result()

    def change_status(self, authset_id, status, auth=None):
        if auth is None:
            auth = self.uauth

        Status = self.client.get_model('Status')
        s = Status(status=status)

        self.client.devices.put_devices_id_status(id=authset_id, status=s, _request_options={"headers": auth}).result()


class ManagementClient(SwaggerApiClient):
    log = logging.getLogger('client.ManagementClient')

    spec_option = 'management_spec'
    api_type = "management"

    # default user auth - single user, single tenant
    uauth = {"Authorization": "Bearer foobarbaz"}

    def setup(self):
        self.setup_swagger()

    def get_all_devices(self, page=1, auth=None):
        if auth is None:
            auth=self.uauth
        r, h = self.client.devices.get_devices(page=page, _request_options={"headers": auth}).result()
        for i in parse_header_links(h.headers["link"]):
            if i["rel"] == "next":
                page = int(dict(urlparse.parse_qs(urlparse.urlsplit(i["url"]).query))["page"][0])
                return r + self.get_all_devices(page=page, auth=auth)
        else:
            return r

    def change_status(self, authset_id, status, auth=None):
        if auth is None:
            auth = self.uauth

        Status = self.client.get_model('Status')
        s = Status(status=status)

        self.client.devices.put_devices_id_status(id=authset_id, status=s, _request_options={"headers": auth}).result()

    def preauthorize(self, identity, key, auth=None):
        """
            Add a preauthorized device.
        """
        if auth is None:
            auth = self.uauth

        AuthSet = self.client.get_model('AuthSet')
        authset = AuthSet(
                device_identity=identity,
                key=key)

        self.client.devices.post_devices(auth_set=authset, _request_options={"headers": auth}).result()

    def make_user_auth(self, user_id, tenant_id=None):
        """
            Prepare an almost-valid JWT auth header, suitable for consumption by deviceadm.
        """
        jwt = common.make_id_jwt(user_id, tenant_id)
        return {"Authorization": "Bearer " + jwt}


class ManagementClientSimple(ManagementClient):
    log = logging.getLogger('client.ManagementClientSimple')

    def __init__(self):
        self.setup_swagger()

class InternalClientSimple(InternalClient):
    log = logging.getLogger('client.InternalClientSimple')

    def __init__(self):
        self.setup_swagger()

class CliClient:
    cmd = '/testing/deviceadm'

    def migrate(self, tenant_id=None):
        args = [
            self.cmd,
            'migrate']

        if tenant_id:
            args += ['--tenant', tenant_id]

        subprocess.run(args, check=True)
