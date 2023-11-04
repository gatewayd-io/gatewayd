package network

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"time"

	"github.com/gatewayd-io/gatewayd-plugin-sdk/databases/postgres"
	v1 "github.com/gatewayd-io/gatewayd-plugin-sdk/plugin/v1"
	"github.com/gatewayd-io/gatewayd/config"
	gerr "github.com/gatewayd-io/gatewayd/errors"
	"github.com/gatewayd-io/gatewayd/metrics"
	"github.com/gatewayd-io/gatewayd/plugin"
	"github.com/gatewayd-io/gatewayd/pool"
	"github.com/getsentry/sentry-go"
	"github.com/go-co-op/gocron"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
)

type IProxy interface {
	Connect(conn *ConnWrapper) *gerr.GatewayDError
	Disconnect(conn *ConnWrapper) *gerr.GatewayDError
	PassThroughToServer(conn *ConnWrapper, stack *Stack) *gerr.GatewayDError
	PassThroughToClient(conn *ConnWrapper, stack *Stack) *gerr.GatewayDError
	IsHealthy(cl *Client) (*Client, *gerr.GatewayDError)
	IsExhausted() bool
	Shutdown()
	AvailableConnections() []string
	BusyConnections() []string
}

type Proxy struct {
	availableConnections pool.IPool
	busyConnections      pool.IPool
	logger               zerolog.Logger
	pluginRegistry       *plugin.Registry
	scheduler            *gocron.Scheduler
	ctx                  context.Context //nolint:containedctx
	pluginTimeout        time.Duration

	Elastic             bool
	ReuseElasticClients bool
	HealthCheckPeriod   time.Duration

	// ClientConfig is used for elastic proxy and reconnection
	ClientConfig *config.Client
}

var _ IProxy = &Proxy{}

// NewProxy creates a new proxy.
func NewProxy(
	ctx context.Context,
	connPool pool.IPool, pluginRegistry *plugin.Registry,
	elastic, reuseElasticClients bool,
	healthCheckPeriod time.Duration,
	clientConfig *config.Client, logger zerolog.Logger,
	pluginTimeout time.Duration,
) *Proxy {
	proxyCtx, span := otel.Tracer(config.TracerName).Start(ctx, "NewProxy")
	defer span.End()

	proxy := Proxy{
		availableConnections: connPool,
		busyConnections:      pool.NewPool(proxyCtx, config.EmptyPoolCapacity),
		logger:               logger,
		pluginRegistry:       pluginRegistry,
		scheduler:            gocron.NewScheduler(time.UTC),
		ctx:                  proxyCtx,
		pluginTimeout:        pluginTimeout,
		Elastic:              elastic,
		ReuseElasticClients:  reuseElasticClients,
		ClientConfig:         clientConfig,
		HealthCheckPeriod:    healthCheckPeriod,
	}

	startDelay := time.Now().Add(proxy.HealthCheckPeriod)
	// Schedule the client health check.
	if _, err := proxy.scheduler.Every(proxy.HealthCheckPeriod).SingletonMode().StartAt(startDelay).Do(
		func() {
			now := time.Now()
			logger.Trace().Msg("Running the client health check to recycle connection(s).")
			proxy.availableConnections.ForEach(func(_, value interface{}) bool {
				if client, ok := value.(*Client); ok {
					// Connection is probably dead by now.
					proxy.availableConnections.Remove(client.ID)
					client.Close()
					// Create a new client.
					client = NewClient(proxyCtx, proxy.ClientConfig, proxy.logger)
					if client != nil && client.ID != "" {
						if err := proxy.availableConnections.Put(client.ID, client); err != nil {
							proxy.logger.Err(err).Msg("Failed to update the client connection")
							// Close the client, because we don't want to have orphaned connections.
							client.Close()
						}
					} else {
						proxy.logger.Error().Msg("Failed to create a new client connection")
					}
				}
				return true
			})
			logger.Trace().Str("duration", time.Since(now).String()).Msg(
				"Finished the client health check")
			metrics.ProxyHealthChecks.Inc()
		},
	); err != nil {
		proxy.logger.Error().Err(err).Msg("Failed to schedule the client health check")
		sentry.CaptureException(err)
		span.RecordError(err)
	}

	// Start the scheduler.
	proxy.scheduler.StartAsync()
	logger.Info().Fields(
		map[string]interface{}{
			"startDelay":        startDelay.Format(time.RFC3339),
			"healthCheckPeriod": proxy.HealthCheckPeriod.String(),
		},
	).Msg("Started the client health check scheduler")

	return &proxy
}

