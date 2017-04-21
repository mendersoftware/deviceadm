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
package mongo

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/mendersoftware/go-lib-micro/mongo/migrate"
	"github.com/stretchr/testify/assert"

	"github.com/mendersoftware/deviceadm/model"
	"github.com/mendersoftware/deviceadm/store"
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

// randStatus returns a randomly chosen status
func randStatus() string {
	statuses := []string{
		model.DevStatusAccepted,
		model.DevStatusPending,
		model.DevStatusRejected,
	}
	idx := rand.Int() % len(statuses)
	return statuses[idx]
}

// makeDevs generates `count` distinct devices, with `auts`PerDevice` auth data
// sets for each device. Within a device, auth sets have different device key,
// identity data remains the same. Each auth set is given an ID with 0000-0000
// format (<dev-idx>-<auth-for-dev-idx>), eg. 0002-0003 is 3rd device, 4th auth
// set of this device. Device auth statuses are picked randomly.
func makeDevs(count int, authsPerDevice int) []model.DeviceAuth {
	devs := make([]model.DeviceAuth, count*authsPerDevice)

	for i := 0; i < count; i++ {
		base_id := fmt.Sprintf("%04d", i)
		identity := fmt.Sprintf("device-identity-%s", base_id)
		attrs := model.DeviceAuthAttributes{
			"someattr": fmt.Sprintf("00:00:%s", base_id),
		}
		devid := model.DeviceID(fmt.Sprintf("devid-%s", base_id))

		for j := 0; j < authsPerDevice; j++ {
			auth_id := fmt.Sprintf("%s-%04d", base_id, j)
			devs[i*authsPerDevice+j] = model.DeviceAuth{
				ID:             model.AuthID(auth_id),
				DeviceId:       devid,
				DeviceIdentity: identity,
				Key:            fmt.Sprintf("key-%s", auth_id),
				Status:         randStatus(),
				Attributes:     attrs,
			}
		}
	}
	return devs
}

