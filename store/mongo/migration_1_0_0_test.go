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
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/mendersoftware/go-lib-micro/mongo/migrate"
	"github.com/stretchr/testify/assert"
	"gopkg.in/mgo.v2"

	"github.com/mendersoftware/deviceadm/model"
)

func randTime(base time.Time) time.Time {
	diff := time.Duration(rand.Int()%1024) * time.Hour
	return base.Add(-diff)
}

func randDevStatus() string {
	statuses := []string{
		model.DevStatusAccepted,
		model.DevStatusPending,
		model.DevStatusRejected,
	}
	idx := rand.Int() % len(statuses)
	return statuses[idx]
}

type migration_1_0_0_TestData struct {
	// devices by device index
	devices map[int]*model.DeviceAuth
}

func (m *migration_1_0_0_TestData) GetDev(idx int) *model.DeviceAuth {
	return m.devices[idx]
}

// populateDevices creates `count` devices, returns test data it generated
func populateDevices(t *testing.T, s *mgo.Session, count int) migration_1_0_0_TestData {

	td := migration_1_0_0_TestData{
		devices: map[int]*model.DeviceAuth{},
	}

	now := time.Now()
	for i := 0; i < count; i++ {
		devid := fmt.Sprintf("devid-0.1.0-%d", i)

		tm := randTime(now)
		// devices in pre 1.1.0 version had DeviceId unset
		dev := model.DeviceAuth{
			ID:             model.AuthID(devid),
			Key:            fmt.Sprintf("pubkey-0.1.0-%d", i),
			DeviceIdentity: fmt.Sprintf("id-data-0.1.0-%d", i),
			Status:         randDevStatus(),
			RequestTime:    &tm,
			Attributes: model.DeviceAuthAttributes{
				"foo": fmt.Sprintf("attr-0.1.0-%d", i),
			},
		}
		err := s.DB(DbName).C(DbDevicesColl).Insert(dev)
		assert.NoError(t, err)

		td.devices[i] = &dev

	}
	return td
}

func TestMigration_1_0_0(t *testing.T) {
	db := getDb()

	s := db.session
	devCount := 100

	data := populateDevices(t, s, devCount)

	ctx := context.Background()

	mig := migration_1_1_0{ms: db, ctx: ctx}

	err := mig.Up(migrate.MakeVersion(0, 1, 0))
	assert.NoError(t, err)

	// there should be devCount devices
	cnt, err := s.DB(DbName).C(DbDevicesColl).Count()
	assert.NoError(t, err)
	assert.Equal(t, devCount, cnt)

	verifyIndexes(t, s)

	// trying to add a device auth set with same ID should raise conflict
	err = s.DB(DbName).C(DbDevicesColl).Insert(&model.DeviceAuth{
		ID: data.GetDev(10).ID,
	})
	assert.True(t, mgo.IsDup(err))

	// verify that DeviceId was set for every device
	for _, dev := range data.devices {
		dbdev, err := db.GetDeviceAuth(ctx, dev.ID)
		assert.NoError(t, err)

		// DeviceId should have been set to the value of ID
		assert.Equal(t, dbdev.ID.String(), dbdev.DeviceId.String())
	}

	db.session.Close()
}

func verifyIndexes(t *testing.T, session *mgo.Session) {
	// verify index exists
	indexes, err := session.DB(DbName).C(DbDevicesColl).Indexes()
	assert.NoError(t, err, "getting indexes failed")

	assert.Len(t, indexes, 2)
	assert.Equal(t,
		indexes[1],
		mgo.Index{
			Key:        []string{dbDeviceIdIndex},
			Unique:     true,
			Name:       dbDeviceIdIndexName,
			Background: false,
		})
}
