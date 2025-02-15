package raft

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gatewayd-io/gatewayd/config"
	"github.com/gatewayd-io/gatewayd/logging"
	"github.com/gatewayd-io/gatewayd/metrics"
	pb "github.com/gatewayd-io/gatewayd/raft/proto"
	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Configuration constants for Raft operations.
const (
	CommandAddConsistentHashEntry = "ADD_CONSISTENT_HASH_ENTRY"
	CommandAddRoundRobinNext      = "ADD_ROUND_ROBIN_NEXT"
	CommandUpdateWeightedRR       = "UPDATE_WEIGHTED_RR"
	CommandUpdateWeightedRRBatch  = "UPDATE_WEIGHTED_RR_BATCH"
	CommandAddPeer                = "ADD_PEER"
	CommandRemovePeer             = "REMOVE_PEER"
	RaftLeaderState               = raft.Leader
	LeaderElectionTimeout         = 3 * time.Second
	maxSnapshots                  = 3                      // Maximum number of snapshots to retain
	maxPool                       = 3                      // Maximum number of connections to pool
	transportTimeout              = 10 * time.Second       // Timeout for transport operations
	leadershipCheckInterval       = 10 * time.Second       // Interval for checking leadership status
	ApplyTimeout                  = 2 * time.Second        // Timeout for applying commands
	peerConnectionInterval        = 5 * time.Second        // Interval between peer connection attempts
	clusterJoinTimeout            = 5 * time.Minute        // Total timeout for trying to join cluster
	leaderWaitRetryInterval       = 100 * time.Millisecond // Interval for retrying leader wait
	forwardRetryInterval          = 100 * time.Millisecond // Interval for retrying forward requests
	peerSyncInterval              = 30 * time.Second       // Interval for peer synchronization checks
)

// Command represents a general command structure for all operations.
type Command struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// ConsistentHashPayload represents the payload for consistent hash operations.
type ConsistentHashPayload struct {
	Hash      string `json:"hash"`
	BlockName string `json:"blockName"`
}

// RoundRobinPayload represents the payload for round robin operations.
type RoundRobinPayload struct {
	NextIndex uint32 `json:"nextIndex"`
	GroupName string `json:"groupName"`
}

// WeightedProxy represents the weight structure for WeightedRoundRobin operations.
type WeightedProxy struct {
	CurrentWeight   int `json:"currentWeight"`
	EffectiveWeight int `json:"effectiveWeight"`
}

// WeightedRRPayload represents the payload for WeightedRoundRobin operations.
type WeightedRRPayload struct {
	GroupName string        `json:"groupName"`
	ProxyName string        `json:"proxyName"`
	Weight    WeightedProxy `json:"weight"`
}

// WeightedRRBatchPayload represents the payload for batch updates of WeightedRoundRobin operations.
type WeightedRRBatchPayload struct {
	GroupName string                   `json:"groupName"`
	Updates   map[string]WeightedProxy `json:"updates"`
}

// PeerPayload represents the payload for peer operations.
type PeerPayload struct {
	ID          string `json:"id"`
	Address     string `json:"address"`
	GRPCAddress string `json:"grpcAddress"`
}

// Node represents a node in the Raft cluster.
type Node struct {
	raft          *raft.Raft
	config        *raft.Config
	Fsm           *FSM
	logStore      raft.LogStore
	stableStore   raft.StableStore
	snapshotStore raft.SnapshotStore
	transport     raft.Transport
	Logger        zerolog.Logger
	Peers         []config.RaftPeer
	rpcServer     *grpc.Server
	rpcClient     *rpcClient
	grpcAddr      string
	grpcIsSecure  bool
}

