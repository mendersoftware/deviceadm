// Copyright 2018 Northern.tech AS
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
package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"testing"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/ant0ine/go-json-rest/rest/test"
	"github.com/mendersoftware/go-lib-micro/requestid"
	"github.com/mendersoftware/go-lib-micro/requestlog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/mendersoftware/deviceadm/devadm"
	mdevadm "github.com/mendersoftware/deviceadm/devadm/mocks"
	"github.com/mendersoftware/deviceadm/model"
	"github.com/mendersoftware/deviceadm/store"
	"github.com/mendersoftware/deviceadm/utils"
)

func mockListDeviceAuths(num int) []model.DeviceAuth {
	var devs []model.DeviceAuth
	for i := 0; i < num; i++ {
		devs = append(devs, model.DeviceAuth{
			ID:       model.AuthID(strconv.Itoa(i)),
			DeviceId: model.DeviceID(strconv.Itoa(i)),
		})
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
		skip   int
		limit  int
		filter store.Filter

		listDevices    []model.DeviceAuth
		listDevicesErr error

		req *http.Request

		code int
		body string
		hdrs []string
	}{
		{
			// valid pagination, no next page
			skip:        15,
			limit:       6,
			listDevices: mockListDeviceAuths(5),
			req: test.MakeSimpleRequest("GET",
				"http://1.2.3.4/api/management/v1/admission/devices?page=4&per_page=5", nil),
			code: 200,
			body: ToJson(mockListDeviceAuths(5)),
			hdrs: []string{
				fmt.Sprintf(utils.LinkTmpl, "devices",
					"page=3&per_page=5", "prev"),
				fmt.Sprintf(utils.LinkTmpl, "devices",
					"page=1&per_page=5", "first"),
			},
		},
		{
			//valid pagination, with next page
			skip:        15,
			limit:       6,
			listDevices: mockListDeviceAuths(9),
			req: test.MakeSimpleRequest("GET",
				"http://1.2.3.4/api/management/v1/admission/devices?page=4&per_page=5", nil),
			code: 200,
			body: ToJson(mockListDeviceAuths(5)),
			hdrs: []string{
				fmt.Sprintf(utils.LinkTmpl, "devices",
					"page=3&per_page=5", "prev"),
				fmt.Sprintf(utils.LinkTmpl, "devices",
					"page=1&per_page=5", "first"),
			},
		},
		{
			//invalid pagination - format
			req: test.MakeSimpleRequest("GET",
				"http://1.2.3.4/api/management/v1/admission/devices?page=foo&per_page=5", nil),
			code: 400,
			body: RestError(utils.MsgQueryParmInvalid("page")),
		},
		{
			//invalid pagination - format
			req: test.MakeSimpleRequest("GET",
				"http://1.2.3.4/api/management/v1/admission/devices?page=1&per_page=foo", nil),
			code: 400,
			body: RestError(utils.MsgQueryParmInvalid("per_page")),
		},
		{
			//invalid pagination - bounds
			req: test.MakeSimpleRequest("GET",
				"http://1.2.3.4/api/management/v1/admission/devices?page=0&per_page=5", nil),
			code: 400,
			body: RestError(utils.MsgQueryParmLimit("page")),
		},
		{
			//valid status
			skip:        15,
			limit:       6,
			filter:      store.Filter{Status: "accepted"},
			listDevices: mockListDeviceAuths(6),
			req: test.MakeSimpleRequest("GET",
				"http://1.2.3.4/api/management/v1/admission/devices?page=4&per_page=5&status=accepted", nil),
			code: 200,
			body: ToJson(mockListDeviceAuths(5)),
			hdrs: []string{
				fmt.Sprintf(utils.LinkTmpl, "devices",
					"page=3&per_page=5&status=accepted", "prev"),
				fmt.Sprintf(utils.LinkTmpl, "devices",
					"page=1&per_page=5&status=accepted", "first"),
			},
		},
		{
			//invalid status
			req: test.MakeSimpleRequest("GET",
				"http://1.2.3.4/api/management/v1/admission/devices?page=1&per_page=5&status=foo", nil),
			code: 400,
			body: RestError(utils.MsgQueryParmOneOf("status", utils.DevStatuses)),
		},
		{
			//devadm.ListDevices error
			skip:           15,
			limit:          6,
			listDevicesErr: errors.New("devadm error"),
			req: test.MakeSimpleRequest("GET",
				"http://1.2.3.4/api/management/v1/admission/devices?page=4&per_page=5", nil),
			code: 500,
			body: RestError("internal error"),
		},
		{
			limit:       21,
			filter:      store.Filter{DeviceID: "foo"},
			listDevices: mockListDeviceAuths(2),
			req: test.MakeSimpleRequest("GET",
				"http://1.2.3.4/api/management/v1/admission/devices?device_id=foo", nil),
			code: 200,
			body: ToJson(mockListDeviceAuths(2)),
		},
	}

	for idx, tc := range testCases {
		t.Logf("tc: %v", idx)
		devadm := &mdevadm.App{}
		devadm.On("ListDeviceAuths",
			mock.MatchedBy(func(c context.Context) bool { return true }),
			tc.skip, tc.limit, tc.filter).Return(tc.listDevices, tc.listDevicesErr)

		apih := makeMockApiHandler(t, devadm)

		rest.ErrorFieldName = "error"

		recorded := runTestRequest(t, apih, tc.req, tc.code, tc.body)

		for _, h := range tc.hdrs {
			assert.Equal(t, h, ExtractHeader("Link", h, recorded))
		}
	}
}

