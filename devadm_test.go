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
	"context"
	"errors"
	"fmt"
	"github.com/mendersoftware/deviceadm/log"
	"github.com/mendersoftware/deviceadm/requestid"
	"github.com/mendersoftware/deviceadm/requestlog"
	"net/http"
	"net/http/httptest"
	"testing"

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

func devadmWithClientForTest(d DataStore, clientRespStatus int) DevAdmApp {
	clientGetter := func() requestid.ApiRequester {
		return FakeApiRequester{clientRespStatus}
	}
	return &DevAdm{
		db:           d,
		clientGetter: clientGetter,
		log:          log.New(log.Ctx{})}
}

func devadmForTest(d DataStore) DevAdmApp {
	return &DevAdm{
		db:           d,
		clientGetter: simpleApiClientGetter,
		log:          log.New(log.Ctx{})}
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

func TestDevAdmSubmitDevice(t *testing.T) {
	db := &MockDataStore{}
	db.On("PutDevice", mock.AnythingOfType("*main.Device")).
		Return(nil)

	d := devadmWithClientForTest(db, http.StatusNoContent)

	err := d.SubmitDevice(Device{})

	assert.NoError(t, err)
}

func TestDevAdmSubmitDeviceErr(t *testing.T) {
	db := &MockDataStore{}
	db.On("PutDevice", mock.AnythingOfType("*main.Device")).
		Return(errors.New("db connection failed"))

	d := devadmWithClientForTest(db, http.StatusNoContent)

	err := d.SubmitDevice(Device{})

	if assert.Error(t, err) {
		assert.EqualError(t, err, "failed to put device: db connection failed")
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
	db.On("PutDevice", mock.AnythingOfType("*main.Device")).
		Return(nil)

	d := devadmWithClientForTest(db, http.StatusNoContent)

	err := d.AcceptDevice("foo")

	assert.NoError(t, err)

	err = d.AcceptDevice("bar")
	assert.Error(t, err)
	assert.EqualError(t, err, ErrDevNotFound.Error())
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
			datastoreError: ErrDevNotFound,
			outError:       ErrDevNotFound,
		},
		"datastore error": {
			datastoreError: errors.New("db connection failed"),
			outError:       errors.New("failed to delete device: db connection failed"),
		},
	}

	for name, tc := range testCases {
		t.Run(fmt.Sprintf("test case: %s", name), func(t *testing.T) {

			db := &MockDataStore{}
			db.On("DeleteDevice",
				mock.AnythingOfType("DeviceID"),
			).Return(tc.datastoreError)
			i := devadmForTest(db)

			err := i.DeleteDevice(DeviceID("foo"))

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
	db := &MockDataStore{}
	db.On("GetDevice", DeviceID("foo")).
		Return(&Device{ID: "foo"}, nil)
	db.On("GetDevice", DeviceID("bar")).
		Return(nil, ErrDevNotFound)
	db.On("PutDevice", mock.AnythingOfType("*main.Device")).
		Return(nil)

	d := devadmWithClientForTest(db, http.StatusNoContent)

	err := d.RejectDevice("foo")

	assert.NoError(t, err)

	err = d.RejectDevice("bar")
	assert.Error(t, err)
	assert.EqualError(t, err, ErrDevNotFound.Error())
}

func TestNewDevAdm(t *testing.T) {
	d := NewDevAdm(&MockDataStore{}, DevAuthClientConfig{})

	assert.NotNil(t, d)
}

func TestDevAdmWithContext(t *testing.T) {
	d := devadmForTest(&MockDataStore{})

	l := log.New(log.Ctx{})
	ctx := context.Background()
	ctx = context.WithValue(ctx, requestlog.ReqLog, l)
	dwc := d.WithContext(ctx).(*DevAdmWithContext)
	assert.NotNil(t, dwc)
	assert.NotNil(t, dwc.log)
	assert.Equal(t, dwc.log, l)
}
