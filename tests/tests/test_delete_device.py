from client import ManagementClientSimple, InternalClient
from common import create_devices, create_devices_mt, api_client_mgmt
import pytest

@pytest.mark.usefixtures("create_devices")
class TestDeleteDevice(InternalClient):
    def do_test_delete(self, auth=None):
        #first pull existing device sets to get some existing device_id
        mc = ManagementClientSimple()
        devs = mc.get_all_devices(auth=auth)
        num_devs_base = len(devs)

        assert num_devs_base > 0

        existing = devs[0]

        #delete existing device set, verify it's gone
        rsp = self.delete_device(existing.device_id, auth)
        assert rsp.status_code == 204

        devs = mc.get_all_devices(auth=auth)
        num_devs = len(devs)
        assert num_devs == num_devs_base - 1

        found = [x for x in devs if x.device_id == existing.device_id]
        assert len(found) == 0

        #delete nonexistent device set, verify nothing's missing
        rsp = self.delete_device('foobar')
        assert rsp.status_code == 204

        devs = mc.get_all_devices(auth=auth)
        assert len(devs) == num_devs

    def test_delete(self):
        self.do_test_delete()

@pytest.mark.usefixtures("create_devices_mt")
class TestDeleteDeviceMultitenant(TestDeleteDevice):
    @pytest.mark.parametrize("tenant_id", ["tenant1", "tenant2"])
    def test_delete(self, tenant_id):
        auth = self.make_id_auth("foo", tenant_id)
        self.do_test_delete(auth=auth)
