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
package utils

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	test "github.com/mendersoftware/deviceauth/utils/testing"
)

func TestParsePubKey(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		pubkey string
		err    error
	}{
		"ok": {
			pubkey: test.LoadPubKeyStr("testdata/public.pem", t),
		},
		"error, bad pem block": {
			pubkey: test.LoadPubKeyStr("testdata/public_bad_pem.pem", t),
			err:    errors.New("cannot decode public key"),
		},
		"error, pem ok, but bad key content": {
			pubkey: test.LoadPubKeyStr("testdata/public_bad_key_content.pem", t),
			err:    errors.New("cannot decode public key"),
		},
	}

	for i := range testCases {
		tc := testCases[i]
		t.Run(fmt.Sprintf("tc %s", i), func(t *testing.T) {
			t.Parallel()

			key, err := ParsePubKey(tc.pubkey)

			if tc.err != nil {
				assert.EqualError(t, err, tc.err.Error())
				assert.Nil(t, key)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, key)
			}
		})
	}
}

func TestSerializePubKey(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		keyPath string
		out     string
		err     error
	}{
		"ok": {
			keyPath: "testdata/public.pem",
			out: `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAzogVU7RGDilbsoUt/DdH
VJvcepl0A5+xzGQ50cq1VE/Dyyy8Zp0jzRXCnnu9nu395mAFSZGotZVr+sWEpO3c
yC3VmXdBZmXmQdZqbdD/GuixJOYfqta2ytbIUPRXFN7/I7sgzxnXWBYXYmObYvdP
okP0mQanY+WKxp7Q16pt1RoqoAd0kmV39g13rFl35muSHbSBoAW3GBF3gO+mF5Ty
1ddp/XcgLOsmvNNjY+2HOD5F/RX0fs07mWnbD7x+xz7KEKjF+H7ZpkqCwmwCXaf0
iyYyh1852rti3Afw4mDxuVSD7sd9ggvYMc0QHIpQNkD4YWOhNiE1AB0zH57VbUYG
UwIDAQAB
-----END PUBLIC KEY-----
`,
		},
	}

	for i := range testCases {
		tc := testCases[i]
		t.Run(fmt.Sprintf("tc %s", i), func(t *testing.T) {
			t.Parallel()

			pubkey := test.LoadPubKeyStr(tc.keyPath, t)

			block, _ := pem.Decode([]byte(pubkey))
			assert.NotNil(t, block)
			assert.Equal(t, PubKeyBlockType, block.Type)

			key, err := x509.ParsePKIXPublicKey(block.Bytes)
			assert.NoError(t, err)

			out, err := SerializePubKey(key)

			if tc.err != nil {
				assert.EqualError(t, err, tc.err.Error())
				assert.Equal(t, "", out)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.out, out)
			}
		})
	}
}