// Connect maps a server connection from the available connection pool to a incoming connection.
// It returns an error if the pool is exhausted. If the pool is elastic, it creates a new client
// and maps it to the incoming connection.
func (pr *Proxy) Connect(conn *ConnWrapper) *gerr.GatewayDError {
	_, span := otel.Tracer(config.TracerName).Start(pr.ctx, "Connect")
	defer span.End()

	var clientID string
	// Get the first available client from the pool.
	pr.availableConnections.ForEach(func(key, _ interface{}) bool {
		if cid, ok := key.(string); ok {
			clientID = cid
			return false // stop the loop.
		}
		return true
	})

	var client *Client
	if pr.IsExhausted() {
		// Pool is exhausted or is elastic.
		if pr.Elastic {
			// Create a new client.
			client = NewClient(pr.ctx, pr.ClientConfig, pr.logger)
			span.AddEvent("Created a new client connection")
			pr.logger.Debug().Str("id", client.ID[:7]).Msg("Reused the client connection")
		} else {
			span.AddEvent(gerr.ErrPoolExhausted.Error())
			return gerr.ErrPoolExhausted
		}
	} else {
		// Get the client from the pool with the given clientID.
		if cl, ok := pr.availableConnections.Pop(clientID).(*Client); ok {
			client = cl
		}
	}

	client, err := pr.IsHealthy(client)
	if err != nil {
		pr.logger.Error().Err(err).Msg("Failed to connect to the client")
		span.RecordError(err)
	}

	if err := pr.busyConnections.Put(conn, client); err != nil {
		// This should never happen.
		span.RecordError(err)
		return err
	}

	metrics.ProxiedConnections.Inc()

	fields := map[string]interface{}{
		"function": "proxy.connect",
		"client":   "unknown",
		"server":   RemoteAddr(conn.Conn()),
	}
	if client.ID != "" {
		fields["client"] = client.ID[:7]
	}
	pr.logger.Debug().Fields(fields).Msg("Client has been assigned")

	pr.logger.Debug().Fields(
		map[string]interface{}{
			"function": "proxy.connect",
			"count":    pr.availableConnections.Size(),
		},
	).Msg("Available client connections")
	pr.logger.Debug().Fields(
		map[string]interface{}{
			"function": "proxy.connect",
			"count":    pr.busyConnections.Size(),
		},
	).Msg("Busy client connections")

	return nil
}

// Disconnect removes the client from the busy connection pool and tries to recycle
// the server connection.
func (pr *Proxy) Disconnect(conn *ConnWrapper) *gerr.GatewayDError {
	_, span := otel.Tracer(config.TracerName).Start(pr.ctx, "Disconnect")
	defer span.End()

	client := pr.busyConnections.Pop(conn)
	if client == nil {
		// If this ever happens, it means that the client connection
		// is pre-empted from the busy connections pool.
		pr.logger.Debug().Msg("Client connection is pre-empted from the busy connections pool")
		span.RecordError(gerr.ErrClientNotFound)
		return gerr.ErrClientNotFound
	}

	//nolint:nestif
	if client, ok := client.(*Client); ok {
		if (pr.Elastic && pr.ReuseElasticClients) || !pr.Elastic {
			// Recycle the server connection by reconnecting.
			if err := client.Reconnect(); err != nil {
				pr.logger.Error().Err(err).Msg("Failed to reconnect to the client")
				span.RecordError(err)
			}

			// If the client is not in the pool, put it back.
			if err := pr.availableConnections.Put(client.ID, client); err != nil {
				pr.logger.Error().Err(err).Msg("Failed to put the client back in the pool")
				span.RecordError(err)
			}
		} else {
			span.RecordError(gerr.ErrClientNotConnected)
			return gerr.ErrClientNotConnected
		}
	} else {
		// This should never happen, but if it does,
		// then there are some serious issues with the pool.
		pr.logger.Error().Msg("Failed to cast the client to the Client type")
		span.RecordError(gerr.ErrCastFailed)
		return gerr.ErrCastFailed
	}

	metrics.ProxiedConnections.Dec()

	pr.logger.Debug().Fields(
		map[string]interface{}{
			"function": "proxy.disconnect",
			"count":    pr.availableConnections.Size(),
		},
	).Msg("Available client connections")
	pr.logger.Debug().Fields(
		map[string]interface{}{
			"function": "proxy.disconnect",
			"count":    pr.busyConnections.Size(),
		},
	).Msg("Busy client connections")

	return nil
}