// NewRaftNode creates and initializes a new Raft node.
func NewRaftNode(logger zerolog.Logger, raftConfig config.Raft) (*Node, error) {
	RaftNodeConfig := raft.DefaultConfig()

	// Create HcLogAdapter to wrap zerolog logger
	hcLogger := logging.NewHcLogAdapter(&logger, "raft")
	RaftNodeConfig.Logger = hcLogger

	var err error
	nodeID := raftConfig.NodeID
	if raftConfig.NodeID == "" {
		nodeID, err = os.Hostname()
		if err != nil {
			return nil, fmt.Errorf("error getting hostname: %w", err)
		}
	}
	raftAddr := raftConfig.Address
	RaftNodeConfig.LocalID = raft.ServerID(nodeID)
	raftDir := filepath.Join(raftConfig.Directory, nodeID)
	err = os.MkdirAll(raftDir, os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("error creating raft directory: %w", err)
	}

	// Create the FSM
	fsm := NewFSM()

	// Add all peers to FSM if not already present
	fsm.mu.Lock()
	for _, peer := range raftConfig.Peers {
		if _, exists := fsm.raftPeers[peer.ID]; !exists {
			fsm.raftPeers[peer.ID] = peer
		}
	}
	fsm.mu.Unlock()

	// Create the log store and stable store
	logStore, err := raftboltdb.NewBoltStore(filepath.Join(raftDir, "raft-log.db"))
	if err != nil {
		return nil, fmt.Errorf("error creating log store: %w", err)
	}

	stableStore, err := raftboltdb.NewBoltStore(filepath.Join(raftDir, "raft-stable.db"))
	if err != nil {
		return nil, fmt.Errorf("error creating stable store: %w", err)
	}

	// Create the snapshot store
	snapshotStore, err := raft.NewFileSnapshotStore(raftDir, maxSnapshots, os.Stderr)
	if err != nil {
		return nil, fmt.Errorf("error creating snapshot store: %w", err)
	}

	// Setup Raft communication
	addr, err := net.ResolveTCPAddr("tcp", raftAddr)
	if err != nil {
		return nil, fmt.Errorf("error resolving TCP address: %w", err)
	}
	transport, err := raft.NewTCPTransport(raftAddr, addr, maxPool, transportTimeout, os.Stderr)
	if err != nil {
		return nil, fmt.Errorf("error creating TCP transport: %w", err)
	}

	// Create the Raft node
	raftNode, err := raft.NewRaft(RaftNodeConfig, fsm, logStore, stableStore, snapshotStore, transport)
	if err != nil {
		return nil, fmt.Errorf("error creating Raft: %w", err)
	}

	node := &Node{
		raft:          raftNode,
		config:        RaftNodeConfig,
		Fsm:           fsm,
		logStore:      logStore,
		stableStore:   stableStore,
		snapshotStore: snapshotStore,
		transport:     transport,
		Logger:        logger,
		Peers:         raftConfig.Peers,
		grpcAddr:      raftConfig.GRPCAddress,
		grpcIsSecure:  raftConfig.IsSecure,
	}

	// Initialize RPC client
	node.rpcClient = newRPCClient(node)

	// Start RPC server
	if err := node.startGRPCServer(raftConfig.CertFile, raftConfig.KeyFile); err != nil {
		return nil, fmt.Errorf("failed to start RPC server: %w", err)
	}

	ctx := context.Background()
	node.StartPeerSynchronizer(ctx)

	// Handle bootstrapping
	if raftConfig.IsBootstrap {
		configuration := raft.Configuration{
			Servers: make([]raft.Server, len(node.Peers)),
		}
		for i, peer := range node.Peers {
			configuration.Servers[i] = raft.Server{
				ID:      raft.ServerID(peer.ID),
				Address: raft.ServerAddress(peer.Address),
			}
		}

		selfPeer := config.RaftPeer{
			ID:          string(node.config.LocalID),
			Address:     raftAddr,
			GRPCAddress: node.grpcAddr,
		}
		configuration.Servers = append(configuration.Servers, raft.Server{
			ID:      RaftNodeConfig.LocalID,
			Address: transport.LocalAddr(),
		})

		// Add self to both configuration and FSM
		fsm.mu.Lock()
		if _, exists := fsm.raftPeers[string(node.config.LocalID)]; !exists {
			fsm.raftPeers[string(node.config.LocalID)] = selfPeer
		}
		fsm.mu.Unlock()

		node.raft.BootstrapCluster(configuration)
	} else {
		// if peers not exists skip tryConnectToCluster
		if len(node.Peers) == 0 {
			node.Logger.Info().Msg("no peers found, skipping cluster connection")
			return node, nil
		}

		go func() {
			if err := node.tryConnectToCluster(raftAddr); err != nil {
				node.Logger.Error().Err(err).Msg("failed to connect to cluster")
			}
		}()
	}

	return node, nil
}

