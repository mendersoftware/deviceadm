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
	"github.com/mendersoftware/deviceadm/config"
)

const (
	SettingListen        = "listen"
	SettingListenDefault = ":8080"

	SettingMiddleware        = "middleware"
	SettingMiddlewareDefault = EnvProd

	SettingDb        = "mongo"
	SettingDbDefault = "mongo-device-adm:27017"

	SettingDevAuthUrl        = "devauthurl"
	SettingDevAuthUrlDefault = "http://mender-device-auth:8080/api/0.1.0/devices/{id}"
)

var (
	configValidators = []config.Validator{}
	configDefaults   = []config.Default{
		{SettingListen, SettingListenDefault},
		{SettingMiddleware, SettingMiddlewareDefault},
		{SettingDb, SettingDbDefault},
		{SettingDevAuthUrl, SettingDevAuthUrlDefault},
	}
)
