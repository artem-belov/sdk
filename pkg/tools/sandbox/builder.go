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

package sandbox

import (
	"context"
	"net"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	registryapi "github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/endpoint"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/nsmgr"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/nsmgrproxy"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/authorize"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/connect"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/registry/chains/memory"
	"github.com/networkservicemesh/sdk/pkg/registry/chains/proxydns"
	"github.com/networkservicemesh/sdk/pkg/registry/common/dnsresolve"
	interpose_reg "github.com/networkservicemesh/sdk/pkg/registry/common/interpose"
	adapter_registry "github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/registry/core/chain"
	"github.com/networkservicemesh/sdk/pkg/tools/addressof"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	"github.com/networkservicemesh/sdk/pkg/tools/logger"
	"github.com/networkservicemesh/sdk/pkg/tools/opentracing"
	"github.com/networkservicemesh/sdk/pkg/tools/token"
)

const defaultContextTimeout = time.Second * 15

type NodeConfig struct {
	NsmgrCtx                   context.Context
	NsmgrGenerateTokenFunc     token.GeneratorFunc
	ForwarderCtx               context.Context
	ForwarderGenerateTokenFunc token.GeneratorFunc
}

// Builder implements builder pattern for building NSM Domain
type Builder struct {
	require             *require.Assertions
	resources           []context.CancelFunc
	nodesCount          int
	DNSDomainName       string
	Resolver            dnsresolve.Resolver
	supplyForwarder     SupplyForwarderFunc
	supplyNSMgr         SupplyNSMgrFunc
	supplyNSMgrProxy    SupplyNSMgrProxyFunc
	supplyRegistry      SupplyRegistryFunc
	supplyRegistryProxy SupplyRegistryProxyFunc
	generateTokenFunc   token.GeneratorFunc
	ctx                 context.Context
	nodesConfig         []*NodeConfig
}

// NewBuilder creates new SandboxBuilder
func NewBuilder(t *testing.T) *Builder {
	return &Builder{
		nodesCount:          1,
		require:             require.New(t),
		Resolver:            net.DefaultResolver,
		supplyNSMgr:         nsmgr.NewServer,
		supplyForwarder:     supplyDummyForwarder,
		DNSDomainName:       "cluster.local",
		supplyRegistry:      memory.NewServer,
		supplyRegistryProxy: proxydns.NewServer,
		supplyNSMgrProxy:    nsmgrproxy.NewServer,
		generateTokenFunc:   GenerateTestToken,
	}
}

// Build builds Domain and Supplier
func (b *Builder) Build() *Domain {
	initCtx := b.ctx
	if initCtx == nil {
		var cancel context.CancelFunc
		initCtx, cancel = context.WithTimeout(context.Background(), defaultContextTimeout)
		b.resources = append(b.resources, cancel)
	}
	ctx := logger.WithLog(initCtx)
	domain := &Domain{}
	domain.NSMgrProxy = b.newNSMgrProxy(ctx)
	if domain.NSMgrProxy == nil {
		domain.RegistryProxy = b.newRegistryProxy(ctx, &url.URL{})
	} else {
		domain.RegistryProxy = b.newRegistryProxy(ctx, domain.NSMgrProxy.URL)
	}
	if domain.RegistryProxy == nil {
		domain.Registry = b.newRegistry(ctx, nil)
	} else {
		domain.Registry = b.newRegistry(ctx, domain.RegistryProxy.URL)
	}
	for i := 0; i < b.nodesCount; i++ {
		nodeConfig := b.nodesConfig[i]
		var node = new(Node)

		// Setup NSMgr
		nsmgrCtx := nodeConfig.NsmgrCtx
		if nsmgrCtx != initCtx {
			var nsmgrCancel context.CancelFunc
			nsmgrCtx, nsmgrCancel = context.WithCancel(nsmgrCtx)
			b.resources = append(b.resources, nsmgrCancel)
			nsmgrCtx = logger.WithLog(nsmgrCtx)
		} else {
			nsmgrCtx = ctx
		}
		node.NSMgr = b.newNSMgr(nsmgrCtx, "127.0.0.1:0", domain.Registry.URL, nodeConfig.NsmgrGenerateTokenFunc)

		// Setup Forwarder
		forwarderCtx := nodeConfig.ForwarderCtx
		if forwarderCtx != initCtx {
			var forwarderCancel context.CancelFunc
			forwarderCtx, forwarderCancel = context.WithCancel(forwarderCtx)
			b.resources = append(b.resources, forwarderCancel)
			forwarderCtx = logger.WithLog(forwarderCtx)
		} else {
			forwarderCtx = ctx
		}
		forwarderName := "cross-nse-" + uuid.New().String()
		node.Forwarder = b.NewCrossConnectNSE(forwarderCtx, forwarderName, node.NSMgr, nodeConfig.ForwarderGenerateTokenFunc)

		domain.Nodes = append(domain.Nodes, node)
	}
	domain.resources, b.resources = b.resources, nil
	return domain
}

