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
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/mendersoftware/deviceadm/client"
	"github.com/mendersoftware/deviceadm/client/deviceauth"
	"github.com/mendersoftware/deviceadm/model"
	"github.com/mendersoftware/deviceadm/store"
	mstore "github.com/mendersoftware/deviceadm/store/mocks"
)

type FakeApiRequester struct {
	status int
}

func (f FakeApiRequester) Do(r *http.Request) (*http.Response, error) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(f.status)
	}))
	defer ts.Close()

	res, err := http.Get(ts.URL)
	return res, err
}

func devadmWithClientForTest(d store.DataStore, clientRespStatus int) App {
	clientGetter := func() client.HttpRunner {
		return FakeApiRequester{clientRespStatus}
	}
	return &DevAdm{
		db:           d,
		clientGetter: clientGetter,
	}
}

func devadmForTest(d store.DataStore) App {
	return &DevAdm{
		db:           d,
		clientGetter: simpleApiClientGetter,
	}
}

func TestDevAdmListDevicesEmpty(t *testing.T) {
	ctx := context.Background()

	db := &mstore.DataStore{}
	db.On("GetDeviceAuths", ctx, 0, 1, store.Filter{}).
		Return([]model.DeviceAuth{}, nil)

	d := devadmForTest(db)

	l, _ := d.ListDeviceAuths(ctx, 0, 1, store.Filter{})
	assert.Len(t, l, 0)
}

func TestDevAdmListDevices(t *testing.T) {
	ctx := context.Background()

	db := &mstore.DataStore{}
	db.On("GetDeviceAuths", ctx, 0, 1, store.Filter{}).
		Return([]model.DeviceAuth{{}, {}, {}}, nil)

	d := devadmForTest(db)

	l, _ := d.ListDeviceAuths(ctx, 0, 1, store.Filter{})
	assert.Len(t, l, 3)
}

func TestDevAdmListDevicesErr(t *testing.T) {
	ctx := context.Background()

	db := &mstore.DataStore{}
	db.On("GetDeviceAuths", ctx, 0, 1, store.Filter{}).
		Return([]model.DeviceAuth{}, errors.New("error"))

	d := devadmForTest(db)

	_, err := d.ListDeviceAuths(ctx, 0, 1, store.Filter{})
	assert.NotNil(t, err)
}

func TestDevAdmSubmitDevice(t *testing.T) {
	ctx := context.Background()

	db := &mstore.DataStore{}
	db.On("PutDeviceAuth", ctx,
		mock.AnythingOfType("*model.DeviceAuth")).
		Return(nil)

	d := devadmWithClientForTest(db, http.StatusNoContent)

	err := d.SubmitDeviceAuth(ctx, model.DeviceAuth{})

	assert.NoError(t, err)
}

func TestDevAdmSubmitDeviceErr(t *testing.T) {
	ctx := context.Background()

	db := &mstore.DataStore{}
	db.On("PutDeviceAuth", ctx,
		mock.AnythingOfType("*model.DeviceAuth")).
		Return(errors.New("db connection failed"))

	d := devadmWithClientForTest(db, http.StatusNoContent)

	err := d.SubmitDeviceAuth(ctx, model.DeviceAuth{})

	if assert.Error(t, err) {
		assert.EqualError(t, err, "failed to put device: db connection failed")
	}
}

func makeGetDevice(id model.AuthID) func(id model.AuthID) (*model.DeviceAuth, error) {
	return func(aid model.AuthID) (*model.DeviceAuth, error) {
		if aid == "" {
			return nil, errors.New("unsupported device auth ID")
		}

		if aid != id {
			return nil, store.ErrNotFound
		}
		return &model.DeviceAuth{
			ID:       id,
			DeviceId: model.DeviceID(id),
		}, nil
	}
}

func TestDevAdmGetDevice(t *testing.T) {
	ctx := context.Background()

	db := &mstore.DataStore{}
	db.On("GetDeviceAuth", ctx, model.AuthID("foo")).
		Return(&model.DeviceAuth{ID: "foo", DeviceId: "foo"}, nil)
	db.On("GetDeviceAuth", ctx, model.AuthID("bar")).
		Return(nil, store.ErrNotFound)
	db.On("GetDeviceAuth", ctx, model.AuthID("baz")).
		Return(nil, errors.New("error"))

	d := devadmForTest(db)

	dev, err := d.GetDeviceAuth(ctx, "foo")
	assert.NotNil(t, dev)
	assert.NoError(t, err)

	dev, err = d.GetDeviceAuth(ctx, "bar")
	assert.Nil(t, dev)
	assert.EqualError(t, err, store.ErrNotFound.Error())

	dev, err = d.GetDeviceAuth(ctx, "baz")
	assert.Nil(t, dev)
	assert.Error(t, err)
}

func TestDevAdmAcceptDevice(t *testing.T) {
	ctx := context.Background()

	db := &mstore.DataStore{}
	db.On("GetDeviceAuth", ctx, model.AuthID("foo")).
		Return(&model.DeviceAuth{ID: "foo"}, nil)
	db.On("GetDeviceAuth", ctx, model.AuthID("bar")).
		Return(nil, store.ErrNotFound)
	db.On("PutDeviceAuth", ctx,
		mock.AnythingOfType("*model.DeviceAuth")).
		Return(nil)

	d := devadmWithClientForTest(db, http.StatusNoContent)

	err := d.AcceptDeviceAuth(ctx, "foo")

	assert.NoError(t, err)

	err = d.AcceptDeviceAuth(ctx, "bar")
	assert.Error(t, err)
	assert.EqualError(t, err, store.ErrNotFound.Error())
}

