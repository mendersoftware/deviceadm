// Copyright 2016 Mender Software AS
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/mendersoftware/go-lib-micro/mongo/migrate"
	"github.com/stretchr/testify/assert"
)

const (
	testDataFolder  = "testdata"
	allDevsInputSet = "get_devices_input.json"
)

// db and test management funcs
func getDb() *DataStoreMongo {
	db.Wipe()
	return NewDataStoreMongoWithSession(db.Session())
}

func setUp(db *DataStoreMongo, dataset string) error {
	devs, err := parseDevs(dataset)
	if err != nil {
		return err
	}

	s := db.session.Copy()
	defer s.Close()

	c := s.DB(DbName).C(DbDevicesColl)

	for _, d := range devs {
		err = c.Insert(d)
		if err != nil {
			return err
		}
	}

	return nil
}

func wipe(db *DataStoreMongo) error {
	s := db.session.Copy()
	defer s.Close()

	c := s.DB(DbName).C(DbDevicesColl)

	_, err := c.RemoveAll(nil)
	if err != nil {
		return err
	}

	return nil
}

func parseDevs(dataset string) ([]Device, error) {
	f, err := os.Open(filepath.Join(testDataFolder, dataset))
	if err != nil {
		return nil, err
	}

	var devs []Device

	j := json.NewDecoder(f)
	if err = j.Decode(&devs); err != nil {
		return nil, err
	}

	return devs, nil
}

// test funcs
func TestMongoGetDevices(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping TestMongoGetDevices in short mode.")
	}

	d := getDb()
	defer d.session.Close()

	var err error

	_, err = d.GetDevices(0, 5, "")
	if err != nil {
		t.Fatalf(err.Error())
	}

	testCases := []struct {
		input    string
		expected string
		skip     int
		limit    int
		status   string
	}{
		{
			//all devs, no skip, no limit
			"get_devices_input.json",
			"get_devices_input.json",
			0,
			20,
			"",
		},
		{
			//all devs, with skip
			"get_devices_input.json",
			"get_devices_expected_skip.json",
			7,
			20,
			"",
		},
		{
			//all devs, no skip, with limit
			"get_devices_input.json",
			"get_devices_expected_limit.json",
			0,
			3,
			"",
		},
		{
			//skip + limit
			"get_devices_input.json",
			"get_devices_expected_skip_limit.json",
			3,
			5,
			"",
		},
		{
			//status = accepted
			"get_devices_input.json",
			"get_devices_expected_status_acc.json",
			0,
			20,
			"accepted",
		},
		{
			//status = pending, skip, limit
			"get_devices_input.json",
			"get_devices_expected_status_skip_limit.json",
			3,
			2,
			"pending",
		},
	}

	for _, tc := range testCases {
		//setup
		err = wipe(d)
		assert.NoError(t, err, "failed to wipe data")

		err = setUp(d, tc.input)
		assert.NoError(t, err, "failed to setup input data %s", tc.expected)

		expected, err := parseDevs(tc.expected)
		assert.NoError(t, err, "failed to parse expected devs %s", tc.expected)

		//test
		devs, err := d.GetDevices(tc.skip, tc.limit, tc.status)
		assert.NoError(t, err, "failed to get devices")

		if !reflect.DeepEqual(expected, devs) {
			assert.Fail(t, "expected: %v\nhave: %v", expected, devs)
		}
	}
}

func TestFailedConn(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping TestMongoGetDevices in short mode.")
	}
	_, err := NewDataStoreMongo("invalid:27017")
	assert.NotNil(t, err)
}

