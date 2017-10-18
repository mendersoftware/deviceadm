// Copyright 2017 Northern.tech AS
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
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/mendersoftware/go-lib-micro/identity"
	"github.com/mendersoftware/go-lib-micro/mongo/migrate"
	ctx_store "github.com/mendersoftware/go-lib-micro/store"
	"github.com/stretchr/testify/assert"
	"gopkg.in/mgo.v2/bson"

	"github.com/mendersoftware/deviceadm/model"
	"github.com/mendersoftware/deviceadm/store"
)

// getDb returns a new, clean, data store.
func getDb() *DataStoreMongo {
	db.Wipe()
	return NewDataStoreMongoWithSession(db.Session())
}

// getMigratedDb returns a new data store and applies migrations
func getMigratedDb(t *testing.T, ctx context.Context) *DataStoreMongo {
	ds := getDb()

	ds = ds.WithAutomigrate().(*DataStoreMongo)
	err := ds.Migrate(ctx, DbVersion)
	assert.NoError(t, err)

	return ds
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

func setUp(ctx context.Context, db *DataStoreMongo, devs []model.DeviceAuth) error {
	s := db.session.Copy()
	defer s.Close()

	c := s.DB(ctx_store.DbFromContext(ctx, DbName)).C(DbDevicesColl)

	for _, d := range devs {
		err := c.Insert(d)
		if err != nil {
			return err
		}
	}

	return nil
}

// test funcs
func TestMongoGetDevicesEmpty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping TestMongoGetDevicesEmpty in short mode.")
	}

	ctx := context.Background()

	d := getMigratedDb(t, ctx)
	defer d.session.Close()

	var err error

	_, err = d.GetDeviceAuths(ctx, 0, 5, store.Filter{})
	if err != nil {
		t.Fatalf(err.Error())
	}
}

func TestMongoGetDevices(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping TestMongoGetDevices in short mode.")
	}

	testCases := []struct {
		skip   int
		limit  int
		filter store.Filter
		tenant string
	}{
		{
			limit: 20,
		},
		{
			skip:  7,
			limit: 20,
		},
		{
			limit: 3,
		},
		{
			skip:  3,
			limit: 5,
		},
		{
			limit:  20,
			filter: store.Filter{Status: model.DevStatusAccepted},
		},
		{
			skip:   3,
			limit:  2,
			filter: store.Filter{Status: model.DevStatusPending},
		},
		{
			filter: store.Filter{DeviceID: "devid-0001"},
		},
		{
			filter: store.Filter{
				DeviceID: "devid-0000",
				Status:   model.DevStatusAccepted,
			},
		},
		{
			filter: store.Filter{
				DeviceID: "devid-0000",
				Status:   model.DevStatusAccepted,
			},
			tenant: "acme",
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

	for idx := range testCases {
		tc := testCases[idx]
		t.Run(fmt.Sprintf("tc: %v", idx), func(t *testing.T) {
			ctx := context.Background()

			//setup
			d := getMigratedDb(t, ctx)
			defer d.session.Close()

			if tc.tenant != "" {
				ctx = identity.WithContext(ctx, &identity.Identity{
					Subject: "foo",
					Tenant:  tc.tenant,
				})
			}
			err := setUp(ctx, d, devs)
			assert.NoError(t, err, "failed to setup input data")

			//test

			if tc.tenant != "" {
				// tenant identity is setup and kept in context, try
				// fetching devices using a context without identity,
				// this should return no devices
				dbdevs, err := d.GetDeviceAuths(context.Background(),
					tc.skip, tc.limit, tc.filter)
				assert.NoError(t, err, "failed to get devices")
				assert.Len(t, dbdevs, 0)
			}
			dbdevs, err := d.GetDeviceAuths(ctx, tc.skip, tc.limit, tc.filter)
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
		})
	}
}

func TestMongoGetDevice(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping TestMongoGetDevice in short mode.")
	}

	ctx := context.Background()

	d := getMigratedDb(t, ctx)
	defer d.session.Close()
	var err error

	_, err = d.GetDeviceAuth(ctx, "")
	assert.Error(t, err, "expected error")

	// populate DB
	devs := makeDevs(100, 3)
	err = setUp(ctx, d, devs)
	assert.NoError(t, err, "failed to setup input data")

	// we're going to go through all expected devices just for the
	// sake of it
	expected := devs

	for _, dev := range expected {
		// we expect to find a device that was present in the
		// input set
		dbdev, err := d.GetDeviceAuth(ctx, dev.ID)
		assert.NoError(t, err, "expected no error")
		assert.NotNil(t, dbdev, "expected to device of ID %s to be found",
			dev.ID)
		// obviously the found device should be identical
		assert.True(t, reflect.DeepEqual(dev, *dbdev), "expected dev %+v to be equal to %+v",
			dbdev, dev)

		// modify device ID by appending bogus string to it
		dbdev, err = d.GetDeviceAuth(ctx, dev.ID+"-foobar")
		assert.Nil(t, dbdev, "expected nil got %+v", dbdev)
		assert.EqualError(t, err, store.ErrNotFound.Error(), "expected error")
	}

	// now with tenant
	tenDevs := makeDevs(2, 1)
	// make sure that tenant devices have different ID
	for i := range tenDevs {
		tenDevs[i].ID = tenDevs[i].ID + "-foo"
	}
	tenCtx := identity.WithContext(ctx, &identity.Identity{
		Subject: "foo",
		Tenant:  "bar",
	})
	setUp(tenCtx, d, tenDevs)
	// a non-tenant device should not be found in tenant's DB
	_, err = d.GetDeviceAuth(tenCtx, devs[0].ID)
	assert.EqualError(t, err, store.ErrNotFound.Error())
	// try tenant's device now
	_, err = d.GetDeviceAuth(tenCtx, tenDevs[0].ID)
	assert.NoError(t, err)

}