// tryConnectToCluster attempts to connect to the cluster by sending AddPeer requests to all peers.
// It returns an error if the timeout is reached or if the connection fails.
func (n *Node) tryConnectToCluster(localAddress string) error {
	timeoutCh := time.After(clusterJoinTimeout)
	ticker := time.NewTicker(peerConnectionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-timeoutCh:
			n.Logger.Error().Msg("timeout while trying to connect to cluster")
			return errors.New("timeout while trying to connect to cluster")
		case <-ticker.C:
			for _, peer := range n.Peers {
				client, err := n.rpcClient.getClient(peer.GRPCAddress)
				if err != nil {
					n.Logger.Debug().
						Err(err).
						Str("peer_id", peer.ID).
						Str("grpc_address", peer.GRPCAddress).
						Msg("Failed to get client for peer")
					continue
				}

				ctx, cancel := context.WithTimeout(context.Background(), transportTimeout)
				resp, err := client.AddPeer(ctx, &pb.AddPeerRequest{
					PeerId:      string(n.config.LocalID),
					PeerAddress: localAddress,
					GrpcAddress: n.grpcAddr,
				})
				cancel()

				if err == nil && resp.GetSuccess() {
					n.Logger.Info().
						Str("peer_id", peer.ID).
						Msg("Successfully joined cluster through peer")
					return nil
				}

				n.Logger.Debug().
					Err(err).
					Str("peer_id", peer.ID).
					Msg("Failed to join cluster through peer")
			}
		}
	}
}

func (n *Node) GetPeers() []raft.Server {
	peers := n.raft.GetConfiguration().Configuration().Servers
	return peers
}

// getLeaderClient is a helper function that returns a gRPC client connected to the current leader.
func (n *Node) getLeaderClient() (pb.RaftServiceClient, error) {
	// Wait for leader with timeout
	leaderID, err := n.waitForLeader()
	if err != nil {
		return nil, err
	}

	n.Fsm.mu.RLock()
	peer, exists := n.Fsm.raftPeers[string(leaderID)]
	n.Fsm.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("leader %s not found in peer list", leaderID)
	}

	// Get the RPC client for the leader
	client, err := n.rpcClient.getClient(peer.GRPCAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get client for leader: %w", err)
	}

	return client, nil
}

// waitForLeader waits for a leader to be elected with a timeout.
func (n *Node) waitForLeader() (raft.ServerID, error) {
	ctx, cancel := context.WithTimeout(context.Background(), transportTimeout)
	defer cancel()

	ticker := time.NewTicker(leaderWaitRetryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("timeout waiting for leader: %w", ctx.Err())
		case <-ticker.C:
			_, leaderID := n.raft.LeaderWithID()
			if leaderID != "" {
				return leaderID, nil
			}
		}
	}
}

// AddPeer adds a new peer to the Raft cluster.
func (n *Node) AddPeer(ctx context.Context, peerID, peerAddr, grpcAddr string) error {
	if n.raft.State() != raft.Leader {
		// Get the leader client
		client, err := n.getLeaderClient()
		if err != nil {
			return err
		}

		resp, err := client.AddPeer(ctx, &pb.AddPeerRequest{
			PeerId:      peerID,
			PeerAddress: peerAddr,
			GrpcAddress: grpcAddr,
		})
		if err != nil {
			return fmt.Errorf("failed to forward AddPeer request: %w", err)
		}

		if !resp.GetSuccess() {
			return fmt.Errorf("leader failed to add peer: %s", resp.GetError())
		}

		return nil
	}

	return n.AddPeerInternal(ctx, peerID, peerAddr, grpcAddr)
}

// AddPeerInternal adds a new peer to the Raft cluster.
func (n *Node) AddPeerInternal(ctx context.Context, peerID, peerAddress, grpcAddress string) error {
	if n.raft.State() != raft.Leader {
		return errors.New("only the leader can add peers")
	}

	// there is a chance that leader changed to the new peer, so need to update FSM raftPeers
	n.Fsm.mu.Lock()
	if _, exists := n.Fsm.raftPeers[peerID]; !exists {
		n.Fsm.raftPeers[peerID] = config.RaftPeer{
			ID:          peerID,
			Address:     peerAddress,
			GRPCAddress: grpcAddress,
		}
	}
	n.Fsm.mu.Unlock()

	// Create and apply peer command
	cmd := Command{
		Type: CommandAddPeer,
		Payload: PeerPayload{
			ID:          peerID,
			Address:     peerAddress,
			GRPCAddress: grpcAddress,
		},
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("failed to marshal peer command: %w", err)
	}

	if err := n.Apply(ctx, data, ApplyTimeout); err != nil {
		return fmt.Errorf("failed to apply peer command: %w", err)
	}

	// Add to Raft cluster
	future := n.raft.AddVoter(raft.ServerID(peerID), raft.ServerAddress(peerAddress), 0, 0)
	if err := future.Error(); err != nil {
		// need to remove peer from FSM by applying remove peer command
		cmd := Command{
			Type: CommandRemovePeer,
			Payload: PeerPayload{
				ID: peerID,
			},
		}

		data, err := json.Marshal(cmd)
		if err != nil {
			return fmt.Errorf("failed to marshal peer command: %w", err)
		}

		if err := n.Apply(ctx, data, ApplyTimeout); err != nil {
			return fmt.Errorf("failed to apply peer command: %w", err)
		}
		return fmt.Errorf("failed to add peer: %w", err)
	}

	metrics.RaftPeerAdditions.Inc()
	return nil
}