func TestMongoGetDevice(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping TestMongoGetDevice in short mode.")
	}

	d := getDb()
	defer d.session.Close()
	var err error

	_, err = d.GetDevice("")
	assert.Error(t, err, "expected error")

	// populate DB
	err = setUp(d, allDevsInputSet)
	assert.NoError(t, err, "failed to setup input data %s", allDevsInputSet)

	// we're going to go through all expected devices just for the
	// sake of it
	expected, err := parseDevs(allDevsInputSet)
	assert.NoError(t, err, "failed to parse expected devs %s", allDevsInputSet)

	for _, dev := range expected {
		// we expect to find a device that was present in the
		// input set
		dbdev, err := d.GetDevice(dev.ID)
		assert.NoError(t, err, "expected no error")
		assert.NotNil(t, dbdev, "expected to device of ID %s to be found",
			dev.ID)
		// obviously the found device should be identical
		assert.True(t, reflect.DeepEqual(dev, *dbdev), "expected dev %+v to be equal to %+v",
			dbdev, dev)

		// modify device ID by appending bogus string to it
		dbdev, err = d.GetDevice(dev.ID + "-foobar")
		assert.Nil(t, dbdev, "expected nil got %+v", dbdev)
		assert.EqualError(t, err, ErrDevNotFound.Error(), "expected error")
	}

}

func TestMongoPutDevice(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping TestMongoGetDevice in short mode.")
	}

	d := getDb()
	defer d.session.Close()
	var err error

	_, err = d.GetDevice("")
	assert.Error(t, err, "expected error")

	// populate DB
	err = setUp(d, allDevsInputSet)
	assert.NoError(t, err, "failed to setup input data %s", allDevsInputSet)

	// all dataset of all devices
	devs, err := parseDevs(allDevsInputSet)
	assert.NoError(t, err, "failed to parse expected devs %s", allDevsInputSet)

	// insert all devices to DB
	for _, dev := range devs {
		err := d.PutDevice(&dev)
		assert.NoError(t, err, "expected no error inserting to data store")
	}

	// get devices, one by one
	for _, dev := range devs {
		// we expect to find a device that was present in the
		// input set
		dbdev, err := d.GetDevice(dev.ID)
		assert.NoError(t, err, "expected no error")
		assert.NotNil(t, dbdev, "expected to device of ID %s to be found",
			dev.ID)
		t.Logf("stored dev: %+v", dbdev)

		// obviously the found device should be identical
		assert.True(t, reflect.DeepEqual(dev, *dbdev), "expected dev %+v to be equal to %+v",
			dbdev, dev)

		// modify device staus
		ndev := Device{
			Status: "accepted",
			ID:     dbdev.ID,
		}

		// update device key
		err = d.PutDevice(&ndev)
		assert.NoError(t, err, "expected no error updating devices in DB")
	}

	// get devices, one by one, check if status is set to accepted
	for _, dev := range devs {
		// we expect to find a device that was present in the
		// input set
		dbdev, err := d.GetDevice(dev.ID)
		assert.NoError(t, err, "expected no error")
		assert.NotNil(t, dbdev, "expected to device of ID %s to be found",
			dev.ID)
		t.Logf("updated dev: %+v", dbdev)

		assert.Equal(t, "accepted", dbdev.Status)
		// other fields should be identical
		assert.Equal(t, dev.ID, dbdev.ID)
		assert.Equal(t, dev.DeviceIdentity, dbdev.DeviceIdentity)
		assert.Equal(t, dev.Key, dbdev.Key)
		assert.True(t, reflect.DeepEqual(dev.Attributes, dbdev.Attributes))
	}
}

func TestMongoPutDeviceTime(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping TestMongoPutDeviceTime in short mode.")
	}

	d := getDb()
	defer d.session.Close()
	var err error

	dev, err := d.GetDevice(DeviceID("foobar"))
	assert.Nil(t, dev)
	assert.EqualError(t, err, ErrDevNotFound.Error())

	now := time.Now()
	expdev := Device{
		ID:          DeviceID("foobar"),
		RequestTime: &now,
		Attributes: DeviceAttributes{
			"foo": "bar",
		},
	}
	err = d.PutDevice(&expdev)
	assert.NoError(t, err)

	dev, err = d.GetDevice(DeviceID("foobar"))
	assert.NotNil(t, dev)
	assert.NoError(t, err)

	t.Logf("go device: %v", dev)
	// cannot just compare expected device with one we got from db because
	// RequestTime might have been trimmed by mongo
	assert.ObjectsAreEqualValues(expdev.Attributes, dev.Attributes)
	assert.Equal(t, expdev.ID, dev.ID)
	// time round off should be within 1s
	if assert.NotNil(t, dev.RequestTime) {
		assert.WithinDuration(t, time.Now(), *dev.RequestTime, time.Second)
	}
}

