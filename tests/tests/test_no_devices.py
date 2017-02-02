from client import Client
import bravado
from common import create_devices
import pytest


class TestPrebootstrap(Client):


    def test_get_all_devices_zero(self, expected_total=0):
        """
            Test no devices are returned when getting devices prebootstrap
        """
        assert len(self.get_all_devices()) == expected_total


    def test_get_non_existant_device(self):
        """
            Test getting a device when non exist results in 404
        """
        try:
            self.client.devices.get_devices_id(id="0c396e0032f2b4367d6abe709c889ced728df1f97eb0c368a41465aa24a89454", _request_options=self.uauth).result()
        except bravado.exception.HTTPError, e:
            assert e.response.status_code == 404
        else:
            pytest.fail("Error code 404 not returned")


    def test_get_non_existant_device_status(self):
        """
            Test getting a device status when non exist results in 404
        """
        try:
            self.client.devices.get_devices_id_status(id="ac396e0032f2b4367d6abe709c889ced728df1f97eb0c368a41465aa24a89454", _request_options=self.uauth).result()
        except bravado.exception.HTTPError, e:
            assert e.response.status_code == 404
        else:
            pytest.fail("Error code 404 not returned")


    def test_put_non_existant_device_status_change(self):
        """
            Test setting a device status when device does not exist results in 404
        """
        try:
            Status = self.client.get_model('Status')
            s = Status(status="accepted")
            self.client.devices.put_devices_id_status(id="ac396e0032f2b4367d6abe709c889ced728df1f97eb0c368a41465aa24a89454", status=s, _request_options=self.uauth).result()
        except bravado.exception.HTTPError, e:
            assert e.response.status_code == 404
        else:
            pytest.fail("Error code 404 not returned")
