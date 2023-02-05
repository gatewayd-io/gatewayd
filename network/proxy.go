package network

import (
	"context"
	"time"

	sdkPlugin "github.com/gatewayd-io/gatewayd-plugin-sdk/plugin"
	"github.com/gatewayd-io/gatewayd/config"
	gerr "github.com/gatewayd-io/gatewayd/errors"
	"github.com/gatewayd-io/gatewayd/metrics"
	"github.com/gatewayd-io/gatewayd/plugin"
	"github.com/gatewayd-io/gatewayd/pool"
	"github.com/getsentry/sentry-go"
	"github.com/go-co-op/gocron"
	"github.com/panjf2000/gnet/v2"
	"github.com/rs/zerolog"
)

type IProxy interface {
	Connect(gconn gnet.Conn) *gerr.GatewayDError
	Disconnect(gconn gnet.Conn) *gerr.GatewayDError
	PassThrough(gconn gnet.Conn) *gerr.GatewayDError
	IsHealty(cl *Client) (*Client, *gerr.GatewayDError)
	IsExhausted() bool
	Shutdown()
}

type Proxy struct {
	availableConnections pool.IPool
	busyConnections      pool.IPool
	logger               zerolog.Logger
	pluginRegistry       *plugin.Registry
	scheduler            *gocron.Scheduler

	Elastic             bool
	ReuseElasticClients bool
	HealthCheckPeriod   time.Duration

	// ClientConfig is used for elastic proxy and reconnection
	ClientConfig *config.Client
}

var _ IProxy = &Proxy{}

// NewProxy creates a new proxy.
func NewProxy(
	connPool pool.IPool, pluginRegistry *plugin.Registry,
	elastic, reuseElasticClients bool,
	healthCheckPeriod time.Duration,
	clientConfig *config.Client, logger zerolog.Logger,
) *Proxy {
	proxy := Proxy{
		availableConnections: connPool,
		busyConnections:      pool.NewPool(config.EmptyPoolCapacity),
		logger:               logger,
		pluginRegistry:       pluginRegistry,
		scheduler:            gocron.NewScheduler(time.UTC),
		Elastic:              elastic,
		ReuseElasticClients:  reuseElasticClients,
		ClientConfig:         clientConfig,
	}

	if healthCheckPeriod == 0 {
		proxy.HealthCheckPeriod = config.DefaultHealthCheckPeriod
	} else {
		proxy.HealthCheckPeriod = healthCheckPeriod
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
					client = NewClient(proxy.ClientConfig, proxy.logger)
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
	}

	// Start the scheduler.
	proxy.scheduler.StartAsync()
	logger.Info().Fields(
		map[string]interface{}{
			"startDelay":        startDelay,
			"healthCheckPeriod": proxy.HealthCheckPeriod.String(),
		},
	).Msg("Started the client health check scheduler")

	return &proxy
}

