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

package store

import (
	"errors"

	"github.com/mendersoftware/deviceadm/model"
)

var (
	// device not found
	ErrDevNotFound = errors.New("not found")
)

type DataStore interface {
	GetDevices(skip, limit int, status string) ([]model.Device, error)

	// find a device with given `id`, returns the device or nil,
	// if device was not found, error is set to ErrDevNotFound
	GetDevice(id model.AuthID) (*model.Device, error)

	// update or insert device into data store, only non-empty
	// fields will be stored/updated, for instance, to update a
	// status of device "foo":
	//
	// ds.PutDevice(&Device{
	// 	ID: "foo",
	//      DeviceId: "bar",
	// 	Status: "accepted",
	// })
	PutDevice(dev *model.Device) error

	// remove device
	DeleteDevice(id model.AuthID) error
}
