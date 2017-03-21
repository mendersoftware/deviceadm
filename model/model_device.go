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
package model

import (
	"time"
)

type DeviceID string
type AuthID string

// wrapper for device attributes data
type DeviceAttributes map[string]string

const (
	DevStatusAccepted = "accepted"
	DevStatusRejected = "rejected"
	DevStatusPending  = "pending"
)

// Device wrapper
type Device struct {
	//system-generated authentication data set ID
	ID AuthID `json:"id" bson:",omitempty"`

	//device ID
	DeviceId DeviceID `json:"device_id" bson:",omitempty"`

	//blob of encrypted identity attributes
	DeviceIdentity string `json:"device_identity" bson:",omitempty"`

	//device's public key
	Key string `json:"key" bson:",omitempty"`

	//admission status('accepted', 'rejected', 'pending')
	Status string `json:"status" bson:",omitempty"`

	//decoded, human-readable identity attribute set
	Attributes DeviceAttributes `json:"attributes" bson:",omitempty"`

	//admission request reception time
	RequestTime *time.Time `json:"request_time" bson:"request_time,omitempty"`
}

func (did DeviceID) String() string {
	return string(did)
}

func (aid AuthID) String() string {
	return string(aid)
}
