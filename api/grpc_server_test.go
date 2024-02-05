package api

import (
	"context"
	"sync"
	"testing"
	"time"

	v1 "github.com/gatewayd-io/gatewayd/api/v1"
	"github.com/gatewayd-io/gatewayd/config"
	"github.com/gatewayd-io/gatewayd/network"
	"github.com/gatewayd-io/gatewayd/plugin"
	"github.com/gatewayd-io/gatewayd/pool"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/protobuf/types/known/emptypb"
)

func Test_GRPCServer(t *testing.T) {
	clientConfig := &config.Client{
		Network: config.DefaultNetwork,
		Address: config.DefaultAddress,
	}
	client := network.NewClient(context.TODO(), clientConfig, zerolog.Logger{}, nil)
	newPool := pool.NewPool(context.TODO(), 1)
	assert.Nil(t, newPool.Put(client.ID, client))

	proxy := network.NewProxy(
		context.TODO(),
		newPool,
		nil,
		config.DefaultHealthCheckPeriod,
		&config.Client{
			Network: config.DefaultNetwork,
			Address: config.DefaultAddress,
		},
		zerolog.Logger{},
		config.DefaultPluginTimeout,
	)

	pluginRegistry := plugin.NewRegistry(
		context.TODO(),
		config.Loose,
		config.PassDown,
		config.Accept,
		config.Stop,
		zerolog.Logger{},
		true,
	)

	server := network.NewServer(
		context.TODO(),
		config.DefaultNetwork,
		config.DefaultAddress,
		config.DefaultTickInterval,
		network.Option{
			EnableTicker: false,
		},
		proxy,
		zerolog.Logger{},
		pluginRegistry,
		config.DefaultPluginTimeout,
		false,
		"",
		"",
		config.DefaultHandshakeTimeout,
	)

	var waitGroup sync.WaitGroup
	waitGroup.Add(2)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ctx, timeoutCancel := context.WithTimeout(ctx, 5*time.Second)
	defer timeoutCancel()

	go func(waitGroup *sync.WaitGroup, ctx context.Context) {
		defer waitGroup.Done()

		select {
		case <-ctx.Done():
			return
		default:
			StartGRPCAPI(
				&API{
					Pools: map[string]*pool.Pool{
						config.Default: newPool,
					},
					Proxies: map[string]*network.Proxy{
						config.Default: proxy,
					},
					Servers: map[string]*network.Server{
						config.Default: server,
					},
					Options: &Options{
						GRPCNetwork: "tcp",
						GRPCAddress: "localhost:19090",
					},
				},
				&HealthChecker{Servers: map[string]*network.Server{config.Default: server}})
		}
	}(&waitGroup, ctx)

	go func(t *testing.T, waitGroup *sync.WaitGroup, ctx context.Context) {
		defer waitGroup.Done()

		// Wait for the server to start
		time.Sleep(500 * time.Millisecond)

		grpcConn, err := grpc.DialContext(
			ctx, "localhost:19090", grpc.WithTransportCredentials(insecure.NewCredentials()))
		require.NoError(t, err)
		defer grpcConn.Close()

		healthzClient := grpc_health_v1.NewHealthClient(grpcConn)
		resp, err := healthzClient.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
		require.NoError(t, err)
		assert.Equal(t, grpc_health_v1.HealthCheckResponse_SERVING, resp.Status)

		client := v1.NewGatewayDAdminAPIServiceClient(grpcConn)
		result, err := client.GetServers(ctx, &emptypb.Empty{})
		require.NoError(t, err)
		assert.NotEmpty(t, result)

		// var health grpc_health_v1.HealthClient
		// err = grpcConn.Invoke(ctx, "/grpc.health.v1.Health/Check", nil, health)
		// assert.NotEmpty(t, err)
		// assert.Equal(t, health, nil)
		timeoutCancel()
		cancel()
	}(t, &waitGroup, ctx)

	waitGroup.Wait()
}
