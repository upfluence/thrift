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
	"github.com/upfluence/errors"
	"github.com/upfluence/errors/base"
)

// Generic Thrift exception
type TException interface {
	error
}

func prependWrappedError(prepend string, err error) error {
	var baseErr = base.UnwrapOnce(err)

	if baseErr == nil {
		baseErr = errors.New(err.Error())
	}

	return errors.Wrap(baseErr, prepend)
}

// Prepends additional information to an error without losing the Thrift exception interface
func PrependError(prepend string, err error) error {
	var (
		terr TTransportException
		perr TProtocolException
		aerr TApplicationException
	)

	switch {
	case errors.As(err, &terr):
		return NewTTransportExceptionWithType(terr.TypeId(), prependWrappedError(prepend, err))
	case errors.As(err, &perr):
		return NewTProtocolExceptionWithType(perr.TypeId(), prependWrappedError(prepend, err))
	case errors.As(err, &aerr):
		return NewTApplicationExceptionFromError(aerr.TypeId(), prependWrappedError(prepend, err))
	default:
		return errors.Wrap(err, prepend)
	}
}

func Cause(err error) error {
	return errors.Cause(err)
}
