package status

import (
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
	// pointer to data storage
	storage *storage.Storage
}

// New constructs a Status struct, automatically recovering state from disk, if any
func New(nodeAddress PeerAddress, peerAddresses []PeerAddress,
	storage *storage.Storage) *Status {

	var (
		err              error
		currentTermSlice []byte
		currentTerm      int64
		votedForSlice    []byte
		votedFor         PeerAddress
		nextIndex        map[PeerAddress]int64
		matchIndex       map[PeerAddress]int64
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

	nextIndex = make(map[PeerAddress]int64)
	matchIndex = make(map[PeerAddress]int64)

	for _, address := range peerAddresses {
		nextIndex[address] = 0
		matchIndex[address] = -1
	}

	return &Status{
		nodeAddress:   nodeAddress,
		currentTerm:   currentTerm,
		votedFor:      votedFor,
		commitIndex:   -1,
		lastApplied:   -1,
		peerAddresses: peerAddresses,
		nextIndex:     nextIndex,
		matchIndex:    matchIndex,
		storage:       storage,
	}
}

// NodeAddress returns the network address which can be used to contact this node
func (status *Status) NodeAddress() PeerAddress {
	return status.nodeAddress
}

// CurrentTerm is the current raft turn, as seen by a raft node (0 in the beggining)
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
// committed by this raft node (-1 in the beggining)
func (status *Status) CommitIndex() int64 {
	return status.commitIndex
}

// SetCommitIndex does not persist the change (not needed for correctness)
func (status *Status) SetCommitIndex(newCommitIndex int64) {
	status.commitIndex = newCommitIndex
}

// LastApplied is the index of the last committed log entry that has already
// been applied by the raft state machine (-1 in the beggining)
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

// NextIndex is the (as known by leader) next log index a peer is waiting for
// (0 in the beggining)
func (status *Status) NextIndex(peer PeerAddress) int64 {
	return status.nextIndex[peer]
}

// SetNextIndex does not persist the change (not needed for correctness)
func (status *Status) SetNextIndex(peer PeerAddress, nextIndex int64) {
	status.nextIndex[peer] = nextIndex
}

// MatchIndex is the (as known by leader) last log index where the leader's log
// matches the peer's log (-1 in the beggining)
func (status *Status) MatchIndex(peer PeerAddress) int64 {
	return status.matchIndex[peer]
}

// SetMatchIndex does not persist the change (not needed for correctness)
func (status *Status) SetMatchIndex(peer PeerAddress, matchIndex int64) {
	status.matchIndex[peer] = matchIndex
}
