from client import ManagementClient
import bravado
from common import create_devices, create_devices_mt, api_client_mgmt
from tenantadm import fake_tenantadm
import pytest

@pytest.mark.usefixtures("create_devices")
class TestPrebootstrap(ManagementClient):

    def change_status(self, device_id, expected_initial, expected_final, expected_error_code=None, auth=None):
        if auth is None:
            auth = self.uauth
        Status = self.client.get_model('Status')
        s = Status(status=expected_final)
        try:
            actual_initial = self.client.devices.get_devices_id(id=device_id, _request_options={"headers": auth}).result()[0].status
            assert actual_initial == expected_initial
            self.client.devices.put_devices_id_status(id=device_id, status=s, _request_options={"headers": auth}).result()
        except bravado.exception.HTTPError as e:
            assert e.response.status_code == expected_error_code
            return
        else:
            if expected_error_code is not None:
                pytest.fail("Expected an exception, but didnt get any!")
                return

        assert self.client.devices.get_devices_id(id=device_id, _request_options={"headers": auth}).result()[0].status == expected_final

    def do_test_change_status(self, auth):
        r, h = self.client.devices.get_devices(_request_options={"headers": auth}).result()

        firstDevice = r[0]
        secondDevice = r[1]
        thirdDevice = r[2]

        # go from pending => accepted
        self.change_status(firstDevice.id, expected_initial="pending", expected_final="accepted", auth=auth)

        # go from pending => rejected
        self.change_status(secondDevice.id, expected_initial="pending", expected_final="rejected", auth=auth)

        # go from rejected => accepted
        self.change_status(secondDevice.id, expected_initial="rejected", expected_final="accepted", auth=auth)

        # go from accepted => rejected
        self.change_status(firstDevice.id, expected_initial="accepted", expected_final="rejected", auth=auth)

        # go from pending => rejected => accepted
        self.change_status(thirdDevice.id, expected_initial="pending", expected_final="rejected", auth=auth)
        self.change_status(thirdDevice.id, expected_initial="rejected", expected_final="accepted", auth=auth)

        # go from rejected => pending
        self.change_status(firstDevice.id, expected_initial="rejected", expected_final="pending", expected_error_code=400, auth=auth)

        # go from accepted => pending
        self.change_status(secondDevice.id, expected_initial="accepted", expected_final="pending", expected_error_code=400, auth=auth)

        # go from accepted => blah
        self.change_status(secondDevice.id, expected_initial="accepted", expected_final="blah", expected_error_code=400, auth=auth)

        # device not found
        self.change_status(secondDevice.id+'1', expected_initial="accepted", expected_final="pending", expected_error_code=404, auth=auth)

    def test_change_status(self):
        """
            Test every possible status transition works, invalid and non-specified transitions fail
        """
        self.do_test_change_status(auth=self.uauth)

@pytest.mark.usefixtures("create_devices_mt")
class TestPrebootstrapMultitenant(TestPrebootstrap):
    @pytest.mark.parametrize("tenant_id", ["tenant1", "tenant2"])
    def test_change_status(self, tenant_id):
        """
            Test every possible status transition works, invalid and non-specified transitions fail
        """
        auth = self.make_user_auth("user", tenant_id)
        with fake_tenantadm():
            self.do_test_change_status(auth)