// RemovePeer removes a peer from the Raft cluster.
func (n *Node) RemovePeer(ctx context.Context, peerID string) error {
	_, leaderID := n.raft.LeaderWithID()
	if leaderID == "" {
		return errors.New("no leader available")
	}

	if n.raft.State() != raft.Leader {
		client, err := n.getLeaderClient()
		if err != nil {
			return fmt.Errorf("failed to get leader client: %w", err)
		}

		resp, err := client.RemovePeer(ctx, &pb.RemovePeerRequest{
			PeerId: peerID,
		})
		if err != nil {
			return fmt.Errorf("failed to forward RemovePeer request: %w", err)
		}

		if !resp.GetSuccess() {
			return fmt.Errorf("leader failed to remove peer: %s", resp.GetError())
		}

		return nil
	}

	return n.RemovePeerInternal(ctx, peerID)
}

// RemovePeerInternal removes a peer from the Raft cluster.
func (n *Node) RemovePeerInternal(ctx context.Context, peerID string) error {
	if n.raft.State() != raft.Leader {
		return errors.New("only the leader can remove peers")
	}

	// get peer address and grpc address from FSM
	n.Fsm.mu.RLock()
	peer, exists := n.Fsm.raftPeers[peerID]
	n.Fsm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("peer %s not found in FSM", peerID)
	}

	// send remove peer command to peers
	// need to remove peer from FSM before removing from Raft to handle that leader want to remove itself
	cmd := Command{
		Type: CommandRemovePeer,
		Payload: PeerPayload{
			ID: peerID,
		},
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("failed to marshal peer command: %w", err)
	}

	if err := n.Apply(ctx, data, ApplyTimeout); err != nil {
		return fmt.Errorf("failed to apply peer command: %w", err)
	}

	future := n.raft.RemoveServer(raft.ServerID(peerID), 0, 0)
	if err := future.Error(); err != nil {
		// need to add again peer to FSM by applying add peer command
		cmd := Command{
			Type: CommandAddPeer,
			Payload: PeerPayload{
				ID:          peerID,
				Address:     peer.Address,
				GRPCAddress: peer.GRPCAddress,
			},
		}

		data, err := json.Marshal(cmd)
		if err != nil {
			return fmt.Errorf("failed to marshal peer command: %w", err)
		}

		if err := n.Apply(ctx, data, ApplyTimeout); err != nil {
			return fmt.Errorf("failed to apply peer command: %w", err)
		}
		return fmt.Errorf("failed to remove peer: %w", err)
	}

	metrics.RaftPeerRemovals.Inc()
	return nil
}

// DiscoverPeers discovers new peers and adds them to the cluster.
func (n *Node) DiscoverPeers() error {
	// Implement peer discovery logic here
	// For example, you can use a service registry or DNS to discover new peers

	// Example: Adding a discovered peer
	peerID := "new-peer-id"
	peerAddress := "new-peer-address"
	return n.AddPeer(context.Background(), peerID, peerAddress, "")
}

// GracefulShutdown gracefully removes the node from the cluster.
func (n *Node) GracefulShutdown() error {
	if n.raft.State() != raft.Leader {
		return errors.New("only the leader can initiate graceful shutdown")
	}

	// Remove the current node from the cluster
	peerID := string(n.config.LocalID)
	return n.RemovePeer(context.Background(), peerID)
}

// Apply is the public method that handles forwarding if necessary.
func (n *Node) Apply(ctx context.Context, data []byte, timeout time.Duration) error {
	if n.raft.State() != raft.Leader {
		return n.forwardToLeader(ctx, data, timeout)
	}
	return n.applyInternal(ctx, data, timeout)
}

// applyInternal is the internal method that actually applies the data.
func (n *Node) applyInternal(_ context.Context, data []byte, timeout time.Duration) error {
	future := n.raft.Apply(data, timeout)
	if err := future.Error(); err != nil {
		return fmt.Errorf("failed to apply log entry: %w", err)
	}
	return nil
}

