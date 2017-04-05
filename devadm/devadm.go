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
type App interface {
	ListDeviceAuths(skip int, limit int, filter store.Filter) ([]model.DeviceAuth, error)
	SubmitDeviceAuth(d model.DeviceAuth) error
	GetDeviceAuth(id model.AuthID) (*model.DeviceAuth, error)
	AcceptDeviceAuth(id model.AuthID) error
	RejectDeviceAuth(id model.AuthID) error
	DeleteDeviceAuth(id model.AuthID) error

	DeleteDeviceData(id model.DeviceID) error

	WithContext(c context.Context) App
}

func NewDevAdm(d store.DataStore, authclientconf deviceauth.Config) App {
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
	authclientconf deviceauth.Config
	clientGetter   ApiClientGetter
}

func (d *DevAdm) ListDeviceAuths(skip int, limit int, filter store.Filter) ([]model.DeviceAuth, error) {
	devs, err := d.db.GetDeviceAuths(skip, limit, filter)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch devices")
	}

	return devs, nil
}

func (d *DevAdm) SubmitDeviceAuth(dev model.DeviceAuth) error {
	now := time.Now()
	dev.RequestTime = &now

	err := d.db.PutDeviceAuth(&dev)
	if err != nil {
		return errors.Wrap(err, "failed to put device")
	}
	return nil
}

func (d *DevAdm) GetDeviceAuth(id model.AuthID) (*model.DeviceAuth, error) {
	dev, err := d.db.GetDeviceAuth(id)
	if err != nil {
		return nil, err
	}
	return dev, nil
}

func (d *DevAdm) DeleteDeviceAuth(id model.AuthID) error {
	err := d.db.DeleteDeviceAuth(id)
	switch err {
	case nil:
		return nil
	case store.ErrNotFound:
		return err
	default:
		return errors.Wrap(err, "failed to delete device")
	}
}

func (d *DevAdm) propagateDeviceAuthUpdate(dev *model.DeviceAuth) error {
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

func (d *DevAdm) updateDeviceAuthStatus(id model.AuthID, status string) error {
	dev, err := d.db.GetDeviceAuth(id)
	if err != nil {
		return err
	}

	dev.Status = status

	err = d.propagateDeviceAuthUpdate(dev)
	if err != nil {
		return err
	}

	// update only status and attributes fields
	err = d.db.PutDeviceAuth(&model.DeviceAuth{
		ID:       dev.ID,
		DeviceId: dev.DeviceId,
		Status:   dev.Status,
	})
	if err != nil {
		return err
	}

	return nil
}

func (d *DevAdm) AcceptDeviceAuth(id model.AuthID) error {
	return d.updateDeviceAuthStatus(id, model.DevStatusAccepted)
}

func (d *DevAdm) RejectDeviceAuth(id model.AuthID) error {
	return d.updateDeviceAuthStatus(id, model.DevStatusRejected)
}

func (d *DevAdm) DeleteDeviceData(devid model.DeviceID) error {
	return d.db.DeleteDeviceAuthByDevice(devid)
}

func (d *DevAdm) WithContext(ctx context.Context) App {
	dwc := &DevAdmWithContext{
		DevAdm: *d,
		ctx:    ctx,
	}
	dwc.log = log.FromContext(ctx)
	dwc.clientGetter = dwc.contextClientGetter
	return dwc
}
