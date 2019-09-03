// Copyright 2019 Google Cloud Platform Proxy Authors
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

package configgenerator

import (
	"reflect"
	"strings"
	"testing"

	"cloudesf.googlesource.com/gcpproxy/src/go/configinfo"
	"github.com/golang/protobuf/ptypes/wrappers"

	routepb "github.com/envoyproxy/data-plane-api/api/route"
)

func TestMakeRouteConfigForCors(t *testing.T) {
	testData := []struct {
		desc string
		// Test parameters, in the order of "cors_preset", "cors_allow_origin"
		// "cors_allow_origin_regex", "cors_allow_methods", "cors_allow_headers"
		// "cors_expose_headers"
		params           []string
		allowCredentials bool
		wantedError      string
		wantRoute        *routepb.CorsPolicy
	}{
		{
			desc:      "No Cors",
			wantRoute: nil,
		},
		{
			desc:        "Incorrect configured basic Cors",
			params:      []string{"basic", "", `^https?://.+\\.example\\.com$`, "", "", ""},
			wantedError: "cors_allow_origin cannot be empty when cors_preset=basic",
		},
		{
			desc:        "Incorrect configured  Cors",
			params:      []string{"", "", "", "GET", "", ""},
			wantedError: "cors_preset must be set in order to enable CORS support",
		},
		{
			desc:        "Incorrect configured regex Cors",
			params:      []string{"cors_with_regexs", "", `^https?://.+\\.example\\.com$`, "", "", ""},
			wantedError: `cors_preset must be either "basic" or "cors_with_regex"`,
		},
		{
			desc:   "Correct configured basic Cors, with allow methods",
			params: []string{"basic", "http://example.com", "", "GET,POST,PUT,OPTIONS", "", ""},
			wantRoute: &routepb.CorsPolicy{
				AllowOrigin:      []string{"http://example.com"},
				AllowMethods:     "GET,POST,PUT,OPTIONS",
				AllowCredentials: &wrappers.BoolValue{Value: false},
			},
		},
		{
			desc:   "Correct configured regex Cors, with allow headers",
			params: []string{"cors_with_regex", "", `^https?://.+\\.example\\.com$`, "", "Origin,Content-Type,Accept", ""},
			wantRoute: &routepb.CorsPolicy{
				AllowOriginRegex: []string{`^https?://.+\\.example\\.com$`},
				AllowHeaders:     "Origin,Content-Type,Accept",
				AllowCredentials: &wrappers.BoolValue{Value: false},
			},
		},
		{
			desc:             "Correct configured regex Cors, with expose headers",
			params:           []string{"cors_with_regex", "", `^https?://.+\\.example\\.com$`, "", "", "Content-Length"},
			allowCredentials: true,
			wantRoute: &routepb.CorsPolicy{
				AllowOriginRegex: []string{`^https?://.+\\.example\\.com$`},
				ExposeHeaders:    "Content-Length",
				AllowCredentials: &wrappers.BoolValue{Value: true},
			},
		},
	}

	for _, tc := range testData {
		options := configinfo.DefaultEnvoyConfigOptions()
		if tc.params != nil {
			options.CorsPreset = tc.params[0]
			options.CorsAllowOrigin = tc.params[1]
			options.CorsAllowOriginRegex = tc.params[2]
			options.CorsAllowMethods = tc.params[3]
			options.CorsAllowHeaders = tc.params[4]
			options.CorsExposeHeaders = tc.params[5]
		}
		options.CorsAllowCredentials = tc.allowCredentials

		gotRoute, err := MakeRouteConfig(&configinfo.ServiceInfo{
			Name:    "test-api",
			Options: options,
		})
		if tc.wantedError != "" {
			if err == nil || !strings.Contains(err.Error(), tc.wantedError) {
				t.Errorf("Test (%s): expected err: %v, got: %v", tc.desc, tc.wantedError, err)
			}
			continue
		}

		gotHost := gotRoute.GetVirtualHosts()
		if len(gotHost) != 1 {
			t.Errorf("Test (%s): got expected number of virtual host", tc.desc)
		}
		gotCors := gotHost[0].GetCors()
		if !reflect.DeepEqual(gotCors, tc.wantRoute) {
			t.Errorf("Test (%s): makeRouteConfig failed, got Cors: %s, want: %s", tc.desc, gotCors, tc.wantRoute)
		}
	}
}
