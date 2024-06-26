// Copyright 2013 Beego Authors
// Copyright 2014 The Macaron Authors
//
// Licensed under the Apache License, Version 2.0 (the "License"): you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package session

import (
	"bytes"
	"crypto/rand"
	"encoding/gob"
	"encoding/json"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/unknwon/com"
)

func init() {
	gob.Register([]interface{}{})
	gob.Register(map[int]interface{}{})
	gob.Register(map[string]interface{}{})
	gob.Register(map[interface{}]interface{}{})
	gob.Register(map[string]string{})
	gob.Register(map[int]string{})
	gob.Register(map[int]int{})
	gob.Register(map[int]int64{})
	gob.Register(SessInfo{})
	gob.Register(time.Time{})
	gob.Register(GeoLocation{})
}

func EncodeGob(obj map[interface{}]interface{}) ([]byte, error) {
	for _, v := range obj {
		gob.Register(v)
	}
	buf := bytes.NewBuffer(nil)
	err := gob.NewEncoder(buf).Encode(obj)
	return buf.Bytes(), err
}

func DecodeGob(encoded []byte) (out map[interface{}]interface{}, err error) {
	buf := bytes.NewBuffer(encoded)
	err = gob.NewDecoder(buf).Decode(&out)
	return out, err
}

func EncodedUserSessionHub(obj UserSessionHub) ([]byte, error) {
	return json.Marshal(obj)
}

func DecodeUserSessionHub(encoded []byte) (UserSessionHub, error) {
	hubData := new(UserSessionHub)
	err := json.Unmarshal(encoded, hubData)
	if err != nil {
		return UserSessionHub{}, err
	}

	return *hubData, nil
}

// NOTE: A local copy in case of underlying package change
var alphanum = []byte("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz")

// generateRandomKey creates a random key with the given strength.
func generateRandomKey(strength int) []byte {
	k := make([]byte, strength)
	if n, err := io.ReadFull(rand.Reader, k); n != strength || err != nil {
		return com.RandomCreateBytes(strength, alphanum...)
	}
	return k
}

func IsZerothReplica() bool {
	hostname, found := os.LookupEnv("HOSTNAME")
	if !found {
		return false
	}

	parts := strings.Split(hostname, "-")
	sn, err := strconv.ParseInt(parts[len(parts)-1], 10, 64)
	if err != nil {
		return false
	}

	return sn == 0
}