func TestDevAdmDeleteDevice(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		datastoreError error
		outError       error
	}{
		"ok": {
			datastoreError: nil,
			outError:       nil,
		},
		"no device": {
			datastoreError: store.ErrNotFound,
			outError:       store.ErrNotFound,
		},
		"datastore error": {
			datastoreError: errors.New("db connection failed"),
			outError:       errors.New("failed to delete device: db connection failed"),
		},
	}

	for name, tc := range testCases {
		t.Run(fmt.Sprintf("test case: %s", name), func(t *testing.T) {
			ctx := context.Background()

			db := &mstore.DataStore{}
			db.On("DeleteDeviceAuth", ctx,
				mock.AnythingOfType("model.AuthID"),
			).Return(tc.datastoreError)
			i := devadmForTest(db)

			err := i.DeleteDeviceAuth(ctx, "foo")

			if tc.outError != nil {
				if assert.Error(t, err) {
					assert.EqualError(t, err, tc.outError.Error())
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDevAdmRejectDevice(t *testing.T) {
	ctx := context.Background()

	db := &mstore.DataStore{}
	db.On("GetDeviceAuth", ctx, model.AuthID("foo")).
		Return(&model.DeviceAuth{ID: "foo"}, nil)
	db.On("GetDeviceAuth", ctx, model.AuthID("bar")).
		Return(nil, store.ErrNotFound)
	db.On("PutDeviceAuth", ctx,
		mock.AnythingOfType("*model.DeviceAuth")).
		Return(nil)

	d := devadmWithClientForTest(db, http.StatusNoContent)

	err := d.RejectDeviceAuth(ctx, "foo")

	assert.NoError(t, err)

	err = d.RejectDeviceAuth(ctx, "bar")
	assert.Error(t, err)
	assert.EqualError(t, err, store.ErrNotFound.Error())
}

func TestDevAdmProvisionTenant(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		datastoreError error
		outError       error
	}{
		"ok": {
			datastoreError: nil,
			outError:       nil,
		},
		"generic error": {
			datastoreError: errors.New("failed to provision tenant"),
			outError:       errors.New("failed to provision tenant"),
		},
	}

	for name, tc := range testCases {
		t.Run(fmt.Sprintf("test case: %s", name), func(t *testing.T) {
			ctx := context.Background()

			db := &mstore.DataStore{}
			db.On("MigrateTenant", ctx,
				"1.1.0",
				mock.AnythingOfType("string"),
			).Return(tc.datastoreError)
			db.On("WithAutomigrate").Return(db)
			i := devadmForTest(db)

			err := i.ProvisionTenant(ctx, "foo")

			if tc.outError != nil {
				if assert.Error(t, err) {
					assert.EqualError(t, err, tc.outError.Error())
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDevAdmAcceptDevicePreAuth(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		id model.AuthID

		storeAuth      *model.DeviceAuth
		storeGetErr    error
		storeUpdateErr error

		err error
	}{
		"ok": {
			id: model.AuthID("1"),

			storeAuth: &model.DeviceAuth{
				ID:             "11",
				DeviceId:       model.DeviceID("1"),
				DeviceIdentity: "foo-1",
				Key:            "key1",
				Status:         model.DevStatusPreauthorized,
			},
		},
		"error: not found": {
			id: model.AuthID("1"),

			storeGetErr: store.ErrNotFound,

			err: ErrAuthNotFound,
		},
		"error: not preauthorzed": {
			id: model.AuthID("1"),

			storeAuth: &model.DeviceAuth{
				ID:             "11",
				DeviceId:       model.DeviceID("1"),
				DeviceIdentity: "foo-1",
				Key:            "key1",
				Status:         model.DevStatusPending,
			},

			err: errors.New("auth set must be in 'preauthorized' state"),
		},
		"error: generic on get": {
			id: model.AuthID("1"),

			storeGetErr: errors.New("db error"),

			err: errors.New("failed to fetch auth set: db error"),
		},
		"error: generic on update": {
			id: model.AuthID("1"),

			storeAuth: &model.DeviceAuth{
				ID:             "11",
				DeviceId:       model.DeviceID("1"),
				DeviceIdentity: "foo-1",
				Key:            "key1",
				Status:         model.DevStatusPreauthorized,
			},

			storeUpdateErr: errors.New("db error"),

			err: errors.New("failed to update auth set: db error"),
		},
	}

	for name, tc := range testCases {
		t.Run(fmt.Sprintf("test case: %s", name), func(t *testing.T) {
			ctx := context.Background()

			db := &mstore.DataStore{}
			db.On("GetDeviceAuth",
				ctx,
				tc.id,
			).Return(tc.storeAuth, tc.storeGetErr)

			db.On("UpdateDeviceAuth",
				ctx,
				mock.AnythingOfType("*model.DeviceAuth"),
			).Return(tc.storeUpdateErr)

			d := devadmForTest(db)

			err := d.AcceptDevicePreAuth(ctx, tc.id)

			if tc.err != nil {
				assert.EqualError(t, err, tc.err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNewDevAdm(t *testing.T) {
	d := NewDevAdm(&mstore.DataStore{}, deviceauth.Config{})

	assert.NotNil(t, d)
}
