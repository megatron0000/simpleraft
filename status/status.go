package status

import (
	"strconv"

	"modernc.org/kv"
)

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
	db *kv.DB
}

func toString(x int64) string {
	return strconv.FormatInt(x, 10)
}

func New(nodeID int64, db *kv.DB) *Status {

	var (
		err              error
		currentTermSlice []byte
		currentTerm      int64
		votedForSlice    []byte
		votedFor         int64
	)

	// retrieve currentTerm from datastore
	currentTermSlice, err = db.Get(nil, []byte("/raft/nodeID="+toString(nodeID)+"/currentTerm"))

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
	votedForSlice, err = db.Get(nil, []byte("/raft/nodeID="+toString(nodeID)+"/votedFor"))

	// TODO: how to handle an error here ?
	if err != nil {
		panic(err)
	}

	if votedForSlice == nil {
		votedFor = 0
	} else {
		votedFor, err = strconv.ParseInt(string(votedForSlice), 10, 64)
		// TODO: how to handle an error here ?
		if err != nil {
			panic(err)
		}
	}

	return &Status{
		nodeID: nodeID,
		currentTerm: currentTerm,
		votedFor: votedFor,
		commitIndex: -1,
		lastApplied: -1,
		db: db
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

// VotedFor is the address (a positive number) of another raft node for which
// this raft node has voted in this turn.
//
// If this node has not voted yet, it should be set to -1
func (status *Status) VotedFor() int64 {
	return status.votedFor
}

// CommitIndex is the index of the most up-to-date log entry known to be
// committed by this raft node
func (status *Status) CommitIndex() int64 {
	return status.commitIndex
}

// LastApplied is the index of the last committed log entry that has already
// been applied by the raft state machine
func (status *Status) LastApplied() int64 {
	return status.lastApplied
}