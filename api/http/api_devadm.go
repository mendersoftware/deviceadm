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
package http

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/mendersoftware/deviceadm/devadm"
	"github.com/mendersoftware/deviceadm/model"
	"github.com/mendersoftware/deviceadm/store"
	"github.com/mendersoftware/deviceadm/utils"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/mendersoftware/go-lib-micro/log"
	"github.com/mendersoftware/go-lib-micro/requestid"
	"github.com/mendersoftware/go-lib-micro/requestlog"
	"github.com/pkg/errors"
)

const (
	uriDevices      = "/api/0.1.0/devices"
	uriDevice       = "/api/0.1.0/devices/:id"
	uriDeviceStatus = "/api/0.1.0/devices/:id/status"
)

// model of device status response at /devices/:id/status endpoint,
// the response is a stripped down version of the device containing
// only the status field
type DevAdmApiStatus struct {
	Status string `json:"status"`
}

type DevAdmHandlers struct {
	DevAdm devadm.DevAdmApp
}

// return an ApiHandler for device admission app
func NewDevAdmApiHandlers(devadm devadm.DevAdmApp) ApiHandler {
	return &DevAdmHandlers{
		devadm,
	}
}

func (d *DevAdmHandlers) GetApp() (rest.App, error) {
	routes := []*rest.Route{
		rest.Get(uriDevices, d.GetDevicesHandler),
		rest.Delete(uriDevices, d.DeleteDevicesHandler),

		rest.Put(uriDevice, d.SubmitDeviceHandler),
		rest.Get(uriDevice, d.GetDeviceHandler),
		rest.Delete(uriDevice, d.DeleteDeviceHandler),

		rest.Get(uriDeviceStatus, d.GetDeviceStatusHandler),
		rest.Put(uriDeviceStatus, d.UpdateDeviceStatusHandler),
	}

	routes = append(routes)

	app, err := rest.MakeRouter(
		// augment routes with OPTIONS handler
		AutogenOptionsRoutes(routes, AllowHeaderOptionsGenerator)...,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create router")
	}

	return app, nil

}

func (d *DevAdmHandlers) GetDevicesHandler(w rest.ResponseWriter, r *rest.Request) {
	l := requestlog.GetRequestLogger(r.Env)

	page, perPage, err := utils.ParsePagination(r)
	if err != nil {
		restErrWithLog(w, r, l, err, http.StatusBadRequest)
		return
	}

	status, err := utils.ParseQueryParmStr(r, utils.StatusName, false, utils.DevStatuses)
	if err != nil {
		restErrWithLog(w, r, l, err, http.StatusBadRequest)
		return
	}

	da := d.DevAdm.WithContext(restToContext(r))

	//get one extra device to see if there's a 'next' page
	devs, err := da.ListDeviceAuths(int((page-1)*perPage), int(perPage+1), status)
	if err != nil {
		restErrWithLogInternal(w, r, l, errors.Wrap(err, "failed to list devices"))
		return
	}

	len := len(devs)
	hasNext := false
	if uint64(len) > perPage {
		hasNext = true
		len = int(perPage)
	}

	links := utils.MakePageLinkHdrs(r, page, perPage, hasNext)

	for _, l := range links {
		w.Header().Add("Link", l)
	}
	w.WriteJson(devs[:len])
}