// PassThroughToServer sends the data from the client to the server.
func (pr *Proxy) PassThroughToServer(conn *ConnWrapper, stack *Stack) *gerr.GatewayDError {
	_, span := otel.Tracer(config.TracerName).Start(pr.ctx, "PassThrough")
	defer span.End()

	var client *Client
	// Check if the proxy has a egress client for the incoming connection.
	if pr.busyConnections.Get(conn) == nil {
		span.RecordError(gerr.ErrClientNotFound)
		return gerr.ErrClientNotFound
	}

	// Get the client from the busy connection pool.
	if cl, ok := pr.busyConnections.Get(conn).(*Client); ok {
		client = cl
	} else {
		span.RecordError(gerr.ErrCastFailed)
		return gerr.ErrCastFailed
	}
	span.AddEvent("Got the client from the busy connection pool")

	if !client.IsConnected() {
		return gerr.ErrClientNotConnected
	}

	// Receive the request from the client.
	request, origErr := pr.receiveTrafficFromClient(conn.Conn())
	span.AddEvent("Received traffic from client")

	// Run the OnTrafficFromClient hooks.
	pluginTimeoutCtx, cancel := context.WithTimeout(context.Background(), pr.pluginTimeout)
	defer cancel()

	result, err := pr.pluginRegistry.Run(
		pluginTimeoutCtx,
		trafficData(
			conn.Conn(),
			client,
			[]Field{
				{
					Name:  "request",
					Value: request,
				},
			},
			origErr),
		v1.HookName_HOOK_NAME_ON_TRAFFIC_FROM_CLIENT)
	if err != nil {
		pr.logger.Error().Err(err).Msg("Error running hook")
		span.RecordError(err)
	}
	span.AddEvent("Ran the OnTrafficFromClient hooks")

	if origErr != nil && errors.Is(origErr, io.EOF) {
		// Client closed the connection.
		span.AddEvent("Client closed the connection")
		return gerr.ErrClientNotConnected.Wrap(origErr)
	}

	// Check if the client sent a SSL request.
	if postgres.IsPostgresSSLRequest(request) {
		// Perform TLS handshake.
		conn.UpgradeToTLS(func(c net.Conn) {
			// Acknowledge the SSL request.
			if sent, err := conn.Write([]byte{'S'}); err != nil {
				pr.logger.Error().Err(err).Msg("Failed to acknowledge the SSL request")
				span.RecordError(err)
			} else {
				pr.logger.Debug().Fields(
					map[string]interface{}{
						"function": "upgradeToTLS",
						"local":    LocalAddr(conn.Conn()),
						"remote":   RemoteAddr(conn.Conn()),
						"length":   sent,
					},
				).Msg("Sent data to database")
			}
		})

		if conn.IsTLSEnabled() {
			pr.logger.Debug().Fields(
				map[string]interface{}{
					"local":  LocalAddr(conn.Conn()),
					"remote": RemoteAddr(conn.Conn()),
				},
			).Msg("Performed the TLS handshake")
			span.AddEvent("Performed the TLS handshake")

			return nil
		} else {
			pr.logger.Error().Fields(
				map[string]interface{}{
					"local":  LocalAddr(conn.Conn()),
					"remote": RemoteAddr(conn.Conn()),
				},
			).Msg("Failed to perform the TLS handshake")
			span.AddEvent("Failed to perform the TLS handshake")

			if _, err := conn.Write([]byte{'N'}); err != nil {
				pr.logger.Error().Err(err).Msg("Server does not support SSL, but SSL was required")
				span.RecordError(err)
			}

			return gerr.ErrTLSDisabled
		}
	}

	// Push the client's request to the stack.
	stack.Push(&Request{Data: request})

	// If the hook wants to terminate the connection, do it.
	if pr.shouldTerminate(result) {
		if modResponse, modReceived := pr.getPluginModifiedResponse(result); modResponse != nil {
			metrics.ProxyPassThroughsToClient.Inc()
			metrics.ProxyPassThroughTerminations.Inc()
			metrics.BytesSentToClient.Observe(float64(modReceived))
			metrics.TotalTrafficBytes.Observe(float64(modReceived))

			span.AddEvent("Terminating connection")

			// Remove the request from the stack if the response is modified.
			stack.PopLastRequest()

			return pr.sendTrafficToClient(conn.Conn(), modResponse, modReceived)
		}
		span.RecordError(gerr.ErrHookTerminatedConnection)
		return gerr.ErrHookTerminatedConnection
	}
	// If the hook modified the request, use the modified request.
	if modRequest := pr.getPluginModifiedRequest(result); modRequest != nil {
		request = modRequest
		span.AddEvent("Plugin(s) modified the request")
	}

	stack.UpdateLastRequest(&Request{Data: request})

	// Send the request to the server.
	_, err = pr.sendTrafficToServer(client, request)
	span.AddEvent("Sent traffic to server")

	pluginTimeoutCtx, cancel = context.WithTimeout(context.Background(), pr.pluginTimeout)
	defer cancel()

	// Run the OnTrafficToServer hooks.
	_, err = pr.pluginRegistry.Run(
		pluginTimeoutCtx,
		trafficData(
			conn.Conn(),
			client,
			[]Field{
				{
					Name:  "request",
					Value: request,
				},
			},
			err),
		v1.HookName_HOOK_NAME_ON_TRAFFIC_TO_SERVER)
	if err != nil {
		pr.logger.Error().Err(err).Msg("Error running hook")
		span.RecordError(err)
	}
	span.AddEvent("Ran the OnTrafficToServer hooks")

	metrics.ProxyPassThroughsToServer.Inc()

	return nil
}

