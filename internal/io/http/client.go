// Copyright 2023-2024 EMQ Technologies Co., Ltd.
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

package http

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/lf-edge/ekuiper/contract/v2/api"
	"github.com/lf-edge/ekuiper/v2/internal/compressor"
	"github.com/lf-edge/ekuiper/v2/internal/conf"
	"github.com/lf-edge/ekuiper/v2/internal/pkg/httpx"
	"github.com/lf-edge/ekuiper/v2/pkg/cast"
	"github.com/lf-edge/ekuiper/v2/pkg/cert"
	"github.com/lf-edge/ekuiper/v2/pkg/message"
)

// ClientConf is the configuration for http client
// It is shared by httppull source and rest sink to configure their http client
type ClientConf struct {
	config       *RawConf
	client       *http.Client
	compressor   message.Compressor   // compressor used to payload compression when specifies compressAlgorithm
	decompressor message.Decompressor // decompressor used to payload decompression when specifies compressAlgorithm
}

type RawConf struct {
	Url      string            `json:"url"`
	Method   string            `json:"method"`
	Body     string            `json:"body"`
	BodyType string            `json:"bodyType"`
	Headers  map[string]string `json:"headers"`
	Timeout  int               `json:"timeout"`
	// Could be code or body
	ResponseType string `json:"responseType"`
	Compression  string `json:"compression"` // Compression specifies the algorithms used to payload compression
}

const (
	DefaultTimeout = 5000
)

type bodyResp struct {
	Code int `json:"code"`
}

var bodyTypeMap = map[string]string{"none": "", "text": "text/plain", "json": "application/json", "html": "text/html", "xml": "application/xml", "javascript": "application/javascript", "form": ""}

// newTransport allows EdgeX Foundry, protected by OpenZiti to override and obtain a transport
// protected by OpenZiti's zero trust connectivity. See client_edgex.go where this function is
// set in an init() call
var newTransport = getTransport

func getTransport(tlscfg *tls.Config, logger *logrus.Logger) *http.Transport {
	return &http.Transport{
		TLSClientConfig: tlscfg,
	}
}

func (cc *ClientConf) InitConf(device string, props map[string]interface{}) error {
	c := &RawConf{
		Url:          "http://localhost",
		Method:       http.MethodGet,
		Timeout:      DefaultTimeout,
		ResponseType: "code",
	}

	if err := cast.MapToStruct(props, c); err != nil {
		return fmt.Errorf("fail to parse the properties: %v", err)
	}
	if c.Url == "" {
		return fmt.Errorf("url is required")
	}
	c.Url = c.Url + device
	switch strings.ToUpper(c.Method) {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete:
		c.Method = strings.ToUpper(c.Method)
	default:
		return fmt.Errorf("Not supported HTTP method %s.", c.Method)
	}
	if c.Timeout < 0 {
		return fmt.Errorf("timeout must be greater than or equal to 0")
	}
	// Set default body type if not set
	if c.BodyType == "" {
		switch c.Method {
		case http.MethodGet, http.MethodHead:
			c.BodyType = "none"
		default:
			c.BodyType = "json"
		}
	}
	if _, ok2 := bodyTypeMap[strings.ToLower(c.BodyType)]; ok2 {
		c.BodyType = strings.ToLower(c.BodyType)
	} else {
		return fmt.Errorf("Not valid body type value %v.", c.BodyType)
	}
	switch c.ResponseType {
	case "code", "body":
		// correct
	default:
		return fmt.Errorf("Not valid response type value %v.", c.ResponseType)
	}
	err := httpx.IsHttpUrl(c.Url)
	if err != nil {
		return err
	}
	tlscfg, err := cert.GenTLSConfig(props, "http")
	if err != nil {
		return err
	}
	tr := newTransport(tlscfg, conf.Log)
	cc.client = &http.Client{
		Transport: tr,
		Timeout:   time.Duration(c.Timeout) * time.Millisecond,
	}
	cc.config = c
	// that means payload need compression and decompression, so we need initialize compressor and decompressor
	if c.Compression != "" {
		cc.compressor, err = compressor.GetCompressor(c.Compression)
		if err != nil {
			return fmt.Errorf("init payload compressor failed, %w", err)
		}

		cc.decompressor, err = compressor.GetDecompressor(c.Compression)
		if err != nil {
			return fmt.Errorf("init payload decompressor failed, %w", err)
		}
	}
	return nil
}

