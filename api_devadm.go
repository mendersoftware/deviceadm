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
	"github.com/ant0ine/go-json-rest/rest"
	"github.com/mendersoftware/deviceadm/utils"
	"github.com/pkg/errors"
	"net/http"
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
	DevAdm DevAdmApp
}

// return an ApiHandler for device admission app
func NewDevAdmApiHandlers(devadm DevAdmApp) ApiHandler {
	return &DevAdmHandlers{
		devadm,
	}
}

func (d *DevAdmHandlers) GetApp() (rest.App, error) {
	routes := []*rest.Route{
		rest.Get(uriDevices, d.GetDevicesHandler),
		rest.Post(uriDevices, d.AddDeviceHandler),

		rest.Get(uriDevice, d.GetDeviceHandler),

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
	page, perPage, err := utils.ParsePagination(r)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	status, err := utils.ParseQueryParmStr(r, utils.StatusName, false, utils.DevStatuses)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	//get one extra device to see if there's a 'next' page
	devs, err := d.DevAdm.ListDevices(int((page-1)*perPage), int(perPage+1), status)
	if err != nil {
		rest.Error(w, "failed to list devices", http.StatusInternalServerError)
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

func (d *DevAdmHandlers) AddDeviceHandler(w rest.ResponseWriter, r *rest.Request) {
	dev, err := parseDevice(r)
	if err != nil {
		rest.Error(w,
			err.Error(),
			http.StatusBadRequest)
		return
	}

	//save device in pending state
	dev.Status = "pending"
	err = d.DevAdm.AddDevice(dev)
	if err != nil {
		rest.Error(w,
			"internal error",
			http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func parseDevice(r *rest.Request) (*Device, error) {
	dev := Device{}

	//decode body
	err := r.DecodeJsonPayload(&dev)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode request body")
	}

	//validate id
	if dev.ID == DeviceID("") {
		return nil, errors.New("'id' field required")
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
func (d *DevAdmHandlers) getDeviceOrFail(w rest.ResponseWriter, r *rest.Request) *Device {
	devid := r.PathParam("id")

	dev, err := d.DevAdm.GetDevice(DeviceID(devid))

	if dev == nil {
		if err == ErrDevNotFound {
			rest.Error(w, err.Error(), http.StatusNotFound)
		} else {
			rest.Error(w, "internal error",
				http.StatusInternalServerError)
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
	devid := r.PathParam("id")

	var status DevAdmApiStatus
	err := r.DecodeJsonPayload(&status)
	if err != nil {
		rest.Error(w,
			errors.Wrap(err, "failed to decode status data").Error(),
			http.StatusBadRequest)
		return
	}

	// validate status
	if status.Status != DevStatusAccepted && status.Status != DevStatusRejected {
		rest.Error(w,
			errors.New("incorrect device status").Error(),
			http.StatusBadRequest)
		return
	}

	if status.Status == DevStatusAccepted {
		err = d.DevAdm.AcceptDevice(DeviceID(devid))
	} else if status.Status == DevStatusRejected {
		err = d.DevAdm.RejectDevice(DeviceID(devid))
	}
	if err != nil {
		code := http.StatusInternalServerError
		if err == ErrDevNotFound {
			code = http.StatusNotFound
		}
		rest.Error(w, err.Error(), code)
		return

	}

	devurl := utils.BuildURL(r, uriDevice, map[string]string{
		":id": devid,
	})
	w.Header().Add("Location", devurl.String())
	w.WriteHeader(http.StatusSeeOther)
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