// PassThroughToClient sends the data from the server to the client.
func (pr *Proxy) PassThroughToClient(conn *ConnWrapper, stack *Stack) *gerr.GatewayDError {
	_, span := otel.Tracer(config.TracerName).Start(pr.ctx, "PassThrough")
	defer span.End()

	var client *Client
	// Check if the proxy has a egress client for the incoming connection.
	if pr.busyConnections.Get(conn) == nil {
		span.RecordError(gerr.ErrClientNotFound)
		return gerr.ErrClientNotFound
	}

	// Get the client from the busy connection pool.
	if cl, ok := pr.busyConnections.Get(conn).(*Client); ok {
		client = cl
	} else {
		span.RecordError(gerr.ErrCastFailed)
		return gerr.ErrCastFailed
	}
	span.AddEvent("Got the client from the busy connection pool")

	if !client.IsConnected() {
		return gerr.ErrClientNotConnected
	}

	// Receive the response from the server.
	received, response, err := pr.receiveTrafficFromServer(client)
	span.AddEvent("Received traffic from server")

	// If the response is empty, don't send anything, instead just close the ingress connection.
	if received == 0 || err != nil {
		fields := map[string]interface{}{"function": "proxy.passthrough"}
		if client.LocalAddr() != "" {
			fields["local_addr"] = client.LocalAddr()
		}
		if client.RemoteAddr() != "" {
			fields["remote_addr"] = client.RemoteAddr()
		}
		pr.logger.Debug().Fields(fields).Msg("No data to send to client")
		span.AddEvent("No data to send to client")
		span.RecordError(err)

		stack.PopLastRequest()

		return err
	}

	pluginTimeoutCtx, cancel := context.WithTimeout(context.Background(), pr.pluginTimeout)
	defer cancel()

	// Get the last request from the stack.
	lastRequest := stack.PopLastRequest()
	request := make([]byte, 0)
	if lastRequest != nil {
		request = lastRequest.Data
	}

	// Run the OnTrafficFromServer hooks.
	result, err := pr.pluginRegistry.Run(
		pluginTimeoutCtx,
		trafficData(
			conn.Conn(),
			client,
			[]Field{
				{
					Name:  "request",
					Value: request,
				},
				{
					Name:  "response",
					Value: response[:received],
				},
			},
			err),
		v1.HookName_HOOK_NAME_ON_TRAFFIC_FROM_SERVER)
	if err != nil {
		pr.logger.Error().Err(err).Msg("Error running hook")
		span.RecordError(err)
	}
	span.AddEvent("Ran the OnTrafficFromServer hooks")

	// If the hook modified the response, use the modified response.
	if modResponse, modReceived := pr.getPluginModifiedResponse(result); modResponse != nil {
		response = modResponse
		received = modReceived
		span.AddEvent("Plugin(s) modified the response")
	}

	// Send the response to the client.
	errVerdict := pr.sendTrafficToClient(conn.Conn(), response, received)
	span.AddEvent("Sent traffic to client")

	// Run the OnTrafficToClient hooks.
	pluginTimeoutCtx, cancel = context.WithTimeout(context.Background(), pr.pluginTimeout)
	defer cancel()

	_, err = pr.pluginRegistry.Run(
		pluginTimeoutCtx,
		trafficData(
			conn.Conn(),
			client,
			[]Field{
				{
					Name:  "request",
					Value: request,
				},
				{
					Name:  "response",
					Value: response[:received],
				},
			},
			nil,
		),
		v1.HookName_HOOK_NAME_ON_TRAFFIC_TO_CLIENT)
	if err != nil {
		pr.logger.Error().Err(err).Msg("Error running hook")
		span.RecordError(err)
	}

	if errVerdict != nil {
		span.RecordError(errVerdict)
	}

	metrics.ProxyPassThroughsToClient.Inc()

	return errVerdict
}

