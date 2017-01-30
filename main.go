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
	"flag"
	"github.com/mendersoftware/deviceadm/config"
	"github.com/mendersoftware/deviceadm/log"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

func main() {
	var configPath string
	var printVersion bool
	var devSetup bool
	var debug bool

	flag.StringVar(&configPath, "config",
		"",
		"Configuration file path. Supports JSON, TOML, YAML and HCL formatted configs.")
	flag.BoolVar(&printVersion, "version",
		false, "Show version")
	flag.BoolVar(&devSetup, "dev",
		false, "Use development setup")
	flag.BoolVar(&debug, "debug",
		false, "Enable debug logging")

	flag.Parse()

	log.Setup(debug)

	l := log.New(log.Ctx{})

	conf, err := HandleConfigFile(configPath)
	if err != nil {
		l.Fatalf("error loading configuration: %s", err)
	}

	if devSetup == true {
		l.Infof("setting up development configuration")
		conf.Set(SettingMiddleware, EnvDev)
	}

	l.Printf("Device Admission Service, version %s starting up",
		CreateVersionString())

	db, err := NewDataStoreMongo(conf.GetString(SettingDb))
	if err != nil {
		l.Fatal("failed to connect to db")
	}

	err = db.Migrate(DbVersion, nil)
	if err != nil {
		l.Fatal("failed to run migrations")
	}

	l.Fatal(RunServer(conf))
}

func HandleConfigFile(filePath string) (config.Handler, error) {

	c := viper.New()

	// Set default values for config
	config.SetDefaults(c, configDefaults)

	// Enable setting conig values by environment variables
	c.SetEnvPrefix("DEVICEADM")
	c.AutomaticEnv()

	// Find and read the config file
	if filePath != "" {
		c.SetConfigFile(filePath)
		if err := c.ReadInConfig(); err != nil {
			return nil, errors.Wrap(err, "failed to read configuration")
		}
	}

	// Validate config
	if err := config.ValidateConfig(c, configValidators...); err != nil {
		return nil, errors.Wrap(err, "failed to validate configuration")
	}

	return c, nil
}