// SetContext sets default context for all chains
func (b *Builder) SetContext(ctx context.Context) *Builder {
	b.ctx = ctx
	b.SetCustomConfig([]*NodeConfig{})
	return b
}

// SetCustomConfig sets custom configuration for nodes
func (b *Builder) SetCustomConfig(config []*NodeConfig) *Builder {
	oldConfig := b.nodesConfig
	b.nodesConfig = nil

	for i := 0; i < b.nodesCount; i++ {
		nodeConfig := &NodeConfig{}
		if i < len(config) && config[i] != nil {
			*nodeConfig = *oldConfig[i]
		}

		customConfig := &NodeConfig{}
		if i < len(config) && config[i] != nil {
			*customConfig = *config[i]
		}

		if customConfig.NsmgrCtx != nil {
			nodeConfig.NsmgrCtx = customConfig.NsmgrCtx
		} else if nodeConfig.NsmgrCtx == nil {
			nodeConfig.NsmgrCtx = b.ctx
		}

		if customConfig.NsmgrGenerateTokenFunc != nil {
			nodeConfig.NsmgrGenerateTokenFunc = customConfig.NsmgrGenerateTokenFunc
		} else if nodeConfig.NsmgrGenerateTokenFunc == nil {
			nodeConfig.NsmgrGenerateTokenFunc = b.generateTokenFunc
		}

		if customConfig.ForwarderCtx != nil {
			nodeConfig.ForwarderCtx = customConfig.ForwarderCtx
		} else if nodeConfig.ForwarderCtx == nil {
			nodeConfig.ForwarderCtx = b.ctx
		}

		if customConfig.ForwarderGenerateTokenFunc != nil {
			nodeConfig.ForwarderGenerateTokenFunc = customConfig.ForwarderGenerateTokenFunc
		} else if nodeConfig.ForwarderGenerateTokenFunc == nil {
			nodeConfig.ForwarderGenerateTokenFunc = b.generateTokenFunc
		}

		b.nodesConfig = append(b.nodesConfig, nodeConfig)
	}
	return b
}

// SetNodesCount sets nodes count
func (b *Builder) SetNodesCount(nodesCount int) *Builder {
	b.nodesCount = nodesCount
	b.SetCustomConfig([]*NodeConfig{})
	return b
}

// SetDNSResolver sets DNS resolver for proxy registries
func (b *Builder) SetDNSResolver(d dnsresolve.Resolver) *Builder {
	b.Resolver = d
	return b
}

// SetTokenGenerateFunc sets function for the token generation
func (b *Builder) SetTokenGenerateFunc(f token.GeneratorFunc) *Builder {
	b.generateTokenFunc = f
	return b
}

// SetRegistryProxySupplier replaces default memory registry supplier to custom function
func (b *Builder) SetRegistryProxySupplier(f SupplyRegistryProxyFunc) *Builder {
	b.supplyRegistryProxy = f
	return b
}

// SetRegistrySupplier replaces default memory registry supplier to custom function
func (b *Builder) SetRegistrySupplier(f SupplyRegistryFunc) *Builder {
	b.supplyRegistry = f
	return b
}

