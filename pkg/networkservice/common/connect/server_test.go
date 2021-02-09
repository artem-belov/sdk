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

package connect_test

import (
	"context"
	"net/url"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/memif"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vfio"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/connect"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/inject/injecterror"
	"github.com/networkservicemesh/sdk/pkg/tools/clienturlctx"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

const (
	parallelCount = 1000
)

func startServer(ctx context.Context, listenOn *url.URL, server networkservice.NetworkServiceServer) error {
	grpcServer := grpc.NewServer()
	networkservice.RegisterNetworkServiceServer(grpcServer, server)
	grpcutils.RegisterHealthServices(grpcServer, server)

	errCh := grpcutils.ListenAndServe(ctx, listenOn, grpcServer)
	select {
	case err := <-errCh:
		return err
	default:
		return nil
	}
}

func checkServer(target *url.URL, server networkservice.NetworkServiceServer) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	cc, err := grpc.DialContext(ctx, grpcutils.URLToTarget(target), grpc.WithBlock(), grpc.WithInsecure())
	defer func() {
		_ = cc.Close()
	}()
	if err != nil {
		return err
	}

	serviceNames := networkservice.ServiceNames(server)
	if len(serviceNames) == 0 {
		return errors.Errorf("grpc Service Name was not found")
	}
	serviceName := serviceNames[0]

	client := grpc_health_v1.NewHealthClient(cc)
	for ctx.Err() == nil {
		response, err := client.Check(ctx, &grpc_health_v1.HealthCheckRequest{Service: serviceName})
		if err != nil {
			return err
		}
		if response.Status == grpc_health_v1.HealthCheckResponse_SERVING {
			return nil
		}
	}

	return ctx.Err()
}

func TestConnectServer_Request(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	// 1. Create connectServer

	serverNext := new(captureServer)
	serverClient := new(captureServer)

	s := next.NewNetworkServiceServer(
		connect.NewServer(context.TODO(),
			func(_ context.Context, cc grpc.ClientConnInterface) networkservice.NetworkServiceClient {
				return next.NewNetworkServiceClient(
					adapters.NewServerToClient(serverClient),
					networkservice.NewNetworkServiceClient(cc),
				)
			},
			grpc.WithInsecure(),
		),
		serverNext,
	)

	func() {
		ctx, cancel := context.WithCancel(context.Background())
		ctx = log.WithLog(ctx)
		defer cancel()

		// 3. Setup servers

		urlA := &url.URL{Scheme: "tcp", Host: "127.0.0.1:10000"}
		serverA := new(captureServer)
		nsServerA := next.NewNetworkServiceServer(
			serverA,
			newEditServer("a", "A", &networkservice.Mechanism{
				Cls:  cls.LOCAL,
				Type: kernel.MECHANISM,
			}),
		)
		err := startServer(ctx, urlA, nsServerA)
		require.NoError(t, err)
		require.NoError(t, checkServer(urlA, nsServerA))

		urlB := &url.URL{Scheme: "tcp", Host: "127.0.0.1:10001"}
		serverB := new(captureServer)
		nsServerB := next.NewNetworkServiceServer(
			serverB,
			newEditServer("b", "B", &networkservice.Mechanism{
				Cls:  cls.LOCAL,
				Type: memif.MECHANISM,
			}),
		)
		err = startServer(ctx, urlB, nsServerB)
		require.NoError(t, err)
		require.NoError(t, checkServer(urlB, nsServerB))

		defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

		// 4. Create request

		request := &networkservice.NetworkServiceRequest{
			Connection: &networkservice.Connection{
				Id:             "id",
				NetworkService: "network-service",
				Mechanism: &networkservice.Mechanism{
					Cls:  cls.LOCAL,
					Type: vfio.MECHANISM,
				},
				Context: &networkservice.ConnectionContext{
					ExtraContext: map[string]string{
						"not": "empty",
					},
				},
			},
		}

		// 5. Request A

		conn, err := s.Request(clienturlctx.WithClientURL(ctx, urlA), request.Clone())
		require.NoError(t, err)

		requestClient := request.Clone()
		require.Equal(t, requestClient.String(), serverClient.capturedRequest.String())

		requestA := request.Clone()
		require.Equal(t, requestA.String(), serverA.capturedRequest.String())

		requestNext := request.Clone()
		requestNext.Connection.Mechanism.Type = kernel.MECHANISM
		requestNext.Connection.Context.ExtraContext["a"] = "A"
		require.Equal(t, requestNext.String(), serverNext.capturedRequest.String())

		require.Equal(t, requestNext.Connection.String(), conn.String())

		// 6. Request B

		request.Connection = conn

		conn, err = s.Request(clienturlctx.WithClientURL(ctx, urlB), request.Clone())
		require.NoError(t, err)

		requestClient = request.Clone()
		require.Equal(t, requestClient.String(), serverClient.capturedRequest.String())

		require.Nil(t, serverA.capturedRequest)

		requestB := request.Clone()
		require.Equal(t, requestB.String(), serverB.capturedRequest.String())

		requestNext = request.Clone()
		requestNext.Connection.Mechanism.Type = memif.MECHANISM
		requestNext.Connection.Context.ExtraContext["b"] = "B"
		require.Equal(t, requestNext.String(), serverNext.capturedRequest.String())

		require.Equal(t, requestNext.Connection.String(), conn.String())

		// 8. Close B

		_, err = s.Close(ctx, conn)
		require.NoError(t, err)

		require.Nil(t, serverClient.capturedRequest)
		require.Nil(t, serverB.capturedRequest)
		require.Nil(t, serverNext.capturedRequest)
	}()
}

