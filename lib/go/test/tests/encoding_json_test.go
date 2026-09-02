/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements. See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership. The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License. You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied. See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

package tests

import (
	"encoding/json"
	"testing"

	"github.com/upfluence/thrift/lib/go/test/gen/thrifttest"
)

func TestEnumString(t *testing.T) {
	for _, tt := range []struct {
		name          string
		have          thrifttest.Numberz
		want          string
		wantHumanized string
	}{
		{
			name:          "known value",
			have:          thrifttest.Numberz_ONE,
			want:          "Numberz_ONE",
			wantHumanized: "ONE",
		},
		{
			name:          "unknown value",
			have:          thrifttest.Numberz(0),
			want:          "Numberz_<UNSET>",
			wantHumanized: "<UNSET>",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.have.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}

			if got := tt.have.HumanizedString(); got != tt.wantHumanized {
				t.Errorf("HumanizedString() = %q, want %q", got, tt.wantHumanized)
			}
		})
	}
}

func TestJSONMarshalUnmarshal(t *testing.T) {
	s1 := thrifttest.StructB{
		Aa: &thrifttest.StructA{S: "Aa"},
		Ab: &thrifttest.StructA{S: "Ab"},
	}

	b, err := json.Marshal(s1)
	if err != nil {
		t.Fatalf("Unexpected error from json.Marshal: %s", err)
	}

	s2 := thrifttest.StructB{}
	err = json.Unmarshal(b, &s2)
	if err != nil {
		t.Fatalf("Unexpected error from json.Unmarshal: %s", err)
	}

	if *s1.Aa != *s2.Aa || *s1.Ab != *s2.Ab {
		t.Logf("s1 = %+v", s1)
		t.Logf("s2 = %+v", s2)
		t.Errorf("json: Unmarshal(Marshal(s)) != s")
	}
}
