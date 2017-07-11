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
package devadm

import (
	"context"
	"time"

	"github.com/pkg/errors"

	"github.com/mendersoftware/deviceadm/client"
	"github.com/mendersoftware/deviceadm/client/deviceauth"
	"github.com/mendersoftware/deviceadm/model"
	"github.com/mendersoftware/deviceadm/store"
)

// helper for obtaining API clients
type ApiClientGetter func() client.HttpRunner

func simpleApiClientGetter() client.HttpRunner {
	return &client.HttpApi{}
}

// this device admission service interface
type App interface {
	ListDeviceAuths(ctx context.Context, skip int, limit int, filter store.Filter) ([]model.DeviceAuth, error)
	SubmitDeviceAuth(ctx context.Context, d model.DeviceAuth) error
	GetDeviceAuth(ctx context.Context, id model.AuthID) (*model.DeviceAuth, error)
	AcceptDeviceAuth(ctx context.Context, id model.AuthID) error
	RejectDeviceAuth(ctx context.Context, id model.AuthID) error
	DeleteDeviceAuth(ctx context.Context, id model.AuthID) error

	DeleteDeviceData(ctx context.Context, id model.DeviceID) error
}

func NewDevAdm(d store.DataStore, authclientconf deviceauth.Config) App {
	return &DevAdm{
		db:             d,
		authclientconf: authclientconf,
		clientGetter:   simpleApiClientGetter,
	}
}

type DevAdm struct {
	db             store.DataStore
	authclientconf deviceauth.Config
	clientGetter   ApiClientGetter
}

func (d *DevAdm) ListDeviceAuths(ctx context.Context, skip int, limit int, filter store.Filter) ([]model.DeviceAuth, error) {
	devs, err := d.db.GetDeviceAuths(ctx, skip, limit, filter)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch devices")
	}

	return devs, nil
}

func (d *DevAdm) SubmitDeviceAuth(ctx context.Context, dev model.DeviceAuth) error {
	now := time.Now()
	dev.RequestTime = &now

	err := d.db.PutDeviceAuth(ctx, &dev)
	if err != nil {
		return errors.Wrap(err, "failed to put device")
	}
	return nil
}

func (d *DevAdm) GetDeviceAuth(ctx context.Context, id model.AuthID) (*model.DeviceAuth, error) {
	dev, err := d.db.GetDeviceAuth(ctx, id)
	if err != nil {
		return nil, err
	}
	return dev, nil
}

func (d *DevAdm) DeleteDeviceAuth(ctx context.Context, id model.AuthID) error {
	err := d.db.DeleteDeviceAuth(ctx, id)
	switch err {
	case nil:
		return nil
	case store.ErrNotFound:
		return err
	default:
		return errors.Wrap(err, "failed to delete device")
	}
}

func (d *DevAdm) propagateDeviceAuthUpdate(ctx context.Context, dev *model.DeviceAuth) error {
	// forward device state to auth service
	cl := deviceauth.NewClient(d.authclientconf, d.clientGetter())
	err := cl.UpdateDevice(ctx, deviceauth.StatusReq{
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

func (d *DevAdm) updateDeviceAuthStatus(ctx context.Context, id model.AuthID, status string) error {
	dev, err := d.db.GetDeviceAuth(ctx, id)
	if err != nil {
		return err
	}

	dev.Status = status

	err = d.propagateDeviceAuthUpdate(ctx, dev)
	if err != nil {
		return err
	}

	// update only status and attributes fields
	err = d.db.PutDeviceAuth(ctx, &model.DeviceAuth{
		ID:       dev.ID,
		DeviceId: dev.DeviceId,
		Status:   dev.Status,
	})
	if err != nil {
		return err
	}

	return nil
}

func (d *DevAdm) AcceptDeviceAuth(ctx context.Context, id model.AuthID) error {
	return d.updateDeviceAuthStatus(ctx, id, model.DevStatusAccepted)
}

func (d *DevAdm) RejectDeviceAuth(ctx context.Context, id model.AuthID) error {
	return d.updateDeviceAuthStatus(ctx, id, model.DevStatusRejected)
}

func (d *DevAdm) DeleteDeviceData(ctx context.Context, devid model.DeviceID) error {
	return d.db.DeleteDeviceAuthByDevice(ctx, devid)
}
