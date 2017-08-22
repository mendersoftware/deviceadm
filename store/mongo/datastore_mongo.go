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
	"crypto/tls"
	"net"
	"time"

	"github.com/mendersoftware/go-lib-micro/mongo/migrate"
	ctx_store "github.com/mendersoftware/go-lib-micro/store"
	"github.com/pkg/errors"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/mendersoftware/deviceadm/model"
	"github.com/mendersoftware/deviceadm/store"
)

const (
	DbVersion     = "1.1.0"
	DbName        = "deviceadm"
	DbDevicesColl = "devices"
)

type DataStoreMongo struct {
	session *mgo.Session
}

func NewDataStoreMongoWithSession(s *mgo.Session) *DataStoreMongo {
	return &DataStoreMongo{session: s}
}

type DataStoreMongoConfig struct {
	// MGO connection string
	ConnectionString string

	// SSL support
	SSL           bool
	SSLSkipVerify bool

	// Overwrites credentials provided in connection string if provided
	Username string
	Password string
}

func NewDataStoreMongo(config DataStoreMongoConfig) (*DataStoreMongo, error) {
	dialInfo, err := mgo.ParseURL(config.ConnectionString)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open mgo session")
	}

	// Set 10s timeout - same as set by Dial
	dialInfo.Timeout = 10 * time.Second

	if config.Username != "" {
		dialInfo.Username = config.Username
	}
	if config.Password != "" {
		dialInfo.Password = config.Password
	}

	if config.SSL {
		dialInfo.DialServer = func(addr *mgo.ServerAddr) (net.Conn, error) {

			// Setup TLS
			tlsConfig := &tls.Config{}
			tlsConfig.InsecureSkipVerify = config.SSLSkipVerify

			conn, err := tls.Dial("tcp", addr.String(), tlsConfig)
			return conn, err
		}
	}

	masterSession, err := mgo.DialWithInfo(dialInfo)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open mgo session")
	}

	// Validate connection
	if err := masterSession.Ping(); err != nil {
		return nil, errors.Wrap(err, "failed to open mgo session")
	}

	// force write ack with immediate journal file fsync
	masterSession.SetSafe(&mgo.Safe{
		W: 1,
		J: true,
	})

	return NewDataStoreMongoWithSession(masterSession), nil
}

func (db *DataStoreMongo) GetDeviceAuths(ctx context.Context, skip, limit int, filter store.Filter) ([]model.DeviceAuth, error) {
	s := db.session.Copy()
	defer s.Close()
	c := s.DB(ctx_store.DbFromContext(ctx, DbName)).C(DbDevicesColl)
	res := []model.DeviceAuth{}

	var dbFilter = &model.DeviceAuth{}
	if filter.Status != "" {
		dbFilter.Status = filter.Status
	}
	if filter.DeviceID != "" {
		dbFilter.DeviceId = filter.DeviceID
	}

	err := c.Find(dbFilter).Sort("id").Skip(skip).Limit(limit).All(&res)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch device list")
	}

	return res, nil
}

func (db *DataStoreMongo) GetDeviceAuth(ctx context.Context, id model.AuthID) (*model.DeviceAuth, error) {
	s := db.session.Copy()
	defer s.Close()
	c := s.DB(ctx_store.DbFromContext(ctx, DbName)).C(DbDevicesColl)

	filter := bson.M{"id": id}
	res := model.DeviceAuth{}

	err := c.Find(filter).One(&res)

	if err != nil {
		if err == mgo.ErrNotFound {
			return nil, store.ErrNotFound
		} else {
			return nil, errors.Wrap(err, "failed to fetch device")
		}
	}

	return &res, nil
}

func (db *DataStoreMongo) DeleteDeviceAuth(ctx context.Context, id model.AuthID) error {
	s := db.session.Copy()
	defer s.Close()

	filter := bson.M{"id": id}
	err := s.DB(ctx_store.DbFromContext(ctx, DbName)).
		C(DbDevicesColl).Remove(filter)

	switch err {
	case nil:
		return nil
	case mgo.ErrNotFound:
		return store.ErrNotFound
	default:
		return errors.Wrap(err, "failed to delete device")
	}
}

func (db *DataStoreMongo) DeleteDeviceAuthByDevice(ctx context.Context, id model.DeviceID) error {
	s := db.session.Copy()
	defer s.Close()

	filter := model.DeviceAuth{DeviceId: id}
	ci, err := s.DB(ctx_store.DbFromContext(ctx, DbName)).
		C(DbDevicesColl).RemoveAll(filter)

	switch {
	case err != nil:
		return nil
	case ci != nil && ci.Removed == 0:
		return store.ErrNotFound
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
func (db *DataStoreMongo) PutDeviceAuth(ctx context.Context, dev *model.DeviceAuth) error {
	s := db.session.Copy()
	defer s.Close()
	c := s.DB(ctx_store.DbFromContext(ctx, DbName)).C(DbDevicesColl)

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

func (db *DataStoreMongo) Migrate(ctx context.Context, version string) error {
	m := migrate.SimpleMigrator{
		Session: db.session,
		Db:      ctx_store.DbFromContext(ctx, DbName),
	}

	ver, err := migrate.NewVersion(version)
	if err != nil {
		return errors.Wrap(err, "failed to parse service version")
	}

	migrations := []migrate.Migration{
		&migration_1_1_0{
			ms:  db,
			ctx: ctx,
		},
	}
	err = m.Apply(ctx, *ver, migrations)
	if err != nil {
		return errors.Wrap(err, "failed to apply migrations")
	}

	return nil
}
