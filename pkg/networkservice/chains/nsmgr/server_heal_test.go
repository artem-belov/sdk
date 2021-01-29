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

// Package nsmgr_test define a tests for NSMGR chain element.
package nsmgr_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	kernelmech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/sdk/pkg/tools/sandbox"
)

func TestNSMGR_HealEndpoint(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)

	defer cancel()
	domain := sandbox.NewBuilder(t).
		SetNodesCount(2).
		SetRegistryProxySupplier(nil).
		SetContext(ctx).
		Build()
	defer domain.Cleanup()

	expireTime, err := ptypes.TimestampProto(time.Now().Add(time.Second))
	require.NoError(t, err)

	nseReg := &registry.NetworkServiceEndpoint{
		Name:                "final-endpoint",
		NetworkServiceNames: []string{"my-service-remote"},
		ExpirationTime:      expireTime,
	}

	counter := &counterServer{}
	nseCtx, nseCtxCancel := context.WithTimeout(context.Background(), time.Second)
	defer nseCtxCancel()
	_, err = sandbox.NewEndpoint(nseCtx, nseReg, sandbox.GenerateExpiringToken(time.Second), domain.Nodes[0].NSMgr, counter)
	require.NoError(t, err)

	request := &networkservice.NetworkServiceRequest{
		MechanismPreferences: []*networkservice.Mechanism{
			{Cls: cls.LOCAL, Type: kernelmech.MECHANISM},
		},
		Connection: &networkservice.Connection{
			Id:             "1",
			NetworkService: "my-service-remote",
			Context:        &networkservice.ConnectionContext{},
		},
	}

	nsc := sandbox.NewClient(ctx, sandbox.GenerateTestToken, domain.Nodes[1].NSMgr.URL)

	conn, err := nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, 1, counter.UniqueRequests())
	require.Equal(t, 8, len(conn.Path.PathSegments))

	nseReg2 := &registry.NetworkServiceEndpoint{
		Name:                "final-endpoint2",
		NetworkServiceNames: []string{"my-service-remote"},
	}
	_, err = sandbox.NewEndpoint(ctx, nseReg2, sandbox.GenerateTestToken, domain.Nodes[0].NSMgr, counter)
	require.NoError(t, err)

	// Wait NSE expired and reconnecting to the new NSE
	<-time.After(5 * time.Second)

	// Close.
	e, err := nsc.Close(ctx, conn)
	require.NoError(t, err)
	require.NotNil(t, e)
	require.Equal(t, 2, counter.UniqueRequests())
	require.Equal(t, 1, counter.UniqueCloses())
}

func TestNSMGR_HealLocalForwarder(t *testing.T) {
	forwarderCtx, forwarderCtxCancel := context.WithTimeout(context.Background(), time.Second)
	defer forwarderCtxCancel()

	customConfig := []*sandbox.NodeConfig{
		nil,
		{
			ForwarderCtx:               forwarderCtx,
			ForwarderGenerateTokenFunc: sandbox.GenerateExpiringToken(time.Second),
		},
	}

	testNSMGR_HealForwarder(t, 1, customConfig)
}

func TestNSMGR_HealRemoteForwarder(t *testing.T) {
	forwarderCtx, forwarderCtxCancel := context.WithTimeout(context.Background(), time.Second)
	defer forwarderCtxCancel()

	customConfig := []*sandbox.NodeConfig{
		{
			ForwarderCtx:               forwarderCtx,
			ForwarderGenerateTokenFunc: sandbox.GenerateExpiringToken(time.Second),
		},
	}

	testNSMGR_HealForwarder(t, 0, customConfig)
}