func makeMockApiHandler(t *testing.T, mocka *mdevadm.App) http.Handler {
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
		dev *model.DeviceAuth
		err error
	}{
		"foo": {
			&model.DeviceAuth{
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

	getDeviceAuth := func(id model.AuthID) (*model.DeviceAuth, error) {
		d, ok := devs[id.String()]
		if ok == false {
			return nil, store.ErrNotFound
		}
		if d.err != nil {
			return nil, d.err
		}
		return d.dev, nil
	}
	devadm := &mdevadm.App{}
	devadm.On("GetDeviceAuth",
		mock.MatchedBy(func(c context.Context) bool { return true }),
		mock.AnythingOfType("model.AuthID")).Return(
		func(_ context.Context, id model.AuthID) *model.DeviceAuth {
			da, _ := getDeviceAuth(id)
			return da
		},
		func(_ context.Context, id model.AuthID) error {
			_, err := getDeviceAuth(id)
			return err
		},
	)

	apih := makeMockApiHandler(t, devadm)

	// enforce specific field naming in errors returned by API
	rest.ErrorFieldName = "error"

	tcases := []struct {
		req  *http.Request
		code int
		body string
	}{
		{
			test.MakeSimpleRequest("GET",
				"http://1.2.3.4/api/management/v1/admission/devices/foo", nil),
			200,
			ToJson(devs["foo"].dev),
		},
		{
			test.MakeSimpleRequest("GET",
				"http://1.2.3.4/api/management/v1/admission/devices/foo/status", nil),
			200,
			ToJson(struct {
				Status string `json:"status"`
			}{
				devs["foo"].dev.Status,
			}),
		},
		{
			test.MakeSimpleRequest("GET",
				"http://1.2.3.4/api/management/v1/admission/devices/bar", nil),
			500,
			RestError(devs["bar"].err.Error()),
		},
		{
			test.MakeSimpleRequest("GET",
				"http://1.2.3.4/api/management/v1/admission/devices/baz", nil),
			404,
			RestError(store.ErrNotFound.Error()),
		},
		{
			test.MakeSimpleRequest("GET", "http://1.2.3.4/api/management/v1/admission/devices/baz/status", nil),
			404,
			RestError(store.ErrNotFound.Error()),
		},
	}

	for _, tc := range tcases {
		runTestRequest(t, apih, tc.req, tc.code, tc.body)
	}
}

func TestApiDevAdmUpdateStatusDevice(t *testing.T) {
	devs := map[string]struct {
		dev *model.DeviceAuth
		err error
	}{
		"foo": {
			&model.DeviceAuth{
				ID:             "foo",
				DeviceId:       "bar",
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
		"abovelimit": {
			&model.DeviceAuth{
				ID:             "foo",
				DeviceId:       "bar",
				Key:            "foobar",
				Status:         "accepted",
				DeviceIdentity: "deadcafe",
			},
			&utils.UsageError{UserMsg: "max dev count limit reached"},
		},
	}

	mockaction := func(_ context.Context, id model.AuthID) error {
		d, ok := devs[id.String()]
		if ok == false {
			return store.ErrNotFound
		}
		if d.err != nil {
			return d.err
		}
		return nil
	}
	devadm := &mdevadm.App{}
	devadm.On("AcceptDeviceAuth",
		mock.MatchedBy(func(c context.Context) bool { return true }),
		mock.AnythingOfType("model.AuthID")).Return(mockaction)
	devadm.On("RejectDeviceAuth",
		mock.MatchedBy(func(c context.Context) bool { return true }),
		mock.AnythingOfType("model.AuthID")).Return(mockaction)

	apih := makeMockApiHandler(t, devadm)
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
				"http://1.2.3.4/api/management/v1/admission/devices/foo/status", nil),
			code: 400,
			body: RestError("failed to decode status data: JSON payload is empty"),
		},
		{
			req: test.MakeSimpleRequest("PUT",
				"http://1.2.3.4/api/management/v1/admission/devices/foo/status",
				DevAdmApiStatus{"foo"}),
			code: 400,
			body: RestError("incorrect device status"),
		},
		{
			req: test.MakeSimpleRequest("PUT",
				"http://1.2.3.4/api/management/v1/admission/devices/foo/status",
				accstatus),
			code: 200,
			body: ToJson(accstatus),
		},
		{
			req: test.MakeSimpleRequest("PUT",
				"http://1.2.3.4/api/management/v1/admission/devices/bar/status",
				accstatus),
			code: 500,
			body: RestError(devs["bar"].err.Error()),
		},
		{
			req: test.MakeSimpleRequest("PUT",
				"http://1.2.3.4/api/management/v1/admission/devices/baz/status",
				accstatus),
			code: 404,
			body: RestError(store.ErrNotFound.Error()),
		},
		{
			req: test.MakeSimpleRequest("PUT",
				"http://1.2.3.4/api/management/v1/admission/devices/foo/status",
				rejstatus),
			code: 200,
			body: ToJson(rejstatus),
		},
		{
			req: test.MakeSimpleRequest("PUT",
				"http://1.2.3.4/api/management/v1/admission/devices/abovelimit/status",
				accstatus),
			code: 422,
			body: RestError("max dev count limit reached"),
		},
	}

	for _, tc := range tcases {
		runTestRequest(t, apih, tc.req, tc.code, tc.body)
	}

}

func TestApiDevAdmAcceptPreauthorized(t *testing.T) {
	testCases := map[string]struct {
		id        string
		body      interface{}
		devAdmErr error
		respCode  int
		respBody  string
	}{
		"ok": {
			id:       "1",
			body:     DevAdmApiStatus{"accepted"},
			respCode: 200,
			respBody: ToJson(DevAdmApiStatus{"accepted"}),
		},
		"error: bad request": {
			id:       "1",
			body:     DevAdmApiStatus{"unknown"},
			respCode: 400,
			respBody: RestError("incorrect device status"),
		},
		"error: bad request 2": {
			id:       "1",
			body:     DevAdmApiStatus{"rejected"},
			respCode: 400,
			respBody: RestError("incorrect device status"),
		},
		"error: not found": {
			id:   "2",
			body: DevAdmApiStatus{"accepted"},

			devAdmErr: devadm.ErrAuthNotFound,
			respCode:  404,
			respBody:  RestError("device auth set not found"),
		},
		"error: conflict": {
			id:   "3",
			body: DevAdmApiStatus{"accepted"},

			devAdmErr: devadm.ErrNotPreauthorized,
			respCode:  409,
			respBody:  RestError("auth set must be in 'preauthorized' state"),
		},
		"error: generic": {
			id:   "3",
			body: DevAdmApiStatus{"accepted"},

			devAdmErr: errors.New("some error"),
			respCode:  500,
			respBody:  RestError("internal error"),
		},
	}

	for name, tc := range testCases {
		t.Run(fmt.Sprintf("test case: %s", name), func(t *testing.T) {
			devadm := &mdevadm.App{}

			devadm.On("AcceptDevicePreAuth",
				mock.MatchedBy(func(c context.Context) bool { return true }),
				model.AuthID(tc.id)).Return(tc.devAdmErr)

			apih := makeMockApiHandler(t, devadm)

			rest.ErrorFieldName = "error"

			req := test.MakeSimpleRequest("PUT",
				fmt.Sprintf("http://1.2.3.4/api/internal/v1/admission/devices/%s/status", tc.id),
				tc.body)

			runTestRequest(t, apih, req, tc.respCode, tc.respBody)
		})
	}
}

func TestNewDevAdmApiHandlers(t *testing.T) {
	h := NewDevAdmApiHandlers(&mdevadm.App{})
	assert.NotNil(t, h)
}

func TestApiDevAdmGetApp(t *testing.T) {
	h := NewDevAdmApiHandlers(&mdevadm.App{})
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
		devAdmErr error
		id        model.AuthID
		respCode  int
		respBody  string
	}{
		"empty body": {
			req: test.MakeSimpleRequest("PUT",
				"http://1.2.3.4/api/management/v1/admission/devices/id-0001",
				nil),
			id:       "id-0001",
			respCode: 400,
			respBody: RestError("failed to decode request body: JSON payload is empty"),
		},
		"garbled body": {
			req: test.MakeSimpleRequest("PUT",
				"http://1.2.3.4/api/management/v1/admission/devices/id-0001",
				"foo bar"),
			id:       "id-0001",
			respCode: 400,
			respBody: RestError("failed to decode request body: json: cannot unmarshal string into Go value of type model.DeviceAuth"),
		},
		"body formatted ok, all fields present": {
			req: test.MakeSimpleRequest("PUT",
				"http://1.2.3.4/api/management/v1/admission/devices/id-0001",
				map[string]string{
					"key":       "key-0001",
					"device_id": "123",
					"device_identity": makeJson(t,
						map[string]string{
							"mac": "00:00:00:01",
						}),
				},
			),
			id:       "id-0001",
			respCode: 204,
		},
		"body formatted ok, 'key' missing": {
			req: test.MakeSimpleRequest("PUT",
				"http://1.2.3.4/api/management/v1/admission/devices/id-0001",
				map[string]string{
					"device_id": "123",
					"device_identity": makeJson(t,
						map[string]string{
							"mac": "00:00:00:01",
						}),
				},
			),
			id:       "id-0001",
			respCode: 400,
			respBody: RestError("'key' field required"),
		},
		"body formatted ok, identity missing": {
			req: test.MakeSimpleRequest("PUT",
				"http://1.2.3.4/api/management/v1/admission/devices/id-0001",
				map[string]string{
					"device_id": "123",
					"key":       "key-0001",
				},
			),
			id:       "id-0001",
			respCode: 400,
			respBody: RestError("'device_identity' field required"),
		},
		"body formatted ok, identity garbled": {
			req: test.MakeSimpleRequest("PUT",
				"http://1.2.3.4/api/management/v1/admission/devices/id-0001",
				map[string]string{
					"device_id":       "123",
					"key":             "key-0001",
					"device_identity": "{mac: foobar}",
				},
			),
			id:       "id-0001",
			respCode: 400,
			respBody: RestError("failed to decode attributes data: invalid character 'm' looking for beginning of object key string"),
		},
		"body formatted ok, identity empty": {
			req: test.MakeSimpleRequest("PUT",
				"http://1.2.3.4/api/management/v1/admission/devices/id-0001",
				map[string]string{
					"device_id":       "123",
					"key":             "key-0001",
					"device_identity": "{}",
				},
			),
			id:       "id-0001",
			respCode: 400,
			respBody: RestError("no attributes provided"),
		},
		"body formatted ok, devadm error": {
			req: test.MakeSimpleRequest("PUT",
				"http://1.2.3.4/api/management/v1/admission/devices/id-0001",
				map[string]string{
					"device_id": "123",
					"key":       "key-0001",
					"device_identity": makeJson(t,
						map[string]string{
							"mac": "00:00:00:01",
						}),
				},
			),
			devAdmErr: errors.New("internal error"),
			id:        "id-0001",
			respCode:  500,
			respBody:  RestError("internal error"),
		},
		"body formatted ok, missing device_id": {
			req: test.MakeSimpleRequest("PUT",
				"http://1.2.3.4/api/management/v1/admission/devices/id-0001",
				map[string]string{
					"key": "key-0001",
					"device_identity": makeJson(t,
						map[string]string{
							"mac": "00:00:00:01",
						}),
				},
			),
			devAdmErr: errors.New("internal error"),
			id:        "id-0001",
			respCode:  400,
			respBody:  RestError("'device_id' field required"),
		},
	}

	for name, tc := range testCases {
		t.Logf("test case: %s", name)
		devadm := &mdevadm.App{}
		devadm.On("SubmitDeviceAuth",
			mock.MatchedBy(func(c context.Context) bool { return true }),
			mock.MatchedBy(
				func(d model.DeviceAuth) bool {
					return assert.NotEmpty(t, d.Attributes) &&
						assert.NotEmpty(t, d.DeviceId) &&
						assert.Equal(t, tc.id, d.ID)
				})).Return(tc.devAdmErr)

		apih := makeMockApiHandler(t, devadm)

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
		id        model.AuthID

		code int
		body string
	}{
		"success": {
			req: test.MakeSimpleRequest("DELETE", "http://1.2.3.4/api/internal/v1/admission/devices/2", nil),

			devadmErr: nil,
			id:        "2",

			code: http.StatusNoContent,
			body: "",
		},
		"error: no device": {
			req: test.MakeSimpleRequest("DELETE", "http://1.2.3.4/api/internal/v1/admission/devices/1", nil),

			devadmErr: store.ErrNotFound,
			id:        "1",

			code: http.StatusNoContent,
		},
		"error: generic": {
			req: test.MakeSimpleRequest("DELETE", "http://1.2.3.4/api/internal/v1/admission/devices/3", nil),

			devadmErr: errors.New("some internal error"),
			id:        "3",

			code: http.StatusInternalServerError,
			body: RestError("internal error"),
		},
	}

	for name, tc := range tcases {
		t.Run(fmt.Sprintf("test case: %s", name), func(t *testing.T) {
			devadm := &mdevadm.App{}
			devadm.On("DeleteDeviceAuth",
				mock.MatchedBy(func(c context.Context) bool { return true }),
				tc.id).Return(tc.devadmErr)

			apih := makeMockApiHandler(t, devadm)

			runTestRequest(t, apih, tc.req, tc.code, tc.body)
		})
	}
}

