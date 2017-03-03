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
	"github.com/pkg/errors"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/mendersoftware/go-lib-micro/mongo/migrate"
)

const (
	DbVersion     = "0.1.0"
	DbName        = "deviceadm"
	DbDevicesColl = "devices"
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

func (db *DataStoreMongo) GetDevices(skip, limit int, status string) ([]Device, error) {
	s := db.session.Copy()
	defer s.Close()
	c := s.DB(DbName).C(DbDevicesColl)
	res := []Device{}

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

func (db *DataStoreMongo) GetDevice(id DeviceID) (*Device, error) {
	s := db.session.Copy()
	defer s.Close()
	c := s.DB(DbName).C(DbDevicesColl)

	filter := bson.M{"id": id}
	res := Device{}

	err := c.Find(filter).One(&res)

	if err != nil {
		if err == mgo.ErrNotFound {
			return nil, ErrDevNotFound
		} else {
			return nil, errors.Wrap(err, "failed to fetch device")
		}
	}

	return &res, nil
}

func (db *DataStoreMongo) DeleteDevice(id DeviceID) error {
	s := db.session.Copy()
	defer s.Close()

	filter := bson.M{"id": id}
	err := s.DB(DbName).C(DbDevicesColl).Remove(filter)

	switch err {
	case nil:
		return nil
	case mgo.ErrNotFound:
		return ErrDevNotFound
	default:
		return errors.Wrap(err, "failed to delete device")
	}
}

// produce a Device wrapper suitable for passing in an Upsert() as
// '$set' fields
func genDeviceUpdate(dev *Device) *Device {
	updev := Device{}

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
func (db *DataStoreMongo) PutDevice(dev *Device) error {
	s := db.session.Copy()
	defer s.Close()
	c := s.DB(DbName).C(DbDevicesColl)

	filter := bson.M{"id": dev.ID}

	// use $set operator so that fields values are replaced
	data := bson.M{"$set": genDeviceUpdate(dev)}

	// does insert or update
	_, err := c.Upsert(filter, data)
	if err != nil {
		return errors.Wrap(err, "failed to store device")
	}
	return nil
}

func (db *DataStoreMongo) Migrate(version string, migrations []migrate.Migration) error {
	m := migrate.DummyMigrator{
		Session: db.session,
		Db:      DbName,
	}

	ver, err := migrate.NewVersion(version)
	if err != nil {
		return errors.Wrap(err, "failed to parse service version")
	}

	err = m.Apply(ver, migrations)
	if err != nil {
		return errors.Wrap(err, "failed to apply migrations")
	}

	return nil
}