func TestMongoPutDevice(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping TestMongoGetDevice in short mode.")
	}

	ctx := context.Background()

	d := getMigratedDb(t, ctx)
	defer d.session.Close()

	var err error

	_, err = d.GetDeviceAuth(ctx, "")
	assert.Error(t, err, "expected error")

	// populate DB
	devs := makeDevs(100, 3)
	err = setUp(ctx, d, devs)
	assert.NoError(t, err, "failed to setup input data")

	// insert all devices to DB
	for _, dev := range devs {
		err := d.PutDeviceAuth(ctx, &dev)
		assert.NoError(t, err, "expected no error inserting to data store")
	}

	// get devices, one by one
	for _, dev := range devs {
		// we expect to find a device that was present in the
		// input set
		dbdev, err := d.GetDeviceAuth(ctx, dev.ID)
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
		err = d.PutDeviceAuth(ctx, &ndev)
		assert.NoError(t, err, "expected no error updating devices in DB")
	}

	// get devices, one by one, check if status is set to accepted
	for _, dev := range devs {
		// we expect to find a device that was present in the
		// input set
		dbdev, err := d.GetDeviceAuth(ctx, dev.ID)
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

	// since everything work a tenant case is rather simple
	tenCtx := identity.WithContext(ctx, &identity.Identity{
		Subject: "foo",
		Tenant:  "bar",
	})
	devs = makeDevs(1, 1)
	err = setUp(tenCtx, d, devs)
	dbdev, err := d.GetDeviceAuth(tenCtx, devs[0].ID)
	err = d.PutDeviceAuth(tenCtx, &model.DeviceAuth{
		Status: "rejected",
		ID:     devs[0].ID,
	})
	assert.NoError(t, err)
	dbdev, err = d.GetDeviceAuth(tenCtx, devs[0].ID)
	assert.Equal(t, "rejected", dbdev.Status)

}

