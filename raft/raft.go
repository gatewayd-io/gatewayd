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
	"github.com/gatewayd-io/gatewayd/metrics"
	pb "github.com/gatewayd-io/gatewayd/raft/proto"
	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
)

// Configuration constants for Raft operations.
const (
	CommandAddConsistentHashEntry = "ADD_CONSISTENT_HASH_ENTRY"
	CommandAddRoundRobinNext      = "ADD_ROUND_ROBIN_NEXT"
	CommandUpdateWeightedRR       = "UPDATE_WEIGHTED_RR"
	CommandUpdateWeightedRRBatch  = "UPDATE_WEIGHTED_RR_BATCH"
	RaftLeaderState               = raft.Leader
	LeaderElectionTimeout         = 3 * time.Second
	maxSnapshots                  = 3                // Maximum number of snapshots to retain
	maxPool                       = 3                // Maximum number of connections to pool
	transportTimeout              = 10 * time.Second // Timeout for transport operations
	leadershipCheckInterval       = 10 * time.Second // Interval for checking leadership status
	ApplyTimeout                  = 2 * time.Second  // Timeout for applying commands
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
}

// NewRaftNode creates and initializes a new Raft node.
func NewRaftNode(logger zerolog.Logger, raftConfig config.Raft) (*Node, error) {
	config := raft.DefaultConfig()

	var err error
	nodeID := raftConfig.NodeID
	if raftConfig.NodeID == "" {
		nodeID, err = os.Hostname()
		if err != nil {
			return nil, fmt.Errorf("error getting hostname: %w", err)
		}
	}
	raftAddr := raftConfig.Address
	config.LocalID = raft.ServerID(nodeID)
	raftDir := filepath.Join(raftConfig.Directory, nodeID)
	err = os.MkdirAll(raftDir, os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("error creating raft directory: %w", err)
	}

	// Create the FSM
	fsm := NewFSM()

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
	raftNode, err := raft.NewRaft(config, fsm, logStore, stableStore, snapshotStore, transport)
	if err != nil {
		return nil, fmt.Errorf("error creating Raft: %w", err)
	}

	node := &Node{
		raft:          raftNode,
		config:        config,
		Fsm:           fsm,
		logStore:      logStore,
		stableStore:   stableStore,
		snapshotStore: snapshotStore,
		transport:     transport,
		Logger:        logger,
		Peers:         raftConfig.Peers,
		grpcAddr:      raftConfig.GRPCAddress,
	}

	// Initialize RPC client
	node.rpcClient = newRPCClient(node)

	// Start RPC server
	if err := node.startRPCServer(); err != nil {
		return nil, fmt.Errorf("failed to start RPC server: %w", err)
	}

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
		configuration.Servers = append(configuration.Servers, raft.Server{
			ID:      config.LocalID,
			Address: transport.LocalAddr(),
		})
		node.raft.BootstrapCluster(configuration)
	}

	go node.monitorLeadership()

	return node, nil
}

// monitorLeadership checks if the node is the Raft leader and logs state changes.
func (n *Node) monitorLeadership() {
	for {
		if n.raft.State() == raft.Leader {
			n.Logger.Info().Msg("This node is the Raft leader")

			for _, peer := range n.Peers {
				// Check if peer already exists in cluster configuration
				existingConfig := n.raft.GetConfiguration().Configuration()
				peerExists := false
				for _, server := range existingConfig.Servers {
					if server.ID == raft.ServerID(peer.ID) {
						peerExists = true
						n.Logger.Debug().Msgf("Peer %s already exists in Raft cluster, skipping", peer.ID)
						break
					}
				}
				if peerExists {
					continue
				}
				err := n.AddPeer(peer.ID, peer.Address)
				if err != nil {
					n.Logger.Error().Err(err).Msgf("Failed to add node %s to Raft cluster", peer.ID)
				}
			}
		} else {
			n.Logger.Debug().Msg("This node is a Raft follower")
		}

		time.Sleep(leadershipCheckInterval)
	}
}

// AddPeer adds a new peer to the Raft cluster.
func (n *Node) AddPeer(peerID, peerAddr string) error {
	if err := n.raft.AddVoter(raft.ServerID(peerID), raft.ServerAddress(peerAddr), 0, 0).Error(); err != nil {
		return fmt.Errorf("failed to add voter: %w", err)
	}
	return nil
}

// RemovePeer removes a peer from the Raft cluster.
func (n *Node) RemovePeer(peerID string) error {
	if err := n.raft.RemoveServer(raft.ServerID(peerID), 0, 0).Error(); err != nil {
		return fmt.Errorf("failed to remove server: %w", err)
	}
	return nil
}

// Apply is the public method that handles forwarding if necessary.
func (n *Node) Apply(data []byte, timeout time.Duration) error {
	if n.raft.State() != raft.Leader {
		return n.forwardToLeader(data, timeout)
	}
	return n.applyInternal(data, timeout)
}

// applyInternal is the internal method that actually applies the data.
func (n *Node) applyInternal(data []byte, timeout time.Duration) error {
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
func (n *Node) forwardToLeader(data []byte, timeout time.Duration) error {
	leaderAddr, leaderID := n.raft.LeaderWithID()
	if leaderID == "" {
		return errors.New("no leader available")
	}

	n.Logger.Debug().
		Str("leader_id", string(leaderID)).
		Str("leader_addr", string(leaderAddr)).
		Msg("forwarding request to leader")

	var leaderGrpcAddr string
	for _, peer := range n.Peers {
		if raft.ServerID(peer.ID) == leaderID {
			leaderGrpcAddr = peer.GRPCAddress
			break
		}
	}
	// Get the RPC client for the leader
	client, err := n.rpcClient.getClient(leaderGrpcAddr)
	if err != nil {
		return fmt.Errorf("failed to get client for leader: %w", err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Forward the request
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

// GetState returns the current Raft state.
func (n *Node) GetState() raft.RaftState {
	return n.raft.State()
}

// startRPCServer starts a gRPC server on the configured address to handle Raft RPC requests.
// It returns an error if the server fails to start listening on the configured address.
func (n *Node) startRPCServer() error {
	listener, err := net.Listen("tcp", n.grpcAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", n.grpcAddr, err)
	}

	n.rpcServer = grpc.NewServer()
	pb.RegisterRaftServiceServer(n.rpcServer, &rpcServer{node: n})

	go func() {
		if err := n.rpcServer.Serve(listener); err != nil {
			n.Logger.Error().Err(err).Msg("RPC server failed")
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