func TestConnectServer_RequestParallel(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	// 1. Create connectServer

	serverNext := new(countServer)
	serverClient := new(countServer)

	s := next.NewNetworkServiceServer(
		connect.NewServer(context.TODO(),
			func(_ context.Context, cc grpc.ClientConnInterface) networkservice.NetworkServiceClient {
				return next.NewNetworkServiceClient(
					adapters.NewServerToClient(serverClient),
					networkservice.NewNetworkServiceClient(cc),
				)
			},
			grpc.WithInsecure(),
		),
		serverNext,
	)

	func() {
		ctx, cancel := context.WithCancel(context.Background())
		ctx = log.WithLog(ctx)
		defer cancel()

		// 3. Setup servers

		urlA := &url.URL{Scheme: "tcp", Host: "127.0.0.1:10000"}
		serverA := new(countServer)

		err := startServer(ctx, urlA, serverA)
		require.NoError(t, err)
		require.NoError(t, checkServer(urlA, serverA))

		defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

		// 4. Request A

		wg := new(sync.WaitGroup)
		wg.Add(parallelCount)

		barrier := new(sync.WaitGroup)
		barrier.Add(1)

		for i := 0; i < parallelCount; i++ {
			go func(k int) {
				// 4.1. Create request
				request := &networkservice.NetworkServiceRequest{
					Connection: &networkservice.Connection{
						Id: strconv.Itoa(k),
					},
				}

				// 4.2. Request A
				_, err := s.Request(clienturlctx.WithClientURL(ctx, urlA), request)
				assert.NoError(t, err)
				wg.Done()

				barrier.Wait()

				// 4.3. Re request A
				conn, err := s.Request(clienturlctx.WithClientURL(ctx, urlA), request)
				assert.NoError(t, err)

				// 4.4. Close A
				_, err = s.Close(ctx, conn)
				assert.NoError(t, err)
				wg.Done()
			}(i)
		}

		wg.Wait()
		wg.Add(parallelCount)

		require.Equal(t, int32(parallelCount), serverClient.count)
		require.Equal(t, int32(parallelCount), serverA.count)
		require.Equal(t, int32(parallelCount), serverNext.count)

		barrier.Done()
		wg.Wait()

		require.Equal(t, int32(parallelCount), serverClient.count)
		require.Equal(t, int32(parallelCount), serverA.count)
		require.Equal(t, int32(parallelCount), serverNext.count)
	}()
}

func TestConnectServer_RequestFail(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	// 1. Create connectServer

	s := connect.NewServer(context.TODO(),
		func(_ context.Context, cc grpc.ClientConnInterface) networkservice.NetworkServiceClient {
			return injecterror.NewClient()
		},
		grpc.WithInsecure(),
	)

	func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// 2. Setup servers

		urlA := &url.URL{Scheme: "tcp", Host: "127.0.0.1:10000"}
		serverA := new(captureServer)

		err := startServer(ctx, urlA, next.NewNetworkServiceServer(
			serverA,
			newEditServer("a", "A", &networkservice.Mechanism{
				Cls:  cls.LOCAL,
				Type: kernel.MECHANISM,
			}),
		))
		require.NoError(t, err)
		require.NoError(t, checkServer(urlA, serverA))

		defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

		request := &networkservice.NetworkServiceRequest{
			Connection: &networkservice.Connection{
				Id:             "id",
				NetworkService: "network-service",
				Mechanism: &networkservice.Mechanism{
					Cls:  cls.LOCAL,
					Type: vfio.MECHANISM,
				},
				Context: &networkservice.ConnectionContext{
					ExtraContext: map[string]string{
						"not": "empty",
					},
				},
			},
		}

		// 4. Request A
		_, err = s.Request(clienturlctx.WithClientURL(ctx, urlA), request.Clone())
		require.Error(t, err)
	}()
}

type editServer struct {
	key       string
	value     string
	mechanism *networkservice.Mechanism
}

func newEditServer(key, value string, mechanism *networkservice.Mechanism) *editServer {
	return &editServer{
		key:       key,
		value:     value,
		mechanism: mechanism,
	}
}

func (s *editServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	request.Connection.Context.ExtraContext[s.key] = s.value
	request.Connection.Mechanism = s.mechanism

	return next.Server(ctx).Request(ctx, request)
}

func (s *editServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}

type captureServer struct {
	capturedRequest *networkservice.NetworkServiceRequest
}

func (s *captureServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	s.capturedRequest = request.Clone()
	return next.Server(ctx).Request(ctx, request)
}

func (s *captureServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	s.capturedRequest = nil
	return next.Server(ctx).Close(ctx, conn)
}

type countServer struct {
	count int32
}

func (s *countServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	atomic.AddInt32(&s.count, 1)
	return next.Server(ctx).Request(ctx, request)
}

func (s *countServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	atomic.AddInt32(&s.count, -1)
	return next.Server(ctx).Close(ctx, conn)
}
