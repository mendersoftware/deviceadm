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
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ant0ine/go-json-rest/rest"
	"github.com/ant0ine/go-json-rest/rest/test"
	"github.com/mendersoftware/deviceadm/requestid"
	"github.com/mendersoftware/deviceadm/requestlog"
	"github.com/mendersoftware/deviceadm/utils"
	"github.com/stretchr/testify/assert"
	"net/http"
	"strconv"
	"testing"
)

type MockDevAdm struct {
	mockListDevices  func(skip int, limit int, status string) ([]Device, error)
	mockGetDevice    func(id DeviceID) (*Device, error)
	mockAcceptDevice func(id DeviceID) error
	mockRejectDevice func(id DeviceID) error
	mockSubmitDevice func(d Device) error
	mockDeleteDevice func(id DeviceID) error
	mockWithContext  func(c context.Context) DevAdmApp
}

func (mda *MockDevAdm) ListDevices(skip int, limit int, status string) ([]Device, error) {
	return mda.mockListDevices(skip, limit, status)
}

func (mda *MockDevAdm) SubmitDevice(dev Device) error {
	return mda.mockSubmitDevice(dev)
}

func (mda *MockDevAdm) GetDevice(id DeviceID) (*Device, error) {
	return mda.mockGetDevice(id)
}

func (mda *MockDevAdm) AcceptDevice(id DeviceID) error {
	return mda.mockAcceptDevice(id)
}

func (mda *MockDevAdm) RejectDevice(id DeviceID) error {
	return mda.mockRejectDevice(id)
}

func (mda *MockDevAdm) DeleteDevice(id DeviceID) error {
	return mda.mockDeleteDevice(id)
}

func (mda *MockDevAdm) WithContext(c context.Context) DevAdmApp {
	return mda
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
	msg, _ := json.Marshal(map[string]interface{}{"error": status, "request_id": "test"})
	return string(msg)
}

func runTestRequest(t *testing.T, handler http.Handler, req *http.Request, code int, body string) *test.Recorded {
	req.Header.Add(requestid.RequestIdHeader, "test")
	recorded := test.RunRequest(t, handler, req)
	recorded.CodeIs(code)
	recorded.BodyIs(body)
	return recorded
}

func TestApiDevAdmGetDevices(t *testing.T) {
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
				fmt.Sprintf(utils.LinkTmpl, "r", "page=3&per_page=5", "prev"),
				fmt.Sprintf(utils.LinkTmpl, "r", "page=1&per_page=5", "first"),
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
				fmt.Sprintf(utils.LinkTmpl, "r", "page=3&per_page=5", "prev"),
				fmt.Sprintf(utils.LinkTmpl, "r", "page=1&per_page=5", "first"),
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
				fmt.Sprintf(utils.LinkTmpl, "r", "page=3&per_page=5&status=allowed", "prev"),
				fmt.Sprintf(utils.LinkTmpl, "r", "page=1&per_page=5&status=allowed", "first"),
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
			RestError("internal error"),
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
		router, err := rest.MakeRouter(rest.Get("/r", handlers.GetDevicesHandler))
		assert.NotNil(t, router)
		assert.NoError(t, err)

		api := rest.NewApi()
		api.Use(
			&requestlog.RequestLogMiddleware{},
			&requestid.RequestIdMiddleware{},
		)
		api.SetApp(router)

		rest.ErrorFieldName = "error"

		recorded := runTestRequest(t, api.MakeHandler(), testCase.inReq,
			testCase.outResponseCode, testCase.outResponseBody)

		for _, h := range testCase.outHdrs {
			assert.Equal(t, h, ExtractHeader("Link", h, recorded))
		}
	}
}

func makeMockApiHandler(t *testing.T, mocka *MockDevAdm) http.Handler {
	handlers := NewDevAdmApiHandlers(mocka)
	assert.NotNil(t, handlers)

	app, err := handlers.GetApp()
	assert.NotNil(t, app)
	assert.NoError(t, err)

	api := rest.NewApi()
	api.Use(
		&requestlog.RequestLogMiddleware{},
		&requestid.RequestIdMiddleware{},
	)
	api.SetApp(app)

	return api.MakeHandler()
}

