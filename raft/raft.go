package raft

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gatewayd-io/gatewayd/config"
	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
	"github.com/rs/zerolog"
)

// Command types for Raft operations.
const (
	CommandAddConsistentHashEntry = "ADD_CONSISTENT_HASH_ENTRY"
	RaftLeaderState               = raft.Leader
	LeaderElectionTimeout         = 3 * time.Second
	maxSnapshots                  = 3                // Maximum number of snapshots to retain
	maxPool                       = 3                // Maximum number of connections to pool
	transportTimeout              = 10 * time.Second // Timeout for transport operations
	leadershipCheckInterval       = 10 * time.Second // Interval for checking leadership status
)

// ConsistentHashCommand represents a command to modify the consistent hash.
type ConsistentHashCommand struct {
	Type      string `json:"type"`
	Hash      uint64 `json:"hash"`
	BlockName string `json:"blockName"`
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
	Peers         []raft.Server
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
		Peers:         convertPeers(raftConfig.Peers),
	}

	// Handle bootstrapping
	isBootstrap := raftConfig.LeaderID == nodeID
	if isBootstrap {
		configuration := raft.Configuration{
			Servers: make([]raft.Server, len(node.Peers)),
		}
		for i, peer := range node.Peers {
			configuration.Servers[i] = raft.Server{
				ID:      peer.ID,
				Address: peer.Address,
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

func convertPeers(configPeers []config.RaftPeer) []raft.Server {
	peers := make([]raft.Server, len(configPeers))
	for i, peer := range configPeers {
		peers[i] = raft.Server{
			ID:      raft.ServerID(peer.ID),
			Address: raft.ServerAddress(peer.Address),
		}
	}
	return peers
}

// monitorLeadership checks if the node is the Raft leader and logs state changes.
func (n *Node) monitorLeadership() {
	for {
		isLeader := n.raft.State() == raft.Leader
		if isLeader {
			n.Logger.Info().Msg("This node is the Raft leader")

			for _, peer := range n.Peers {
				// Check if peer already exists in cluster configuration
				existingConfig := n.raft.GetConfiguration().Configuration()
				peerExists := false
				for _, server := range existingConfig.Servers {
					if server.ID == peer.ID {
						peerExists = true
						n.Logger.Info().Msgf("Peer %s already exists in Raft cluster, skipping", peer.ID)
						break
					}
				}
				if peerExists {
					continue
				}
				err := n.AddPeer(string(peer.ID), string(peer.Address))
				if err != nil {
					n.Logger.Error().Err(err).Msgf("Failed to add node %s to Raft cluster", peer.ID)
				}
			}
		} else {
			n.Logger.Info().Msg("This node is a Raft follower")
		}

		time.Sleep(leadershipCheckInterval) // Use the named constant here
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

// Apply applies a new log entry to the Raft log.
func (n *Node) Apply(data []byte, timeout time.Duration) error {
	future := n.raft.Apply(data, timeout)
	if err := future.Error(); err != nil {
		return fmt.Errorf("failed to apply log entry: %w", err)
	}
	return nil
}

// Shutdown gracefully shuts down the Raft node.
func (n *Node) Shutdown() error {
	if err := n.raft.Shutdown().Error(); err != nil {
		return fmt.Errorf("failed to shutdown raft node: %w", err)
	}
	return nil
}

// FSM represents the Finite State Machine for the Raft cluster.
type FSM struct {
	lbHashToBlockName map[uint64]string
	mu                sync.RWMutex
}

// GetProxyBlock safely retrieves the block name for a given hash.
func (f *FSM) GetProxyBlock(hash uint64) (string, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	if blockName, exists := f.lbHashToBlockName[hash]; exists {
		return blockName, true
	}
	return "", false
}

// NewFSM creates a new FSM instance.
func NewFSM() *FSM {
	return &FSM{
		lbHashToBlockName: make(map[uint64]string),
	}
}

// Apply implements the raft.FSM interface.
func (f *FSM) Apply(log *raft.Log) interface{} {
	var cmd ConsistentHashCommand
	if err := json.Unmarshal(log.Data, &cmd); err != nil {
		return fmt.Errorf("failed to unmarshal command: %w", err)
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	switch cmd.Type {
	case CommandAddConsistentHashEntry:
		f.lbHashToBlockName[cmd.Hash] = cmd.BlockName
		return nil
	default:
		return fmt.Errorf("unknown command type: %s", cmd.Type)
	}
}

// Snapshot returns a snapshot of the FSM.
func (f *FSM) Snapshot() (raft.FSMSnapshot, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	// Create a copy of the hash map
	hashMapCopy := make(map[uint64]string)
	for k, v := range f.lbHashToBlockName {
		hashMapCopy[k] = v
	}

	return &FSMSnapshot{lbHashToBlockName: hashMapCopy}, nil
}

// Restore restores the FSM from a snapshot.
func (f *FSM) Restore(rc io.ReadCloser) error {
	decoder := json.NewDecoder(rc)
	var lbHashToBlockName map[uint64]string
	if err := decoder.Decode(&lbHashToBlockName); err != nil {
		return fmt.Errorf("error decoding snapshot: %w", err)
	}

	f.mu.Lock()
	f.lbHashToBlockName = lbHashToBlockName
	f.mu.Unlock()

	return nil
}

// FSMSnapshot represents a snapshot of the FSM.
type FSMSnapshot struct {
	lbHashToBlockName map[uint64]string
}

func (f *FSMSnapshot) Persist(sink raft.SnapshotSink) error {
	err := json.NewEncoder(sink).Encode(f.lbHashToBlockName)
	if err != nil {
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
