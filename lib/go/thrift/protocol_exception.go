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

package thrift

import (
	"encoding/base64"

	"github.com/upfluence/errors"
)

// Thrift Protocol exception
type TProtocolException interface {
	TException
	TypeId() int
}

const (
	UNKNOWN_PROTOCOL_EXCEPTION = 0
	INVALID_DATA               = 1
	NEGATIVE_SIZE              = 2
	SIZE_LIMIT                 = 3
	BAD_VERSION                = 4
	NOT_IMPLEMENTED            = 5
	DEPTH_LIMIT                = 6
)

type tProtocolException struct {
	typeId int
	err    error
}

func (p *tProtocolException) TypeId() int {
	return p.typeId
}

func (p *tProtocolException) String() string {
	return p.Error()
}

func (p *tProtocolException) Error() string {
	return p.err.Error()
}

func (p *tProtocolException) Unwrap() error {
	return p.err
}

func NewTProtocolException(err error) TProtocolException {
	if err == nil {
		return nil
	}

	var (
		perr TProtocolException
		berr base64.CorruptInputError

		typeID = UNKNOWN_PROTOCOL_EXCEPTION
	)

	switch {
	case errors.As(err, &perr):
		return perr
	case errors.As(err, &berr):
		typeID = INVALID_DATA
	}

	return NewTProtocolExceptionWithType(typeID, err)
}

func NewTProtocolExceptionWithType(errType int, err error) TProtocolException {
	if err == nil {
		return nil
	}

	return &tProtocolException{typeId: errType, err: errors.WithStack(err)}
}