func TestMigrate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping TestMigrate in short mode.")
	}

	testCases := map[string]struct {
		version string
		err     string
	}{
		"0.1.0": {
			version: "0.1.0",
			err:     "",
		},
		"1.2.3": {
			version: "1.2.3",
			err:     "",
		},
		"0.1 error": {
			version: "0.1",
			err:     "failed to parse service version: failed to parse Version: unexpected EOF",
		},
	}

	for name, tc := range testCases {
		t.Logf("case: %s", name)
		db.Wipe()
		session := db.Session()

		store := NewDataStoreMongoWithSession(session)

		err := store.Migrate(tc.version, nil)
		if tc.err == "" {
			assert.NoError(t, err)
			var out []migrate.MigrationEntry
			session.DB(DbName).C(migrate.DbMigrationsColl).Find(nil).All(&out)
			assert.Len(t, out, 1)
			v, _ := migrate.NewVersion(tc.version)
			assert.Equal(t, v, out[0].Version)
		} else {
			assert.EqualError(t, err, tc.err)
		}

		session.Close()
	}

}

func TestMongoDeleteDevice(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping TestMongoDeleteDevice in short mode.")
	}

	inDevs := []Device{
		Device{
			ID:             DeviceID("0001"),
			DeviceIdentity: "0001-id",
			Key:            "0001-key",
			Status:         "pending",
		},
		Device{
			ID:             DeviceID("0002"),
			DeviceIdentity: "0002-id",
			Key:            "0002-key",
			Status:         "pending",
		},
	}

	testCases := map[string]struct {
		id  DeviceID
		out []Device
		err error
	}{
		"exists 1": {
			id: DeviceID("0001"),
			out: []Device{
				Device{
					ID:             DeviceID("0002"),
					DeviceIdentity: "0002-id",
					Key:            "0002-key",
					Status:         "pending",
				},
			},
			err: nil,
		},
		"exists 2": {
			id: DeviceID("0002"),
			out: []Device{
				Device{
					ID:             DeviceID("0001"),
					DeviceIdentity: "0001-id",
					Key:            "0001-key",
					Status:         "pending",
				},
			},
			err: nil,
		},
		"doesn't exist": {
			id: DeviceID("foo"),
			out: []Device{
				Device{
					ID:             DeviceID("0001"),
					DeviceIdentity: "0001-id",
					Key:            "0001-key",
					Status:         "pending",
				},
				Device{
					ID:             DeviceID("0002"),
					DeviceIdentity: "0002-id",
					Key:            "0002-key",
					Status:         "pending",
				},
			},
			err: ErrDevNotFound,
		},
	}

	for name, tc := range testCases {
		t.Logf("case: %s", name)
		db.Wipe()
		session := db.Session()

		for _, d := range inDevs {
			err := session.DB(DbName).C(DbDevicesColl).Insert(d)
			assert.NoError(t, err, "failed to setup input data")
		}

		store := NewDataStoreMongoWithSession(session)

		err := store.DeleteDevice(tc.id)
		if tc.err != nil {
			assert.EqualError(t, err, tc.err.Error())
		} else {
			assert.NoError(t, err, "failed to delete device")
		}

		var outDevs []Device
		err = session.DB(DbName).C(DbDevicesColl).Find(nil).All(&outDevs)
		assert.NoError(t, err, "failed to verify devices")

		assert.True(t, reflect.DeepEqual(tc.out, outDevs))

		session.Close()
	}

}
