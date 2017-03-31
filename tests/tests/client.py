import logging
import os

import pytest
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

        self.client.swagger_spec.api_url = "http://{}/api/{}/".format(pytest.config.getoption("host"),
                                                                      pytest.config.getoption("api"))

    def make_api_url(self, path):
        return os.path.join(self.client.swagger_spec.api_url,
                            path if not path.startswith("/") else path[1:])

class InternalClient(SwaggerApiClient):
    log = logging.getLogger('client.InternalClient')

    spec_option = 'internal_spec'

    def setup(self):
        self.setup_swagger()

    def delete_device(self, id):
        return requests.delete(self.make_api_url('/devices?device_id={}'.format(id)))

class ManagementClient(SwaggerApiClient):
    log = logging.getLogger('client.ManagementClient')

    spec_option = 'management_spec'

    #user auth - dummy, just to make swagger client happy
    uauth = {"headers": {"Authorization": "Bearer foobarbaz"}}

    def setup(self):
        self.setup_swagger()

    def get_all_devices(self, page=1):
        r, h = self.client.devices.get_devices(page=page, _request_options=self.uauth).result()
        for i in parse_header_links(h.headers["link"]):
            if i["rel"] == "next":
                page = int(dict(urlparse.parse_qs(urlparse.urlsplit(i["url"]).query))["page"][0])
                return r + self.get_all_devices(page=page)
        else:
            return r

class ManagementClientSimple(ManagementClient):
    log = logging.getLogger('client.ManagementClientSimple')

    def __init__(self):
        self.setup_swagger()
