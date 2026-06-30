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
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/upfluence/thrift/lib/go/test/gen/ignoreinitialismstest"
)

func TestIgnoreInitialismsFlagIsHonoured(t *testing.T) {
	st := reflect.TypeOf(ignoreinitialismstest.IgnoreInitialismsTest{})

	for _, tc := range []struct {
		name          string
		haveFieldName string
	}{
		{
			name:          "Id",
			haveFieldName: "Id",
		},
		{
			name:          "MyId",
			haveFieldName: "MyId",
		},
		{
			name:          "NumCpu",
			haveFieldName: "NumCpu",
		},
		{
			name:          "NumGpu",
			haveFieldName: "NumGpu",
		},
		{
			name:          "My_ID",
			haveFieldName: "My_ID",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, ok := st.FieldByName(tc.haveFieldName)

			assert.True(t, ok)
		})
	}
}
