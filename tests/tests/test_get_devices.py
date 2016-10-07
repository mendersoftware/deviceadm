from client import Client
import bravado
from common import create_devices
import pytest

@pytest.mark.usefixtures("create_devices")
class TestPrebootstrap(Client):

    "Test all devices appear after being bootstrapped"
    def test_get_all_devices(self, expected_total=pytest.config.getoption("devices")):
        assert len(self.get_all_devices()) == int(expected_total)

    "Test getting a specific device"
    def test_get_devices(self):
        devices = self.get_all_devices()
        first = devices[0]
        assert first.status == "pending"

    "Test getting a specific device results in 404"
    def test_get_non_existant_device(self):
        try:
            self.client.devices.get_devices_id(id="0c396e0032f2b4367d6abe709c889ced728df1f97eb0c368a41465aa24a89454").result()
        except bravado.exception.HTTPError, e:
            assert e.response.status_code == 404
        else:
            pytest.fail("Error code 404 not returned")