func TestApiDeleteDeviceData(t *testing.T) {
	t.Parallel()
	rest.ErrorFieldName = "error"

	tcases := map[string]struct {
		req *http.Request

		devadmErr error
		devid     model.DeviceID

		code int
		body string
	}{
		"success": {
			req: test.MakeSimpleRequest("DELETE", "http://1.2.3.4/api/internal/v1/admission/devices?device_id=2", nil),

			devadmErr: nil,
			devid:     "2",

			code: http.StatusNoContent,
			body: "",
		},
		"error: no device": {
			req: test.MakeSimpleRequest("DELETE", "http://1.2.3.4/api/internal/v1/admission/devices?device_id=1", nil),

			devadmErr: store.ErrNotFound,
			devid:     "1",

			code: http.StatusNoContent,
		},
		"error: generic": {
			req: test.MakeSimpleRequest("DELETE", "http://1.2.3.4/api/internal/v1/admission/devices?device_id=3", nil),

			devadmErr: errors.New("some internal error"),
			devid:     "3",

			code: http.StatusInternalServerError,
			body: RestError("internal error"),
		},
	}

	for name, tc := range tcases {
		t.Run(fmt.Sprintf("test case: %s", name), func(t *testing.T) {
			devadm := &mdevadm.App{}
			devadm.On("DeleteDeviceData",
				mock.MatchedBy(func(c context.Context) bool { return true }),
				tc.devid).Return(tc.devadmErr)

			apih := makeMockApiHandler(t, devadm)

			runTestRequest(t, apih, tc.req, tc.code, tc.body)
		})
	}
}

