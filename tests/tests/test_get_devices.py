from client import ManagementClient
import bravado
from common import create_devices, create_devices_mt, api_client_mgmt
import pytest

@pytest.mark.usefixtures("create_devices")
class TestPrebootstrap(ManagementClient):
    def do_test_get_non_existant_device(self, auth):
        """
            Test getting a specific device results in 404
        """
        try:
            self.client.devices.get_devices_id(id="0c396e0032f2b4367d6abe709c889ced728df1f97eb0c368a41465aa24a89454", _request_options={"headers": auth}).result()
        except bravado.exception.HTTPError as e:
            assert e.response.status_code == 404
        else:
            pytest.fail("Error code 404 not returned")

    def test_get_non_existant_device(self):
        self.do_test_get_non_existant_device(auth=self.uauth)


@pytest.mark.usefixtures("create_devices_mt")
class TestPrebootstrapMultitenant(TestPrebootstrap):
    @pytest.mark.parametrize("tenant_id", ["tenant1", "tenant2"])
    def test_get_non_existant_device(self, tenant_id):
        auth = self.make_user_auth("user", tenant_id)
        self.do_test_get_non_existant_device(auth=auth)