// IsHealthy checks if the pool is exhausted or the client is disconnected.
func (pr *Proxy) IsHealthy(client *Client) (*Client, *gerr.GatewayDError) {
	_, span := otel.Tracer(config.TracerName).Start(pr.ctx, "IsHealthy")
	defer span.End()

	if pr.IsExhausted() {
		pr.logger.Error().Msg("No more available connections")
		span.RecordError(gerr.ErrPoolExhausted)
		return client, gerr.ErrPoolExhausted
	}

	if !client.IsConnected() {
		pr.logger.Error().Msg("Client is disconnected")
		span.RecordError(gerr.ErrClientNotConnected)
	}

	return client, nil
}

// IsExhausted checks if the available connection pool is exhausted.
func (pr *Proxy) IsExhausted() bool {
	_, span := otel.Tracer(config.TracerName).Start(pr.ctx, "IsExhausted")
	defer span.End()

	if pr.Elastic {
		return false
	}

	return pr.availableConnections.Size() == 0 && pr.availableConnections.Cap() > 0
}

// Shutdown closes all connections and clears the connection pools.
func (pr *Proxy) Shutdown() {
	_, span := otel.Tracer(config.TracerName).Start(pr.ctx, "Shutdown")
	defer span.End()

	pr.availableConnections.ForEach(func(key, value interface{}) bool {
		if client, ok := value.(*Client); ok {
			if client.IsConnected() {
				client.Close()
			}
		}
		return true
	})
	pr.availableConnections.Clear()
	pr.logger.Debug().Msg("All available connections have been closed")

	pr.busyConnections.ForEach(func(key, value interface{}) bool {
		if conn, ok := key.(net.Conn); ok {
			// This will stop all the Conn.Read() and Conn.Write() calls.
			if err := conn.SetDeadline(time.Now()); err != nil {
				pr.logger.Error().Err(err).Msg("Error setting the deadline")
				span.RecordError(err)
			}
			if err := conn.Close(); err != nil {
				pr.logger.Error().Err(err).Msg("Failed to close the connection")
				span.RecordError(err)
			}
		}
		if client, ok := value.(*Client); ok {
			if client != nil {
				client.Close()
			}
		}
		return true
	})
	pr.busyConnections.Clear()
	pr.scheduler.Stop()
	pr.scheduler.Clear()
	pr.logger.Debug().Msg("All busy connections have been closed")
}