func TestApiDeleteDeviceManagement(t *testing.T) {
	t.Parallel()
	rest.ErrorFieldName = "error"

	tcases := map[string]struct {
		req *http.Request

		devadmErr error
		id        model.AuthID

		code int
		body string
	}{
		"success": {
			req: test.MakeSimpleRequest("DELETE", "http://1.2.3.4/api/management/v1/admission/devices/2", nil),

			devadmErr: nil,
			id:        "2",

			code: http.StatusNoContent,
			body: "",
		},
		"error: no device": {
			req: test.MakeSimpleRequest("DELETE", "http://1.2.3.4/api/management/v1/admission/devices/1", nil),

			devadmErr: store.ErrNotFound,
			id:        "1",

			code: http.StatusNoContent,
		},
		"error: generic": {
			req: test.MakeSimpleRequest("DELETE", "http://1.2.3.4/api/management/v1/admission/devices/3", nil),

			devadmErr: errors.New("some internal error"),
			id:        "3",

			code: http.StatusInternalServerError,
			body: RestError("internal error"),
		},
	}

	for name, tc := range tcases {
		t.Run(fmt.Sprintf("test case: %s", name), func(t *testing.T) {
			devadm := &mdevadm.App{}
			devadm.On("DeleteDeviceAuthPropagate",
				mock.MatchedBy(func(c context.Context) bool { return true }),
				tc.id, mock.AnythingOfType("string")).Return(tc.devadmErr)

			apih := makeMockApiHandler(t, devadm)

			runTestRequest(t, apih, tc.req, tc.code, tc.body)
		})
	}
}

