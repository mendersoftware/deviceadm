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
	"github.com/pkg/errors"
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
}

func (d *DevAdmHandlers) AddDevice(w rest.ResponseWriter, r *rest.Request) {
}

func (d *DevAdmHandlers) GetDevice(w rest.ResponseWriter, r *rest.Request) {
}

func (d *DevAdmHandlers) UpdateDevice(w rest.ResponseWriter, r *rest.Request) {
}

func (d *DevAdmHandlers) GetDeviceStatus(w rest.ResponseWriter, r *rest.Request) {
}
