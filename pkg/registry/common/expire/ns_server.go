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

package expire

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/extend"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
)

type nsServer struct {
	nseClient  registry.NetworkServiceEndpointRegistryClient
	monitorErr error
	nsCounts   intMap
	contexts   contextMap
	once       sync.Once
	chainCtx   context.Context
}

func (n *nsServer) checkUpdates() {
	stream, err := n.nseClient.Find(n.chainCtx, &registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{},
		Watch:                  true,
	})
	if err != nil {
		n.monitorErr = err
		return
	}

	var timers = make(map[string]*time.Timer)

	unregister := func(ns string) {
		if ctx, ok := n.contexts.Load(ns); ok {
			_, _ = n.Unregister(ctx, &registry.NetworkService{Name: ns})
		}
	}

	for nse := range registry.ReadNetworkServiceEndpointChannel(stream) {
		for i := range nse.NetworkServiceNames {
			ns := nse.NetworkServiceNames[i]

			timer, cotains := timers[nse.Name]

			var value int32
			refCount, _ := n.nsCounts.LoadOrStore(ns, &value)

			if cotains && atomic.LoadInt32(refCount) != 0 {
				if nse.ExpirationTime != nil && nse.ExpirationTime.Seconds < 0 {
					if timer.Stop() && atomic.AddInt32(refCount, -1) <= 0 {
						unregister(ns)
					}
					delete(timers, nse.Name)
					continue
				}
				timer.Stop()
				timer.Reset(time.Until(nse.ExpirationTime.AsTime().Local()))
				continue
			}

			atomic.AddInt32(refCount, 1)
			timers[nse.Name] = time.AfterFunc(time.Until(nse.ExpirationTime.AsTime().Local()), func() {
				if atomic.AddInt32(refCount, -1) <= 0 {
					unregister(ns)
				}
			})
		}
	}
}

func (n *nsServer) Register(ctx context.Context, request *registry.NetworkService) (*registry.NetworkService, error) {
	n.once.Do(func() {
		go func() {
			for n.monitorErr == nil && n.chainCtx.Err() == nil {
				n.checkUpdates()
			}
		}()
	})
	if n.monitorErr != nil {
		return nil, n.monitorErr
	}
	n.contexts.Store(request.Name, extend.WithValuesFromContext(n.chainCtx, ctx))
	return next.NetworkServiceRegistryServer(ctx).Register(ctx, request)
}

func (n *nsServer) Find(query *registry.NetworkServiceQuery, s registry.NetworkServiceRegistry_FindServer) error {
	if n.monitorErr != nil {
		return n.monitorErr
	}
	return next.NetworkServiceRegistryServer(s.Context()).Find(query, s)
}

func (n *nsServer) Unregister(ctx context.Context, request *registry.NetworkService) (*empty.Empty, error) {
	if n.monitorErr != nil {
		return nil, n.monitorErr
	}
	if v, ok := n.nsCounts.Load(request.Name); ok {
		if atomic.LoadInt32(v) > 0 {
			return nil, errors.New("cannot delete network service: resource already in use")
		}
	}
	n.contexts.Delete(request.Name)
	return next.NetworkServiceRegistryServer(ctx).Unregister(ctx, request)
}

// NewNetworkServiceServer wraps passed NetworkServiceRegistryServer and monitor NetworkServiceEndpoints via passed NetworkServiceEndpointRegistryClient
func NewNetworkServiceServer(ctx context.Context, nseClient registry.NetworkServiceEndpointRegistryClient) registry.NetworkServiceRegistryServer {
	return &nsServer{nseClient: nseClient, chainCtx: ctx}
}
