package status

import (
	"encoding/json"
	"simpleraft/storage"
	"strconv"
)

// helper
func toString(x int64) string {
	return strconv.FormatInt(x, 10)
}

// PeerAddress is the network-address where a peer can be contacted
type PeerAddress string

// Status holds information the raft node needs to operate.
// All struct members are non-exported to enforce usage of
// functions for reading/writing these values
type Status struct {
	nodeAddress   PeerAddress
	currentTerm   int64
	votedFor      PeerAddress
	commitIndex   int64
	lastApplied   int64
	peerAddresses []PeerAddress
	nextIndex     map[PeerAddress]int64
	matchIndex    map[PeerAddress]int64
	// The latest UNCOMITTED log index corresponding to a cluster-change command
	// -1 if none
	clusterChangeIndex int64
	// The term of log entry whose index is `clusterChangeIndex`
	// -1 if none
	clusterChangeTerm int64
	// pointer to data storage
	storage *storage.Storage
}

// New constructs a Status struct, automatically recovering state from disk, if any.
// `peerAddresses` works as a default value: it will only be used if there is no
// alternative record already in the disk storage
func New(nodeAddress PeerAddress, peerAddresses []PeerAddress,
	storage *storage.Storage) *Status {

	var (
		err                     error
		currentTermSlice        []byte
		currentTerm             int64
		votedForSlice           []byte
		votedFor                PeerAddress
		nextIndex               map[PeerAddress]int64
		matchIndex              map[PeerAddress]int64
		clusterChangeIndexSlice []byte
		clusterChangeIndex      int64
		clusterChangeTermSlice  []byte
		clusterChangeTerm       int64
		oldPeerAddresses        []byte
		peerAddressesJSON       []byte
	)

	// retrieve currentTerm from datastore
	currentTermSlice, err = storage.Get(
		nil,
		[]byte("/raft/currentTerm"))

	// TODO: how to handle an error here ?
	if err != nil {
		panic(err)
	}

	if currentTermSlice == nil {
		currentTerm = 0
	} else {
		currentTerm, err = strconv.ParseInt(string(currentTermSlice), 10, 64)
		// TODO: how to handle an error here ?
		if err != nil {
			panic(err)
		}
	}

	// retrieve votedFor from datastore
	votedForSlice, err = storage.Get(
		nil,
		[]byte("/raft/votedFor"))

	// TODO: how to handle an error here ?
	if err != nil {
		panic(err)
	}

	if votedForSlice == nil {
		votedFor = ""
	} else {
		votedFor = PeerAddress(string(votedForSlice))
	}

	// retrieve peerAddresses from datastore (if any)
	oldPeerAddresses, err = storage.Get(
		nil,
		[]byte("/raft/peerAddresses"))

	// TODO: how to handle an error here ?
	if err != nil {
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

	nextIndex = make(map[PeerAddress]int64)
	matchIndex = make(map[PeerAddress]int64)

	for _, address := range peerAddresses {
		nextIndex[address] = 0
		matchIndex[address] = -1
	}

	// retrieve clusterChangeIndex from datastore
	clusterChangeIndexSlice, err = storage.Get(
		nil,
		[]byte("/raft/clusterChangeIndex"))

	// TODO: how to handle an error here ?
	if err != nil {
		panic(err)
	}

	if clusterChangeIndexSlice == nil {
		clusterChangeIndex = -1
	} else {
		clusterChangeIndex, err = strconv.ParseInt(string(clusterChangeIndexSlice), 10, 64)
		// TODO: how to handle an error here ?
		if err != nil {
			panic(err)
		}
	}

	// retrieve clusterChangeTerm from datastore
	clusterChangeTermSlice, err = storage.Get(
		nil,
		[]byte("/raft/clusterChangeTerm"))

	// TODO: how to handle an error here ?
	if err != nil {
		panic(err)
	}

	if clusterChangeTermSlice == nil {
		clusterChangeTerm = -1
	} else {
		clusterChangeTerm, err = strconv.ParseInt(string(clusterChangeTermSlice), 10, 64)
		// TODO: how to handle an error here ?
		if err != nil {
			panic(err)
		}
	}

	return &Status{
		nodeAddress:        nodeAddress,
		currentTerm:        currentTerm,
		votedFor:           votedFor,
		commitIndex:        -1,
		lastApplied:        -1,
		peerAddresses:      peerAddresses,
		nextIndex:          nextIndex,
		matchIndex:         matchIndex,
		clusterChangeIndex: clusterChangeIndex,
		clusterChangeTerm:  clusterChangeTerm,
		storage:            storage,
	}
}

// NodeAddress returns the network address which can be used to contact this node
func (status *Status) NodeAddress() PeerAddress {
	return status.nodeAddress
}

// CurrentTerm is the current raft turn, as seen by a raft node (0 in the beginning)
func (status *Status) CurrentTerm() int64 {
	return status.currentTerm
}

// SetCurrentTerm automatically persists the change (necessary for correctness)
func (status *Status) SetCurrentTerm(newTerm int64) {
	status.currentTerm = newTerm
	err := status.storage.Set(
		[]byte("/raft/currentTerm"),
		[]byte(toString(newTerm)))

	// TODO: how to handle an error here ?
	if err != nil {
		panic(err)
	}
}

// VotedFor is the address of another raft node for which
// this raft node has voted in this turn.
//
// If this node has not voted yet, it should be set to the empty string (i.e. "")
func (status *Status) VotedFor() PeerAddress {
	return status.votedFor
}

// SetVotedFor automatically persists the change (necessary for correctness)
func (status *Status) SetVotedFor(newVotedFor PeerAddress) {
	status.votedFor = newVotedFor
	err := status.storage.Set(
		[]byte("/raft/votedFor"),
		[]byte(newVotedFor))

	// TODO: how to handle an error here ?
	if err != nil {
		panic(err)
	}
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
func (status *Status) PeerAddresses() []PeerAddress {
	return status.peerAddresses
}

// SetPeerAddresses automatically persists the change (necessary for correctness)
func (status *Status) SetPeerAddresses(newPeers []PeerAddress) {
	var (
		marshal       []byte
		err           error
		newNextIndex  map[PeerAddress]int64
		newMatchIndex map[PeerAddress]int64
		addr          PeerAddress
		nextIndex     int64
		matchIndex    int64
		ok            bool
	)

	if marshal, err = json.Marshal(newPeers); err != nil {
		panic(err)
	}
	status.storage.Set([]byte("/raft/peerAddresses"), marshal)

	newNextIndex = make(map[PeerAddress]int64)
	newMatchIndex = make(map[PeerAddress]int64)

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
func (status *Status) NextIndex(peer PeerAddress) int64 {
	return status.nextIndex[peer]
}

// SetNextIndex does not persist the change (not needed for correctness)
func (status *Status) SetNextIndex(peer PeerAddress, nextIndex int64) {
	status.nextIndex[peer] = nextIndex
}

// MatchIndex is the (as known by leader) last log index where the leader's log
// matches the peer's log (-1 in the beginning)
func (status *Status) MatchIndex(peer PeerAddress) int64 {
	return status.matchIndex[peer]
}

// SetMatchIndex does not persist the change (not needed for correctness)
func (status *Status) SetMatchIndex(peer PeerAddress, matchIndex int64) {
	status.matchIndex[peer] = matchIndex
}

// ClusterChangeIndex is the latest UNCOMITTED log index
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
	err = status.storage.Set(
		[]byte("/raft/clusterChangeIndex"),
		[]byte(toString(newClusterChangeIndex)))

	// TODO: how to handle an error here ?
	if err != nil {
		panic(err)
	}

	status.clusterChangeTerm = newClusterChangeTerm
	err = status.storage.Set(
		[]byte("/raft/clusterChangeTerm"),
		[]byte(toString(newClusterChangeTerm)))

	// TODO: how to handle an error here ?
	if err != nil {
		panic(err)
	}

	status.storage.Commit()
}
