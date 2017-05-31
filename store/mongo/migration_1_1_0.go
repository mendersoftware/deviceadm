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

	"github.com/mendersoftware/go-lib-micro/mongo/migrate"
	ctx_store "github.com/mendersoftware/go-lib-micro/store"
	"github.com/pkg/errors"
	"gopkg.in/mgo.v2"

	"github.com/mendersoftware/deviceadm/model"
)

const (
	dbDeviceIdIndex     = "id"
	dbDeviceIdIndexName = "uniqueDeviceIdIndex"
)

type migration_1_1_0 struct {
	ms  *DataStoreMongo
	ctx context.Context
}

// Up applies a migration to version 1.1.0.
//
// In pre 1.1.x DB, the `id` field meant 'device ID'. In 1.1.0 device ID was
// moved to `device_id` field, while `id` is now an authentication set ID.
func (m *migration_1_1_0) Up(from migrate.Version) error {
	s := m.ms.session.Copy()

	defer s.Close()

	iter := s.DB(ctx_store.DbFromContext(m.ctx, DbName)).
		C(DbDevicesColl).Find(nil).Iter()

	var olddev model.DeviceAuth

	for iter.Next(&olddev) {
		newdev := olddev
		newdev.DeviceId = model.DeviceID(newdev.ID)

		_, err := s.DB(ctx_store.DbFromContext(m.ctx, DbName)).
			C(DbDevicesColl).Upsert(&olddev, &newdev)
		if err != nil {
			return errors.Wrapf(err, "failed to insert auth set for device %v",
				olddev.ID)
		}
	}

	if err := iter.Close(); err != nil {
		return errors.Wrap(err, "failed to close DB iterator")
	}

	if err := m.ensureIndexes(s); err != nil {
		return errors.Wrap(err, "database indexing failed")
	}

	return nil
}

func (m *migration_1_1_0) ensureIndexes(s *mgo.Session) error {
	uniqueDevIdIdx := mgo.Index{
		Key:        []string{dbDeviceIdIndex},
		Unique:     true,
		Name:       dbDeviceIdIndexName,
		Background: false,
	}

	return s.DB(ctx_store.DbFromContext(m.ctx, DbName)).
		C(DbDevicesColl).EnsureIndex(uniqueDevIdIdx)

}

func (m *migration_1_1_0) Version() migrate.Version {
	return migrate.MakeVersion(1, 1, 0)
}
