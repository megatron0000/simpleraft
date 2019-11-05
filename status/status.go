package status

import (
	"simpleraft/storage"
	"strconv"
)

// helper
func toString(x int64) string {
	return strconv.FormatInt(x, 10)
}

// Status holds information the raft node needs to operate.
// All struct members are non-exported to enforce usage of
// functions for reading/writing these values
type Status struct {
	nodeID      int64
	currentTerm int64
	votedFor    int64
	commitIndex int64
	lastApplied int64
	// pointer to data storage
	storage *storage.Storage
}

// New constructs a Status struct, automatically recovering state from disk, if any
func New(nodeID int64, storage *storage.Storage) *Status {

	var (
		err              error
		currentTermSlice []byte
		currentTerm      int64
		votedForSlice    []byte
		votedFor         int64
	)

	// retrieve currentTerm from datastore
	currentTermSlice, err = storage.Get(
		nil,
		[]byte("/raft/nodeID="+toString(nodeID)+"/currentTerm"))

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
		[]byte("/raft/nodeID="+toString(nodeID)+"/votedFor"))

	// TODO: how to handle an error here ?
	if err != nil {
		panic(err)
	}

	if votedForSlice == nil {
		votedFor = -1
	} else {
		votedFor, err = strconv.ParseInt(string(votedForSlice), 10, 64)
		// TODO: how to handle an error here ?
		if err != nil {
			panic(err)
		}
	}

	return &Status{
		nodeID:      nodeID,
		currentTerm: currentTerm,
		votedFor:    votedFor,
		commitIndex: -1,
		lastApplied: -1,
		storage:     storage,
	}
}

// NodeID is the unique identification of this raft node (a positive number)
func (status *Status) NodeID() int64 {
	return status.nodeID
}

// CurrentTerm is the current raft turn, as seen by a raft node
func (status *Status) CurrentTerm() int64 {
	return status.currentTerm
}

// SetCurrentTerm automatically persists the change (necessary for correctness)
func (status *Status) SetCurrentTerm(newTerm int64) {
	status.currentTerm = newTerm
	err := status.storage.Set(
		[]byte("/raft/nodeID="+toString(status.nodeID)+"/currentTerm"),
		[]byte(toString(newTerm)))

	// TODO: how to handle an error here ?
	if err != nil {
		panic(err)
	}
}

// VotedFor is the address (a positive number) of another raft node for which
// this raft node has voted in this turn.
//
// If this node has not voted yet, it should be set to -1
func (status *Status) VotedFor() int64 {
	return status.votedFor
}

// SetVotedFor automatically persists the change (necessary for correctness)
func (status *Status) SetVotedFor(newVotedFor int64) {
	status.votedFor = newVotedFor
	err := status.storage.Set(
		[]byte("/raft/nodeID="+toString(status.nodeID)+"/votedFor"),
		[]byte(toString(newVotedFor)))

	// TODO: how to handle an error here ?
	if err != nil {
		panic(err)
	}
}

// CommitIndex is the index of the most up-to-date log entry known to be
// committed by this raft node
func (status *Status) CommitIndex() int64 {
	return status.commitIndex
}

// SetCommitIndex does not persist the change (not needed for correctness)
func (status *Status) SetCommitIndex(newCommitIndex int64) {
	status.commitIndex = newCommitIndex
}

// LastApplied is the index of the last committed log entry that has already
// been applied by the raft state machine
func (status *Status) LastApplied() int64 {
	return status.lastApplied
}

// SetLastApplied does not persist the change (not needed for correctness)
func (status *Status) SetLastApplied(newLastApplied int64) {
	status.lastApplied = newLastApplied
}