// forwardToLeader forwards a request to the current Raft leader node. It first identifies
// the leader's gRPC address by matching the leader ID with the peer list. Then it establishes
// a gRPC connection to forward the request. The method handles timeouts and returns any errors
// that occur during forwarding.
func (n *Node) forwardToLeader(ctx context.Context, data []byte, timeout time.Duration) error {
	// Create deadline-based context for overall operation
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Try to get leader client with retries until timeout
	var client pb.RaftServiceClient
	var err error
	retryInterval := forwardRetryInterval

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout while waiting for leader: %w", ctx.Err())
		default:
			client, err = n.getLeaderClient()
			if err == nil {
				// Successfully got leader client, proceed with forwarding
				resp, err := client.ForwardApply(ctx, &pb.ForwardApplyRequest{
					Data:      data,
					TimeoutMs: timeout.Milliseconds(),
				})
				if err != nil {
					return fmt.Errorf("failed to forward request: %w", err)
				}

				if !resp.GetSuccess() {
					return fmt.Errorf("leader failed to apply: %s", resp.GetError())
				}

				return nil
			}

			// Log retry attempt at debug level
			n.Logger.Debug().Err(err).Msg("Failed to get leader client, retrying...")
			time.Sleep(retryInterval)
		}
	}
}

// Shutdown gracefully stops the Node by stopping the gRPC server, closing RPC client connections,
// and shutting down the underlying Raft node. It returns an error if the Raft node fails to
// shutdown properly, ignoring the ErrRaftShutdown error which indicates the node was already
// shutdown.
func (n *Node) Shutdown() error {
	if n.rpcServer != nil {
		n.rpcServer.GracefulStop()
	}
	if n.rpcClient != nil {
		n.rpcClient.close()
	}

	if err := n.raft.Shutdown().Error(); err != nil && !errors.Is(err, raft.ErrRaftShutdown) {
		return fmt.Errorf("failed to shutdown raft node: %w", err)
	}
	return nil
}

// FSM represents the Finite State Machine for the Raft cluster.
type FSM struct {
	lbHashToBlockName map[string]string
	roundRobinIndex   map[string]*atomic.Uint32
	weightedRRStates  map[string]map[string]WeightedProxy
	raftPeers         map[string]config.RaftPeer
	mu                sync.RWMutex
}

// GetProxyBlock safely retrieves the block name for a given hash.
func (f *FSM) GetProxyBlock(hash string) (string, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	if blockName, exists := f.lbHashToBlockName[hash]; exists {
		return blockName, true
	}
	return "", false
}

// GetRoundRobinNext retrieves the next index for a given group name.
func (f *FSM) GetRoundRobinNext(groupName string) uint32 {
	if index, ok := f.roundRobinIndex[groupName]; ok {
		return index.Load()
	}
	return 0
}

// GetWeightedRRState retrieves the weight for a given group name and proxy name.
func (f *FSM) GetWeightedRRState(groupName, proxyName string) (WeightedProxy, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	if group, exists := f.weightedRRStates[groupName]; exists {
		if weight, ok := group[proxyName]; ok {
			return weight, true
		}
	}
	return WeightedProxy{}, false
}

// NewFSM creates a new FSM instance.
func NewFSM() *FSM {
	return &FSM{
		lbHashToBlockName: make(map[string]string),
		roundRobinIndex:   make(map[string]*atomic.Uint32),
		weightedRRStates:  make(map[string]map[string]WeightedProxy),
		raftPeers:         make(map[string]config.RaftPeer),
	}
}

