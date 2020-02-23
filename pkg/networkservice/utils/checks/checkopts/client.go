// Copyright (c) 2020 Cisco and/or its affiliates.
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

package checkopts

import (
	"context"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type checkOptsClient struct {
	*testing.T
	opts []grpc.CallOption
}

// NewClient - returns a NetworkServiceClient that checks for the specified opts... being passed into it
//             t - *testing.T for checks
//             opts... grpc.CallOptions expected at the end of the list of opts... passed to Request/Close
func NewClient(t *testing.T, opts ...grpc.CallOption) networkservice.NetworkServiceClient {
	if len(opts) == 0 {
		opts = append(opts, grpc.EmptyCallOption{})
	}
	return &checkOptsClient{
		T:    t,
		opts: opts,
	}
}

func (c *checkOptsClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	require.GreaterOrEqual(c.T, len(opts), len(c.opts))
	for i, opt := range c.opts {
		assert.Equal(c.T, opt, opts[i+len(c.opts)-len(opts)])
	}
	return next.Client(ctx).Request(ctx, request, opts...)
}

func (c *checkOptsClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	require.GreaterOrEqual(c.T, len(opts), len(c.opts))
	for i, opt := range c.opts {
		assert.Equal(c.T, opt, opts[i+len(c.opts)-len(opts)])
	}
	return next.Client(ctx).Close(ctx, conn, opts...)
}