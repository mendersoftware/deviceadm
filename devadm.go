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
	ListDevices() []Device
	AddDevice(d Device) error
	GetDevice(id DeviceID) *Device
	UpdateDevice(id DeviceID, d Device) error
}

func NewDevAdm() DevAdmApp {
	return &DevAdm{}
}

type DevAdm struct {
	// nothing
}

func (d *DevAdm) ListDevices() []Device {
	return []Device{}
}

func (d *DevAdm) AddDevice(dev Device) error {
	return errors.New("not implemented")
}

func (d *DevAdm) GetDevice(id DeviceID) *Device {
	return nil
}

func (d *DevAdm) UpdateDevice(id DeviceID, dev Device) error {
	return errors.New("not implemented")
}
