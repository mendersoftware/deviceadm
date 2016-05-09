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
	"github.com/ant0ine/go-json-rest/rest"
	"github.com/mendersoftware/deviceadm/utils"
	"github.com/pkg/errors"
	"net/http"
)

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
		rest.Get("/api/0.1.0/devices", d.GetDevices),
		rest.Post("/api/0.1.0/devices", d.AddDevice),

		rest.Get("/api/0.1.0/devices/:id", d.GetDevice),
		rest.Put("/api/0.1.0/devices/:id", d.UpdateDevice),

		rest.Get("/api/0.1.0/devices/:id/status", d.GetDeviceStatus),
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

func (d *DevAdmHandlers) GetDevices(w rest.ResponseWriter, r *rest.Request) {
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

func (d *DevAdmHandlers) AddDevice(w rest.ResponseWriter, r *rest.Request) {
}

func (d *DevAdmHandlers) GetDevice(w rest.ResponseWriter, r *rest.Request) {
}

func (d *DevAdmHandlers) UpdateDevice(w rest.ResponseWriter, r *rest.Request) {
}

func (d *DevAdmHandlers) GetDeviceStatus(w rest.ResponseWriter, r *rest.Request) {
}
