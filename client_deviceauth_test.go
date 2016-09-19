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
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDevAuthClientUrl(t *testing.T) {
	da := NewDevAuthClient(DevAuthClientConfig{
		DevauthUrl: "http://devauth:9999",
	}, &http.Client{}, log.New(log.Ctx{}))

	s := da.buildDevAuthUpdateUrl(Device{
		ID: "foobar",
	})

	assert.Equal(t, "http://devauth:9999/api/0.1.0/devices/foobar/status", s)
}

// return mock http server returning status code 'status'
func newMockServer(status int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
	}))

}

func TestDevAuthClientReqSuccess(t *testing.T) {
	s := newMockServer(200)
	defer s.Close()

	c := NewDevAuthClient(DevAuthClientConfig{
		DevauthUrl: s.URL,
	}, &http.Client{}, log.New(log.Ctx{}))

	err := c.UpdateDevice(Device{
		ID: "123",
	})
	assert.NoError(t, err, "expected no errors")
}

func TestDevAuthClientReqFail(t *testing.T) {
	s := newMockServer(400)
	defer s.Close()

	c := NewDevAuthClient(DevAuthClientConfig{
		DevauthUrl: s.URL,
	}, &http.Client{}, log.New(log.Ctx{}))

	err := c.UpdateDevice(Device{
		ID: "123",
	})
	assert.Error(t, err, "expected an error")
}

func TestDevAuthClientReqUrl(t *testing.T) {
	var urlPath string

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// this is just URI endpoint, without host/port/scheme
		// etc.
		urlPath = r.URL.Path
		w.WriteHeader(200)
	}))
	defer s.Close()

	c := NewDevAuthClient(DevAuthClientConfig{
		DevauthUrl: s.URL,
	}, &http.Client{}, log.New(log.Ctx{}))

	devid := "123"
	err := c.UpdateDevice(Device{
		ID: DeviceID(devid),
	})

	assert.NoError(t, err, "expected no errors")
}

func TestDevAuthClientReqNoHost(t *testing.T) {
	c := NewDevAuthClient(DevAuthClientConfig{}, &http.Client{}, log.New(log.Ctx{}))

	devid := "123"
	err := c.UpdateDevice(Device{
		ID: DeviceID(devid),
	})

	assert.Error(t, err, "expected an error")
}

func TestDevAuthClientTImeout(t *testing.T) {

	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	// channel for nofitying the responder that the test is
	// complete
	testdone := make(chan bool)

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// wait for the test to notify us about timeout
		select {
		case <-testdone:
			// test finished, can leave now
		case <-time.After(defaultDevAuthReqTimeout * 2):
			// don't block longer than default timeout * 2
		}
		w.WriteHeader(400)
	}))

	c := NewDevAuthClient(DevAuthClientConfig{
		DevauthUrl: s.URL,
	}, &http.Client{Timeout: defaultDevAuthReqTimeout}, log.New(log.Ctx{}))

	devid := "123"

	t1 := time.Now()
	err := c.UpdateDevice(Device{
		ID: DeviceID(devid),
	})
	t2 := time.Now()

	// let the responder know we're done
	testdone <- true

	s.Close()

	assert.Error(t, err, "expected timeout error")
	// allow some slack in timeout, add 20% of the default timeout
	maxdur := defaultDevAuthReqTimeout +
		time.Duration(0.2*float64(defaultDevAuthReqTimeout))

	assert.WithinDuration(t, t2, t1, maxdur, "timeout took too long")
}

func TestUseLog(t *testing.T) {
	c := NewDevAuthClient(DevAuthClientConfig{}, &http.Client{}, log.New(log.Ctx{}))

	l2 := log.New(log.Ctx{"test": "test"})
	c.UseLog(l2)
	assert.Equal(t, l2, c.log)
}
