from client import Client
import bravado
from common import create_devices
import pytest

@pytest.mark.usefixtures("create_devices")
class TestPrebootstrap(Client):

    def change_status(self, device_id, expected_initial, expected_final, expected_error_code=None):
        Status = self.client.get_model('Status')
        s = Status(status=expected_final)
        try:
            actual_initial = self.client.devices.get_devices_id(id=device_id).result()[0].status 
            assert actual_initial == expected_initial
            self.client.devices.put_devices_id_status(id=device_id, status=s).result()
        except bravado.exception.HTTPError, e:
            assert e.response.status_code == expected_error_code
            return
        else:
            if expected_error_code is not None:
                pytest.fail("Expected an exception, but didnt get any!")
                return

        assert self.client.devices.get_devices_id(id=device_id).result()[0].status == expected_final


    def test_change_status(self):
        """
            Test every possible status transition works, invalid and non-specified transitions fail
        """
        r, h = self.client.devices.get_devices().result()

        firstDevice = r[0]
        secondDevice = r[1]
        thirdDevice = r[2]

        # go from pending => accepted
        self.change_status(firstDevice.id, expected_initial="pending", expected_final="accepted")

        # go from pending => rejected
        self.change_status(secondDevice.id, expected_initial="pending", expected_final="rejected")

        # go from rejected => accepted
        self.change_status(secondDevice.id, expected_initial="rejected", expected_final="accepted")

        # go from accepted => rejected
        self.change_status(firstDevice.id, expected_initial="accepted", expected_final="rejected")

        # go from pending => rejected => accepted
        self.change_status(thirdDevice.id, expected_initial="pending", expected_final="rejected")
        self.change_status(thirdDevice.id, expected_initial="rejected", expected_final="accepted")

        # go from rejected => pending
        self.change_status(firstDevice.id, expected_initial="rejected", expected_final="pending", expected_error_code=400)

        # go from accepted => pending
        self.change_status(secondDevice.id, expected_initial="accepted", expected_final="pending", expected_error_code=400)

        # go from accepted => blah
        self.change_status(secondDevice.id, expected_initial="accepted", expected_final="blah", expected_error_code=400)

        # device not found
        self.change_status(secondDevice.id+'1', expected_initial="accepted", expected_final="pending", expected_error_code=404)
