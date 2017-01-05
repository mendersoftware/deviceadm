#!/bin/bash
py.test -s --tb=short --api=0.1.0  --host mender-device-adm:8080 --devices 101 --verbose --junitxml=results.xml tests/{test_no_devices.py,test_get_devices.py,test_put_devices.py}