// Apply implements the raft.FSM interface.
func (f *FSM) Apply(log *raft.Log) interface{} {
	var cmd Command
	if err := json.Unmarshal(log.Data, &cmd); err != nil {
		return fmt.Errorf("failed to unmarshal command: %w", err)
	}

	switch cmd.Type {
	case CommandAddConsistentHashEntry:
		f.mu.Lock()
		defer f.mu.Unlock()
		payload, err := convertPayload[ConsistentHashPayload](cmd.Payload)
		if err != nil {
			return fmt.Errorf("failed to convert payload: %w", err)
		}
		f.lbHashToBlockName[payload.Hash] = payload.BlockName
		return nil
	case CommandAddRoundRobinNext:
		payload, err := convertPayload[RoundRobinPayload](cmd.Payload)
		if err != nil {
			return fmt.Errorf("failed to convert payload: %w", err)
		}
		if _, exists := f.roundRobinIndex[payload.GroupName]; !exists {
			f.roundRobinIndex[payload.GroupName] = &atomic.Uint32{}
		}
		f.roundRobinIndex[payload.GroupName].Store(payload.NextIndex)
		return nil
	case CommandUpdateWeightedRR:
		f.mu.Lock()
		defer f.mu.Unlock()
		payload, err := convertPayload[WeightedRRPayload](cmd.Payload)
		if err != nil {
			return fmt.Errorf("failed to convert payload: %w", err)
		}

		if _, exists := f.weightedRRStates[payload.GroupName]; !exists {
			f.weightedRRStates[payload.GroupName] = make(map[string]WeightedProxy)
		}
		f.weightedRRStates[payload.GroupName][payload.ProxyName] = payload.Weight
		return nil
	case CommandUpdateWeightedRRBatch:
		f.mu.Lock()
		defer f.mu.Unlock()
		payload, err := convertPayload[WeightedRRBatchPayload](cmd.Payload)
		if err != nil {
			return fmt.Errorf("failed to convert payload: %w", err)
		}

		if _, exists := f.weightedRRStates[payload.GroupName]; !exists {
			f.weightedRRStates[payload.GroupName] = make(map[string]WeightedProxy)
		}

		// Update all proxies in a single operation
		for proxyName, weight := range payload.Updates {
			f.weightedRRStates[payload.GroupName][proxyName] = weight
		}
		return nil
	case CommandAddPeer:
		f.mu.Lock()
		defer f.mu.Unlock()
		payload, err := convertPayload[PeerPayload](cmd.Payload)
		if err != nil {
			return fmt.Errorf("failed to convert payload: %w", err)
		}
		if f.raftPeers == nil {
			f.raftPeers = make(map[string]config.RaftPeer)
		}
		f.raftPeers[payload.ID] = config.RaftPeer{
			ID:          payload.ID,
			Address:     payload.Address,
			GRPCAddress: payload.GRPCAddress,
		}
		return nil
	case CommandRemovePeer:
		f.mu.Lock()
		defer f.mu.Unlock()
		payload, err := convertPayload[PeerPayload](cmd.Payload)
		if err != nil {
			return fmt.Errorf("failed to convert payload: %w", err)
		}
		delete(f.raftPeers, payload.ID)
		return nil
	default:
		return fmt.Errorf("unknown command type: %s", cmd.Type)
	}
}

