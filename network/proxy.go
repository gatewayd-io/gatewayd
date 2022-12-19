package network

import (
	"context"

	gerr "github.com/gatewayd-io/gatewayd/errors"
	"github.com/gatewayd-io/gatewayd/plugin"
	"github.com/gatewayd-io/gatewayd/pool"
	"github.com/panjf2000/gnet/v2"
	"github.com/rs/zerolog"
	"golang.org/x/exp/maps"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	EmptyPoolCapacity int = 0
)

type Proxy interface {
	Connect(gconn gnet.Conn) error
	Disconnect(gconn gnet.Conn) error
	PassThrough(gconn gnet.Conn) error
	TryReconnect(cl *Client) (*Client, error)
	Shutdown()
	IsExhausted() bool
}

type ProxyImpl struct {
	availableConnections pool.Pool
	busyConnections      pool.Pool
	logger               zerolog.Logger
	hookConfig           *plugin.HookConfig

	Elastic             bool
	ReuseElasticClients bool

	// ClientConfig is used for elastic proxy and reconnection
	ClientConfig *Client
}

var _ Proxy = &ProxyImpl{}

func NewProxy(
	p pool.Pool, hookConfig *plugin.HookConfig,
	elastic, reuseElasticClients bool,
	clientConfig *Client, logger zerolog.Logger,
) *ProxyImpl {
	return &ProxyImpl{
		availableConnections: p,
		busyConnections:      pool.NewPool(EmptyPoolCapacity),
		logger:               logger,
		hookConfig:           hookConfig,
		Elastic:              elastic,
		ReuseElasticClients:  reuseElasticClients,
		ClientConfig:         clientConfig,
	}
}

func (pr *ProxyImpl) Connect(gconn gnet.Conn) error {
	var clientID string
	// Get the first available client from the pool
	pr.availableConnections.ForEach(func(key, _ interface{}) bool {
		if cid, ok := key.(string); ok {
			clientID = cid
			return false // stop the loop
		}
		return true
	})

	var client *Client
	if pr.IsExhausted() {
		// Pool is exhausted or is elastic
		if pr.Elastic {
			// Create a new client
			client = NewClient(
				pr.ClientConfig.Network,
				pr.ClientConfig.Address,
				pr.ClientConfig.ReceiveBufferSize,
				pr.logger,
			)
			pr.logger.Debug().Msgf("Reused the client %s by putting it back in the pool", client.ID)
		} else {
			return gerr.ErrPoolExhausted
		}
	} else {
		// Get the client from the pool with the given clientID
		if cl, ok := pr.availableConnections.Pop(clientID).(*Client); ok {
			client = cl
		}
	}

	client, err := pr.TryReconnect(client)
	if err != nil {
		return err
	}

	if err := pr.busyConnections.Put(gconn, client); err != nil {
		// This should never happen
		return gerr.ErrPutFailed
	}
	pr.logger.Debug().Msgf(
		"Client %s has been assigned to %s", client.ID, gconn.RemoteAddr().String())

	pr.logger.Debug().Str("function", "Proxy.Connect").Msgf(
		"There are %d available clients", pr.availableConnections.Size())
	pr.logger.Debug().Str("function", "Proxy.Connect").Msgf(
		"There are %d busy clients", pr.busyConnections.Size())

	return nil
}

func (pr *ProxyImpl) Disconnect(gconn gnet.Conn) error {
	client := pr.busyConnections.Pop(gconn)
	//nolint:nestif
	if client != nil {
		if cl, ok := client.(*Client); ok {
			if (pr.Elastic && pr.ReuseElasticClients) || !pr.Elastic {
				cl, err := pr.TryReconnect(cl)
				if err != nil {
					return err
				}
				// If the client is not in the pool, put it back
				err = pr.availableConnections.Put(cl.ID, cl)
				if err != nil {
					return err //nolint:wrapcheck
				}
			} else {
				return gerr.ErrClientNotConnected
			}
		} else {
			// This should never happen, but if it does,
			// then there are some serious issues with the pool
			return gerr.ErrCastFailed
		}
	} else {
		return gerr.ErrClientNotFound
	}

	pr.logger.Debug().Str("function", "Proxy.Disconnect").Msgf(
		"There are %d available clients", pr.availableConnections.Size())
	pr.logger.Debug().Str("function", "Proxy.Disconnect").Msgf(
		"There are %d busy clients", pr.busyConnections.Size())

	return nil
}

