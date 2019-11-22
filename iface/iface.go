package iface

var (
	// StateFollower represents that the raft node is a follower
	StateFollower string = "follower"

	// StateCandidate represents that the raft node is a candidate
	StateCandidate string = "candidate"

	// StateLeader represents that the raft node is a leader
	StateLeader string = "leader"
)

var (
	// EntryStateMachineCommand : log entry contains a command
	// to be executed by the state machine (application layer
	// on top of raft)
	EntryStateMachineCommand = "state machine command"

	// EntryAddServer : log entry contains a PeerAddress
	// which is the node to be added to the cluster
	// configuration
	EntryAddServer = "add server to cluster"

	// EntryRemoveServer : log entry contains a PeerAddress
	// which is the node to be removed from the cluster
	// configuration
	EntryRemoveServer = "remove server from cluster"

	// EntryNoOp : log entry contains no command at all. It
	// is there just for the new leader to establish at least
	// one log entry from its own term (guaranteeing that it
	// can update its commitIndex upon entry commitment)
	EntryNoOp = "no operation"
)

// LogEntry represents an entry in the log.
// `command` is []byte because there is no a-priori structure
// to be imposed (on the contrary: each application on top of raft has
// its intended semantics for the command)
type LogEntry struct {
	Term    int64
	Kind    string
	Command []byte
}

