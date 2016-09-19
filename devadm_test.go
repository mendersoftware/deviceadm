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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func devadmForTest(d DataStore) DevAdmApp {
	return &DevAdm{db: d}
}

func TestDevAdmListDevicesEmpty(t *testing.T) {
	db := &MockDataStore{}
	db.On("GetDevices", 0, 1, "").
		Return([]Device{}, nil)

	d := devadmForTest(db)

	l, _ := d.ListDevices(0, 1, "")
	assert.Len(t, l, 0)
}

func TestDevAdmListDevices(t *testing.T) {
	db := &MockDataStore{}
	db.On("GetDevices", 0, 1, "").
		Return([]Device{{}, {}, {}}, nil)

	d := devadmForTest(db)

	l, _ := d.ListDevices(0, 1, "")
	assert.Len(t, l, 3)
}

func TestDevAdmListDevicesErr(t *testing.T) {
	db := &MockDataStore{}
	db.On("GetDevices", 0, 1, "").
		Return([]Device{}, errors.New("error"))

	d := devadmForTest(db)

	_, err := d.ListDevices(0, 1, "")
	assert.NotNil(t, err)
}

func TestDevAdmAddDevice(t *testing.T) {
	db := &MockDataStore{}
	db.On("PutDevice", mock.AnythingOfType("*main.Device")).
		Return(nil)

	d := devadmForTest(db)

	err := d.AddDevice(Device{})

	assert.NoError(t, err)
}

func TestDevAdmAddDeviceErr(t *testing.T) {
	db := &MockDataStore{}
	db.On("PutDevice", mock.AnythingOfType("*main.Device")).
		Return(errors.New("db connection failed"))

	d := devadmForTest(db)

	err := d.AddDevice(Device{})

	if assert.Error(t, err) {
		assert.EqualError(t, err, "failed to add device: db connection failed")
	}
}

func makeGetDevice(id DeviceID) func(id DeviceID) (*Device, error) {
	return func(did DeviceID) (*Device, error) {
		if did == "" {
			return nil, errors.New("unsupported device ID")
		}

		if did != id {
			return nil, ErrDevNotFound
		}
		return &Device{ID: id}, nil
	}
}

func TestDevAdmGetDevice(t *testing.T) {
	db := &MockDataStore{}
	db.On("GetDevice", DeviceID("foo")).
		Return(&Device{ID: "foo"}, nil)
	db.On("GetDevice", DeviceID("bar")).
		Return(nil, ErrDevNotFound)
	db.On("GetDevice", DeviceID("baz")).
		Return(nil, errors.New("error"))

	d := devadmForTest(db)

	dev, err := d.GetDevice("foo")
	assert.NotNil(t, dev)
	assert.NoError(t, err)

	dev, err = d.GetDevice("bar")
	assert.Nil(t, dev)
	assert.EqualError(t, err, ErrDevNotFound.Error())

	dev, err = d.GetDevice("baz")
	assert.Nil(t, dev)
	assert.Error(t, err)
}

func TestDevAdmAcceptDevice(t *testing.T) {
	db := &MockDataStore{}
	db.On("GetDevice", DeviceID("foo")).
		Return(&Device{ID: "foo"}, nil)
	db.On("GetDevice", DeviceID("bar")).
		Return(nil, ErrDevNotFound)

	d := devadmForTest(db)

	err := d.AcceptDevice("foo")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to propagate")
	// check error type?

	err = d.AcceptDevice("bar")
	assert.Error(t, err)
	assert.EqualError(t, err, ErrDevNotFound.Error())
}

func TestDevAdmRejectDevice(t *testing.T) {
	db := &MockDataStore{}
	db.On("GetDevice", DeviceID("foo")).
		Return(&Device{ID: "foo"}, nil)
	db.On("GetDevice", DeviceID("bar")).
		Return(nil, ErrDevNotFound)

	d := devadmForTest(db)

	err := d.RejectDevice("foo")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to propagate")

	err = d.RejectDevice("bar")
	assert.Error(t, err)
	assert.EqualError(t, err, ErrDevNotFound.Error())
}

func TestNewDevAdm(t *testing.T) {
	d := NewDevAdm(&MockDataStore{}, DevAuthClientConfig{})

	assert.NotNil(t, d)
}