func testNSMGR_HealForwarder(t *testing.T, nodeNum int, customConfig []*sandbox.NodeConfig) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	builder := sandbox.NewBuilder(t)
	domain := builder.
		SetNodesCount(2).
		SetRegistryProxySupplier(nil).
		SetContext(ctx).
		SetCustomConfig(customConfig).
		Build()
	defer domain.Cleanup()

	nseReg := &registry.NetworkServiceEndpoint{
		Name:                "final-endpoint",
		NetworkServiceNames: []string{"my-service-remote"},
	}

	counter := &counterServer{}
	_, err := sandbox.NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken, domain.Nodes[0].NSMgr, counter)
	require.NoError(t, err)

	request := &networkservice.NetworkServiceRequest{
		MechanismPreferences: []*networkservice.Mechanism{
			{Cls: cls.LOCAL, Type: kernelmech.MECHANISM},
		},
		Connection: &networkservice.Connection{
			Id:             "1",
			NetworkService: "my-service-remote",
			Context:        &networkservice.ConnectionContext{},
		},
	}

	nsc := sandbox.NewClient(ctx, sandbox.GenerateTestToken, domain.Nodes[1].NSMgr.URL)

	conn, err := nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, 1, counter.UniqueRequests())
	require.Equal(t, 8, len(conn.Path.PathSegments))

	forwarderName := "cross-nse-restored"
	builder.NewCrossConnectNSE(ctx, forwarderName, domain.Nodes[nodeNum], sandbox.GenerateTestToken)

	// Wait Cross NSE expired and reconnecting through the new Cross NSE
	<-time.After(5 * time.Second)

	// Close.
	e, err := nsc.Close(ctx, conn)
	require.NoError(t, err)
	require.NotNil(t, e)
	require.Equal(t, 2, counter.UniqueRequests())
	require.Equal(t, 2, counter.UniqueCloses())
}

func TestNSMGR_HealLocalNSMgr(t *testing.T) {
	nsmgrCtx, nsmgrCtxCancel := context.WithTimeout(context.Background(), time.Second)
	defer nsmgrCtxCancel()

	customConfig := []*sandbox.NodeConfig{
		nil,
		{
			NsmgrCtx:                   nsmgrCtx,
			NsmgrGenerateTokenFunc:     sandbox.GenerateExpiringToken(time.Second),
			ForwarderGenerateTokenFunc: sandbox.GenerateExpiringToken(time.Second),
		},
	}

	testNSMGR_HealNSMgr(t, 1, customConfig, nsmgrCtxCancel)
}

func TestNSMGR_HealRemoteNSMgr(t *testing.T) {
	nsmgrCtx, nsmgrCtxCancel := context.WithTimeout(context.Background(), time.Second)
	defer nsmgrCtxCancel()

	customConfig := []*sandbox.NodeConfig{
		{
			NsmgrCtx:                   nsmgrCtx,
			NsmgrGenerateTokenFunc:     sandbox.GenerateExpiringToken(time.Second),
			ForwarderGenerateTokenFunc: sandbox.GenerateExpiringToken(time.Second),
		},
	}

	testNSMGR_HealNSMgr(t, 0, customConfig, nsmgrCtxCancel)
}

func testNSMGR_HealNSMgr(t *testing.T, nodeNum int, customConfig []*sandbox.NodeConfig, nsmgrCtxCancel context.CancelFunc) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	builder := sandbox.NewBuilder(t)
	domain := builder.
		SetNodesCount(2).
		SetRegistryProxySupplier(nil).
		SetContext(ctx).
		SetCustomConfig(customConfig).
		Build()
	defer domain.Cleanup()

	nseReg := &registry.NetworkServiceEndpoint{
		Name:                "final-endpoint",
		NetworkServiceNames: []string{"my-service-remote"},
	}

	counter := &counterServer{}
	_, err := sandbox.NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken, domain.Nodes[0].NSMgr, counter)
	require.NoError(t, err)

	request := &networkservice.NetworkServiceRequest{
		MechanismPreferences: []*networkservice.Mechanism{
			{Cls: cls.LOCAL, Type: kernelmech.MECHANISM},
		},
		Connection: &networkservice.Connection{
			Id:             "1",
			NetworkService: "my-service-remote",
			Context:        &networkservice.ConnectionContext{},
		},
	}

	nsc := sandbox.NewClient(ctx, sandbox.GenerateTestToken, domain.Nodes[1].NSMgr.URL)

	conn, err := nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, 1, counter.UniqueRequests())
	require.Equal(t, 8, len(conn.Path.PathSegments))

	nsmgrCtxCancel()
	builder.NewNSMgr(ctx, domain.Nodes[nodeNum].NSMgr.URL.Host, domain.Registry.URL, sandbox.GenerateTestToken)

	// Wait Cross NSE expired and reconnecting through the new Cross NSE
	<-time.After(5 * time.Second)

	// Close.
	e, err := nsc.Close(ctx, conn)
	require.NoError(t, err)
	require.NotNil(t, e)
	require.Equal(t, 2, counter.UniqueRequests())
	require.Equal(t, 2, counter.UniqueCloses())
}