// Helper function to convert interface{} to specific payload type.
func convertPayload[T any](payload interface{}) (T, error) {
	var result T

	// Convert the payload to JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return result, fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Unmarshal into the specific type
	if err := json.Unmarshal(payloadBytes, &result); err != nil {
		return result, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	return result, nil
}

// Snapshot returns a snapshot of the FSM.
func (f *FSM) Snapshot() (raft.FSMSnapshot, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	// Create copies of all maps
	hashMapCopy := make(map[string]string)
	for k, v := range f.lbHashToBlockName {
		hashMapCopy[k] = v
	}

	roundRobinCopy := make(map[string]uint32)
	for k, v := range f.roundRobinIndex {
		roundRobinCopy[k] = v.Load()
	}

	// Copy weightedRRStates
	weightedRRCopy := make(map[string]map[string]WeightedProxy)
	for groupName, group := range f.weightedRRStates {
		weightedRRCopy[groupName] = make(map[string]WeightedProxy)
		for proxyName, weight := range group {
			weightedRRCopy[groupName][proxyName] = weight
		}
	}

	return &FSMSnapshot{
		lbHashToBlockName: hashMapCopy,
		roundRobinIndex:   roundRobinCopy,
		weightedRRStates:  weightedRRCopy,
	}, nil
}

// Restore restores the FSM from a snapshot.
func (f *FSM) Restore(readCloser io.ReadCloser) error {
	var data struct {
		HashToBlock     map[string]string                   `json:"hashToBlock"`
		RoundRobin      map[string]uint32                   `json:"roundRobin"`
		WeightedRRState map[string]map[string]WeightedProxy `json:"weightedRrState"`
	}

	if err := json.NewDecoder(readCloser).Decode(&data); err != nil {
		return fmt.Errorf("error decoding snapshot: %w", err)
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.lbHashToBlockName = data.HashToBlock
	f.roundRobinIndex = make(map[string]*atomic.Uint32)
	for k, v := range data.RoundRobin {
		atomicVal := &atomic.Uint32{}
		atomicVal.Store(v)
		f.roundRobinIndex[k] = atomicVal
	}
	f.weightedRRStates = data.WeightedRRState

	return nil
}

// FSMSnapshot represents a snapshot of the FSM.
type FSMSnapshot struct {
	lbHashToBlockName map[string]string
	roundRobinIndex   map[string]uint32
	weightedRRStates  map[string]map[string]WeightedProxy
}

// Persist writes the FSMSnapshot data to the given SnapshotSink.
func (f *FSMSnapshot) Persist(sink raft.SnapshotSink) error {
	data := struct {
		HashToBlock     map[string]string                   `json:"hashToBlock"`
		RoundRobin      map[string]uint32                   `json:"roundRobin"`
		WeightedRRState map[string]map[string]WeightedProxy `json:"weightedRrState"`
	}{
		HashToBlock:     f.lbHashToBlockName,
		RoundRobin:      f.roundRobinIndex,
		WeightedRRState: f.weightedRRStates,
	}

	if err := json.NewEncoder(sink).Encode(data); err != nil {
		if cancelErr := sink.Cancel(); cancelErr != nil {
			return fmt.Errorf("error canceling snapshot: %w (original error: %w)", cancelErr, err)
		}
		return fmt.Errorf("error encoding snapshot: %w", err)
	}

	if err := sink.Close(); err != nil {
		return fmt.Errorf("error closing snapshot sink: %w", err)
	}
	return nil
}

func (f *FSMSnapshot) Release() {}

// startRPCServer starts a gRPC server on the configured address to handle Raft RPC requests.
// It returns an error if the server fails to start listening on the configured address.
func (n *Node) startGRPCServer(certFile, keyFile string) error {
	var opts []grpc.ServerOption

	// Configure TLS if secure mode is enabled
	if n.grpcIsSecure {
		if certFile == "" || keyFile == "" {
			return errors.New("TLS certificate and key files are required when secure mode is enabled")
		}

		creds, err := credentials.NewServerTLSFromFile(certFile, keyFile)
		if err != nil {
			return fmt.Errorf("failed to load TLS credentials: %w", err)
		}
		opts = append(opts, grpc.Creds(creds))
	}

	// Create listener
	listener, err := net.Listen("tcp", n.grpcAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", n.grpcAddr, err)
	}

	// Create gRPC server with options
	n.rpcServer = grpc.NewServer(opts...)
	pb.RegisterRaftServiceServer(n.rpcServer, &rpcServer{node: n})

	// Start server in a goroutine
	go func() {
		n.Logger.Info().
			Str("address", n.grpcAddr).
			Bool("secure", n.grpcIsSecure).
			Msg("Starting gRPC server")

		if err := n.rpcServer.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			n.Logger.Error().Err(err).Msg("gRPC server failed unexpectedly")
		}
	}()

	return nil
}

type HealthStatus struct {
	IsHealthy   bool
	HasLeader   bool
	IsLeader    bool
	LastContact time.Duration
	Error       error
}

// GetHealthStatus returns the health status of the Raft node.
func (n *Node) GetHealthStatus() HealthStatus {
	// Handle uninitialized raft node
	if n == nil || n.raft == nil {
		return HealthStatus{
			IsHealthy: false,
			Error:     errors.New("raft node not initialized"),
		}
	}

	raftState := n.raft.State()
	stats := n.raft.Stats()
	nodeID := string(n.config.LocalID)

	// Determine leadership status
	_, leaderID := n.raft.LeaderWithID()
	isLeader := raftState == raft.Leader
	hasLeader := leaderID != ""

	// Update metrics for leadership status
	metrics.RaftLeaderStatus.WithLabelValues(nodeID).Set(boolToFloat(isLeader))

	// Parse last contact with leader
	lastContact, lastContactErr := parseLastContact(stats["last_contact"])
	communicatesWithLeader := isLeader || (hasLeader && lastContactErr == nil && lastContact <= LeaderElectionTimeout)

	// Determine health status
	isHealthy := communicatesWithLeader
	metrics.RaftHealthStatus.WithLabelValues(nodeID).Set(boolToFloat(isHealthy))

	// Update latency metric if last contact is valid
	if lastContactErr == nil && lastContact >= 0 {
		metrics.RaftLastContactLatency.WithLabelValues(nodeID).Set(float64(lastContact.Milliseconds()))
	}

	return HealthStatus{
		IsHealthy:   isHealthy,
		HasLeader:   hasLeader,
		IsLeader:    isLeader,
		LastContact: lastContact,
		Error:       lastContactErr,
	}
}

// Helper function to parse last contact time safely.
func parseLastContact(value string) (time.Duration, error) {
	switch value {
	case "", "never":
		return -1, errors.New("no contact with leader")
	case "0":
		return 0, nil
	default:
		duration, err := time.ParseDuration(value)
		if err != nil {
			return 0, fmt.Errorf("invalid last_contact format: %w", err)
		}
		return duration, nil
	}
}

// Convert bool to float for metric values.
func boolToFloat(val bool) float64 {
	if val {
		return 1
	}
	return 0
}

// StopGRPCServer stops the gRPC server.
func (n *Node) StopGRPCServer() {
	if n.rpcServer != nil {
		n.rpcServer.GracefulStop()
	}
}

// // GetState returns the current Raft state and leader ID.
func (n *Node) GetState() (raft.RaftState, raft.ServerID) {
	state := n.raft.State()
	_, leaderID := n.raft.LeaderWithID()
	return state, leaderID
}

// StartPeerSynchronizer starts a goroutine that synchronizes peers between Raft and FSM
func (n *Node) StartPeerSynchronizer(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(peerSyncInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				n.Logger.Info().Msg("Stopping peer synchronizer")
				return
			case <-ticker.C:
				if err := n.syncPeers(ctx); err != nil {
					n.Logger.Error().Err(err).Msg("Failed to synchronize peers")
				}
			}
		}
	}()
}

