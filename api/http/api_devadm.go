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
package http

import (
	"encoding/json"
	"net/http"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/mendersoftware/go-lib-micro/log"
	"github.com/mendersoftware/go-lib-micro/requestid"
	"github.com/pkg/errors"

	"github.com/mendersoftware/deviceadm/devadm"
	"github.com/mendersoftware/deviceadm/model"
	"github.com/mendersoftware/deviceadm/store"
	"github.com/mendersoftware/deviceadm/utils"
)

const (
	uriDevices      = "/api/management/v1/admission/devices"
	uriDevice       = "/api/management/v1/admission/devices/:id"
	uriDeviceStatus = "/api/management/v1/admission/devices/:id/status"

	//internal api
	uriDevicesInternal      = "/api/internal/v1/admission/devices"
	uriDeviceInternal       = "/api/internal/v1/admission/devices/:id"
	uriDeviceStatusInternal = "/api/internal/v1/admission/devices/:id/status"

	uriTenants = "/api/internal/v1/admission/tenants"
)

// model of device status response at /devices/:id/status endpoint,
// the response is a stripped down version of the device containing
// only the status field
type DevAdmApiStatus struct {
	Status string `json:"status"`
}

type DevAdmHandlers struct {
	DevAdm devadm.App
}

// return an ApiHandler for device admission app
func NewDevAdmApiHandlers(devadm devadm.App) ApiHandler {
	return &DevAdmHandlers{
		devadm,
	}
}

