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

	"github.com/mendersoftware/deviceadm/model"
	"github.com/mendersoftware/deviceadm/store"

	"github.com/mendersoftware/go-lib-micro/mongo/migrate"
	"github.com/pkg/errors"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const (
	DbVersion     = "1.1.0"
	DbName        = "deviceadm"
	DbDevicesColl = "devices"

	DbDeviceIdIndex     = "id"
	DbDeviceIdIndexName = "uniqueDeviceIdIndex"
)

type DataStoreMongo struct {
	session *mgo.Session
}

func NewDataStoreMongoWithSession(s *mgo.Session) *DataStoreMongo {
	return &DataStoreMongo{session: s}
}

func NewDataStoreMongo(host string) (*DataStoreMongo, error) {
	s, err := mgo.Dial(host)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open mgo session")
	}
	return NewDataStoreMongoWithSession(s), nil
}

func (db *DataStoreMongo) GetDeviceAuths(skip, limit int, status string) ([]model.DeviceAuth, error) {
	s := db.session.Copy()
	defer s.Close()
	c := s.DB(DbName).C(DbDevicesColl)
	res := []model.DeviceAuth{}

	var filter bson.M
	if status != "" {
		filter = bson.M{"status": status}
	}

	err := c.Find(filter).Skip(skip).Limit(limit).All(&res)

	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch device list")
	}

	return res, nil
}

func (db *DataStoreMongo) GetDeviceAuth(id model.AuthID) (*model.DeviceAuth, error) {
	s := db.session.Copy()
	defer s.Close()
	c := s.DB(DbName).C(DbDevicesColl)

	filter := bson.M{"id": id}
	res := model.DeviceAuth{}

	err := c.Find(filter).One(&res)

	if err != nil {
		if err == mgo.ErrNotFound {
			return nil, store.ErrDevNotFound
		} else {
			return nil, errors.Wrap(err, "failed to fetch device")
		}
	}

	return &res, nil
}

func (db *DataStoreMongo) DeleteDeviceAuth(id model.AuthID) error {
	s := db.session.Copy()
	defer s.Close()

	filter := bson.M{"id": id}
	err := s.DB(DbName).C(DbDevicesColl).Remove(filter)

	switch err {
	case nil:
		return nil
	case mgo.ErrNotFound:
		return store.ErrDevNotFound
	default:
		return errors.Wrap(err, "failed to delete device")
	}
}

// produce a DeviceAuth wrapper suitable for passing in an Upsert() as
// '$set' fields
func genDeviceAuthUpdate(dev *model.DeviceAuth) *model.DeviceAuth {
	updev := model.DeviceAuth{}

	if dev.DeviceId != "" {
		updev.DeviceId = dev.DeviceId
	}

	if dev.Status != "" {
		updev.Status = dev.Status
	}

	if dev.Key != "" {
		updev.Key = dev.Key
	}

	if dev.DeviceIdentity != "" {
		updev.DeviceIdentity = dev.DeviceIdentity
	}

	// TODO: should attributes be merged?
	if len(dev.Attributes) != 0 {
		updev.Attributes = dev.Attributes
	}

	if dev.RequestTime != nil {
		updev.RequestTime = dev.RequestTime
	}

	return &updev
}

//
func (db *DataStoreMongo) PutDeviceAuth(dev *model.DeviceAuth) error {
	s := db.session.Copy()
	defer s.Close()
	c := s.DB(DbName).C(DbDevicesColl)

	filter := bson.M{"id": dev.ID}

	// use $set operator so that fields values are replaced
	data := bson.M{"$set": genDeviceAuthUpdate(dev)}

	// does insert or update
	_, err := c.Upsert(filter, data)
	if err != nil {
		return errors.Wrap(err, "failed to store device")
	}
	return nil
}

func (db *DataStoreMongo) Migrate(version string) error {
	m := migrate.SimpleMigrator{
		Session: db.session,
		Db:      DbName,
	}

	ver, err := migrate.NewVersion(version)
	if err != nil {
		return errors.Wrap(err, "failed to parse service version")
	}

	migrations := []migrate.Migration{
		&migration_1_1_0{ms: db},
	}
	err = m.Apply(context.Background(), *ver, migrations)
	if err != nil {
		return errors.Wrap(err, "failed to apply migrations")
	}

	return nil
}

func (db *DataStoreMongo) EnsureIndexes() error {
	s := db.session.Copy()
	defer s.Close()

	uniqueDevIdIdx := mgo.Index{
		Key:        []string{DbDeviceIdIndex},
		Unique:     true,
		Name:       DbDeviceIdIndexName,
		Background: false,
	}

	return s.DB(DbName).C(DbDevicesColl).EnsureIndex(uniqueDevIdIdx)

}