//nolint:funlen
func (pr *ProxyImpl) PassThrough(gconn gnet.Conn) error {
	// TODO: Handle bi-directional traffic
	// Currently the passthrough is a one-way street from the client to the server, that is,
	// the client can send data to the server and receive the response back, but the server
	// cannot take initiative and send data to the client. So, there should be another event-loop
	// that listens for data from the server and sends it to the client

	var client *Client
	if pr.busyConnections.Get(gconn) == nil {
		return gerr.ErrClientNotFound
	}

	if cl, ok := pr.busyConnections.Get(gconn).(*Client); ok {
		client = cl
	} else {
		return gerr.ErrCastFailed
	}

	// buf contains the data from the client (<type>, length, query)
	buf, err := gconn.Next(-1)
	if err != nil {
		pr.logger.Error().Err(err).Msgf("Error reading from client: %v", err)
	}
	pr.logger.Debug().Fields(
		map[string]interface{}{
			"length": len(buf),
			"local":  gconn.LocalAddr().String(),
			"remote": gconn.RemoteAddr().String(),
		},
	).Msg("Received data from client")

	addresses := map[string]interface{}{
		"client": map[string]interface{}{
			"local":  gconn.LocalAddr().String(),
			"remote": gconn.RemoteAddr().String(),
		},
		"server": map[string]interface{}{
			"local":  client.Conn.LocalAddr().String(),
			"remote": client.Conn.RemoteAddr().String(),
		},
	}

	ingress := map[string]interface{}{
		"buffer": buf, // Will be converted to base64-encoded string
		"error":  "",
	}
	if err != nil {
		ingress["error"] = err.Error()
	}
	maps.Copy(ingress, addresses)

	//nolint:nestif
	if ingressData, intErr := structpb.NewStruct(ingress); intErr != nil {
		pr.logger.Error().Err(intErr).Msgf("Error creating ingress data: %v", err)
	} else {
		result, err := pr.hookConfig.Run(
			context.Background(),
			ingressData,
			plugin.OnIngressTraffic,
			pr.hookConfig.Verification)
		if err != nil {
			pr.logger.Error().Err(err).Msgf("Error running hook: %v", err)
		}

		if result != nil {
			if buffer, ok := result.AsMap()["buffer"].([]byte); ok {
				buf = buffer
			}
			if errMsg, ok := result.AsMap()["error"].(string); ok && errMsg != "" {
				pr.logger.Error().Msgf("Error in hook: %s", errMsg)
			}
		}
	}

	// Send the query to the server
	sent, err := client.Send(buf)
	if err != nil {
		// return err
		pr.logger.Error().Err(err).Msgf("Error sending data to database")
	}
	pr.logger.Debug().Fields(
		map[string]interface{}{
			"function": "Proxy.PassThrough",
			"length":   sent,
			"local":    client.Conn.LocalAddr().String(),
			"remote":   client.Conn.RemoteAddr().String(),
		},
	).Msg("Sent data to database")

	// Receive the response from the server
	received, response, err := client.Receive()
	pr.logger.Debug().Fields(
		map[string]interface{}{
			"function": "Proxy.PassThrough",
			"length":   received,
			"local":    client.Conn.LocalAddr().String(),
			"remote":   client.Conn.RemoteAddr().String(),
		},
	).Msg("Received data from database")

	egress := map[string]interface{}{
		"response": response[:received], // Will be converted to base64-encoded string
		"error":    "",
	}
	if err != nil {
		egress["error"] = err.Error()
	}
	maps.Copy(egress, addresses)

	//nolint:nestif
	if egressData, intErr := structpb.NewStruct(egress); intErr != nil {
		pr.logger.Error().Err(intErr).Msgf("Error creating egress data: %v", err)
	} else {
		result, err := pr.hookConfig.Run(
			context.Background(),
			egressData,
			plugin.OnEgressTraffic,
			pr.hookConfig.Verification)
		if err != nil {
			pr.logger.Error().Err(err).Msgf("Error running hook: %v", err)
		}

		if result != nil {
			if resp, ok := result.AsMap()["response"].([]byte); ok {
				response = resp
			}
			if errMsg, ok := result.AsMap()["error"].(string); ok && errMsg != "" {
				pr.logger.Error().Msgf("Error in hook: %s", errMsg)
			}
		}
	}

	err = gconn.AsyncWrite(response[:received], func(gconn gnet.Conn, err error) error {
		pr.logger.Debug().Fields(
			map[string]interface{}{
				"function": "Proxy.PassThrough",
				"length":   received,
				"local":    gconn.LocalAddr().String(),
				"remote":   gconn.RemoteAddr().String(),
			},
		).Msg("Sent data to client")
		return err
	})
	if err != nil {
		pr.logger.Error().Err(err).Msgf("Error writing to client")
		return err //nolint:wrapcheck
	}

	return nil
}

func (pr *ProxyImpl) TryReconnect(client *Client) (*Client, error) {
	// TODO: try retriable connection?

	if pr.IsExhausted() {
		pr.logger.Error().Msg("No more available connections :: TryReconnect")
		return client, gerr.ErrPoolExhausted
	}

	if !client.IsConnected() {
		pr.logger.Error().Msg("Client is disconnected")
		return client, gerr.ErrClientNotConnected
	}

	return client, nil
}

func (pr *ProxyImpl) Shutdown() {
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
	pr.logger.Debug().Msg("All busy connections have been closed")
}

func (pr *ProxyImpl) IsExhausted() bool {
	if pr.Elastic {
		return false
	}

	return pr.availableConnections.Size() == 0 && pr.availableConnections.Cap() > 0
}