func TestApiDevAdmPostTenants(t *testing.T) {
	testCases := map[string]struct {
		req       *http.Request
		devAdmErr error
		respCode  int
		respBody  string
	}{
		"ok": {
			req: test.MakeSimpleRequest("POST",
				"http://1.2.3.4/api/internal/v1/admission/tenants",
				model.NewTenant{TenantId: "foo"}),
			respCode: 201,
			respBody: "",
		},
		"error: empty request": {
			req: test.MakeSimpleRequest("POST",
				"http://1.2.3.4/api/internal/v1/admission/tenants",
				nil),
			respCode: 400,
			respBody: RestError("EOF"),
		},
		"error: no tenant_id": {
			req: test.MakeSimpleRequest("POST",
				"http://1.2.3.4/api/internal/v1/admission/tenants",
				model.NewTenant{TenantId: ""},
			),
			respCode: 400,
			respBody: RestError("tenant_id must be provided"),
		},
		"error: generic": {
			req: test.MakeSimpleRequest("POST",
				"http://1.2.3.4/api/internal/v1/admission/tenants",
				model.NewTenant{TenantId: "foo"},
			),
			devAdmErr: errors.New("can't provision tenant"),
			respCode:  500,
			respBody:  RestError("internal error"),
		},
	}

	for name, tc := range testCases {
		t.Logf("test case: %s", name)
		devadm := &mdevadm.App{}

		devadm.On("ProvisionTenant",
			mock.MatchedBy(func(c context.Context) bool { return true }),
			mock.AnythingOfType("string")).Return(tc.devAdmErr)

		apih := makeMockApiHandler(t, devadm)

		rest.ErrorFieldName = "error"

		runTestRequest(t, apih, tc.req, tc.respCode, tc.respBody)
	}
}