// Connect maps a server connection from the available connection pool to a incoming connection.
// It returns an error if the pool is exhausted. If the pool is elastic, it creates a new client
// and maps it to the incoming connection.
func (pr *Proxy) Connect(gconn gnet.Conn) *gerr.GatewayDError {
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
			client = NewClient(pr.ClientConfig, pr.logger)
			pr.logger.Debug().Str("id", client.ID[:7]).Msg("Reused the client connection")
		} else {
			return gerr.ErrPoolExhausted
		}
	} else {
		// Get the client from the pool with the given clientID.
		if cl, ok := pr.availableConnections.Pop(clientID).(*Client); ok {
			client = cl
		}
	}

	client, err := pr.IsHealty(client)
	if err != nil {
		pr.logger.Error().Err(err).Msg("Failed to connect to the client")
	}

	if err := pr.busyConnections.Put(gconn, client); err != nil {
		// This should never happen.
		return err
	}

	metrics.ProxiedConnections.Inc()

	pr.logger.Debug().Fields(
		map[string]interface{}{
			"function": "proxy.connect",
			"client":   client.ID[:7],
			"server":   gconn.RemoteAddr().String(),
		},
	).Msg("Client has been assigned")

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
func (pr *Proxy) Disconnect(gconn gnet.Conn) *gerr.GatewayDError {
	client := pr.busyConnections.Pop(gconn)
	//nolint:nestif
	if client != nil {
		if client, ok := client.(*Client); ok {
			if (pr.Elastic && pr.ReuseElasticClients) || !pr.Elastic {
				_, err := pr.IsHealty(client)
				if err != nil {
					pr.logger.Error().Err(err).Msg("Failed to reconnect to the client")
				}
				// If the client is not in the pool, put it back.
				err = pr.availableConnections.Put(client.ID, client)
				if err != nil {
					pr.logger.Error().Err(err).Msg("Failed to put the client back in the pool")
				}
			} else {
				return gerr.ErrClientNotConnected
			}
		} else {
			// This should never happen, but if it does,
			// then there are some serious issues with the pool.
			return gerr.ErrCastFailed
		}
	} else {
		return gerr.ErrClientNotFound
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

// PassThrough sends the data from the client to the server and vice versa.
//
// TODO: refactor this mess! My eye burns even looking at it.
func (pr *Proxy) PassThrough(gconn gnet.Conn) *gerr.GatewayDError {
	// TODO: Handle bi-directional traffic
	// Currently the passthrough is a one-way street from the client to the server, that is,
	// the client can send data to the server and receive the response back, but the server
	// cannot take initiative and send data to the client. So, there should be another event-loop
	// that listens for data from the server and sends it to the client.

	var client *Client
	if pr.busyConnections.Get(gconn) == nil {
		return gerr.ErrClientNotFound
	}

	// Get the client from the busy connection pool.
	if cl, ok := pr.busyConnections.Get(gconn).(*Client); ok {
		client = cl
	} else {
		return gerr.ErrCastFailed
	}

	// getPluginModifiedRequest is a function that retrieves the modified request
	// from the hook result.
	getPluginModifiedRequest := func(result map[string]interface{}) []byte {
		// If the hook modified the request, use the modified request.
		//nolint:gocritic
		if modRequest, errMsg, convErr := extractFieldValue(result, "request"); errMsg != "" {
			pr.logger.Error().Str("error", errMsg).Msg("Error in hook")
		} else if convErr != nil {
			pr.logger.Error().Err(convErr).Msg("Error in data conversion")
		} else if modRequest != nil {
			return modRequest
		}

		return nil
	}

	// getPluginModifiedResponse is a function that retrieves the modified response
	// from the hook result.
	getPluginModifiedResponse := func(result map[string]interface{}) ([]byte, int) {
		// If the hook returns a response, use it instead of the original response.
		//nolint:gocritic
		if modResponse, errMsg, convErr := extractFieldValue(result, "response"); errMsg != "" {
			pr.logger.Error().Str("error", errMsg).Msg("Error in hook")
		} else if convErr != nil {
			pr.logger.Error().Err(convErr).Msg("Error in data conversion")
		} else if modResponse != nil {
			return modResponse, len(modResponse)
		}

		return nil, 0
	}

	// Receive the request from the client.
	request, origErr := pr.receiveTrafficFromClient(gconn)

	// Run the OnTrafficFromClient hooks.
	result, err := pr.pluginRegistry.Run(
		context.Background(),
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
		sdkPlugin.OnTrafficFromClient)
	if err != nil {
		pr.logger.Error().Err(err).Msg("Error running hook")
	}
	// If the hook wants to terminate the connection, do it.
	if pr.shouldTerminate(result) {
		if modResponse, modReceived := getPluginModifiedResponse(result); modResponse != nil {
			metrics.ProxyPassThroughs.Inc()
			metrics.ProxyPassThroughTerminations.Inc()
			metrics.BytesSentToClient.Observe(float64(modReceived))
			metrics.TotalTrafficBytes.Observe(float64(modReceived))

			return pr.sendTrafficToClient(gconn, modResponse, modReceived)
		}
		return gerr.ErrHookTerminatedConnection
	}
	// If the hook modified the request, use the modified request.
	if modRequest := getPluginModifiedRequest(result); modRequest != nil {
		request = modRequest
	}

	// Send the request to the server.
	_, err = pr.sendTrafficToServer(client, request)

	// Run the OnTrafficToServer hooks.
	_, err = pr.pluginRegistry.Run(
		context.Background(),
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
		sdkPlugin.OnTrafficToServer)
	if err != nil {
		pr.logger.Error().Err(err).Msg("Error running hook")
	}

	// Receive the response from the server.
	received, response, err := pr.receiveTrafficFromServer(client)

	// The connection to the server is closed, so we MUST reconnect,
	// otherwise the client will be stuck.
	if IsConnClosed(received, err) || IsConnTimedOut(err) {
		pr.logger.Debug().Fields(
			map[string]interface{}{
				"function": "proxy.passthrough",
				"local":    client.Conn.LocalAddr().String(),
				"remote":   client.Conn.RemoteAddr().String(),
			}).Msg("Client disconnected")

		client.Close()
		client = NewClient(pr.ClientConfig, pr.logger)
		pr.busyConnections.Remove(gconn)
		if err := pr.busyConnections.Put(gconn, client); err != nil {
			// This should never happen
			return err
		}
	}

	// If the response is empty, don't send anything, instead just close the ingress connection.
	if received == 0 {
		pr.logger.Debug().Fields(
			map[string]interface{}{
				"function": "proxy.passthrough",
				"local":    client.Conn.LocalAddr().String(),
				"remote":   client.Conn.RemoteAddr().String(),
			}).Msg("No data to send to client")
		return err
	}

	// Run the OnTrafficFromServer hooks.
	result, err = pr.pluginRegistry.Run(
		context.Background(),
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
		sdkPlugin.OnTrafficFromServer)
	if err != nil {
		pr.logger.Error().Err(err).Msg("Error running hook")
	}
	// If the hook modified the response, use the modified response.
	if modResponse, modReceived := getPluginModifiedResponse(result); modResponse != nil {
		response = modResponse
		received = modReceived
	}

	// Send the response to the client.
	errVerdict := pr.sendTrafficToClient(gconn, response, received)

	// Run the OnTrafficToClient hooks.
	_, err = pr.pluginRegistry.Run(
		context.Background(),
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
		sdkPlugin.OnTrafficToClient)
	if err != nil {
		pr.logger.Error().Err(err).Msg("Error running hook")
	}

	metrics.ProxyPassThroughs.Inc()

	return errVerdict
}

// IsHealty checks if the pool is exhausted or the client is disconnected.
func (pr *Proxy) IsHealty(client *Client) (*Client, *gerr.GatewayDError) {
	if pr.IsExhausted() {
		pr.logger.Error().Msg("No more available connections")
		return client, gerr.ErrPoolExhausted
	}

	if !client.IsConnected() {
		pr.logger.Error().Msg("Client is disconnected")
	}

	return client, nil
}

// IsExhausted checks if the available connection pool is exhausted.
func (pr *Proxy) IsExhausted() bool {
	if pr.Elastic {
		return false
	}

	return pr.availableConnections.Size() == 0 && pr.availableConnections.Cap() > 0
}

// Shutdown closes all connections and clears the connection pools.
func (pr *Proxy) Shutdown() {
	pr.availableConnections.ForEach(func(key, value interface{}) bool {
		if cl, ok := value.(*Client); ok {
			cl.Close()
		}
		return true
	})
	pr.availableConnections.Clear()
	pr.logger.Debug().Msg("All available connections have been closed")

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
	pr.logger.Debug().Msg("All busy connections have been closed")
}

// receiveTrafficFromClient is a function that receives data from the client.
func (pr *Proxy) receiveTrafficFromClient(gconn gnet.Conn) ([]byte, error) {
	// request contains the data from the client.
	request, err := gconn.Next(-1)
	if err != nil {
		pr.logger.Error().Err(err).Msg("Error reading from client")
	}
	pr.logger.Debug().Fields(
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
	// Send the request to the server.
	sent, err := client.Send(request)
	if err != nil {
		pr.logger.Error().Err(err).Msg("Error sending request to database")
	}
	pr.logger.Debug().Fields(
		map[string]interface{}{
			"function": "proxy.passthrough",
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
	// Receive the response from the server.
	received, response, err := client.Receive()
	pr.logger.Debug().Fields(
		map[string]interface{}{
			"function": "proxy.passthrough",
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
	// Send the response to the client async.
	origErr := gconn.AsyncWrite(response[:received], func(gconn gnet.Conn, err error) error {
		pr.logger.Debug().Fields(
			map[string]interface{}{
				"function": "proxy.passthrough",
				"length":   received,
				"local":    gconn.LocalAddr().String(),
				"remote":   gconn.RemoteAddr().String(),
			},
		).Msg("Sent data to client")
		return err
	})
	if origErr != nil {
		pr.logger.Error().Err(origErr).Msg("Error writing to client")
		return gerr.ErrServerSendFailed.Wrap(origErr)
	}

	metrics.BytesSentToClient.Observe(float64(received))
	metrics.TotalTrafficBytes.Observe(float64(received))

	return nil
}

// shouldTerminate is a function that retrieves the terminate field from the hook result.
// Only the OnTrafficFromClient hook will terminate the connection.
func (pr *Proxy) shouldTerminate(result map[string]interface{}) bool {
	// If the hook wants to terminate the connection, do it.
	if result != nil {
		if terminate, ok := result["terminate"].(bool); ok && terminate {
			pr.logger.Debug().Str("function", "proxy.passthrough").Msg("Terminating connection")
			return true
		}
	}

	return false
}
