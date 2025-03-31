// Copyright 2025 EMQ Technologies Co., Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"encoding/json"
	"net/http"

	"github.com/lf-edge/ekuiper/v2/internal/keyedstate"
)

func keyedStateHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		req := &keyedStateRequest{}
		defer r.Body.Close()
		err := json.NewDecoder(r.Body).Decode(req)
		if err != nil {
			handleError(w, err, "Invalid body: Error decoding req", logger)
			return
		}
		if err := keyedstate.SetKeyedState(req.Key, req.Value); err != nil {
			handleError(w, err, "", logger)
			return
		}
		w.Write([]byte("success"))
	case "GET":
		k := r.URL.Query().Get("key")
		v, err := keyedstate.GetKeyedState(k)
		if err != nil {
			handleError(w, err, "", logger)
			return
		}
		jsonResponse(keyedStateRequest{Key: k, Value: v}, w, logger)
	}
}

type keyedStateRequest struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}
