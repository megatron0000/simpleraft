package status

import (
	"encoding/json"
	"simpleraft/iface"
	"simpleraft/storage"
	"strconv"
	"time"
)

// helper
func toString(x int64) string {
	return strconv.FormatInt(x, 10)
}

// Status holds information the raft node needs to operate.
// All struct members are non-exported to enforce usage of
// functions for reading/writing these values
type Status struct {
	state         string
	nodeAddress   iface.PeerAddress
	currentTerm   int64
	votedFor      iface.PeerAddress
	voteCount     int64
	commitIndex   int64
	lastApplied   int64
	peerAddresses []iface.PeerAddress
	nextIndex     map[iface.PeerAddress]int64
	matchIndex    map[iface.PeerAddress]int64
	// The latest log index corresponding to a cluster-change command
	// -1 if none
	clusterChangeIndex int64
	// The term of log entry whose index is `clusterChangeIndex`
	// -1 if none
	clusterChangeTerm int64
	// last instant we heard from the leader
	leaderLastHeard time.Time
	// least amount of time the raft node is configured
	// to wait before starting an election
	minElectionTimeout time.Duration
	// pointer to data storage
	storage *storage.Storage
}

// New constructs a Status struct, automatically recovering state from disk, if any.
// `peerAddresses` and `nodeAddress` work as default values: they will only be used
// if there is no alternative record present in the disk storage
func New(nodeAddress iface.PeerAddress, peerAddresses []iface.PeerAddress,
	storage *storage.Storage, minElectionTimeout time.Duration) *Status {

	var (
		err                     error
		nodeAddressSlice        []byte
		currentTermSlice        []byte
		currentTerm             int64
		votedForSlice           []byte
		votedFor                iface.PeerAddress
		nextIndex               map[iface.PeerAddress]int64
		matchIndex              map[iface.PeerAddress]int64
		clusterChangeIndexSlice []byte
		clusterChangeIndex      int64
		clusterChangeTermSlice  []byte
		clusterChangeTerm       int64
		oldPeerAddresses        []byte
		peerAddressesJSON       []byte
	)

	// retrieve nodeAddress from datastore
	if nodeAddressSlice, err = storage.Get(nil, []byte("/raft/nodeAddress")); err != nil {
		panic(err)
	}

	if nodeAddressSlice == nil {
		// save it to disk
		if nodeAddressSlice, err = json.Marshal(nodeAddress); err != nil {
			panic(err)
		}

		if err = storage.Set([]byte("/raft/nodeAddress"), nodeAddressSlice); err != nil {
			panic(err)
		}
	} else {
		if err = json.Unmarshal(nodeAddressSlice, &nodeAddress); err != nil {
			panic(err)
		}
	}

	// retrieve currentTerm from datastore
	if currentTermSlice, err = storage.Get(nil, []byte("/raft/currentTerm")); err != nil {
		panic(err)
	}

	if currentTermSlice == nil {
		currentTerm = 0
	} else {
		if currentTerm, err = strconv.ParseInt(string(currentTermSlice), 10, 64); err != nil {
			panic(err)
		}
	}

	// retrieve votedFor from datastore
	if votedForSlice, err = storage.Get(
		nil,
		[]byte("/raft/votedFor")); err != nil {
		panic(err)
	}

	if votedForSlice == nil {
		votedFor = ""
	} else {
		votedFor = iface.PeerAddress(string(votedForSlice))
	}

	// retrieve peerAddresses from datastore (if any)
	if oldPeerAddresses, err = storage.Get(
		nil,
		[]byte("/raft/peerAddresses")); err != nil {
		panic(err)
	}

	if oldPeerAddresses == nil {
		// save it to disk
		if peerAddressesJSON, err = json.Marshal(peerAddresses); err != nil {
			panic(err)
		}

		if err = storage.Set([]byte("/raft/peerAddresses"), peerAddressesJSON); err != nil {
			panic(err)
		}

	} else {
		if err = json.Unmarshal(oldPeerAddresses, &peerAddresses); err != nil {
			panic(err)
		}
	}

	nextIndex = make(map[iface.PeerAddress]int64)
	matchIndex = make(map[iface.PeerAddress]int64)

	for _, address := range peerAddresses {
		nextIndex[address] = 0
		matchIndex[address] = -1
	}

	// retrieve clusterChangeIndex from datastore
	if clusterChangeIndexSlice, err = storage.Get(
		nil,
		[]byte("/raft/clusterChangeIndex")); err != nil {
		panic(err)
	}

	if clusterChangeIndexSlice == nil {
		clusterChangeIndex = -1
	} else {
		if clusterChangeIndex, err = strconv.ParseInt(
			string(clusterChangeIndexSlice), 10, 64); err != nil {
			panic(err)
		}
	}

	// retrieve clusterChangeTerm from datastore
	if clusterChangeTermSlice, err = storage.Get(
		nil,
		[]byte("/raft/clusterChangeTerm")); err != nil {
		panic(err)
	}

	if clusterChangeTermSlice == nil {
		clusterChangeTerm = -1
	} else {
		if clusterChangeTerm, err = strconv.ParseInt(
			string(clusterChangeTermSlice), 10, 64); err != nil {
			panic(err)
		}
	}

	return &Status{
		state:              iface.StateFollower,
		nodeAddress:        nodeAddress,
		currentTerm:        currentTerm,
		votedFor:           votedFor,
		voteCount:          0,
		commitIndex:        -1,
		lastApplied:        -1,
		peerAddresses:      peerAddresses,
		nextIndex:          nextIndex,
		matchIndex:         matchIndex,
		clusterChangeIndex: clusterChangeIndex,
		clusterChangeTerm:  clusterChangeTerm,
		leaderLastHeard:    time.Now().AddDate(-1, 0, 0),
		minElectionTimeout: minElectionTimeout,
		storage:            storage,
	}
}