// SetDNSDomainName sets DNS domain name for the building NSM domain
func (b *Builder) SetDNSDomainName(name string) *Builder {
	b.DNSDomainName = name
	return b
}

// SetForwarderSupplier replaces default dummy forwarder supplier to custom function
func (b *Builder) SetForwarderSupplier(f SupplyForwarderFunc) *Builder {
	b.supplyForwarder = f
	return b
}

// SetNSMgrProxySupplier replaces default nsmgr-proxy supplier to custom function
func (b *Builder) SetNSMgrProxySupplier(f SupplyNSMgrProxyFunc) *Builder {
	b.supplyNSMgrProxy = f
	return b
}

// SetNSMgrSupplier replaces default nsmgr supplier to custom function
func (b *Builder) SetNSMgrSupplier(f SupplyNSMgrFunc) *Builder {
	b.supplyNSMgr = f
	return b
}

func (b *Builder) dialContext(ctx context.Context, u *url.URL) *grpc.ClientConn {
	conn, err := grpc.DialContext(ctx, grpcutils.URLToTarget(u),
		grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)))
	b.resources = append(b.resources, func() {
		_ = conn.Close()
	})
	b.require.NoError(err, "Can not dial to", u)
	return conn
}

func (b *Builder) newNSMgrProxy(ctx context.Context) *EndpointEntry {
	if b.supplyRegistryProxy == nil {
		return nil
	}
	name := "nsmgr-proxy-" + uuid.New().String()
	mgr := b.supplyNSMgrProxy(ctx, name, b.generateTokenFunc, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithDefaultCallOptions(grpc.WaitForReady(true)))
	serveURL := &url.URL{Scheme: "tcp", Host: "127.0.0.1:0"}
	serve(ctx, serveURL, mgr.Register)
	logger.Log(ctx).Infof("%v listen on: %v", name, serveURL)
	return &EndpointEntry{
		Endpoint: mgr,
		URL:      serveURL,
	}
}

func (b *Builder) NewNSMgr(ctx context.Context, address string, registryURL *url.URL, generateTokenFunc token.GeneratorFunc) (entry *NSMgrEntry, resources []context.CancelFunc) {
	nsmgrCtx, nsmgrCancel := context.WithCancel(ctx)
	b.resources = append(b.resources, nsmgrCancel)

	entry = b.newNSMgr(logger.WithLog(nsmgrCtx), address, registryURL, generateTokenFunc)
	resources, b.resources = b.resources, nil
	return
}

func (b *Builder) newNSMgr(ctx context.Context, address string, registryURL *url.URL, generateTokenFunc token.GeneratorFunc) *NSMgrEntry {
	if b.supplyNSMgr == nil {
		panic("nodes without managers are not supported")
	}
	var registryCC *grpc.ClientConn
	if registryURL != nil {
		registryCC = b.dialContext(ctx, registryURL)
	}
	listener, err := net.Listen("tcp", address)
	b.require.NoError(err)
	serveURL := grpcutils.AddressToURL(listener.Addr())
	b.require.NoError(listener.Close())

	nsmgrReg := &registryapi.NetworkServiceEndpoint{
		Name: "nsmgr-" + uuid.New().String(),
		Url:  serveURL.String(),
	}

	mgr := b.supplyNSMgr(ctx, nsmgrReg, authorize.NewServer(), generateTokenFunc, registryCC, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithDefaultCallOptions(grpc.WaitForReady(true)))

	serve(ctx, serveURL, mgr.Register)
	logger.Log(ctx).Infof("%v listen on: %v", nsmgrReg.Name, serveURL)
	return &NSMgrEntry{
		URL:   serveURL,
		Nsmgr: mgr,
	}
}

func serve(ctx context.Context, u *url.URL, register func(server *grpc.Server)) {
	server := grpc.NewServer(opentracing.WithTracing()...)
	register(server)
	errCh := grpcutils.ListenAndServe(ctx, u, server)
	go func() {
		select {
		case <-ctx.Done():
			logger.Log(ctx).Infof("Stop serve: %v", u.String())
			return
		case err := <-errCh:
			if err != nil {
				logger.Log(ctx).Fatalf("An error during serve: %v", err.Error())
			}
		}
	}()
}