func TestMongoPutDeviceTime(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping TestMongoPutDeviceTime in short mode.")
	}

	ctx := context.Background()

	d := getMigratedDb(t, ctx)
	defer d.session.Close()
	var err error

	dev, err := d.GetDeviceAuth(ctx, "foobar")
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
	err = d.PutDeviceAuth(ctx, &expdev)
	assert.NoError(t, err)

	dev, err = d.GetDeviceAuth(ctx, "foobar")
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
		tenantDbs   []string
		automigrate bool

		version string
		err     string
	}{
		DbVersion: {
			automigrate: true,
			version:     DbVersion,
			err:         "",
		},
		DbVersion + " no automigrate": {
			automigrate: false,
			version:     DbVersion,
			err:         "failed to apply migrations: db needs migration: deviceadm has version 0.0.0, needs version 1.1.0",
		},
		DbVersion + " multitenant": {
			automigrate: true,
			tenantDbs:   []string{"deviceadm-tenant1id", "deviceadm-tenant2id"},
			version:     DbVersion,
			err:         "",
		},
		DbVersion + " multitenant, no automigrate": {
			automigrate: false,
			tenantDbs:   []string{"deviceadm-tenant1id", "deviceadm-tenant2id"},
			version:     DbVersion,
			err:         "failed to apply migrations: db needs migration: deviceadm-tenant1id has version 0.0.0, needs version 1.1.0",
		},

		"0.1 error": {
			automigrate: true,
			version:     "0.1",
			err:         "failed to parse service version: failed to parse Version: unexpected EOF",
		},
	}

	for name, tc := range testCases {
		t.Run(fmt.Sprintf("tc: %s", name), func(t *testing.T) {
			db.Wipe()
			db := NewDataStoreMongoWithSession(db.Session())

			// set up automigration
			if tc.automigrate {
				db = db.WithAutomigrate().(*DataStoreMongo)
			}

			// set up multitenancy/tenant dbs
			if len(tc.tenantDbs) != 0 {

				for _, d := range tc.tenantDbs {
					err := db.session.DB(d).C("foo").Insert(bson.M{"foo": "bar"})
					assert.NoError(t, err)
				}
			}

			ctx := context.Background()
			err := db.Migrate(ctx, tc.version)
			if tc.err == "" {
				assert.NoError(t, err)

				// verify migration entry in all databases (>1 if multitenant)
				if tc.automigrate {
					dbs := []string{DbName}
					if len(tc.tenantDbs) > 0 {
						dbs = tc.tenantDbs
					}

					for _, d := range dbs {
						var out []migrate.MigrationEntry
						db.session.DB(d).C(migrate.DbMigrationsColl).Find(nil).All(&out)
						sort.Slice(out, func(i int, j int) bool {
							return migrate.VersionIsLess(out[i].Version, out[j].Version)
						})
						// the last migration should match what we want
						v, _ := migrate.NewVersion(tc.version)
						assert.Equal(t, *v, out[len(out)-1].Version)

						// verify index exists
						c := db.session.DB(d).C(DbDevicesColl)
						indexes, err := c.Indexes()
						assert.NoError(t, err)
						haveIdx := false
						for _, i := range indexes {
							if i.Name == dbDeviceIdIndexName {
								haveIdx = true
							}
						}
						assert.True(t, haveIdx)
					}
				}

			} else {
				assert.EqualError(t, err, tc.err)
			}
			db.session.Close()
		})
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
		id     model.AuthID
		out    []model.DeviceAuth
		err    error
		tenant string
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
		"exists 2, with tenant": {
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
			tenant: "foo",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()

			d := getMigratedDb(t, ctx)
			defer d.session.Close()

			setUp(ctx, d, inDevs)

			if tc.tenant != "" {
				ctx = identity.WithContext(ctx, &identity.Identity{
					Subject: "foo",
					Tenant:  tc.tenant,
				})
				// once again but for tenant DB
				setUp(ctx, d, inDevs)
			}

			err := d.DeleteDeviceAuth(ctx, tc.id)
			if tc.err != nil {
				assert.EqualError(t, err, tc.err.Error())
			} else {
				assert.NoError(t, err, "failed to delete device")
			}

			if tc.tenant != "" {
				// device should still be present in default DB though,
				// as it's a separate storage
				_, err := d.GetDeviceAuth(context.Background(), tc.id)
				assert.NoError(t, err)
			}

			var outDevs []model.DeviceAuth
			err = d.session.DB(ctx_store.DbFromContext(ctx, DbName)).
				C(DbDevicesColl).Find(nil).All(&outDevs)
			assert.NoError(t, err, "failed to verify devices")

			assert.True(t, reflect.DeepEqual(tc.out, outDevs))
		})
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

	ctx := context.Background()
	tenCtx := identity.WithContext(ctx, &identity.Identity{
		Subject: "foo",
		Tenant:  "bar",
	})

	dbstore := NewDataStoreMongoWithSession(session)

	// setup in default DB
	setUp(ctx, dbstore, inDevs)
	// and in tenant DB
	setUp(tenCtx, dbstore, inDevs)

	// delete from default DB
	err := dbstore.DeleteDeviceAuthByDevice(ctx, "0001")
	assert.NoError(t, err)

	for _, aid := range []model.AuthID{"0001", "0002"} {
		_, err := dbstore.GetDeviceAuth(ctx, aid)
		assert.EqualError(t, err, store.ErrNotFound.Error())

		// devices should exist in tenant's DB however
		_, err = dbstore.GetDeviceAuth(tenCtx, aid)
		assert.NoError(t, err)
	}

	aset, err := dbstore.GetDeviceAuth(ctx, "0003")
	assert.NoError(t, err)
	assert.NotNil(t, aset)

	err = dbstore.DeleteDeviceAuthByDevice(ctx, "0004")
	assert.EqualError(t, err, store.ErrNotFound.Error())
}