func setUp(db *DataStoreMongo, devs []model.DeviceAuth) error {
	s := db.session.Copy()
	defer s.Close()

	c := s.DB(DbName).C(DbDevicesColl)

	for _, d := range devs {
		err := c.Insert(d)
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

func parseDevs(dataset string) ([]model.DeviceAuth, error) {
	f, err := os.Open(filepath.Join(testDataFolder, dataset))
	if err != nil {
		return nil, err
	}

	var devs []model.DeviceAuth

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

	_, err = d.GetDeviceAuths(context.Background(), 0, 5, store.Filter{})
	if err != nil {
		t.Fatalf(err.Error())
	}

	testCases := []struct {
		skip   int
		limit  int
		filter store.Filter
	}{
		{
			0,
			20,
			store.Filter{},
		},
		{
			7,
			20,
			store.Filter{},
		},
		{
			0,
			3,
			store.Filter{},
		},
		{
			3,
			5,
			store.Filter{},
		},
		{
			0,
			20,
			store.Filter{Status: model.DevStatusAccepted},
		},
		{
			3,
			2,
			store.Filter{Status: model.DevStatusPending},
		},
		{
			0,
			0,
			store.Filter{DeviceID: "devid-0001"},
		},
		{
			0,
			0,
			store.Filter{
				DeviceID: "devid-0000",
				Status:   model.DevStatusAccepted,
			},
		},
	}

	// 30 devauths, 6 for every device
	devs := makeDevs(5, 6)
	// auth statuses are random, so we need to add an entry for devid-0000
	// with known status 'accepted'
	known := devs[0]
	known.Key = known.Key + "-known"
	known.ID = known.ID + "-known"
	known.Status = model.DevStatusAccepted
	devs = append(devs, known)

	for idx, tc := range testCases {
		t.Logf("tc: %v", idx)
		//setup
		err = wipe(d)
		assert.NoError(t, err, "failed to wipe data")

		err = setUp(d, devs)
		assert.NoError(t, err, "failed to setup input data")

		//test
		dbdevs, err := d.GetDeviceAuths(context.Background(),
			tc.skip, tc.limit, tc.filter)
		assert.NoError(t, err, "failed to get devices")

		if tc.limit != 0 {
			assert.True(t, len(dbdevs) > 0 && len(dbdevs) <= tc.limit)
		} else {
			assert.NotEmpty(t, dbdevs)
		}

		if tc.filter.Status != "" {
			for _, d := range dbdevs {
				assert.Equal(t, tc.filter.Status, d.Status)
			}
		}
		if tc.filter.DeviceID != "" {
			for _, d := range dbdevs {
				assert.Equal(t, tc.filter.DeviceID, d.DeviceId)
			}
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

	_, err = d.GetDeviceAuth(context.Background(), "")
	assert.Error(t, err, "expected error")

	// populate DB
	devs := makeDevs(100, 3)
	err = setUp(d, devs)
	assert.NoError(t, err, "failed to setup input data")

	// we're going to go through all expected devices just for the
	// sake of it
	expected := devs

	for _, dev := range expected {
		// we expect to find a device that was present in the
		// input set
		dbdev, err := d.GetDeviceAuth(context.Background(), dev.ID)
		assert.NoError(t, err, "expected no error")
		assert.NotNil(t, dbdev, "expected to device of ID %s to be found",
			dev.ID)
		// obviously the found device should be identical
		assert.True(t, reflect.DeepEqual(dev, *dbdev), "expected dev %+v to be equal to %+v",
			dbdev, dev)

		// modify device ID by appending bogus string to it
		dbdev, err = d.GetDeviceAuth(context.Background(), dev.ID+"-foobar")
		assert.Nil(t, dbdev, "expected nil got %+v", dbdev)
		assert.EqualError(t, err, store.ErrNotFound.Error(), "expected error")
	}

}

func TestMongoPutDevice(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping TestMongoGetDevice in short mode.")
	}

	d := getDb()
	defer d.session.Close()
	var err error

	_, err = d.GetDeviceAuth(context.Background(), "")
	assert.Error(t, err, "expected error")

	// populate DB
	devs := makeDevs(100, 3)
	err = setUp(d, devs)
	assert.NoError(t, err, "failed to setup input data")

	// insert all devices to DB
	for _, dev := range devs {
		err := d.PutDeviceAuth(context.Background(), &dev)
		assert.NoError(t, err, "expected no error inserting to data store")
	}

	// get devices, one by one
	for _, dev := range devs {
		// we expect to find a device that was present in the
		// input set
		dbdev, err := d.GetDeviceAuth(context.Background(), dev.ID)
		assert.NoError(t, err, "expected no error")
		assert.NotNil(t, dbdev, "expected to device of ID %s to be found",
			dev.ID)

		// obviously the found device should be identical
		assert.True(t, reflect.DeepEqual(dev, *dbdev), "expected dev %+v to be equal to %+v",
			dbdev, dev)

		// modify device staus
		ndev := model.DeviceAuth{
			Status: "accepted",
			ID:     dbdev.ID,
		}

		// update device key
		err = d.PutDeviceAuth(context.Background(), &ndev)
		assert.NoError(t, err, "expected no error updating devices in DB")
	}

	// get devices, one by one, check if status is set to accepted
	for _, dev := range devs {
		// we expect to find a device that was present in the
		// input set
		dbdev, err := d.GetDeviceAuth(context.Background(), dev.ID)
		assert.NoError(t, err, "expected no error")
		assert.NotNil(t, dbdev, "expected to device of ID %s to be found",
			dev.ID)

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

	dev, err := d.GetDeviceAuth(context.Background(), "foobar")
	assert.Nil(t, dev)
	assert.EqualError(t, err, store.ErrNotFound.Error())

	now := time.Now()
	expdev := model.DeviceAuth{
		ID:          "foobar",
		DeviceId:    "bar",
		RequestTime: &now,
		Attributes: model.DeviceAuthAttributes{
			"foo": "bar",
		},
	}
	err = d.PutDeviceAuth(context.Background(), &expdev)
	assert.NoError(t, err)

	dev, err = d.GetDeviceAuth(context.Background(), "foobar")
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
		DbVersion: {
			version: DbVersion,
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

		err := store.Migrate(context.Background(), tc.version)
		if tc.err == "" {
			assert.NoError(t, err)
			// list migrations
			var out []migrate.MigrationEntry
			session.DB(DbName).C(migrate.DbMigrationsColl).Find(nil).All(&out)
			sort.Slice(out, func(i int, j int) bool {
				return migrate.VersionIsLess(out[i].Version, out[j].Version)
			})
			// the last migration should match what we want
			v, _ := migrate.NewVersion(tc.version)
			assert.Equal(t, *v, out[len(out)-1].Version)
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

	inDevs := []model.DeviceAuth{
		{
			ID:             "0001",
			DeviceId:       "0001",
			DeviceIdentity: "0001-id",
			Key:            "0001-key",
			Status:         "pending",
		},
		{
			ID:             "0002",
			DeviceId:       "0002",
			DeviceIdentity: "0002-id",
			Key:            "0002-key",
			Status:         "pending",
		},
	}

	testCases := map[string]struct {
		id  model.AuthID
		out []model.DeviceAuth
		err error
	}{
		"exists 1": {
			id: "0001",
			out: []model.DeviceAuth{
				{
					ID:             "0002",
					DeviceId:       "0002",
					DeviceIdentity: "0002-id",
					Key:            "0002-key",
					Status:         "pending",
				},
			},
			err: nil,
		},
		"exists 2": {
			id: "0002",
			out: []model.DeviceAuth{
				{
					ID:             "0001",
					DeviceId:       "0001",
					DeviceIdentity: "0001-id",
					Key:            "0001-key",
					Status:         "pending",
				},
			},
			err: nil,
		},
		"doesn't exist": {
			id: "foo",
			out: []model.DeviceAuth{
				{
					ID:             "0001",
					DeviceId:       "0001",
					DeviceIdentity: "0001-id",
					Key:            "0001-key",
					Status:         "pending",
				},
				{
					ID:             "0002",
					DeviceId:       "0002",
					DeviceIdentity: "0002-id",
					Key:            "0002-key",
					Status:         "pending",
				},
			},
			err: store.ErrNotFound,
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

		err := store.DeleteDeviceAuth(context.Background(), tc.id)
		if tc.err != nil {
			assert.EqualError(t, err, tc.err.Error())
		} else {
			assert.NoError(t, err, "failed to delete device")
		}

		var outDevs []model.DeviceAuth
		err = session.DB(DbName).C(DbDevicesColl).Find(nil).All(&outDevs)
		assert.NoError(t, err, "failed to verify devices")

		assert.True(t, reflect.DeepEqual(tc.out, outDevs))

		session.Close()
	}

}

func TestMongoDeleteDeviceAuthsByDevice(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode.")
	}

	inDevs := []model.DeviceAuth{
		{
			ID:             "0001",
			DeviceId:       "0001",
			DeviceIdentity: "0001-id",
			Key:            "0001-key",
			Status:         "pending",
		},
		{
			ID:             "0002",
			DeviceId:       "0001",
			DeviceIdentity: "0002-id",
			Key:            "0002-key",
			Status:         "pending",
		},
		{
			ID:             "0003",
			DeviceId:       "0002",
			DeviceIdentity: "0002-id",
			Key:            "0002-key",
			Status:         "pending",
		},
	}

	db.Wipe()
	session := db.Session()
	defer session.Close()

	dbstore := NewDataStoreMongoWithSession(session)

	for _, d := range inDevs {
		err := dbstore.PutDeviceAuth(context.Background(), &d)
		assert.NoError(t, err)
	}

	err := dbstore.DeleteDeviceAuthByDevice(context.Background(), "0001")
	assert.NoError(t, err)

	for _, aid := range []model.AuthID{"0001", "0002"} {
		_, err := dbstore.GetDeviceAuth(context.Background(), aid)
		assert.EqualError(t, err, store.ErrNotFound.Error())
	}

	aset, err := dbstore.GetDeviceAuth(context.Background(), "0003")
	assert.NoError(t, err)
	assert.NotNil(t, aset)

	err = dbstore.DeleteDeviceAuthByDevice(context.Background(), "0004")
	assert.EqualError(t, err, store.ErrNotFound.Error())
}
