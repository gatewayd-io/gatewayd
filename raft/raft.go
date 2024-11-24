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

// Command types for Raft operations
const (
	CommandAddHashMapping = "ADD_HASH_MAPPING"
)

// HashMapCommand represents a command to modify the hash map
type HashMapCommand struct {
	Type      string `json:"type"`
	Hash      uint64 `json:"hash"`
	BlockName string `json:"block_name"`
}

// RaftNode represents a node in the Raft cluster
type RaftNode struct {
	raft           *raft.Raft
	config         *raft.Config
	Fsm            *FSM
	logStore       raft.LogStore
	stableStore    raft.StableStore
	snapshotStore  raft.SnapshotStore
	transport      raft.Transport
	bootstrapPeers []raft.Server
	Logger         zerolog.Logger
	Peers          []raft.Server // Holds Raft peers (for joining an existing cluster)
}

// NewRaftNode creates and initializes a new Raft node
func NewRaftNode(logger zerolog.Logger, raftConfig config.Raft) (*RaftNode, error) {
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
	raftDir := filepath.Join("raft", nodeID)
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
	snapshotStore, err := raft.NewFileSnapshotStore(raftDir, 3, os.Stderr)
	if err != nil {
		return nil, fmt.Errorf("error creating snapshot store: %w", err)
	}

	// Setup Raft communication
	addr, err := net.ResolveTCPAddr("tcp", raftAddr)
	if err != nil {
		return nil, fmt.Errorf("error resolving TCP address: %w", err)
	}
	transport, err := raft.NewTCPTransport(raftAddr, addr, 3, 10*time.Second, os.Stderr)
	if err != nil {
		return nil, fmt.Errorf("error creating TCP transport: %w", err)
	}

	// Create the Raft node
	r, err := raft.NewRaft(config, fsm, logStore, stableStore, snapshotStore, transport)
	if err != nil {
		return nil, fmt.Errorf("error creating Raft: %w", err)
	}

	node := &RaftNode{
		raft:          r,
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

// monitorLeadership checks if the node is the Raft leader and logs state changes
func (n *RaftNode) monitorLeadership() {
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

		time.Sleep(10 * time.Second) // Poll leadership status periodically
	}
}

// AddPeer adds a new peer to the Raft cluster
func (n *RaftNode) AddPeer(peerID, peerAddr string) error {
	return n.raft.AddVoter(raft.ServerID(peerID), raft.ServerAddress(peerAddr), 0, 0).Error()
}

// RemovePeer removes a peer from the Raft cluster
func (n *RaftNode) RemovePeer(peerID string) error {
	return n.raft.RemoveServer(raft.ServerID(peerID), 0, 0).Error()
}

// Apply applies a new log entry to the Raft log
func (n *RaftNode) Apply(data []byte, timeout time.Duration) error {
	future := n.raft.Apply(data, timeout)
	return future.Error()
}

// Shutdown gracefully shuts down the Raft node
func (n *RaftNode) Shutdown() error {
	return n.raft.Shutdown().Error()
}

// FSM represents the Finite State Machine for the Raft cluster
type FSM struct {
	consistentHashMap map[uint64]string
	mu                sync.RWMutex
}

// GetProxyBlock safely retrieves the block name for a given hash
func (f *FSM) GetProxyBlock(hash uint64) (string, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	if blockName, exists := f.consistentHashMap[hash]; exists {
		return blockName, true
	}
	return "", false
}

// NewFSM creates a new FSM instance
func NewFSM() *FSM {
	return &FSM{
		consistentHashMap: make(map[uint64]string),
	}
}

// Apply implements the raft.FSM interface
func (f *FSM) Apply(log *raft.Log) interface{} {
	var cmd HashMapCommand
	if err := json.Unmarshal(log.Data, &cmd); err != nil {
		return fmt.Errorf("failed to unmarshal command: %w", err)
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	switch cmd.Type {
	case CommandAddHashMapping:
		f.consistentHashMap[cmd.Hash] = cmd.BlockName
		return nil
	default:
		return fmt.Errorf("unknown command type: %s", cmd.Type)
	}
}

// Snapshot returns a snapshot of the FSM
func (f *FSM) Snapshot() (raft.FSMSnapshot, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	// Create a copy of the hash map
	hashMapCopy := make(map[uint64]string)
	for k, v := range f.consistentHashMap {
		hashMapCopy[k] = v
	}

	return &FSMSnapshot{hashMap: hashMapCopy}, nil
}

// Restore restores the FSM from a snapshot
func (f *FSM) Restore(rc io.ReadCloser) error {
	decoder := json.NewDecoder(rc)
	var hashMap map[uint64]string
	if err := decoder.Decode(&hashMap); err != nil {
		return err
	}

	f.mu.Lock()
	f.consistentHashMap = hashMap
	f.mu.Unlock()

	return nil
}

// FSMSnapshot represents a snapshot of the FSM
type FSMSnapshot struct {
	hashMap map[uint64]string
}

func (f *FSMSnapshot) Persist(sink raft.SnapshotSink) error {
	err := json.NewEncoder(sink).Encode(f.hashMap)
	if err != nil {
		sink.Cancel()
		return err
	}
	return sink.Close()
}

func (f *FSMSnapshot) Release() {}
