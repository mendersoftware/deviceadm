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
	"github.com/pkg/errors"
	"net/http"
	"strings"
	"time"
)

const (
	// default device ID endpoint
	defaultDevAuthDevicesUri = "/devices/{id}"
	// default request timeout, 10s?
	defaultDevAuthReqTimeout = time.Duration(10) * time.Second
)

type DevAuthClientConfig struct {
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

func (d *DevAuthClient) UpdateDevice(dev Device) error {
	d.log.Debugf("update device %s", dev.ID)

	url := d.buildDevAuthUpdateUrl(dev)
	req, err := http.NewRequest(http.MethodPut, url, nil)

	// TODO: prepare message

	rsp, err := d.Client.Do(req)
	if err != nil {
		return errors.Wrapf(err, "failed to update device status")
	}
	defer rsp.Body.Close()

	if rsp.StatusCode != http.StatusOK {
		return errors.Wrapf(err,
			"device update request failed with status %v", rsp.Status)
	}
	return nil
}

func NewDevAuthClient(c DevAuthClientConfig) *DevAuthClient {
	if c.Timeout == 0 {
		c.Timeout = defaultDevAuthReqTimeout
	}

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
