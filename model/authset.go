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
package model

import (
	"crypto/rsa"
	"encoding/json"
	"io"

	"github.com/asaskevich/govalidator"
	"github.com/pkg/errors"

	"github.com/mendersoftware/deviceadm/utils"
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

	err := json.Unmarshal([]byte(req.DeviceId), &(req.Attributes))
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode attributes data")
	}

	if len(req.Attributes) == 0 {
		return nil, errors.New("no attributes provided")
	}

	return &req, nil
}

func (r *AuthSet) Validate() error {
	if _, err := govalidator.ValidateStruct(*r); err != nil {
		return err
	}

	// validate/normalize key
	key, err := utils.ParsePubKey(r.Key)
	if err != nil {
		return err
	}

	keyStruct, ok := key.(*rsa.PublicKey)
	if !ok {
		return errors.New("cannot decode public key")
	}

	serialized, err := utils.SerializePubKey(keyStruct)
	if err != nil {
		return err
	}

	r.Key = serialized

	return nil
}