func TestApiDevAdmGetDevice(t *testing.T) {
	devs := map[string]struct {
		dev *Device
		err error
	}{
		"foo": {
			&Device{
				ID:             "foo",
				Key:            "foobar",
				Status:         "accepted",
				DeviceIdentity: "deadcafe",
			},
			nil,
		},
		"bar": {
			nil,
			errors.New("internal error"),
		},
	}

	devadm := MockDevAdm{
		mockGetDevice: func(id DeviceID) (*Device, error) {
			d, ok := devs[id.String()]
			if ok == false {
				return nil, ErrDevNotFound
			}
			if d.err != nil {
				return nil, d.err
			}
			return d.dev, nil
		},
	}

	apih := makeMockApiHandler(t, &devadm)

	// enforce specific field naming in errors returned by API
	rest.ErrorFieldName = "error"

	tcases := []struct {
		req  *http.Request
		code int
		body string
	}{
		{
			test.MakeSimpleRequest("GET",
				"http://1.2.3.4/api/0.1.0/devices/foo", nil),
			200,
			ToJson(devs["foo"].dev),
		},
		{
			test.MakeSimpleRequest("GET",
				"http://1.2.3.4/api/0.1.0/devices/foo/status", nil),
			200,
			ToJson(struct {
				Status string `json:"status"`
			}{
				devs["foo"].dev.Status,
			}),
		},
		{
			test.MakeSimpleRequest("GET",
				"http://1.2.3.4/api/0.1.0/devices/bar", nil),
			500,
			RestError(devs["bar"].err.Error()),
		},
		{
			test.MakeSimpleRequest("GET",
				"http://1.2.3.4/api/0.1.0/devices/baz", nil),
			404,
			RestError(ErrDevNotFound.Error()),
		},
		{
			test.MakeSimpleRequest("GET", "http://1.2.3.4/api/0.1.0/devices/baz/status", nil),
			404,
			RestError(ErrDevNotFound.Error()),
		},
	}

	for _, tc := range tcases {
		runTestRequest(t, apih, tc.req, tc.code, tc.body)
	}
}

func TestApiDevAdmUpdateStatusDevice(t *testing.T) {
	devs := map[string]struct {
		dev *Device
		err error
	}{
		"foo": {
			&Device{
				ID:             "foo",
				Key:            "foobar",
				Status:         "accepted",
				DeviceIdentity: "deadcafe",
			},
			nil,
		},
		"bar": {
			nil,
			errors.New("internal error"),
		},
	}

	mockaction := func(id DeviceID) error {
		d, ok := devs[id.String()]
		if ok == false {
			return ErrDevNotFound
		}
		if d.err != nil {
			return d.err
		}
		return nil
	}
	devadm := MockDevAdm{
		mockAcceptDevice: mockaction,
		mockRejectDevice: mockaction,
	}

	apih := makeMockApiHandler(t, &devadm)
	// enforce specific field naming in errors returned by API
	rest.ErrorFieldName = "error"

	accstatus := DevAdmApiStatus{"accepted"}
	rejstatus := DevAdmApiStatus{"rejected"}

	tcases := []struct {
		req  *http.Request
		code int
		body string
	}{
		{
			req: test.MakeSimpleRequest("PUT",
				"http://1.2.3.4/api/0.1.0/devices/foo/status", nil),
			code: 400,
			body: RestError("failed to decode status data: JSON payload is empty"),
		},
		{
			req: test.MakeSimpleRequest("PUT",
				"http://1.2.3.4/api/0.1.0/devices/foo/status",
				DevAdmApiStatus{"foo"}),
			code: 400,
			body: RestError("incorrect device status"),
		},
		{
			req: test.MakeSimpleRequest("PUT",
				"http://1.2.3.4/api/0.1.0/devices/foo/status",
				accstatus),
			code: 200,
			body: ToJson(accstatus),
		},
		{
			req: test.MakeSimpleRequest("PUT",
				"http://1.2.3.4/api/0.1.0/devices/bar/status",
				accstatus),
			code: 500,
			body: RestError(devs["bar"].err.Error()),
		},
		{
			req: test.MakeSimpleRequest("PUT",
				"http://1.2.3.4/api/0.1.0/devices/baz/status",
				accstatus),
			code: 404,
			body: RestError(ErrDevNotFound.Error()),
		},
		{
			req: test.MakeSimpleRequest("PUT",
				"http://1.2.3.4/api/0.1.0/devices/foo/status",
				rejstatus),
			code: 200,
			body: ToJson(rejstatus),
		},
	}

	for _, tc := range tcases {
		runTestRequest(t, apih, tc.req, tc.code, tc.body)
	}

}

func TestNewDevAdmApiHandlers(t *testing.T) {
	h := NewDevAdmApiHandlers(&MockDevAdm{})
	assert.NotNil(t, h)
}

func TestApiDevAdmGetApp(t *testing.T) {
	h := NewDevAdmApiHandlers(&MockDevAdm{})
	a, err := h.GetApp()
	assert.NotNil(t, a)
	assert.NoError(t, err)
}

func makeJson(t *testing.T, d interface{}) string {
	out, err := json.Marshal(d)
	if err != nil {
		t.FailNow()
	}

	return string(out)
}

