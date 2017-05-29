from client import ManagementClient
import bravado
from common import create_devices
import pytest

@pytest.mark.usefixtures("create_devices")
class TestPrebootstrap(ManagementClient):

    def test_get_all_devices(self, expected_total=pytest.config.getoption("devices")):
        """
            Test all devices appear after being bootstrapped
        """
        assert len(self.get_all_devices()) >= int(expected_total)


    def test_get_devices(self):
        """
            Test getting a specific device
        """
        devices = self.get_all_devices()
        first = devices[0]
        assert first.status == "pending"


    def test_get_non_existant_device(self):
        """
            Test getting a specific device results in 404
        """
        try:
            self.client.devices.get_devices_id(id="0c396e0032f2b4367d6abe709c889ced728df1f97eb0c368a41465aa24a89454", _request_options={"headers": auth}).result()
        except bravado.exception.HTTPError as e:
            assert e.response.status_code == 404
        else:
            pytest.fail("Error code 404 not returned")

