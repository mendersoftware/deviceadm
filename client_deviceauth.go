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
	"bytes"
	"encoding/json"
	"github.com/mendersoftware/deviceadm/log"
	"github.com/pkg/errors"
	"net/http"
	"strings"
	"time"
)

const (
	// default device ID endpoint
	defaultDevAuthDevicesUri = "/api/0.1.0/devices/{id}/status"
	// default request timeout, 10s?
	defaultDevAuthReqTimeout = time.Duration(10) * time.Second
)

type DevAuthClientConfig struct {
	// root devauth address
	DevauthUrl string
	// template of update URL, string '{id}' will be replaced with
	// device ID
	UpdateUrl string
	// request timeout
	Timeout time.Duration
}

type DevAuthClient struct {
	Client http.Client
	log    *log.Logger
	conf   DevAuthClientConfig
}

// devauth's status request
type DevAuthStatusReq struct {
	Status string `json:"status"`
}

// TODO rename this and calling funcs to UpdateDeviceStatus etc.
// perhaps change the interface - the whole Device isn't needed
// leaving for later, requires large refact in tests etc.
func (d *DevAuthClient) UpdateDevice(dev Device) error {
	d.log.Debugf("update device %s", dev.ID)

	url := d.buildDevAuthUpdateUrl(dev)

	statusReqJson, err := json.Marshal(DevAuthStatusReq{
		Status: dev.Status,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to prepare dev auth request")
	}

	reader := bytes.NewReader(statusReqJson)

	req, err := http.NewRequest(http.MethodPut, url, reader)
	if err != nil {
		return errors.Wrapf(err, "failed to prepare dev auth request")
	}

	req.Header.Set("Content-Type", "application/json")

	rsp, err := d.Client.Do(req)
	if err != nil {
		return errors.Wrapf(err, "failed to update device status")
	}
	defer rsp.Body.Close()

	if rsp.StatusCode != http.StatusOK {
		return errors.Errorf("device status update request failed with status %v",
			rsp.Status)
	}
	return nil
}

func NewDevAuthClient(c DevAuthClientConfig) *DevAuthClient {
	if c.Timeout == 0 {
		c.Timeout = defaultDevAuthReqTimeout
	}

	c.UpdateUrl = c.DevauthUrl + defaultDevAuthDevicesUri

	return &DevAuthClient{
		Client: http.Client{
			// request timeout
			Timeout: c.Timeout,
		},
		log:  log.New("devauth-client"),
		conf: c,
	}
}

func (d *DevAuthClient) buildDevAuthUpdateUrl(dev Device) string {
	return strings.Replace(d.conf.UpdateUrl, "{id}", dev.ID.String(), 1)
}
