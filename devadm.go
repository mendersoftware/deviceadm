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
	"github.com/mendersoftware/deviceadm/log"
	"github.com/mendersoftware/deviceadm/requestid"
	"github.com/pkg/errors"
	"net/http"
	"time"
)

// helper for obtaining API clients
type ApiClientGetter func() requestid.ApiRequester

func simpleApiClientGetter() requestid.ApiRequester {
	return &http.Client{}
}

// this device admission service interface
type DevAdmApp interface {
	ListDevices(skip int, limit int, status string) ([]Device, error)
	AddDevice(d Device) error
	GetDevice(id DeviceID) (*Device, error)
	AcceptDevice(id DeviceID) error
	RejectDevice(id DeviceID) error

	WithContext(c *RequestContext) DevAdmApp
}

func NewDevAdm(d DataStore, authclientconf DevAuthClientConfig) DevAdmApp {
	return &DevAdm{
		log:            log.New(log.Ctx{}),
		db:             d,
		authclientconf: authclientconf,
		clientGetter:   simpleApiClientGetter,
	}
}

type DevAdm struct {
	log            *log.Logger
	db             DataStore
	authclientconf DevAuthClientConfig
	clientGetter   ApiClientGetter
}

func (d *DevAdm) ListDevices(skip int, limit int, status string) ([]Device, error) {
	devs, err := d.db.GetDevices(skip, limit, status)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch devices")
	}

	return devs, nil
}

func (d *DevAdm) AddDevice(dev Device) error {
	now := time.Now()
	dev.RequestTime = &now
	err := d.db.PutDevice(&dev)
	if err != nil {
		return errors.Wrap(err, "failed to add device")
	}
	return nil
}

func (d *DevAdm) GetDevice(id DeviceID) (*Device, error) {
	dev, err := d.db.GetDevice(id)
	if err != nil {
		return nil, err
	}
	return dev, nil
}

func (d *DevAdm) propagateDeviceUpdate(dev *Device) error {
	// forward device state to auth service
	cl := NewDevAuthClient(d.authclientconf, d.clientGetter(), d.log)
	err := cl.UpdateDevice(*dev)
	if err != nil {
		d.log.Errorf("update device failed: %s", err)
		// no good if we cannot propagate device update
		// further
		return errors.New("failed to propagate device status update")
	}
	return nil
}

func (d *DevAdm) updateDeviceStatus(id DeviceID, status string) error {
	dev, err := d.db.GetDevice(id)
	if err != nil {
		return err
	}

	dev.Status = status

	err = d.propagateDeviceUpdate(dev)
	if err != nil {
		return err
	}

	// update only status and attributes fields
	err = d.db.PutDevice(&Device{
		ID:     dev.ID,
		Status: dev.Status,
	})
	if err != nil {
		return err
	}

	return nil
}

func (d *DevAdm) AcceptDevice(id DeviceID) error {
	return d.updateDeviceStatus(id, DevStatusAccepted)
}

func (d *DevAdm) RejectDevice(id DeviceID) error {
	return d.updateDeviceStatus(id, DevStatusRejected)
}

func (d *DevAdm) WithContext(ctx *RequestContext) DevAdmApp {
	dwc := &DevAdmWithContext{
		*d,
		ctx,
	}
	dwc.log = ctx.Logger
	dwc.clientGetter = dwc.contextClientGetter
	return dwc
}
