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
	"github.com/stretchr/testify/assert"
	"testing"
)

//mock db with interface methods as fields
//allows monkey patching the methods without
//redefining the struct for each case
type TestDataStore struct {
	MockGetDevices func(skip, limit int, status string) ([]Device, error)
	MockGetDevice  func(id DeviceID) (*Device, error)
}

func (ds *TestDataStore) GetDevices(skip, limit int, status string) ([]Device, error) {
	return ds.MockGetDevices(skip, limit, status)
}

func (ds *TestDataStore) GetDevice(id DeviceID) (*Device, error) {
	return ds.MockGetDevice(id)
}

func (ds *TestDataStore) PutDevice(dev *Device) error {
	return errors.New("not implemented")
}

//mock db methods for multiple cases
func getDevicesErr(skip, limit int, status string) ([]Device, error) {
	return nil, errors.New("test error")
}

func getDevicesEmpty(skip, limit int, status string) ([]Device, error) {
	return []Device{}, nil
}

func getDevices(skip, limit int, status string) ([]Device, error) {
	ret := []Device{
		Device{},
		Device{},
		Device{},
	}
	return ret, nil
}

func devadmForTest(d DataStore) DevAdmApp {
	return &DevAdm{db: d}
}

func TestDevAdmListDevicesEmpty(t *testing.T) {
	db := &TestDataStore{
		MockGetDevices: getDevicesEmpty,
	}

	d := devadmForTest(db)

	l, _ := d.ListDevices(0, 1, "")
	assert.Len(t, l, 0)
}

func TestDevAdmListDevices(t *testing.T) {
	db := &TestDataStore{
		MockGetDevices: getDevices,
	}

	d := devadmForTest(db)

	l, _ := d.ListDevices(0, 1, "")
	assert.Len(t, l, 3)
}

func TestDevAdmListDevicesErr(t *testing.T) {
	db := &TestDataStore{
		MockGetDevices: getDevicesErr,
	}

	d := devadmForTest(db)

	_, err := d.ListDevices(0, 1, "")
	assert.NotNil(t, err)
}

func TestDevAdmAddDevice(t *testing.T) {
	d := devadmForTest(nil)

	err := d.AddDevice(&Device{})

	assert.Error(t, err)
	// check error type?
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
	db := &TestDataStore{
		MockGetDevice: makeGetDevice("foo"),
	}

	d := devadmForTest(db)

	dev, err := d.GetDevice("foo")
	assert.NotNil(t, dev)
	assert.NoError(t, err)

	dev, err = d.GetDevice("bar")
	assert.Nil(t, dev)
	assert.EqualError(t, err, ErrDevNotFound.Error())

	// invoke special case that generates error other than ErrDevNotFound
	dev, err = d.GetDevice("")
	assert.Nil(t, dev)
	assert.Error(t, err)
}

func TestDevAdmUpdateDevice(t *testing.T) {
	d := devadmForTest(nil)

	err := d.UpdateDevice("foo", &Device{})

	assert.Error(t, err)
	// check error type?
}

func TestNewDevAdm(t *testing.T) {
	d := NewDevAdm(&TestDataStore{})

	assert.NotNil(t, d)
}