// syncPeers synchronizes peers between Raft configuration and FSM
func (n *Node) syncPeers(ctx context.Context) error {
	n.Logger.Info().Msg("Test syncPeers")
	// Get current Raft peers
	raftPeers := n.GetPeers()
	raftPeerMap := make(map[string]raft.Server)
	for _, peer := range raftPeers {
		raftPeerMap[string(peer.ID)] = peer
	}

	// Get FSM peers
	n.Fsm.mu.RLock()
	fsmPeerMap := make(map[string]config.RaftPeer)
	for id, peer := range n.Fsm.raftPeers {
		fsmPeerMap[id] = peer
	}
	n.Fsm.mu.RUnlock()

	// Check for peers in Raft but not in FSM
	for id, raftPeer := range raftPeerMap {
		if _, exists := fsmPeerMap[id]; !exists {
			n.Logger.Info().Str("peer_id", id).Msg("Found peer in Raft but not in FSM, querying other peers")
			if err := n.queryPeerInfo(ctx, id, string(raftPeer.Address)); err != nil {
				n.Logger.Error().Err(err).Str("peer_id", id).Msg("Failed to query peer info")
			}
		}
	}

	// Check for peers in FSM but not in Raft
	for id := range fsmPeerMap {
		if _, exists := raftPeerMap[id]; !exists {
			n.Logger.Info().Str("peer_id", id).Msg("Found peer in FSM but not in Raft, removing from FSM")
			// remove peer from FSM
			n.Fsm.mu.Lock()
			delete(n.Fsm.raftPeers, id)
			n.Fsm.mu.Unlock()
		}
	}

	return nil
}

// queryPeerInfo queries other peers for information about an unknown peer
func (n *Node) queryPeerInfo(ctx context.Context, peerID, peerAddr string) error {
	n.Fsm.mu.RLock()
	peers := make([]config.RaftPeer, 0, len(n.Fsm.raftPeers))
	for _, peer := range n.Fsm.raftPeers {
		if peer.ID != peerID && peer.ID != string(n.config.LocalID) {
			peers = append(peers, peer)
		}
	}
	n.Fsm.mu.RUnlock()

	for _, peer := range peers {
		client, err := n.rpcClient.getClient(peer.GRPCAddress)
		if err != nil {
			n.Logger.Debug().
				Err(err).
				Str("peer_id", peer.ID).
				Str("grpc_address", peer.GRPCAddress).
				Msg("Failed to get client for peer")
			continue
		}

		resp, err := client.GetPeerInfo(ctx, &pb.GetPeerInfoRequest{
			PeerId: peerID,
		})
		if err != nil {
			n.Logger.Debug().
				Err(err).
				Str("peer_id", peer.ID).
				Msg("Failed to get peer info")
			continue
		}

		if resp.GetExists() {
			n.Fsm.mu.Lock()
			n.Fsm.raftPeers[peerID] = config.RaftPeer{
				ID:          peerID,
				Address:     peerAddr,
				GRPCAddress: resp.GetGrpcAddress(),
			}
			n.Fsm.mu.Unlock()
			return nil
		}
	}

	return fmt.Errorf("no peers had information about peer %s", peerID)
}
