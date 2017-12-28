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

package store

import (
	"context"
	"errors"

	"github.com/mendersoftware/deviceadm/model"
)

var (
	// object not found
	ErrNotFound = errors.New("not found")
)

type DataStore interface {
	GetDeviceAuths(ctx context.Context, skip, limit int, filter Filter) ([]model.DeviceAuth, error)

	// find a device auth set with given `id`, returns the device auth set
	// or nil, if auth set was not found, error is set to ErrDevNotFound
	GetDeviceAuth(ctx context.Context, id model.AuthID) (*model.DeviceAuth, error)

	// update or insert device auth set into data store, only non-empty
	// fields will be stored/updated, for instance, to update a status of
	// device auth set with ID "foo":
	//
	// ds.PutDeviceAuth(context.TODO(), &DeviceAuth{
	// 	ID: "foo",
	//      DeviceId: "bar",
	// 	Status: "accepted",
	// })
	PutDeviceAuth(ctx context.Context, dev *model.DeviceAuth) error

	// remove device auth set
	DeleteDeviceAuth(ctx context.Context, id model.AuthID) error

	// remove auth sets owned by device
	DeleteDeviceAuthByDevice(ctx context.Context, id model.DeviceID) error

	// UpdateDeviceAuth updates the auth set (strict update, no upserts).
	UpdateDeviceAuth(ctx context.Context, dev *model.DeviceAuth) error

	MigrateTenant(ctx context.Context, version string, tenant string) error
	WithAutomigrate() DataStore

	InsertDeviceAuth(ctx context.Context, dev *model.DeviceAuth) error

	GetDeviceAuthsByIdentityData(ctx context.Context, idata string) ([]model.DeviceAuth, error)
}
