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
	"github.com/spf13/viper"
	"testing"
)

func TestHandleConfigFile(t *testing.T) {

	if _, err := HandleConfigFile(""); err == nil {
		t.FailNow()
	}

	// Depends on default config being avaiable and correct (which is nice!)
	if _, err := HandleConfigFile("config.yaml"); err != nil {
		t.FailNow()
	}

}

func TestSetDefaultConfigs(t *testing.T) {
	defaults := []ConfigDefault{
		{"foo", "bar"},
		{"baz", 1},
	}

	c := viper.New()

	SetDefaultConfigs(c, defaults)

	val_foo := c.GetString("foo")
	val_baz := c.GetInt("baz")

	if val_foo != "bar" {
		t.FailNow()
	}

	if val_baz != 1 {
		t.FailNow()
	}
}
