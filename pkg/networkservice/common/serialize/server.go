// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package serialize provides chain elements for serial Request, Close event processing
package serialize

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/multiexecutor"
)

type serializeServer struct {
	executor multiexecutor.MultiExecutor
}

// NewServer returns a new serialize server chain element
func NewServer() networkservice.NetworkServiceServer {
	return new(serializeServer)
}

func (s *serializeServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (conn *networkservice.Connection, err error) {
	connID := request.GetConnection().GetId()
	<-s.executor.AsyncExec(connID, func() {
		requestCtx := WithExecutor(ctx, executorFunc(s.executor.Executor(connID)))
		conn, err = next.Server(ctx).Request(requestCtx, request)
	})
	return conn, err
}

func (s *serializeServer) Close(ctx context.Context, conn *networkservice.Connection) (_ *empty.Empty, err error) {
	<-s.executor.AsyncExec(conn.GetId(), func() {
		requestCtx := WithExecutor(ctx, executorFunc(s.executor.Executor(conn.GetId())))
		_, err = next.Server(ctx).Close(requestCtx, conn)
	})
	return new(empty.Empty), err
}
