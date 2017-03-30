from client import ManagementClientSimple, InternalClient
from common import create_devices
import pytest

@pytest.mark.usefixtures("create_devices")
class TestDeleteDevice(InternalClient):
    def test_delete(self):
        #first pull existing device sets to get some existing device_id
        mc = ManagementClientSimple()
        devs = mc.get_all_devices()
        num_devs_base = len(devs)

        assert num_devs_base > 0

        existing = devs[0]

        #delete existing device set, verify it's gone
        rsp = self.delete_device(existing.device_id)
        assert rsp.status_code == 204

        devs = mc.get_all_devices()
        num_devs = len(devs)
        assert num_devs == num_devs_base - 1

        found = [x for x in devs if x.device_id == existing.device_id]
        assert len(found) == 0

        #delete nonexistent device set, verify nothing's missing
        rsp = self.delete_device('foobar')
        assert rsp.status_code == 204

        devs = mc.get_all_devices()
        assert len(devs) == num_devs