// State returns the state of the raft node (one of
// iface.StateLeader, iface.StateCandidate, iface.StateFollower)
func (status *Status) State() string {
	return status.state
}

// SetState does not persist the change (not needed for correctness)
func (status *Status) SetState(newState string) {
	switch newState {
	case iface.StateFollower:
	case iface.StateCandidate:
	case iface.StateLeader:
	default:
		panic("unknown state: tried to set node state to " + newState)
	}

	status.state = newState
}

// NodeAddress returns the network address which can be used to contact this node
func (status *Status) NodeAddress() iface.PeerAddress {
	return status.nodeAddress
}

// SetNodeAddress automatically persists the change (necessary for correctness)
func (status *Status) SetNodeAddress(newAddress iface.PeerAddress) {
	var (
		marshal []byte
		err     error
	)

	status.nodeAddress = newAddress

	if marshal, err = json.Marshal(newAddress); err != nil {
		panic(err)
	}

	if err = status.storage.Set([]byte("/raft/nodeAddress"), marshal); err != nil {
		panic(err)
	}
}

// CurrentTerm is the current raft turn, as seen by a raft node (0 in the beginning)
func (status *Status) CurrentTerm() int64 {
	return status.currentTerm
}

// SetCurrentTerm automatically persists the change (necessary for correctness)
func (status *Status) SetCurrentTerm(newTerm int64) {
	status.currentTerm = newTerm
	if err := status.storage.Set(
		[]byte("/raft/currentTerm"),
		[]byte(toString(newTerm))); err != nil {
		panic(err)
	}

}

// VotedFor is the address of another raft node for which
// this raft node has voted in this turn.
//
// If this node has not voted yet, it should be set to the empty string (i.e. "")
func (status *Status) VotedFor() iface.PeerAddress {
	return status.votedFor
}

// SetVotedFor automatically persists the change (necessary for correctness)
func (status *Status) SetVotedFor(newVotedFor iface.PeerAddress) {
	status.votedFor = newVotedFor
	if err := status.storage.Set(
		[]byte("/raft/votedFor"),
		[]byte(newVotedFor)); err != nil {
		panic(err)
	}

}

// VoteCount counts how many votes this node received until now
// in its candidacy. 0 in the beggining
func (status *Status) VoteCount() int64 {
	return status.voteCount
}

// SetVoteCount does not persist the change (not needed for correctness)
func (status *Status) SetVoteCount(newVoteCount int64) {
	status.voteCount = newVoteCount
}

// CommitIndex is the index of the most up-to-date log entry known to be
// committed by this raft node (-1 in the beginning)
func (status *Status) CommitIndex() int64 {
	return status.commitIndex
}

// SetCommitIndex does not persist the change (not needed for correctness)
func (status *Status) SetCommitIndex(newCommitIndex int64) {
	status.commitIndex = newCommitIndex
}

