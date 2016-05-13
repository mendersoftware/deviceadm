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
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ant0ine/go-json-rest/rest"
	"github.com/ant0ine/go-json-rest/rest/test"
	"github.com/mendersoftware/deviceadm/utils"
	"github.com/stretchr/testify/assert"
	"net/http"
	"strconv"
	"testing"
)

type MockDevAdm struct {
	mockListDevices func(skip int, limit int, status string) ([]Device, error)
}

func (mda *MockDevAdm) ListDevices(skip int, limit int, status string) ([]Device, error) {
	return mda.mockListDevices(skip, limit, status)
}

func (mda *MockDevAdm) AddDevice(dev *Device) error {
	return errors.New("not implemented")
}

func (mda *MockDevAdm) GetDevice(id DeviceID) *Device {
	return nil
}

func (mda *MockDevAdm) UpdateDevice(id DeviceID, dev *Device) error {
	return errors.New("not implemented")
}

func mockListDevices(num int) []Device {
	var devs []Device
	for i := 0; i < num; i++ {
		devs = append(devs, Device{ID: DeviceID(strconv.Itoa(i))})
	}
	return devs
}

func ToJson(data interface{}) string {
	j, _ := json.Marshal(data)
	return string(j)
}

// test.HasHeader only tests the first header,
// so create a wrapper for headers with multiple values
func HasHeader(hdr, val string, r *test.Recorded) bool {
	rec := r.Recorder
	for _, v := range rec.Header()[hdr] {
		if v == val {
			return true
		}
	}

	return false
}

func ExtractHeader(hdr, val string, r *test.Recorded) string {
	rec := r.Recorder
	for _, v := range rec.Header()[hdr] {
		if v == val {
			return v
		}
	}

	return ""
}

func RestError(status string) string {
	msg, _ := json.Marshal(map[string]string{"error": status})
	return string(msg)
}

func TestGetDevices(t *testing.T) {
	testCases := []struct {
		listDevicesNum  int
		listDevicesErr  error
		inReq           *http.Request
		outResponseCode int
		outResponseBody string
		outHdrs         []string
	}{
		{
			//valid pagination, no next page
			5,
			nil,
			test.MakeSimpleRequest("GET", "http://1.2.3.4/r?page=4&per_page=5", nil),
			200,
			ToJson(mockListDevices(5)),
			[]string{
				fmt.Sprintf(utils.LinkTmpl, "http://1.2.3.4/r?page=3&per_page=5", "prev"),
				fmt.Sprintf(utils.LinkTmpl, "http://1.2.3.4/r?page=1&per_page=5", "first"),
			},
		},
		{
			//valid pagination, with next page
			9,
			nil,
			test.MakeSimpleRequest("GET", "http://1.2.3.4/r?page=4&per_page=5", nil),
			200,
			ToJson(mockListDevices(5)),
			[]string{
				fmt.Sprintf(utils.LinkTmpl, "http://1.2.3.4/r?page=3&per_page=5", "prev"),
				fmt.Sprintf(utils.LinkTmpl, "http://1.2.3.4/r?page=1&per_page=5", "first"),
			},
		},
		{
			//invalid pagination - format
			5,
			nil,
			test.MakeSimpleRequest("GET", "http://1.2.3.4/r?page=foo&per_page=5", nil),
			400,
			RestError(utils.MsgQueryParmInvalid("page")),
			nil,
		},
		{
			//invalid pagination - format
			5,
			nil,
			test.MakeSimpleRequest("GET", "http://1.2.3.4/r?page=1&per_page=foo", nil),
			400,
			RestError(utils.MsgQueryParmInvalid("per_page")),
			nil,
		},
		{
			//invalid pagination - bounds
			5,
			nil,
			test.MakeSimpleRequest("GET", "http://1.2.3.4/r?page=0&per_page=5", nil),
			400,
			RestError(utils.MsgQueryParmLimit("page")),
			nil,
		},
		{
			//valid status
			5,
			nil,
			test.MakeSimpleRequest("GET", "http://1.2.3.4/r?page=4&per_page=5&status=allowed", nil),
			200,
			ToJson(mockListDevices(5)),
			[]string{
				fmt.Sprintf(utils.LinkTmpl, "http://1.2.3.4/r?page=3&per_page=5&status=allowed", "prev"),
				fmt.Sprintf(utils.LinkTmpl, "http://1.2.3.4/r?page=1&per_page=5&status=allowed", "first"),
			},
		},
		{
			//invalid status
			5,
			nil,
			test.MakeSimpleRequest("GET", "http://1.2.3.4/r?page=1&per_page=5&status=foo", nil),
			400,
			RestError(utils.MsgQueryParmOneOf("status", utils.DevStatuses)),
			nil,
		},
		{
			//devadm.ListDevices error
			5,
			errors.New("devadm error"),
			test.MakeSimpleRequest("GET", "http://1.2.3.4/r?page=4&per_page=5", nil),
			500,
			RestError("failed to list devices"),
			nil,
		},
	}

	for _, testCase := range testCases {
		devadm := MockDevAdm{
			mockListDevices: func(skip int, limit int, status string) ([]Device, error) {
				if testCase.listDevicesErr != nil {
					return nil, testCase.listDevicesErr
				}

				return mockListDevices(testCase.listDevicesNum), nil
			},
		}

		handlers := DevAdmHandlers{&devadm}
		router, err := rest.MakeRouter(rest.Get("/r", handlers.GetDevices))
		assert.NoError(t, err)

		api := rest.NewApi()
		api.SetApp(router)

		rest.ErrorFieldName = "error"

		recorded := test.RunRequest(t, api.MakeHandler(), testCase.inReq)
		recorded.CodeIs(testCase.outResponseCode)
		recorded.BodyIs(testCase.outResponseBody)

		for _, h := range testCase.outHdrs {
			assert.Equal(t, h, ExtractHeader("Link", h, recorded))
		}
	}
}

func TestNewDevAdmApiHandlers(t *testing.T) {
	h := NewDevAdmApiHandlers(&MockDevAdm{})
	assert.NotNil(t, h)
}

func TestGetApp(t *testing.T) {
	h := NewDevAdmApiHandlers(&MockDevAdm{})
	a, err := h.GetApp()
	assert.NotNil(t, a)
	assert.NoError(t, err)
}
