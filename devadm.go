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
)

// this device admission service interface
type DevAdmApp interface {
	ListDevices(skip int, limit int, status string) ([]Device, error)
	AddDevice(d *Device) error
	GetDevice(id DeviceID) (*Device, error)
	AcceptDevice(id DeviceID, attrs DeviceAttributes) error
	RejectDevice(id DeviceID) error
}

func NewDevAdm(d DataStore) DevAdmApp {
	return &DevAdm{db: d}
}

type DevAdm struct {
	db DataStore
}

func (d *DevAdm) ListDevices(skip int, limit int, status string) ([]Device, error) {
	devs, err := d.db.GetDevices(skip, limit, status)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch devices")
	}

	return devs, nil
}

func (d *DevAdm) AddDevice(dev *Device) error {
	return errors.New("not implemented")
}

func (d *DevAdm) GetDevice(id DeviceID) (*Device, error) {
	dev, err := d.db.GetDevice(id)
	if err != nil {
		return nil, err
	}
	return dev, nil
}

func (d *DevAdm) AcceptDevice(id DeviceID, attrs DeviceAttributes) error {
	return errors.New("not implemented")
}

func (d *DevAdm) RejectDevice(id DeviceID) error {
	return errors.New("not implemented")
}
