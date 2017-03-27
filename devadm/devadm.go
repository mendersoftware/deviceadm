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
package devadm

import (
	"context"
	"net/http"
	"time"

	"github.com/mendersoftware/deviceadm/client/deviceauth"
	"github.com/mendersoftware/deviceadm/model"
	"github.com/mendersoftware/deviceadm/store"

	"github.com/mendersoftware/go-lib-micro/log"
	"github.com/mendersoftware/go-lib-micro/requestid"
	"github.com/pkg/errors"
)

// helper for obtaining API clients
type ApiClientGetter func() requestid.ApiRequester

func simpleApiClientGetter() requestid.ApiRequester {
	return &http.Client{}
}

// this device admission service interface
type DevAdmApp interface {
	ListDevices(skip int, limit int, status string) ([]model.DeviceAuth, error)
	SubmitDevice(d model.DeviceAuth) error
	GetDevice(id model.AuthID) (*model.DeviceAuth, error)
	AcceptDevice(id model.AuthID) error
	RejectDevice(id model.AuthID) error
	DeleteDevice(id model.AuthID) error

	WithContext(c context.Context) DevAdmApp
}

func NewDevAdm(d store.DataStore, authclientconf deviceauth.ClientConfig) DevAdmApp {
	return &DevAdm{
		log:            log.New(log.Ctx{}),
		db:             d,
		authclientconf: authclientconf,
		clientGetter:   simpleApiClientGetter,
	}
}

type DevAdm struct {
	log            *log.Logger
	db             store.DataStore
	authclientconf deviceauth.ClientConfig
	clientGetter   ApiClientGetter
}

func (d *DevAdm) ListDevices(skip int, limit int, status string) ([]model.DeviceAuth, error) {
	devs, err := d.db.GetDevices(skip, limit, status)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch devices")
	}

	return devs, nil
}

func (d *DevAdm) SubmitDevice(dev model.DeviceAuth) error {
	now := time.Now()
	dev.RequestTime = &now

	err := d.db.PutDevice(&dev)
	if err != nil {
		return errors.Wrap(err, "failed to put device")
	}
	return nil
}

func (d *DevAdm) GetDevice(id model.AuthID) (*model.DeviceAuth, error) {
	dev, err := d.db.GetDevice(id)
	if err != nil {
		return nil, err
	}
	return dev, nil
}

func (d *DevAdm) DeleteDevice(id model.AuthID) error {
	err := d.db.DeleteDevice(id)
	switch err {
	case nil:
		return nil
	case store.ErrDevNotFound:
		return err
	default:
		return errors.Wrap(err, "failed to delete device")
	}
}

func (d *DevAdm) propagateDeviceUpdate(dev *model.DeviceAuth) error {
	// forward device state to auth service
	cl := deviceauth.NewClientWithLogger(d.authclientconf, d.clientGetter(), d.log)
	err := cl.UpdateDevice(deviceauth.StatusReq{
		AuthId:   dev.ID.String(),
		DeviceId: dev.DeviceId.String(),
		Status:   dev.Status,
	})
	if err != nil {
		// no good if we cannot propagate device update
		// further
		return errors.Wrap(err, "failed to propagate device status update")
	}
	return nil
}

func (d *DevAdm) updateDeviceStatus(id model.AuthID, status string) error {
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
	err = d.db.PutDevice(&model.DeviceAuth{
		ID:       dev.ID,
		DeviceId: dev.DeviceId,
		Status:   dev.Status,
	})
	if err != nil {
		return err
	}

	return nil
}

func (d *DevAdm) AcceptDevice(id model.AuthID) error {
	return d.updateDeviceStatus(id, model.DevStatusAccepted)
}

func (d *DevAdm) RejectDevice(id model.AuthID) error {
	return d.updateDeviceStatus(id, model.DevStatusRejected)
}

func (d *DevAdm) WithContext(ctx context.Context) DevAdmApp {
	dwc := &DevAdmWithContext{
		DevAdm: *d,
		ctx:    ctx,
	}
	dwc.log = log.FromContext(ctx)
	dwc.clientGetter = dwc.contextClientGetter
	return dwc
}
