package executor

import (
	"math/rand"
	"simpleraft/raftlog"
	"simpleraft/status"
	"simpleraft/storage"
	"simpleraft/transport"
	"time"
)

// LogEntryType distinguishes between the possible
// meanings of a command stored inside a LogEntry.
type LogEntryType int

const (
	// UserCommand means the log entry stores a user command
	UserCommand LogEntryType = iota
	// AddServer means the log entry contains a server address to be added to the cluster
	AddServer LogEntryType = iota
	// RemoveServer means the log entry contains a server address to be removed from the cluster
	RemoveServer LogEntryType = iota
	// NoOp means the log entry contains no command, and is used only so
	// that the new term leader can commit something
	NoOp LogEntryType = iota
)

// Executor is a manager of raft protocol state. The underlying components should
// not be directly interacted with, unless you know such an interaction will not
// lead the system to an inconsistent state. Example of inconsistency:
// Directly changing `status.nodeAddress` while the transport service
// is active and listening
type Executor struct {
	status             *status.Status
	log                *raftlog.RaftLog
	transport          *transport.Transport
	minElectionTimeout int
	maxElectionTimeout int
	broadcastInterval  int
	timer              *time.Timer
}

// New instantiates a configured Executor instance but DOES NOT initiate it
func New(
	nodeAddress status.PeerAddress,
	peerAddresses []status.PeerAddress,
	storageFilePath string,
	minElectionTimeout,
	maxElectionTimeout,
	broadcastInterval int) (*Executor, error) {

	var (
		stat     *status.Status
		store    *storage.Storage
		executor *Executor
		log      *raftlog.RaftLog
		transp   *transport.Transport
		timer    *time.Timer
		err      error
	)

	if store, err = storage.New(storageFilePath); err != nil {
		return nil, err
	}

	stat = status.New(nodeAddress, peerAddresses, store)

	if log, err = raftlog.New(store); err != nil {
		return nil, err
	}

	// important to use stat.NodeAddress() because effective address may not
	// equal the `nodeAddress` argument value (if there was another record on disk)
	transp = transport.New(string(stat.NodeAddress()))

	timer = time.NewTimer(executor.randomElectionTimeout())
	if !timer.Stop() {
		<-timer.C
	}

	executor = &Executor{
		status:             stat,
		log:                log,
		transport:          transp,
		minElectionTimeout: minElectionTimeout,
		maxElectionTimeout: maxElectionTimeout,
		broadcastInterval:  broadcastInterval,
		timer:              timer,
	}

	return executor, nil
}

// Run executes the raft protocol loop. This is a blocking function
// (it does NOT create another goroutine for the loop)
func (executor *Executor) Run() {
	executor.transport.Listen()
	executor.timer.Reset(executor.randomElectionTimeout())

	for {
		select {
		case msg := <-executor.transport.ReceiverChan():
			executor.processIncoming(msg)

		case <-executor.timer.C:
			executor.processTick()
		}
	}
}

func (executor *Executor) randomElectionTimeout() time.Duration {
	timeoutInt := executor.minElectionTimeout + rand.Intn(
		executor.maxElectionTimeout-executor.minElectionTimeout)

	return time.Duration(timeoutInt) * time.Millisecond
}

func (executor *Executor) processIncoming(msg transport.IncomingMessage) {
	switch msg.Endpoint {
	case "/appendEntries":

	case "/requestVote":

	case "/addServer":

	case "/removeServer":

	case "/stateMachineCommand":

	default:
		panic("unknown endpoint: " + msg.Endpoint)
	}

	// make sure something is sent as response, even if RuleHandler did not
	// respond
	msg.ReplyChan <- []byte("Empty response")
}

func (executor *Executor) processTick() {

}
