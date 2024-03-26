package network

import (
	"context"
	"time"

	v1 "github.com/gatewayd-io/gatewayd-plugin-sdk/plugin/v1"
	"github.com/gatewayd-io/gatewayd/config"
	gerr "github.com/gatewayd-io/gatewayd/errors"
	"github.com/gatewayd-io/gatewayd/metrics"
	"github.com/gatewayd-io/gatewayd/plugin"
	"github.com/gatewayd-io/gatewayd/pool"
	"github.com/getsentry/sentry-go"
	"github.com/go-co-op/gocron"
	"github.com/panjf2000/gnet/v2"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
)

type IProxy interface {
	Connect(gconn gnet.Conn) *gerr.GatewayDError
	Disconnect(gconn gnet.Conn) *gerr.GatewayDError
	PassThrough(gconn gnet.Conn) *gerr.GatewayDError
	IsHealty(cl *Client) (*Client, *gerr.GatewayDError)
	IsExhausted() bool
	Shutdown()
	BusyConnections() []string
}

type Proxy struct {
	AvailableConnections pool.IPool
	busyConnections      pool.IPool
	Logger               zerolog.Logger
	PluginRegistry       *plugin.Registry
	scheduler            *gocron.Scheduler
	ctx                  context.Context //nolint:containedctx
	PluginTimeout        time.Duration

	Elastic             bool
	ReuseElasticClients bool
	HealthCheckPeriod   time.Duration

	// Client is used for elastic Proxy and reconnection
	Client *Client
}

var _ IProxy = &Proxy{}