func TestApiDevAdmPostDeviceAuth(t *testing.T) {
	testCases := map[string]struct {
		input     interface{}
		devAdmErr error
		respCode  int
		respBody  string
	}{
		"ok": {
			input: model.AuthSet{Key: "foo-key", DeviceId: makeJson(t,
				map[string]string{
					"mac": "00:00:00:01",
				})},
			respCode: 201,
			respBody: "",
		},
		"error: empty request": {
			input:    nil,
			respCode: 400,
			respBody: RestError("EOF"),
		},

		"error: generic": {
			input: model.AuthSet{Key: "foo-key", DeviceId: makeJson(t,
				map[string]string{
					"mac": "00:00:00:01",
				})},
			devAdmErr: errors.New("can't provision tenant"),
			respCode:  500,
			respBody:  RestError("internal error"),
		},
		"error: no key": {
			input: model.AuthSet{Key: "", DeviceId: makeJson(t,
				map[string]string{
					"mac": "00:00:00:01",
				})},
			respCode: 400,
			respBody: RestError("key: non zero value required"),
		},
		"error: no identity data": {
			input:    model.AuthSet{Key: "foo-key"},
			respCode: 400,
			respBody: RestError("device_identity: non zero value required"),
		},
		"error: conflict": {
			input: model.AuthSet{Key: "foo-key", DeviceId: makeJson(t,
				map[string]string{
					"mac": "00:00:00:01",
				})}, devAdmErr: devadm.AuthSetConflictError,
			respCode: 409,
			respBody: RestError("device already exists"),
		},
	}

	for name, tc := range testCases {
		t.Logf("test case: %s", name)
		devadm := &mdevadm.App{}

		if tc.respCode != 400 {
			devadm.On("PreauthorizeDevice",
				mock.MatchedBy(func(c context.Context) bool { return true }),
				mock.MatchedBy(
					func(d model.AuthSet) bool {
						return assert.NotEmpty(t, d.Attributes) &&
							assert.NotEmpty(t, d.DeviceId) &&
							assert.NotEmpty(t, d.Key)
					}),
				"Bearer: foo-token",
			).Return(tc.devAdmErr)
		}
		apih := makeMockApiHandler(t, devadm)

		rest.ErrorFieldName = "error"

		req := test.MakeSimpleRequest("POST",
			"http://1.2.3.4/api/management/v1/admission/devices",
			tc.input)
		req.Header.Add("Authorization", "Bearer: foo-token")
		runTestRequest(t, apih, req, tc.respCode, tc.respBody)
		devadm.AssertExpectations(t)
	}
}
