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
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

const (
	testDataFolder = "testdata"
)

// these tests need a real instance of mongodb
// hardcoding the address here - valid both for the Travis env and local dev env
const TestDb = "127.0.0.1:27017"

// db and test management funcs
func getDb() (*DataStoreMongo, error) {
	d, err := NewDataStoreMongo(TestDb)
	if err != nil {
		return nil, err
	}

	return d, nil
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

	d, err := getDb()
	if err != nil {
		t.Fatalf(err.Error())
	}

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
		if err != nil {
			t.Fatalf("failed to wipe data, error: %s", err.Error())
		}

		err = setUp(d, tc.input)
		if err != nil {
			t.Fatalf("failed to setup input data %s, error: %s", tc.expected, err.Error())
		}

		expected, err := parseDevs(tc.expected)
		if err != nil {
			t.Fatalf("failed to parse expected devs %s, error: %s", tc.expected, err.Error())
		}

		//test
		devs, err := d.GetDevices(tc.skip, tc.limit, tc.status)
		if err != nil {
			t.Fatalf("failed to get devices, error: %s", err.Error())

		}

		if !reflect.DeepEqual(expected, devs) {
			t.Fatalf("expected: %v\nhave: %v", expected, devs)
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