func (d *DevAdmHandlers) DeleteDevicesHandler(w rest.ResponseWriter, r *rest.Request) {
	l := requestlog.GetRequestLogger(r.Env)

	devid, err := utils.ParseQueryParmStr(r, "device_id", true, nil)
	if err != nil {
		restErrWithLog(w, r, l, err, http.StatusBadRequest)
		return
	}

	da := d.DevAdm.WithContext(restToContext(r))

	err = da.DeleteDeviceData(model.DeviceID(devid))
	switch {
	case err == store.ErrNotFound:
		restErrWithLog(w, r, l, err, http.StatusNotFound)
	case err != nil:
		restErrWithLogInternal(w, r, l, err)
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}

func (d *DevAdmHandlers) SubmitDeviceHandler(w rest.ResponseWriter, r *rest.Request) {
	l := requestlog.GetRequestLogger(r.Env)

	dev, err := parseDevice(r)
	if err != nil {
		restErrWithLog(w, r, l, err, http.StatusBadRequest)
		return
	}

	da := d.DevAdm.WithContext(restToContext(r))

	//save device in pending state
	dev.Status = "pending"
	err = da.SubmitDeviceAuth(*dev)
	if err != nil {
		restErrWithLogInternal(w, r, l, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func parseDevice(r *rest.Request) (*model.DeviceAuth, error) {
	dev := model.DeviceAuth{}

	//decode body
	err := r.DecodeJsonPayload(&dev)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode request body")
	}

	//validate id
	id := r.PathParam("id")
	if id == "" {
		return nil, errors.New("'id' field required")
	}
	dev.ID = model.AuthID(id)

	if dev.DeviceId == "" {
		return nil, errors.New("'device_id' field required")
	}

	//validate identity
	if dev.DeviceIdentity == "" {
		return nil, errors.New("'device_identity' field required")
	}

	//validate key
	if dev.Key == "" {
		return nil, errors.New("'key' field required")
	}

	//decode attributes
	err = json.Unmarshal([]byte(dev.DeviceIdentity), &(dev.Attributes))
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode attributes data")
	}

	if len(dev.Attributes) == 0 {
		return nil, errors.New("no attributes provided")
	}
	return &dev, nil
}

// Helper for find a device of ID passed as path param ('id') in
// request 'r' and return it. If a device was not found returns nil
// and produces a sutabie error response using provided
// rest.ResponseWriter
func (d *DevAdmHandlers) getDeviceOrFail(w rest.ResponseWriter, r *rest.Request) *model.DeviceAuth {
	l := requestlog.GetRequestLogger(r.Env)

	authid := r.PathParam("id")

	da := d.DevAdm.WithContext(restToContext(r))
	dev, err := da.GetDeviceAuth(model.AuthID(authid))

	if dev == nil {
		if err == store.ErrNotFound {
			restErrWithLog(w, r, l, err, http.StatusNotFound)
		} else {
			restErrWithLogInternal(w, r, l, err)
		}
		return nil
	}

	return dev
}

func (d *DevAdmHandlers) GetDeviceHandler(w rest.ResponseWriter, r *rest.Request) {
	dev := d.getDeviceOrFail(w, r)
	// getDeviceOrFail() has already produced a suitable error
	// response if device was not found or something else happened

	if dev != nil {
		w.WriteJson(dev)
	}
}

func (d *DevAdmHandlers) UpdateDeviceStatusHandler(w rest.ResponseWriter, r *rest.Request) {
	l := requestlog.GetRequestLogger(r.Env)

	authid := r.PathParam("id")

	var status DevAdmApiStatus
	err := r.DecodeJsonPayload(&status)
	if err != nil {
		restErrWithLog(w, r, l, errors.Wrap(err, "failed to decode status data"), http.StatusBadRequest)
		return
	}

	// validate status
	if status.Status != model.DevStatusAccepted && status.Status != model.DevStatusRejected {
		restErrWithLog(w, r, l, errors.New("incorrect device status"), http.StatusBadRequest)
		return
	}

	da := d.DevAdm.WithContext(restToContext(r))

	if status.Status == model.DevStatusAccepted {
		err = da.AcceptDeviceAuth(model.AuthID(authid))
	} else if status.Status == model.DevStatusRejected {
		err = da.RejectDeviceAuth(model.AuthID(authid))
	}
	if err != nil {
		if err == store.ErrNotFound {
			restErrWithLog(w, r, l, err, http.StatusNotFound)
		} else {
			restErrWithLogInternal(w, r, l, errors.Wrap(err, "failed to list change device status"))
		}
		return
	}

	w.WriteJson(&status)
}

func (d *DevAdmHandlers) GetDeviceStatusHandler(w rest.ResponseWriter, r *rest.Request) {
	dev := d.getDeviceOrFail(w, r)
	// getDeviceOrFail() has already produced a suitable error
	// response if device was not found or something else happened

	if dev != nil {
		w.WriteJson(DevAdmApiStatus{
			dev.Status,
		})
	}
}

func (d *DevAdmHandlers) DeleteDeviceHandler(w rest.ResponseWriter, r *rest.Request) {
	l := requestlog.GetRequestLogger(r.Env)

	devid := r.PathParam("id")

	da := d.DevAdm.WithContext(restToContext(r))
	err := da.DeleteDeviceAuth(model.AuthID(devid))

	switch err {
	case nil:
		w.WriteHeader(http.StatusNoContent)
	case store.ErrNotFound:
		restErrWithLog(w, r, l, store.ErrNotFound, http.StatusNotFound)
	default:
		restErrWithLogInternal(w, r, l, err)
	}

	return
}

// return selected http code + error message directly taken from error
// log error
func restErrWithLog(w rest.ResponseWriter, r *rest.Request, l *log.Logger, e error, code int) {
	restErrWithLogMsg(w, r, l, e, code, e.Error())
}

// return http 500, with an "internal error" message
// log full error
func restErrWithLogInternal(w rest.ResponseWriter, r *rest.Request, l *log.Logger, e error) {
	msg := "internal error"
	e = errors.Wrap(e, msg)
	restErrWithLogMsg(w, r, l, e, http.StatusInternalServerError, msg)
}

// return an error code with an overriden message (to avoid exposing the details)
// log full error
func restErrWithLogMsg(w rest.ResponseWriter, r *rest.Request, l *log.Logger, e error, code int, msg string) {
	w.WriteHeader(code)
	err := w.WriteJson(map[string]string{
		rest.ErrorFieldName: msg,
		"request_id":        requestid.GetReqId(r),
	})
	if err != nil {
		panic(err)
	}
	l.F(log.Ctx{}).Error(errors.Wrap(e, msg).Error())
}

// unpack contextual request data into context.Context
func restToContext(r *rest.Request) context.Context {
	ctx := r.Context()
	ctx = log.WithContext(ctx, requestlog.GetRequestLogger(r.Env))
	ctx = requestid.WithContext(ctx, requestid.GetReqId(r))
	return ctx
}