// AvailableConnections returns a list of available connections.
func (pr *Proxy) AvailableConnections() []string {
	_, span := otel.Tracer(config.TracerName).Start(pr.ctx, "AvailableConnections")
	defer span.End()

	connections := make([]string, 0)
	pr.availableConnections.ForEach(func(_, value interface{}) bool {
		if cl, ok := value.(*Client); ok {
			connections = append(connections, cl.LocalAddr())
		}
		return true
	})
	return connections
}

// BusyConnections returns a list of busy connections.
func (pr *Proxy) BusyConnections() []string {
	_, span := otel.Tracer(config.TracerName).Start(pr.ctx, "BusyConnections")
	defer span.End()

	connections := make([]string, 0)
	pr.busyConnections.ForEach(func(key, _ interface{}) bool {
		if conn, ok := key.(net.Conn); ok {
			connections = append(connections, RemoteAddr(conn))
		}
		return true
	})
	return connections
}

// receiveTrafficFromClient is a function that waits to receive data from the client.
func (pr *Proxy) receiveTrafficFromClient(conn net.Conn) ([]byte, *gerr.GatewayDError) {
	_, span := otel.Tracer(config.TracerName).Start(pr.ctx, "receiveTrafficFromClient")
	defer span.End()

	// request contains the data from the client.
	received := 0
	buffer := bytes.NewBuffer(nil)
	for {
		chunk := make([]byte, pr.ClientConfig.ReceiveChunkSize)
		read, err := conn.Read(chunk)
		if read == 0 || err != nil {
			pr.logger.Debug().Err(err).Msg("Error reading from client")
			span.RecordError(err)

			metrics.BytesReceivedFromClient.Observe(float64(read))
			metrics.TotalTrafficBytes.Observe(float64(read))

			return chunk[:read], gerr.ErrReadFailed.Wrap(err)
		}

		received += read
		buffer.Write(chunk[:read])

		if received == 0 || received < pr.ClientConfig.ReceiveChunkSize {
			break
		}

		if !pr.isConnectionHealthy(conn) {
			break
		}
	}

	length := len(buffer.Bytes())
	pr.logger.Debug().Fields(
		map[string]interface{}{
			"length": length,
			"local":  LocalAddr(conn),
			"remote": RemoteAddr(conn),
		},
	).Msg("Received data from client")
	metrics.BytesReceivedFromClient.Observe(float64(length))
	metrics.TotalTrafficBytes.Observe(float64(length))

	return buffer.Bytes(), nil
}

// sendTrafficToServer is a function that sends data to the server.
func (pr *Proxy) sendTrafficToServer(client *Client, request []byte) (int, *gerr.GatewayDError) {
	_, span := otel.Tracer(config.TracerName).Start(pr.ctx, "sendTrafficToServer")
	defer span.End()

	if len(request) == 0 {
		pr.logger.Trace().Msg("Empty request")
		return 0, nil
	}

	// Send the request to the server.
	sent, err := client.Send(request)
	if err != nil {
		pr.logger.Error().Err(err).Msg("Error sending request to database")
		span.RecordError(err)
	}
	pr.logger.Debug().Fields(
		map[string]interface{}{
			"function": "proxy.passthrough",
			"length":   sent,
			"local":    client.LocalAddr(),
			"remote":   client.RemoteAddr(),
		},
	).Msg("Sent data to database")

	metrics.BytesSentToServer.Observe(float64(sent))
	metrics.TotalTrafficBytes.Observe(float64(sent))

	return sent, err
}