func (b *Builder) NewCrossConnectNSE(ctx context.Context, name string, nsmgrEntry *NSMgrEntry, generateTokenFunc token.GeneratorFunc) *EndpointEntry {
	registrationClient := chain.NewNetworkServiceEndpointRegistryClient(
		interpose_reg.NewNetworkServiceEndpointRegistryClient(),
		adapter_registry.NetworkServiceEndpointServerToClient(nsmgrEntry.NetworkServiceEndpointRegistryServer()),
	)
	return b.newCrossConnectNSE(logger.WithLog(ctx), name, nsmgrEntry.URL, registrationClient, generateTokenFunc)
}

func (b *Builder) newCrossConnectNSE(ctx context.Context, name string, connectTo *url.URL, forwarderRegistrationClient registryapi.NetworkServiceEndpointRegistryClient, generateTokenFunc token.GeneratorFunc) *EndpointEntry {
	if b.supplyForwarder == nil {
		panic("nodes without forwarder are not supported")
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	b.require.NoError(err)
	serveURL := grpcutils.AddressToURL(listener.Addr())
	b.require.NoError(listener.Close())

	regForwarder, err := forwarderRegistrationClient.Register(context.Background(), &registryapi.NetworkServiceEndpoint{
		Url:  serveURL.String(),
		Name: name,
	})
	b.require.NoError(err)

	crossNSE := b.supplyForwarder(ctx, regForwarder.Name, generateTokenFunc, connectTo, grpc.WithInsecure(), grpc.WithDefaultCallOptions(grpc.WaitForReady(true)))
	serve(ctx, serveURL, crossNSE.Register)
	logger.Log(ctx).Infof("%v listen on: %v", name, serveURL)
	return &EndpointEntry{
		Endpoint: crossNSE,
		URL:      serveURL,
	}
}

func (b *Builder) newRegistryProxy(ctx context.Context, nsmgrProxyURL *url.URL) *RegistryEntry {
	if b.supplyRegistryProxy == nil {
		return nil
	}
	result := b.supplyRegistryProxy(ctx, b.Resolver, b.DNSDomainName, nsmgrProxyURL, grpc.WithInsecure(), grpc.WithBlock())
	serveURL := &url.URL{Scheme: "tcp", Host: "127.0.0.1:0"}
	serve(ctx, serveURL, result.Register)
	logger.Log(ctx).Infof("registry-proxy-dns listen on: %v", serveURL)
	return &RegistryEntry{
		URL:      serveURL,
		Registry: result,
	}
}

func (b *Builder) newRegistry(ctx context.Context, proxyRegistryURL *url.URL) *RegistryEntry {
	if b.supplyRegistry == nil {
		return nil
	}
	result := b.supplyRegistry(ctx, proxyRegistryURL, grpc.WithInsecure(), grpc.WithBlock())
	serveURL := &url.URL{Scheme: "tcp", Host: "127.0.0.1:0"}
	serve(ctx, serveURL, result.Register)
	logger.Log(ctx).Infof("Registry listen on: %v", serveURL)
	return &RegistryEntry{
		URL:      serveURL,
		Registry: result,
	}
}

func supplyDummyForwarder(ctx context.Context, name string, generateToken token.GeneratorFunc, connectTo *url.URL, dialOptions ...grpc.DialOption) endpoint.Endpoint {
	type forwarderServer struct {
		endpoint.Endpoint
	}
	rv := &forwarderServer{}
	rv.Endpoint = endpoint.NewServer(ctx,
		name,
		authorize.NewServer(),
		generateToken,
		// Statically set the url we use to the unix file socket for the NSMgr
		clienturl.NewServer(connectTo),
		connect.NewServer(ctx,
			client.NewCrossConnectClientFactory(
				name,
				// What to call onHeal
				addressof.NetworkServiceClient(adapters.NewServerToClient(rv)),
				generateToken),
			dialOptions...,
		),
	)
	return rv
}