func (d *DevAdmHandlers) GetApp() (rest.App, error) {
	routes := []*rest.Route{
		rest.Get(uriDevices, d.GetDevicesHandler),
		rest.Post(uriDevices, d.PostDevicesHandler),
		rest.Delete(uriDevicesInternal, d.DeleteDevicesHandler),

		rest.Put(uriDevice, d.SubmitDeviceHandler),
		rest.Get(uriDevice, d.GetDeviceHandler),
		rest.Delete(uriDeviceInternal, d.DeleteDeviceHandler),

		rest.Get(uriDeviceStatus, d.GetDeviceStatusHandler),
		rest.Put(uriDeviceStatus, d.UpdateDeviceStatusHandler),
		rest.Put(uriDeviceStatusInternal, d.AcceptPreauthorizedHandler),

		rest.Post(uriTenants, d.ProvisionTenantHandler),
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
	ctx := r.Context()
	l := log.FromContext(ctx)

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

	deviceId, err := utils.ParseQueryParmStr(r, "device_id", false, nil)
	if err != nil {
		restErrWithLog(w, r, l, err, http.StatusBadRequest)
		return
	}

	//get one extra device to see if there's a 'next' page
	devs, err := d.DevAdm.ListDeviceAuths(ctx,
		int((page-1)*perPage), int(perPage+1),
		store.Filter{
			Status:   status,
			DeviceID: model.DeviceID(deviceId),
		})
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

func (d *DevAdmHandlers) PostDevicesHandler(w rest.ResponseWriter, r *rest.Request) {
	ctx := r.Context()
	l := log.FromContext(ctx)

	defer r.Body.Close()
	authSet, err := parseAuthSet(r)
	if err != nil {
		restErrWithLog(w, r, l, err, http.StatusBadRequest)
		return
	}

	err = d.DevAdm.PreauthorizeDevice(ctx, *authSet, r.Header.Get("Authorization"))
	if err != nil {
		if err == devadm.AuthSetConflictError {
			restErrWithLog(w, r, l, err, http.StatusConflict)
			return
		}
		restErrWithLogInternal(w, r, l, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (d *DevAdmHandlers) DeleteDevicesHandler(w rest.ResponseWriter, r *rest.Request) {
	ctx := r.Context()
	l := log.FromContext(ctx)

	devid, err := utils.ParseQueryParmStr(r, "device_id", true, nil)
	if err != nil {
		restErrWithLog(w, r, l, err, http.StatusBadRequest)
		return
	}

	err = d.DevAdm.DeleteDeviceData(ctx, model.DeviceID(devid))

	if err != nil && err != store.ErrNotFound {
		restErrWithLogInternal(w, r, l, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (d *DevAdmHandlers) SubmitDeviceHandler(w rest.ResponseWriter, r *rest.Request) {
	ctx := r.Context()
	l := log.FromContext(ctx)

	dev, err := parseDevice(r)
	if err != nil {
		restErrWithLog(w, r, l, err, http.StatusBadRequest)
		return
	}

	//save device in pending state
	dev.Status = model.DevStatusPending
	err = d.DevAdm.SubmitDeviceAuth(ctx, *dev)
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

func parseAuthSet(r *rest.Request) (*model.AuthSet, error) {
	authSet, err := model.ParseAuthSet(r.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal([]byte(authSet.DeviceId), &(authSet.Attributes))
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode attributes data")
	}

	if len(authSet.Attributes) == 0 {
		return nil, errors.New("no attributes provided")
	}

	return authSet, nil
}

// Helper for find a device of ID passed as path param ('id') in
// request 'r' and return it. If a device was not found returns nil
// and produces a sutabie error response using provided
// rest.ResponseWriter
func (d *DevAdmHandlers) getDeviceOrFail(w rest.ResponseWriter, r *rest.Request) *model.DeviceAuth {
	ctx := r.Context()
	l := log.FromContext(ctx)

	authid := r.PathParam("id")

	dev, err := d.DevAdm.GetDeviceAuth(ctx, model.AuthID(authid))

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
	ctx := r.Context()
	l := log.FromContext(ctx)

	authid := r.PathParam("id")

	var status DevAdmApiStatus
	err := r.DecodeJsonPayload(&status)
	if err != nil {
		restErrWithLog(w, r, l,
			errors.Wrap(err, "failed to decode status data"),
			http.StatusBadRequest)
		return
	}

	// validate status
	if status.Status != model.DevStatusAccepted &&
		status.Status != model.DevStatusRejected {
		restErrWithLog(w, r, l,
			errors.New("incorrect device status"),
			http.StatusBadRequest)
		return
	}

	if status.Status == model.DevStatusAccepted {
		err = d.DevAdm.AcceptDeviceAuth(ctx, model.AuthID(authid))
	} else if status.Status == model.DevStatusRejected {
		err = d.DevAdm.RejectDeviceAuth(ctx, model.AuthID(authid))
	}
	if err != nil {
		if utils.IsUsageError(err) {
			restErrWithLog(w, r, l, err, http.StatusUnprocessableEntity)
		} else if err == store.ErrNotFound {
			restErrWithLog(w, r, l, err, http.StatusNotFound)
		} else {
			restErrWithLogInternal(w, r, l,
				errors.Wrap(err,
					"failed to list change device status"))
		}
		return
	}

	w.WriteJson(&status)
}

func (d *DevAdmHandlers) AcceptPreauthorizedHandler(w rest.ResponseWriter, r *rest.Request) {
	ctx := r.Context()
	l := log.FromContext(ctx)

	authid := r.PathParam("id")

	var status DevAdmApiStatus
	err := r.DecodeJsonPayload(&status)
	if err != nil {
		restErrWithLog(w, r, l,
			errors.Wrap(err, "failed to decode status data"),
			http.StatusBadRequest)
		return
	}

	if status.Status != model.DevStatusAccepted {
		restErrWithLog(w, r, l,
			errors.New("incorrect device status"),
			http.StatusBadRequest)
		return
	}

	err = d.DevAdm.AcceptDevicePreAuth(ctx, model.AuthID(authid))
	switch err {
	case nil:
		w.WriteJson(&status)
	case devadm.ErrNotPreauthorized:
		restErrWithLog(w, r, l, err, http.StatusConflict)
	case devadm.ErrAuthNotFound:
		restErrWithLog(w, r, l, err, http.StatusNotFound)
	default:
		restErrWithLogInternal(w, r, l, err)
	}
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
	ctx := r.Context()
	l := log.FromContext(ctx)

	devid := r.PathParam("id")

	err := d.DevAdm.DeleteDeviceAuth(ctx, model.AuthID(devid))

	if err != nil && err != store.ErrNotFound {
		restErrWithLogInternal(w, r, l, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (d *DevAdmHandlers) ProvisionTenantHandler(w rest.ResponseWriter, r *rest.Request) {
	ctx := r.Context()
	l := log.FromContext(ctx)

	defer r.Body.Close()

	tenant, err := model.ParseNewTenant(r.Body)
	if err != nil {
		restErrWithLog(w, r, l, err, http.StatusBadRequest)
		return
	}

	err = d.DevAdm.ProvisionTenant(ctx, tenant.TenantId)
	if err != nil {
		restErrWithLogInternal(w, r, l, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
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