// receiveTrafficFromServer is a function that receives data from the server.
func (pr *Proxy) receiveTrafficFromServer(client *Client) (int, []byte, *gerr.GatewayDError) {
	_, span := otel.Tracer(config.TracerName).Start(pr.ctx, "receiveTrafficFromServer")
	defer span.End()

	// Receive the response from the server.
	received, response, err := client.Receive()

	fields := map[string]interface{}{
		"function": "proxy.passthrough",
		"length":   received,
	}
	if client.LocalAddr() != "" {
		fields["local"] = client.LocalAddr()
	}
	if client.RemoteAddr() != "" {
		fields["remote"] = client.RemoteAddr()
	}

	pr.logger.Debug().Fields(fields).Msg("Received data from database")

	metrics.BytesReceivedFromServer.Observe(float64(received))
	metrics.TotalTrafficBytes.Observe(float64(received))

	return received, response, err
}

// sendTrafficToClient is a function that sends data to the client.
func (pr *Proxy) sendTrafficToClient(
	conn net.Conn, response []byte, received int,
) *gerr.GatewayDError {
	_, span := otel.Tracer(config.TracerName).Start(pr.ctx, "sendTrafficToClient")
	defer span.End()

	// Send the response to the client async.
	sent := 0
	for {
		if sent >= received {
			break
		}

		written, origErr := conn.Write(response[:received])
		if origErr != nil {
			pr.logger.Error().Err(origErr).Msg("Error writing to client")
			span.RecordError(origErr)
			return gerr.ErrServerSendFailed.Wrap(origErr)
		}

		sent += written
	}

	pr.logger.Debug().Fields(
		map[string]interface{}{
			"function": "proxy.passthrough",
			"length":   sent,
			"local":    LocalAddr(conn),
			"remote":   RemoteAddr(conn),
		},
	).Msg("Sent data to client")

	span.AddEvent("Sent data to client")

	metrics.BytesSentToClient.Observe(float64(received))
	metrics.TotalTrafficBytes.Observe(float64(received))

	return nil
}

// shouldTerminate is a function that retrieves the terminate field from the hook result.
// Only the OnTrafficFromClient hook will terminate the connection.
func (pr *Proxy) shouldTerminate(result map[string]interface{}) bool {
	_, span := otel.Tracer(config.TracerName).Start(pr.ctx, "shouldTerminate")
	defer span.End()

	// If the hook wants to terminate the connection, do it.
	if result != nil {
		if terminate, ok := result["terminate"].(bool); ok && terminate {
			pr.logger.Debug().Str("function", "proxy.passthrough").Msg("Terminating connection")
			return true
		}
	}

	return false
}

// getPluginModifiedRequest is a function that retrieves the modified request
// from the hook result.
func (pr *Proxy) getPluginModifiedRequest(result map[string]interface{}) []byte {
	_, span := otel.Tracer(config.TracerName).Start(pr.ctx, "getPluginModifiedRequest")
	defer span.End()

	// If the hook modified the request, use the modified request.
	if modRequest, errMsg := extractFieldValue(result, "request"); errMsg != "" {
		pr.logger.Error().Str("error", errMsg).Msg("Error in hook")
	} else if modRequest != nil {
		return modRequest
	}

	return nil
}

// getPluginModifiedResponse is a function that retrieves the modified response
// from the hook result.
func (pr *Proxy) getPluginModifiedResponse(result map[string]interface{}) ([]byte, int) {
	_, span := otel.Tracer(config.TracerName).Start(pr.ctx, "getPluginModifiedResponse")
	defer span.End()

	// If the hook returns a response, use it instead of the original response.
	if modResponse, errMsg := extractFieldValue(result, "response"); errMsg != "" {
		pr.logger.Error().Str("error", errMsg).Msg("Error in hook")
	} else if modResponse != nil {
		return modResponse, len(modResponse)
	}

	return nil, 0
}

func (pr *Proxy) isConnectionHealthy(conn net.Conn) bool {
	if n, err := conn.Read([]byte{}); n == 0 && err != nil {
		pr.logger.Debug().Fields(
			map[string]interface{}{
				"remote": RemoteAddr(conn),
				"local":  LocalAddr(conn),
				"reason": "read 0 bytes",
			}).Msg("Connection to client is closed")
		return false
	}

	return true
}