// NewProxy creates a new Proxy.
func NewProxy(
	ctx context.Context,
	pxy Proxy,
) *Proxy {
	proxyCtx, span := otel.Tracer(config.TracerName).Start(ctx, "NewProxy")
	defer span.End()

	proxy := Proxy{
		AvailableConnections: pxy.AvailableConnections,
		busyConnections:      pool.NewPool(proxyCtx, config.EmptyPoolCapacity),
		Logger:               pxy.Logger,
		PluginRegistry:       pxy.PluginRegistry,
		scheduler:            gocron.NewScheduler(time.UTC),
		ctx:                  proxyCtx,
		PluginTimeout:        pxy.PluginTimeout,
		Elastic:              pxy.Elastic,
		ReuseElasticClients:  pxy.ReuseElasticClients,
		Client:               pxy.Client,
		HealthCheckPeriod:    pxy.HealthCheckPeriod,
	}

	startDelay := time.Now().Add(proxy.HealthCheckPeriod)
	// Schedule the client health check.
	if _, err := proxy.scheduler.Every(proxy.HealthCheckPeriod).SingletonMode().StartAt(startDelay).Do(
		func() {
			now := time.Now()
			proxy.Logger.Trace().Msg("Running the client health check to recycle connection(s).")
			proxy.AvailableConnections.ForEach(func(_, value interface{}) bool {
				if client, ok := value.(*Client); ok {
					// Connection is probably dead by now.
					proxy.AvailableConnections.Remove(client.ID)
					client.Close()
					// Create a new client.
					client = NewClient(proxyCtx, proxy.Client)
					if client != nil && client.ID != "" {
						if err := proxy.AvailableConnections.Put(client.ID, client); err != nil {
							proxy.Logger.Err(err).Msg("Failed to update the client connection")
							// Close the client, because we don't want to have orphaned connections.
							client.Close()
						}
					} else {
						proxy.Logger.Error().Msg("Failed to create a new client connection")
					}
				}
				return true
			})
			proxy.Logger.Trace().Str("duration", time.Since(now).String()).Msg(
				"Finished the client health check")
			metrics.ProxyHealthChecks.Inc()
		},
	); err != nil {
		proxy.Logger.Error().Err(err).Msg("Failed to schedule the client health check")
		sentry.CaptureException(err)
		span.RecordError(err)
	}

	// Start the scheduler.
	proxy.scheduler.StartAsync()
	proxy.Logger.Info().Fields(
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
func (pr *Proxy) Connect(gconn gnet.Conn) *gerr.GatewayDError {
	_, span := otel.Tracer(config.TracerName).Start(pr.ctx, "Connect")
	defer span.End()

	var clientID string
	// Get the first available client from the pool.
	pr.AvailableConnections.ForEach(func(key, _ interface{}) bool {
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
			client = NewClient(pr.ctx, pr.Client)
			span.AddEvent("Created a new client connection")
			pr.Logger.Debug().Str("id", client.ID[:7]).Msg("Reused the client connection")
		} else {
			span.AddEvent(gerr.ErrPoolExhausted.Error())
			return gerr.ErrPoolExhausted
		}
	} else {
		// Get the client from the pool with the given clientID.
		if cl, ok := pr.AvailableConnections.Pop(clientID).(*Client); ok {
			client = cl
		}
	}

	client, err := pr.IsHealty(client)
	if err != nil {
		pr.Logger.Error().Err(err).Msg("Failed to connect to the client")
		span.RecordError(err)
	}

	if err := pr.busyConnections.Put(gconn, client); err != nil {
		// This should never happen.
		span.RecordError(err)
		return err
	}

	metrics.ProxiedConnections.Inc()

	fields := map[string]interface{}{
		"function": "Proxy.connect",
		"client":   "unknown",
		"server":   gconn.RemoteAddr().String(),
	}
	if client.ID != "" {
		fields["client"] = client.ID[:7]
	}
	pr.Logger.Debug().Fields(fields).Msg("Client has been assigned")

	pr.Logger.Debug().Fields(
		map[string]interface{}{
			"function": "Proxy.connect",
			"count":    pr.AvailableConnections.Size(),
		},
	).Msg("Available client connections")
	pr.Logger.Debug().Fields(
		map[string]interface{}{
			"function": "Proxy.connect",
			"count":    pr.busyConnections.Size(),
		},
	).Msg("Busy client connections")

	return nil
}

// Disconnect removes the client from the busy connection pool and tries to recycle
// the server connection.
func (pr *Proxy) Disconnect(gconn gnet.Conn) *gerr.GatewayDError {
	_, span := otel.Tracer(config.TracerName).Start(pr.ctx, "Disconnect")
	defer span.End()

	client := pr.busyConnections.Pop(gconn)
	//nolint:nestif
	if client != nil {
		if client, ok := client.(*Client); ok {
			if (pr.Elastic && pr.ReuseElasticClients) || !pr.Elastic {
				_, err := pr.IsHealty(client)
				if err != nil {
					pr.Logger.Error().Err(err).Msg("Failed to reconnect to the client")
					span.RecordError(err)
				}
				// If the client is not in the pool, put it back.
				err = pr.AvailableConnections.Put(client.ID, client)
				if err != nil {
					pr.Logger.Error().Err(err).Msg("Failed to put the client back in the pool")
					span.RecordError(err)
				}
			} else {
				span.RecordError(gerr.ErrClientNotConnected)
				return gerr.ErrClientNotConnected
			}
		} else {
			// This should never happen, but if it does,
			// then there are some serious issues with the pool.
			span.RecordError(gerr.ErrCastFailed)
			return gerr.ErrCastFailed
		}
	} else {
		span.RecordError(gerr.ErrClientNotFound)
		return gerr.ErrClientNotFound
	}

	metrics.ProxiedConnections.Dec()

	pr.Logger.Debug().Fields(
		map[string]interface{}{
			"function": "Proxy.disconnect",
			"count":    pr.AvailableConnections.Size(),
		},
	).Msg("Available client connections")
	pr.Logger.Debug().Fields(
		map[string]interface{}{
			"function": "Proxy.disconnect",
			"count":    pr.busyConnections.Size(),
		},
	).Msg("Busy client connections")

	return nil
}

// PassThrough sends the data from the client to the server and vice versa.
func (pr *Proxy) PassThrough(gconn gnet.Conn) *gerr.GatewayDError {
	_, span := otel.Tracer(config.TracerName).Start(pr.ctx, "PassThrough")
	defer span.End()
	// TODO: Handle bi-directional traffic
	// Currently the passthrough is a one-way street from the client to the server, that is,
	// the client can send data to the server and receive the response back, but the server
	// cannot take initiative and send data to the client. So, there should be another event-loop
	// that listens for data from the server and sends it to the client.

	var client *Client
	if pr.busyConnections.Get(gconn) == nil {
		span.RecordError(gerr.ErrClientNotFound)
		return gerr.ErrClientNotFound
	}

	// Get the client from the busy connection pool.
	if cl, ok := pr.busyConnections.Get(gconn).(*Client); ok {
		client = cl
	} else {
		span.RecordError(gerr.ErrCastFailed)
		return gerr.ErrCastFailed
	}
	span.AddEvent("Got the client from the busy connection pool")

	// Receive the request from the client.
	request, origErr := pr.receiveTrafficFromClient(gconn)
	span.AddEvent("Received traffic from client")

	pluginTimeoutCtx, cancel := context.WithTimeout(context.Background(), pr.PluginTimeout)
	defer cancel()
	// Run the OnTrafficFromClient hooks.
	result, err := pr.PluginRegistry.Run(
		pluginTimeoutCtx,
		trafficData(
			gconn,
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
		pr.Logger.Error().Err(err).Msg("Error running hook")
		span.RecordError(err)
	}
	span.AddEvent("Ran the OnTrafficFromClient hooks")

	// If the hook wants to terminate the connection, do it.
	if pr.shouldTerminate(result) {
		if modResponse, modReceived := pr.getPluginModifiedResponse(result); modResponse != nil {
			metrics.ProxyPassThroughs.Inc()
			metrics.ProxyPassThroughTerminations.Inc()
			metrics.BytesSentToClient.Observe(float64(modReceived))
			metrics.TotalTrafficBytes.Observe(float64(modReceived))

			span.AddEvent("Terminating connection")
			return pr.sendTrafficToClient(gconn, modResponse, modReceived)
		}
		span.RecordError(gerr.ErrHookTerminatedConnection)
		return gerr.ErrHookTerminatedConnection
	}
	// If the hook modified the request, use the modified request.
	if modRequest := pr.getPluginModifiedRequest(result); modRequest != nil {
		request = modRequest
		span.AddEvent("Plugin(s) modified the request")
	}

	// Send the request to the server.
	_, err = pr.sendTrafficToServer(client, request)
	span.AddEvent("Sent traffic to server")

	// Run the OnTrafficToServer hooks.
	_, err = pr.PluginRegistry.Run(
		pluginTimeoutCtx,
		trafficData(
			gconn,
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
		pr.Logger.Error().Err(err).Msg("Error running hook")
		span.RecordError(err)
	}
	span.AddEvent("Ran the OnTrafficToServer hooks")

	// Receive the response from the server.
	received, response, err := pr.receiveTrafficFromServer(client)
	span.AddEvent("Received traffic from server")

	// The connection to the server is closed, so we MUST reconnect,
	// otherwise the client will be stuck.
	if IsConnClosed(received, err) || IsConnTimedOut(err) {
		pr.Logger.Debug().Fields(
			map[string]interface{}{
				"function": "Proxy.passthrough",
				"local":    client.Conn.LocalAddr().String(),
				"remote":   client.Conn.RemoteAddr().String(),
			}).Msg("Client disconnected")

		client.Close()
		client = NewClient(pr.ctx, pr.Client)
		pr.busyConnections.Remove(gconn)
		if err := pr.busyConnections.Put(gconn, client); err != nil {
			span.RecordError(err)
			// This should never happen
			return err
		}
	}

	// If the response is empty, don't send anything, instead just close the ingress connection.
	if received == 0 {
		pr.Logger.Debug().Fields(
			map[string]interface{}{
				"function": "Proxy.passthrough",
				"local":    client.Conn.LocalAddr().String(),
				"remote":   client.Conn.RemoteAddr().String(),
			}).Msg("No data to send to client")
		span.AddEvent("No data to send to client")
		span.RecordError(err)
		return err
	}

	// Run the OnTrafficFromServer hooks.
	result, err = pr.PluginRegistry.Run(
		pluginTimeoutCtx,
		trafficData(
			gconn,
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
		pr.Logger.Error().Err(err).Msg("Error running hook")
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
	errVerdict := pr.sendTrafficToClient(gconn, response, received)
	span.AddEvent("Sent traffic to client")

	// Run the OnTrafficToClient hooks.
	_, err = pr.PluginRegistry.Run(
		pluginTimeoutCtx,
		trafficData(
			gconn,
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
			err,
		),
		v1.HookName_HOOK_NAME_ON_TRAFFIC_TO_CLIENT)
	if err != nil {
		pr.Logger.Error().Err(err).Msg("Error running hook")
		span.RecordError(err)
	}

	if errVerdict != nil {
		span.RecordError(errVerdict)
	}

	metrics.ProxyPassThroughs.Inc()

	return errVerdict
}

// IsHealty checks if the pool is exhausted or the client is disconnected.
func (pr *Proxy) IsHealty(client *Client) (*Client, *gerr.GatewayDError) {
	_, span := otel.Tracer(config.TracerName).Start(pr.ctx, "IsHealty")
	defer span.End()

	if pr.IsExhausted() {
		pr.Logger.Error().Msg("No more available connections")
		span.RecordError(gerr.ErrPoolExhausted)
		return client, gerr.ErrPoolExhausted
	}

	if !client.IsConnected() {
		pr.Logger.Error().Msg("Client is disconnected")
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
	return pr.AvailableConnections.Size() == 0 && pr.AvailableConnections.Cap() > 0
}

// Shutdown closes all connections and clears the connection pools.
func (pr *Proxy) Shutdown() {
	_, span := otel.Tracer(config.TracerName).Start(pr.ctx, "Shutdown")
	defer span.End()

	pr.AvailableConnections.ForEach(func(key, value interface{}) bool {
		if cl, ok := value.(*Client); ok {
			cl.Close()
		}
		return true
	})
	pr.AvailableConnections.Clear()
	pr.Logger.Debug().Msg("All available connections have been closed")

	pr.busyConnections.ForEach(func(key, value interface{}) bool {
		if gconn, ok := key.(gnet.Conn); ok {
			gconn.Close()
		}
		if cl, ok := value.(*Client); ok {
			cl.Close()
		}
		return true
	})
	pr.busyConnections.Clear()
	pr.scheduler.Clear()
	pr.Logger.Debug().Msg("All busy connections have been closed")
}

// BusyConnections returns a list of busy connections.
func (pr *Proxy) BusyConnections() []string {
	_, span := otel.Tracer(config.TracerName).Start(pr.ctx, "BusyConnections")
	defer span.End()

	connections := make([]string, 0)
	pr.busyConnections.ForEach(func(key, _ interface{}) bool {
		if gconn, ok := key.(gnet.Conn); ok {
			connections = append(connections, gconn.RemoteAddr().String())
		}
		return true
	})
	return connections
}

// receiveTrafficFromClient is a function that receives data from the client.
func (pr *Proxy) receiveTrafficFromClient(gconn gnet.Conn) ([]byte, error) {
	_, span := otel.Tracer(config.TracerName).Start(pr.ctx, "receiveTrafficFromClient")
	defer span.End()

	// request contains the data from the client.
	request, err := gconn.Next(-1)
	if err != nil {
		pr.Logger.Error().Err(err).Msg("Error reading from client")
		span.RecordError(err)
	}
	pr.Logger.Debug().Fields(
		map[string]interface{}{
			"length": len(request),
			"local":  gconn.LocalAddr().String(),
			"remote": gconn.RemoteAddr().String(),
		},
	).Msg("Received data from client")

	metrics.BytesReceivedFromClient.Observe(float64(len(request)))
	metrics.TotalTrafficBytes.Observe(float64(len(request)))

	//nolint:wrapcheck
	return request, err
}

// sendTrafficToServer is a function that sends data to the server.
func (pr *Proxy) sendTrafficToServer(client *Client, request []byte) (int, *gerr.GatewayDError) {
	_, span := otel.Tracer(config.TracerName).Start(pr.ctx, "sendTrafficToServer")
	defer span.End()

	// Send the request to the server.
	sent, err := client.Send(request)
	if err != nil {
		pr.Logger.Error().Err(err).Msg("Error sending request to database")
		span.RecordError(err)
	}
	pr.Logger.Debug().Fields(
		map[string]interface{}{
			"function": "Proxy.passthrough",
			"length":   sent,
			"local":    client.Conn.LocalAddr().String(),
			"remote":   client.Conn.RemoteAddr().String(),
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
	pr.Logger.Debug().Fields(
		map[string]interface{}{
			"function": "Proxy.passthrough",
			"length":   received,
			"local":    client.Conn.LocalAddr().String(),
			"remote":   client.Conn.RemoteAddr().String(),
		},
	).Msg("Received data from database")

	metrics.BytesReceivedFromServer.Observe(float64(received))
	metrics.TotalTrafficBytes.Observe(float64(received))

	return received, response, err
}

// sendTrafficToClient is a function that sends data to the client.
func (pr *Proxy) sendTrafficToClient(
	gconn gnet.Conn, response []byte, received int,
) *gerr.GatewayDError {
	_, span := otel.Tracer(config.TracerName).Start(pr.ctx, "sendTrafficToClient")
	defer span.End()

	// Send the response to the client async.
	origErr := gconn.AsyncWrite(response[:received], func(gconn gnet.Conn, err error) error {
		pr.Logger.Debug().Fields(
			map[string]interface{}{
				"function": "Proxy.passthrough",
				"length":   received,
				"local":    gconn.LocalAddr().String(),
				"remote":   gconn.RemoteAddr().String(),
			},
		).Msg("Sent data to client")
		span.RecordError(err)
		return err
	})
	if origErr != nil {
		pr.Logger.Error().Err(origErr).Msg("Error writing to client")
		span.RecordError(origErr)
		return gerr.ErrServerSendFailed.Wrap(origErr)
	}

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
			pr.Logger.Debug().Str("function", "Proxy.passthrough").Msg("Terminating connection")
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
	if modRequest, errMsg, convErr := extractFieldValue(result, "request"); errMsg != "" {
		pr.Logger.Error().Str("error", errMsg).Msg("Error in hook")
	} else if convErr != nil {
		pr.Logger.Error().Err(convErr).Msg("Error in data conversion")
		span.RecordError(convErr)
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
	if modResponse, errMsg, convErr := extractFieldValue(result, "response"); errMsg != "" {
		pr.Logger.Error().Str("error", errMsg).Msg("Error in hook")
	} else if convErr != nil {
		pr.Logger.Error().Err(convErr).Msg("Error in data conversion")
		span.RecordError(convErr)
	} else if modResponse != nil {
		return modResponse, len(modResponse)
	}

	return nil, 0
}