func TestApiDevAdmSubmitDevice(t *testing.T) {
	testCases := map[string]struct {
		req       *http.Request
		devAdmErr string
		respCode  int
		respBody  string
	}{
		"empty body": {
			req: test.MakeSimpleRequest("PUT",
				"http://1.2.3.4/api/0.1.0/devices/id-0001",
				nil),
			devAdmErr: "",
			respCode:  400,
			respBody:  RestError("failed to decode request body: JSON payload is empty"),
		},
		"garbled body": {
			req: test.MakeSimpleRequest("PUT",
				"http://1.2.3.4/api/0.1.0/devices/id-0001",
				"foo bar"),
			devAdmErr: "",
			respCode:  400,
			respBody:  RestError("failed to decode request body: json: cannot unmarshal string into Go value of type main.Device"),
		},
		"body formatted ok, all fields present": {
			req: test.MakeSimpleRequest("PUT",
				"http://1.2.3.4/api/0.1.0/devices/id-0001",
				map[string]string{
					"key": "key-0001",
					"device_identity": makeJson(t,
						map[string]string{
							"mac": "00:00:00:01",
						}),
				},
			),
			devAdmErr: "",
			respCode:  204,
			respBody:  "",
		},
		"body formatted ok, 'key' missing": {
			req: test.MakeSimpleRequest("PUT",
				"http://1.2.3.4/api/0.1.0/devices/id-0001",
				map[string]string{
					"device_identity": makeJson(t,
						map[string]string{
							"mac": "00:00:00:01",
						}),
				},
			),
			devAdmErr: "",
			respCode:  400,
			respBody:  RestError("'key' field required"),
		},
		"body formatted ok, identity missing": {
			req: test.MakeSimpleRequest("PUT",
				"http://1.2.3.4/api/0.1.0/devices/id-0001",
				map[string]string{
					"key": "key-0001",
				},
			),
			devAdmErr: "",
			respCode:  400,
			respBody:  RestError("'device_identity' field required"),
		},
		"body formatted ok, identity garbled": {
			req: test.MakeSimpleRequest("PUT",
				"http://1.2.3.4/api/0.1.0/devices/id-0001",
				map[string]string{
					"key":             "key-0001",
					"device_identity": "{mac: foobar}",
				},
			),
			devAdmErr: "",
			respCode:  400,
			respBody:  RestError("failed to decode attributes data: invalid character 'm' looking for beginning of object key string"),
		},
		"body formatted ok, identity empty": {
			req: test.MakeSimpleRequest("PUT",
				"http://1.2.3.4/api/0.1.0/devices/id-0001",
				map[string]string{
					"key":             "key-0001",
					"device_identity": "{}",
				},
			),
			devAdmErr: "",
			respCode:  400,
			respBody:  RestError("no attributes provided"),
		},
		"body formatted ok, devadm error": {
			req: test.MakeSimpleRequest("PUT",
				"http://1.2.3.4/api/0.1.0/devices/id-0001",
				map[string]string{
					"key": "key-0001",
					"device_identity": makeJson(t,
						map[string]string{
							"mac": "00:00:00:01",
						}),
				},
			),
			devAdmErr: "internal error",
			respCode:  500,
			respBody:  RestError("internal error"),
		},
	}

	for name, tc := range testCases {
		t.Logf("test case: %s", name)
		devadm := MockDevAdm{
			mockSubmitDevice: func(d Device) error {
				if tc.devAdmErr != "" {
					return errors.New(tc.devAdmErr)
				}
				return nil
			},
		}

		apih := makeMockApiHandler(t, &devadm)

		rest.ErrorFieldName = "error"

		runTestRequest(t, apih, tc.req, tc.respCode, tc.respBody)
	}
}

func TestApiDeleteDevice(t *testing.T) {
	t.Parallel()
	rest.ErrorFieldName = "error"

	tcases := map[string]struct {
		req *http.Request

		devadmErr error

		code int
		body string
	}{
		"success": {
			req: test.MakeSimpleRequest("DELETE", "http://1.2.3.4/api/0.1.0/devices/2", nil),

			devadmErr: nil,

			code: http.StatusNoContent,
			body: "",
		},
		"error: no device": {
			req: test.MakeSimpleRequest("DELETE", "http://1.2.3.4/api/0.1.0/devices/1", nil),

			devadmErr: ErrDevNotFound,

			code: http.StatusNotFound,
			body: RestError(ErrDevNotFound.Error()),
		},
		"error: generic": {
			req: test.MakeSimpleRequest("DELETE", "http://1.2.3.4/api/0.1.0/devices/3", nil),

			devadmErr: errors.New("some internal error"),

			code: http.StatusInternalServerError,
			body: RestError("internal error"),
		},
	}

	for name, tc := range tcases {
		t.Run(fmt.Sprintf("test case: %s", name), func(t *testing.T) {
			t.Parallel()

			devadm := MockDevAdm{
				mockDeleteDevice: func(id DeviceID) error {
					return tc.devadmErr
				},
			}

			apih := makeMockApiHandler(t, &devadm)

			runTestRequest(t, apih, tc.req, tc.code, tc.body)
		})
	}
}
