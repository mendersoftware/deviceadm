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
package model

import (
	"encoding/json"
	"io"

	"github.com/asaskevich/govalidator"
)

type AuthSet struct {
	DeviceId string `json:"device_identity" valid:"required"`
	Key      string `json:"key" valid:"required"`
	//decoded, human-readable identity attribute set
	Attributes DeviceAuthAttributes `json:"-"`
}

func ParseAuthSet(source io.Reader) (*AuthSet, error) {
	jd := json.NewDecoder(source)

	var req AuthSet

	if err := jd.Decode(&req); err != nil {
		return nil, err
	}

	if err := req.Validate(); err != nil {
		return nil, err
	}

	return &req, nil
}

func (r *AuthSet) Validate() error {
	if _, err := govalidator.ValidateStruct(*r); err != nil {
		return err
	}

	return nil
}
