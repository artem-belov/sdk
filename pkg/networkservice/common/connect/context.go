// Copyright (c) 2020 Cisco Systems, Inc.
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

package connect

import (
	"context"
)

const (
	serverContextKey contextKeyType = "ServerChainContext"
)

type contextKeyType string

func withServerContext(parent context.Context, ctx context.Context) context.Context {
	if parent == nil {
		panic("cannot create context from nil parent")
	}
	return context.WithValue(parent, serverContextKey, ctx)
}

// ServerContext - Returns the Server chain context
func ServerContext(ctx context.Context) context.Context {
	if rv, ok := ctx.Value(serverContextKey).(context.Context); ok {
		return rv
	}
	return nil
}
