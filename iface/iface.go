package iface

// LogEntry represents an entry in the log.
// `command` is an interface{} because there is no a-priori structure
// to be imposed (on the contrary: each application on top of raft has
// its intended semantics for the command)
type LogEntry struct {
	Term    int64
	Command interface{}
}

// RaftLog provides log read/delete/append functionalities.
//
// A particular instance of the struct represents the log of 1
// raft node
type RaftLog interface {

	// Set writes a log entry to disk (if one exists at `index`, it will be overwritten)
	Set(index int64, entry LogEntry) error

	// Get reads a log entry, returning a pointer to it.
	//
	// If no entry exists at `index`, returns `logEntry == nil`.
	Get(index int64) (*LogEntry, error)

	// LastIndex returns either -1 or the highest index of
	// any log entry present in disk
	LastIndex() int64
}

// PeerAddress is the network-address where a peer can be contacted
type PeerAddress string

// Status holds information the raft node needs to operate
type Status interface {
	// NodeAddress returns the network address which can be used to contact this node
	NodeAddress() PeerAddress

	// CurrentTerm is the current raft turn, as seen by a raft node (0 in the beginning)
	CurrentTerm() int64

	// VotedFor is the address of another raft node for which
	// this raft node has voted in this turn.
	//
	// If this node has not voted yet, it should be set to the empty string (i.e. "")
	VotedFor() PeerAddress

	// CommitIndex is the index of the most up-to-date log entry known to be
	// committed by this raft node (-1 in the beginning)
	CommitIndex() int64

	// LastApplied is the index of the last committed log entry that has already
	// been applied by the raft state machine (-1 in the beginning)
	LastApplied() int64

	// PeerAddresses is the slice of peer addresses (does not include the node itself)
	PeerAddresses() []PeerAddress

	// NextIndex is the (as known by leader) next log index a peer is waiting for
	// (0 in the beginning)
	NextIndex(peer PeerAddress) int64

	// MatchIndex is the (as known by leader) last log index where the leader's log
	// matches the peer's log (-1 in the beginning)
	MatchIndex(peer PeerAddress) int64

	// ClusterChangeIndex is the latest UNCOMITTED log index
	// corresponding to a cluster-change command. -1 if none
	ClusterChangeIndex() int64

	// ClusterChangeTerm is the term of log entry whose
	// index is `clusterChangeIndex`. -1 if none
	ClusterChangeTerm() int64
}