const (
	BODY_ERR = "response body error"
	CODE_ERR = "response code error"
)

// responseBodyDecompress used to decompress the specified response body bytes, decompression algorithm indicated
// by response header 'Content-Encoding' value.
func (cc *ClientConf) responseBodyDecompress(ctx api.StreamContext, resp *http.Response, body []byte) ([]byte, error) {
	var err error
	// we need check response header key Content-Encoding is exist, if not that means remote server probably not support
	// configured compression algorithm and we should throw error.
	if resp.Header.Get("Content-Encoding") == "" {
		ctx.GetLogger().Warnf("Cannot find header with key 'Content-Encoding' when trying to detect response content encoding and decompress it, probably remote server does not support configured algorithm %q", cc.config.Compression)
		return nil, fmt.Errorf("try to detect and decompress payload has error, cannot find header with key 'Content-Encoding' in response")
	}
	body, err = cc.decompressor.Decompress(body)
	if err != nil {
		return nil, fmt.Errorf("try to decompress payload failed, %w", err)
	}
	return body, nil
}

// parse the response status. For rest sink, it will not return the body by default if not need to debug
func (cc *ClientConf) parseResponse(ctx api.StreamContext, resp *http.Response) ([]map[string]interface{}, []byte, error) {
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, nil, fmt.Errorf("%s: %d", CODE_ERR, resp.StatusCode)
	}

	c, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %v", BODY_ERR, err)
	}

	defer func() {
		resp.Body.Close()
	}()

	switch cc.config.ResponseType {
	case "code":
		if cc.config.Compression != "" {
			if c, err = cc.responseBodyDecompress(ctx, resp, c); err != nil {
				return nil, nil, fmt.Errorf("try to decompress payload failed, %w", err)
			}
		}
		m, e := decode(c)
		if e != nil {
			return nil, c, fmt.Errorf("%s: decode fail for %v", BODY_ERR, e)
		}
		return m, c, e
	case "body":
		if cc.config.Compression != "" {
			if c, err = cc.responseBodyDecompress(ctx, resp, c); err != nil {
				return nil, nil, fmt.Errorf("try to decompress payload failed, %w", err)
			}
		}
		payloads, err := decode(c)
		if err != nil {
			return nil, c, fmt.Errorf("%s: decode fail for %v", BODY_ERR, err)
		}
		for _, payload := range payloads {
			ro := &bodyResp{}
			err = cast.MapToStruct(payload, ro)
			if err != nil {
				return nil, c, fmt.Errorf("%s: decode fail for %v", BODY_ERR, err)
			}
			if ro.Code < 200 || ro.Code > 299 {
				return nil, c, fmt.Errorf("%s: %d", CODE_ERR, ro.Code)
			}
		}
		return payloads, c, nil
	default:
		return nil, c, fmt.Errorf("%s: unsupported response type %s", BODY_ERR, cc.config.ResponseType)
	}
}

func decode(data []byte) ([]map[string]interface{}, error) {
	var r1 interface{}
	err := json.Unmarshal(data, &r1)
	if err != nil {
		return nil, err
	}
	switch rt := r1.(type) {
	case map[string]interface{}:
		return []map[string]interface{}{rt}, nil
	case []map[string]interface{}:
		return rt, nil
	case []interface{}:
		r2 := make([]map[string]interface{}, len(rt))
		for i, m := range rt {
			if rm, ok := m.(map[string]interface{}); ok {
				r2[i] = rm
			} else {
				return nil, fmt.Errorf("only map[string]interface{} and []map[string]interface{} is supported")
			}
		}
		return r2, nil
	}
	return nil, fmt.Errorf("only map[string]interface{} and []map[string]interface{} is supported")
}