// LastApplied is the index of the last committed log entry that has already
// been applied by the raft state machine (-1 in the beginning)
func (status *Status) LastApplied() int64 {
	return status.lastApplied
}

// SetLastApplied does not persist the change (not needed for correctness)
func (status *Status) SetLastApplied(newLastApplied int64) {
	status.lastApplied = newLastApplied
}

// PeerAddresses is the slice of peer addresses (does not include the node itself)
func (status *Status) PeerAddresses() []iface.PeerAddress {
	return status.peerAddresses
}

// SetPeerAddresses automatically persists the change (necessary for correctness)
func (status *Status) SetPeerAddresses(newPeers []iface.PeerAddress) {
	var (
		marshal       []byte
		err           error
		newNextIndex  map[iface.PeerAddress]int64
		newMatchIndex map[iface.PeerAddress]int64
		addr          iface.PeerAddress
		nextIndex     int64
		matchIndex    int64
		ok            bool
	)

	if marshal, err = json.Marshal(newPeers); err != nil {
		panic(err)
	}
	status.storage.Set([]byte("/raft/peerAddresses"), marshal)

	newNextIndex = make(map[iface.PeerAddress]int64)
	newMatchIndex = make(map[iface.PeerAddress]int64)

	// rebuild nextIndex and matchIndex maps
	for _, addr = range newPeers {
		if nextIndex, ok = status.nextIndex[addr]; ok {
			newNextIndex[addr] = nextIndex
		} else {
			newNextIndex[addr] = 0
		}

		if matchIndex, ok = status.matchIndex[addr]; ok {
			newMatchIndex[addr] = matchIndex
		} else {
			newMatchIndex[addr] = -1
		}
	}

	status.nextIndex = newNextIndex
	status.matchIndex = newMatchIndex
	status.peerAddresses = newPeers
}

// NextIndex is the (as known by leader) next log index a peer is waiting for
// (0 in the beginning)
func (status *Status) NextIndex(peer iface.PeerAddress) int64 {
	return status.nextIndex[peer]
}

// SetNextIndex does not persist the change (not needed for correctness)
func (status *Status) SetNextIndex(peer iface.PeerAddress, nextIndex int64) {
	status.nextIndex[peer] = nextIndex
}

// MatchIndex is the (as known by leader) last log index where the leader's log
// matches the peer's log (-1 in the beginning)
func (status *Status) MatchIndex(peer iface.PeerAddress) int64 {
	return status.matchIndex[peer]
}

// SetMatchIndex does not persist the change (not needed for correctness)
func (status *Status) SetMatchIndex(peer iface.PeerAddress, matchIndex int64) {
	status.matchIndex[peer] = matchIndex
}

// ClusterChangeIndex is the latest log index
// corresponding to a cluster-change command. -1 if none
func (status *Status) ClusterChangeIndex() int64 {
	return status.clusterChangeIndex
}

// ClusterChangeTerm is the term of log entry whose
// index is `clusterChangeIndex`. -1 if none
func (status *Status) ClusterChangeTerm() int64 {
	return status.clusterChangeTerm
}

// SetClusterChange automatically persists the change (necessary for correctness)
func (status *Status) SetClusterChange(newClusterChangeIndex, newClusterChangeTerm int64) {

	var (
		err error
	)

	status.storage.BeginTransaction()

	status.clusterChangeIndex = newClusterChangeIndex
	if err = status.storage.Set(
		[]byte("/raft/clusterChangeIndex"),
		[]byte(toString(newClusterChangeIndex))); err != nil {
		panic(err)
	}

	status.clusterChangeTerm = newClusterChangeTerm
	if err = status.storage.Set(
		[]byte("/raft/clusterChangeTerm"),
		[]byte(toString(newClusterChangeTerm))); err != nil {
		panic(err)
	}

	status.storage.Commit()
}

// LeaderLastHeard is the moment in time when we last heard
// from the leader (initialized to now - 1 month, in which "now"
// is the moment when the initialization code is executed)
func (status *Status) LeaderLastHeard() time.Time {
	return status.leaderLastHeard
}

// SetLeaderLastHeard does not persist the change (not needed for correctness)
func (status *Status) SetLeaderLastHeard(instant time.Time) {
	status.leaderLastHeard = instant
}

// MinElectionTimeout is the minimum time interval the raft node is configured
// to wait before starting an election
func (status *Status) MinElectionTimeout() time.Duration {
	return status.minElectionTimeout
}
