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
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mendersoftware/deviceadm/client/deviceauth"
	"github.com/mendersoftware/deviceadm/model"
	"github.com/mendersoftware/deviceadm/store"
	mstore "github.com/mendersoftware/deviceadm/store/mocks"

	"github.com/mendersoftware/go-lib-micro/log"
	"github.com/mendersoftware/go-lib-micro/requestid"
	"github.com/mendersoftware/go-lib-micro/requestlog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
	clientGetter := func() requestid.ApiRequester {
		return FakeApiRequester{clientRespStatus}
	}
	return &DevAdm{
		db:           d,
		clientGetter: clientGetter,
		log:          log.New(log.Ctx{})}
}

func devadmForTest(d store.DataStore) App {
	return &DevAdm{
		db:           d,
		clientGetter: simpleApiClientGetter,
		log:          log.New(log.Ctx{})}
}

func TestDevAdmListDevicesEmpty(t *testing.T) {
	db := &mstore.DataStore{}
	db.On("GetDeviceAuths", 0, 1, "").
		Return([]model.DeviceAuth{}, nil)

	d := devadmForTest(db)

	l, _ := d.ListDeviceAuths(0, 1, "")
	assert.Len(t, l, 0)
}

func TestDevAdmListDevices(t *testing.T) {
	db := &mstore.DataStore{}
	db.On("GetDeviceAuths", 0, 1, "").
		Return([]model.DeviceAuth{{}, {}, {}}, nil)

	d := devadmForTest(db)

	l, _ := d.ListDeviceAuths(0, 1, "")
	assert.Len(t, l, 3)
}

func TestDevAdmListDevicesErr(t *testing.T) {
	db := &mstore.DataStore{}
	db.On("GetDeviceAuths", 0, 1, "").
		Return([]model.DeviceAuth{}, errors.New("error"))

	d := devadmForTest(db)

	_, err := d.ListDeviceAuths(0, 1, "")
	assert.NotNil(t, err)
}

func TestDevAdmSubmitDevice(t *testing.T) {
	db := &mstore.DataStore{}
	db.On("PutDeviceAuth", mock.AnythingOfType("*model.DeviceAuth")).
		Return(nil)

	d := devadmWithClientForTest(db, http.StatusNoContent)

	err := d.SubmitDeviceAuth(model.DeviceAuth{})

	assert.NoError(t, err)
}

func TestDevAdmSubmitDeviceErr(t *testing.T) {
	db := &mstore.DataStore{}
	db.On("PutDeviceAuth", mock.AnythingOfType("*model.DeviceAuth")).
		Return(errors.New("db connection failed"))

	d := devadmWithClientForTest(db, http.StatusNoContent)

	err := d.SubmitDeviceAuth(model.DeviceAuth{})

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
	db := &mstore.DataStore{}
	db.On("GetDeviceAuth", model.AuthID("foo")).
		Return(&model.DeviceAuth{ID: "foo", DeviceId: "foo"}, nil)
	db.On("GetDeviceAuth", model.AuthID("bar")).
		Return(nil, store.ErrNotFound)
	db.On("GetDeviceAuth", model.AuthID("baz")).
		Return(nil, errors.New("error"))

	d := devadmForTest(db)

	dev, err := d.GetDeviceAuth("foo")
	assert.NotNil(t, dev)
	assert.NoError(t, err)

	dev, err = d.GetDeviceAuth("bar")
	assert.Nil(t, dev)
	assert.EqualError(t, err, store.ErrNotFound.Error())

	dev, err = d.GetDeviceAuth("baz")
	assert.Nil(t, dev)
	assert.Error(t, err)
}

func TestDevAdmAcceptDevice(t *testing.T) {
	db := &mstore.DataStore{}
	db.On("GetDeviceAuth", model.AuthID("foo")).
		Return(&model.DeviceAuth{ID: "foo"}, nil)
	db.On("GetDeviceAuth", model.AuthID("bar")).
		Return(nil, store.ErrNotFound)
	db.On("PutDeviceAuth", mock.AnythingOfType("*model.DeviceAuth")).
		Return(nil)

	d := devadmWithClientForTest(db, http.StatusNoContent)

	err := d.AcceptDeviceAuth("foo")

	assert.NoError(t, err)

	err = d.AcceptDeviceAuth("bar")
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

			db := &mstore.DataStore{}
			db.On("DeleteDeviceAuth",
				mock.AnythingOfType("model.AuthID"),
			).Return(tc.datastoreError)
			i := devadmForTest(db)

			err := i.DeleteDeviceAuth("foo")

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
	db := &mstore.DataStore{}
	db.On("GetDeviceAuth", model.AuthID("foo")).
		Return(&model.DeviceAuth{ID: "foo"}, nil)
	db.On("GetDeviceAuth", model.AuthID("bar")).
		Return(nil, store.ErrNotFound)
	db.On("PutDeviceAuth", mock.AnythingOfType("*model.DeviceAuth")).
		Return(nil)

	d := devadmWithClientForTest(db, http.StatusNoContent)

	err := d.RejectDeviceAuth("foo")

	assert.NoError(t, err)

	err = d.RejectDeviceAuth("bar")
	assert.Error(t, err)
	assert.EqualError(t, err, store.ErrNotFound.Error())
}

func TestNewDevAdm(t *testing.T) {
	d := NewDevAdm(&mstore.DataStore{}, deviceauth.Config{})

	assert.NotNil(t, d)
}

func TestDevAdmWithContext(t *testing.T) {
	d := devadmForTest(&mstore.DataStore{})

	l := log.New(log.Ctx{})
	ctx := context.Background()
	ctx = context.WithValue(ctx, requestlog.ReqLog, l)
	dwc := d.WithContext(ctx).(*DevAdmWithContext)
	assert.NotNil(t, dwc)
	assert.NotNil(t, dwc.log)
	assert.Equal(t, dwc.log, l)
}
