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
	"net/http"

	api_http "github.com/mendersoftware/deviceadm/api/http"
	"github.com/mendersoftware/deviceadm/client/deviceauth"
	"github.com/mendersoftware/deviceadm/config"
	"github.com/mendersoftware/deviceadm/devadm"
	"github.com/mendersoftware/deviceadm/store/mongo"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/mendersoftware/go-lib-micro/log"
	"github.com/pkg/errors"
	"gopkg.in/mgo.v2"
)

func SetupAPI(stacktype string) (*rest.Api, error) {
	api := rest.NewApi()
	if err := SetupMiddleware(api, stacktype); err != nil {
		return nil, errors.Wrap(err, "failed to setup middleware")
	}

	//this will override the framework's error resp to the desired one:
	// {"error": "msg"}
	// instead of:
	// {"Error": "msg"}
	rest.ErrorFieldName = "error"

	return api, nil
}

func SetupDataStore(url string) (*mongo.DataStoreMongo, error) {
	dbSession, err := mgo.Dial(url)
	if err != nil {
		return nil, err
	}
	dbSession.SetSafe(&mgo.Safe{})

	d := mongo.NewDataStoreMongoWithSession(dbSession)
	return d, nil
}

func RunServer(c config.Reader) error {

	l := log.New(log.Ctx{})

	d, err := SetupDataStore(c.GetString(SettingDb))
	if err != nil {
		return errors.Wrap(err, "database connection failed")
	}

	err = d.EnsureIndexes()
	if err != nil {
		return errors.Wrap(err, "database indexing failed")
	}

	devadm := devadm.NewDevAdm(d, deviceauth.ClientConfig{
		DevauthUrl: c.GetString(SettingDevAuthUrl),
	})

	api, err := SetupAPI(c.GetString(SettingMiddleware))
	if err != nil {
		return errors.Wrap(err, "API setup failed")
	}

	devadmapi := api_http.NewDevAdmApiHandlers(devadm)

	apph, err := devadmapi.GetApp()
	if err != nil {
		return errors.Wrap(err, "device admission API handlers setup failed")
	}
	api.SetApp(apph)

	addr := c.GetString(SettingListen)
	l.Printf("listening on %s", addr)

	return http.ListenAndServe(addr, api.MakeHandler())
}