// RaftLog provides log read/delete/append functionalities.
//
// A particular instance of the struct represents the log of 1
// raft node
//
// The interface exposes a read-only log because that is what is needed
// by the RuleHandler component. A read-write version is exposed at the
// implementation level (subpackage simpleraft/raftlog), which should only
// be used by the Executor component
type RaftLog interface {

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
// The interface exposes a read-only status because that is what is needed
// by the RuleHandler component. A read-write version is exposed at the
// implementation level (subpackage simpleraft/status), which should only
// be used by the Executor component
type Status interface {
	// State returns the state of the raft node (one of
	// iface.StateLeader, iface.StateCandidate, iface.StateFollower)
	State() string

	// NodeAddress returns the network address which can be used to contact this node
	NodeAddress() PeerAddress

	// CurrentTerm is the current raft turn, as seen by a raft node (0 in the beginning)
	CurrentTerm() int64

	// VotedFor is the address of another raft node for which
	// this raft node has voted in this turn.
	//
	// If this node has not voted yet, it should be set to the empty string (i.e. "")
	VotedFor() PeerAddress

	// VoteCount counts how many votes this node received until now
	// in its candidacy. 0 in the beggining
	VoteCount() int64

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

	// ClusterChangeIndex is the latest log index
	// corresponding to a cluster-change command. -1 if none
	ClusterChangeIndex() int64

	// ClusterChangeTerm is the term of log entry whose
	// index is `clusterChangeIndex`. -1 if none
	ClusterChangeTerm() int64
}

// StateMachine is not part of the raft protocol. Actually,
// it belongs to the application layer (on top of raft).
//
// This interface must be implemented by the application
// on the occasion of running raft
type StateMachine interface {
	// Apply means: The state machine implements `command` and returns
	// the `result` of the command.
	//
	// The interpretation of `command` byte-slice is up to the application
	// layer. Also, `result` may be empty, provided the application understands
	// this is an appropriate result.
	Apply(command []byte) (result []byte)
}

// MsgStateChanged means: The raft state (leader/candidate/follower)
// has just changed to a new value, DIFFERENT from the prior one.
type MsgStateChanged struct {
}

// MsgAppendEntries means: the caller (i.e. the raft leader)
// called AppendEntries on the current raft node
type MsgAppendEntries struct {
	Term              int64
	LeaderAddress     PeerAddress
	PrevLogIndex      int64
	PrevLogTerm       int64
	Entries           []LogEntry
	LeaderCommitIndex int64
}

// MsgRequestVote means: the caller (i.e. a raft candidadate)
// called RequestVote on the current raft node
type MsgRequestVote struct {
	Term             int64
	CandidateAddress PeerAddress
	LastLogIndex     int64
	LastLogTerm      int64
}

// MsgAddServer means: the caller called AddServer on
// the current raft node
type MsgAddServer struct {
	NewServerAddress PeerAddress
}

// MsgRemoveServer means: the caller called RemoveServer on
// the current raft node
type MsgRemoveServer struct {
	OldServerAddress PeerAddress
}

// MsgTimeout means the raft node's timer has timed out
type MsgTimeout struct {
}

// MsgStateMachineCommand means: the caller issued a
// state machine command to the current raft node.
//
// The value of `Command` must be interpreted by the specific
// state machine implementation (i.e. application layer)
// running on top of raft
type MsgStateMachineCommand struct {
	Command []byte
}

// ReplyNotLeader means: A client requested an action
// to the current raft node; this node is not the leader,
// but the action requires the leader; so we must warn the
// caller that we are not the leader
type ReplyNotLeader struct {
}

// ReplyCheckLater means: A client requested an action
// to the current raft node; this node cannot complete
// the action right away; so we must warn the client
// that it should check again at a later time to see
// if the action has been performed
type ReplyCheckLater struct {
}

// ReplyFailed means: A client requested an action
// to the current raft node; some kind of failure occured;
// so we must communicate this fact to the client
type ReplyFailed struct {
}

// ReplyCompleted means: A client requested an action
// to the current raft node sometime in the past; now
// the client is probing whether the action has been done
// or not, and it turns out that the action is completed;
// so we must communicate the results of the action to the caller.
//
// The value of `Result` can be whathever the client can understand.
// It is up to the layer above the raft protocol (i.e. application
// state machine) to define what is acceptable as a value for `Result`.
// This is why it is a []byte: so that the upper layer can decode it as
// it wishes
type ReplyCompleted struct {
	Result []byte
}

// ReplyDecidedVote means: A caller request the
// current raft node to vote in them; we have decided
// whether to vote or not; so we must communicate
// this decision to the caller
type ReplyDecidedVote struct {
}

// ReplyAppendEntries means: A caller called
// AppendEntries on the current raft node; we
// have processed this request; so we must
// communicate the results to the caller
type ReplyAppendEntries struct {
}

// ActionAppendLog means some entries should
// be appended to the log
type ActionAppendLog struct {
	Entries []LogEntry
}

// ActionDeleteLog means `Count` entries should be removed
// from the log
type ActionDeleteLog struct {
	Count int64
}

// ActionSetState means the raft node state
// should be changed
type ActionSetState struct {
	NewState string
}

// ActionSetCurrentTerm means the raft node's
// currentTerm should be changed
type ActionSetCurrentTerm struct {
	NewCurrentTerm int64
}

// ActionSetVotedFor means the raft node's
// votedFor should be changed
type ActionSetVotedFor struct {
	NewVotedFor PeerAddress
}

// ActionSetVoteCount means the raft node's
// votedCount should be changed
type ActionSetVoteCount struct {
	NewVoteCount int64
}

// ActionSetCommitIndex means the raft node's
// commitIndex should be changed
type ActionSetCommitIndex struct {
	NewCommitIndex int64
}

// ActionSetLastApplied means the raft node's
// lastApplied should be changed
type ActionSetLastApplied struct {
	NewLastApplied int64
}

// ActionAddServer means the raft node's
// lists/maps of current peers should be updated
// to include an additional address
type ActionAddServer struct {
	NewServerAddress PeerAddress
}

// ActionRemoveServer means the raft node's
// lists/maps of current peers should be updated
// to exclude a specific address
type ActionRemoveServer struct {
	OldServerAddress PeerAddress
}

// ActionSetNextIndex means the raft node's
// nextIndex should be changed for a
// specific peer
type ActionSetNextIndex struct {
	Peer         PeerAddress
	NewNextIndex int64
}

// ActionSetMatchIndex means the raft node's
// matchIndex should be changed for a
// specific peer
type ActionSetMatchIndex struct {
	Peer          PeerAddress
	NewMatchIndex int64
}

// ActionSetClusterChangeIndex means the raft node's
// clusterChangeIndex should be changed
type ActionSetClusterChangeIndex struct {
	NewClusterChangeIndex int64
}

// ActionSetClusterChangeTerm means the raft node's
// clusterChangeTerm should be changed
type ActionSetClusterChangeTerm struct {
	NewClusterChangeTerm int64
}

// ActionResetTimer means the raft node's
// timer should be reset (i.e. it will count
// down until 0 again)
type ActionResetTimer struct {
	// HalfTime means: Reset the timer to half the min election timeout.
	// This is needed for leaders to send AppendEntries before any node's
	// election timeout
	HalfTime bool
}

// ActionStateMachineApply means
// a log entry's command should be
// applied by the application layer's
// state machine
type ActionStateMachineApply struct {
	EntryIndex int64
}

// ActionAppendEntries means the current raft node
// (the raft leader) wants to send AppendEntries
// message to another node. Besides the fields
// included in this struct, all remaining necessary
// information (leader commit index etc.) is
// automatically inferred by the Executor component
type ActionAppendEntries struct {
	Destination  PeerAddress
	Entries      []LogEntry
	PrevLogIndex int64
	PrevLogTerm  int64
}

// RuleHandler is the interface representing the actions performed by the raft node
// when any event (message) arrives at the node.
//
// This interface isolates the abstract rules of the raft protocol from the implementation
// details related to log/status/transport services
//
// All methods must return a slice of structs belonging to the Reply* and Action*
// family of structs
type RuleHandler interface {
	FollowerOnStateChanged(msg MsgStateChanged, log *RaftLog, status *Status) []interface{}
	FollowerOnAppendEntries(msg MsgAppendEntries, log *RaftLog, status *Status) []interface{}
	FollowerOnRequestVote(msg MsgRequestVote, log *RaftLog, status *Status) []interface{}
	FollowerOnAddServer(msg MsgAddServer, log *RaftLog, status *Status) []interface{}
	FollowerOnRemoveServer(msg MsgRemoveServer, log *RaftLog, status *Status) []interface{}
	FollowerOnTimeout(msg MsgTimeout, log *RaftLog, status *Status) []interface{}
	FollowerOnStateMachineCommand(msg MsgStateMachineCommand, log *RaftLog, status *Status) []interface{}

	CandidateOnStateChanged(msg MsgStateChanged, log *RaftLog, status *Status) []interface{}
	CandidateOnAppendEntries(msg MsgAppendEntries, log *RaftLog, status *Status) []interface{}
	CandidateOnRequestVote(msg MsgRequestVote, log *RaftLog, status *Status) []interface{}
	CandidateOnAddServer(msg MsgAddServer, log *RaftLog, status *Status) []interface{}
	CandidateOnRemoveServer(msg MsgRemoveServer, log *RaftLog, status *Status) []interface{}
	CandidateOnTimeout(msg MsgTimeout, log *RaftLog, status *Status) []interface{}
	CandidateOnStateMachineCommand(msg MsgStateMachineCommand, log *RaftLog, status *Status) []interface{}

	LeaderOnStateChanged(msg MsgStateChanged, log *RaftLog, status *Status) []interface{}
	LeaderOnAppendEntries(msg MsgAppendEntries, log *RaftLog, status *Status) []interface{}
	LeaderOnRequestVote(msg MsgRequestVote, log *RaftLog, status *Status) []interface{}
	LeaderOnAddServer(msg MsgAddServer, log *RaftLog, status *Status) []interface{}
	LeaderOnRemoveServer(msg MsgRemoveServer, log *RaftLog, status *Status) []interface{}
	LeaderOnTimeout(msg MsgTimeout, log *RaftLog, status *Status) []interface{}
	LeaderOnStateMachineCommand(msg MsgStateMachineCommand, log *RaftLog, status *Status) []interface{}
}
