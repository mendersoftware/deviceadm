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
	"github.com/stretchr/testify/assert"
	"testing"
)

func devadmForTest() DevAdmApp {
	return &DevAdm{}
}

func TestListDevices(t *testing.T) {
	d := devadmForTest()

	l := d.ListDevices()
	assert.Len(t, l, 0)
}

func TestAddDevice(t *testing.T) {
	d := devadmForTest()

	err := d.AddDevice(Device{})

	assert.Error(t, err)
	// check error type?
}

func TestGetDevice(t *testing.T) {
	d := devadmForTest()

	dev := d.GetDevice("foo")

	assert.Nil(t, dev)
}

func TestUpdateDevice(t *testing.T) {
	d := devadmForTest()

	err := d.UpdateDevice("foo", Device{})

	assert.Error(t, err)
	// check error type?
}

func TestNewDevAdm(t *testing.T) {

	d := NewDevAdm()

	assert.NotNil(t, d)
}
