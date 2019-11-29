package executor

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"simpleraft/iface"
	"simpleraft/raftlog"
	"simpleraft/status"
	"simpleraft/storage"
	"simpleraft/transport"
	"strconv"
	"time"
)

// Executor is a manager of raft protocol state. The underlying components should
// not be directly interacted with, unless you know such an interaction will not
// lead the system to an inconsistent state. Example of inconsistency:
// Directly changing `status.nodeAddress` while the transport service
// is active and listening
type Executor struct {
	// lower level components
	storage   *storage.Storage
	status    *status.Status
	log       *raftlog.RaftLog
	transport *transport.Transport

	// higher level components
	handler      iface.RuleHandler
	stateMachine iface.StateMachine

	// configuration. Should be the same among
	// all nodes in the cluster, lest unexpected
	// results may issue
	minElectionTimeout time.Duration
	maxElectionTimeout time.Duration

	timer *time.Timer
	// buffered channel (when a peer answer one of our AppendEntries, this channel
	// will receive the response)
	appendEntriesReplyChan chan iface.MsgAppendEntriesReply
	// buffered channel (when a peer answer one of our RequestVote, this channel
	// will receive the response)
	requestVoteReplyChan chan iface.MsgRequestVoteReply

	// internal channels to implement Stop() method
	stop    chan bool
	stopped chan bool
}

// New instantiates a configured Executor instance but DOES NOT initiate it.
// `handlerImplementation` and `stateMachineImplementation` must be
// implementations of the respective interfaces.
//
// `nodeAddress` and `peerAddresses` will be treated as default values: if
// there is any prior information about them present in disk, the new values
// WILL NOT be used. The old ones (recovered from disk storage) will be
// preferred instead.
//
// The client of this struct MUST defer executor.TearDown() for correctness
func New(
	nodeAddress iface.PeerAddress,
	peerAddresses []iface.PeerAddress,
	storageFilePath string,
	minElectionTimeout time.Duration,
	maxElectionTimeout time.Duration,
	handlerImplementation iface.RuleHandler,
	stateMachineImplementation iface.StateMachine) (*Executor, error) {

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

	stat = status.New(nodeAddress, peerAddresses, store, minElectionTimeout)

	if log, err = raftlog.New(store); err != nil {
		return nil, err
	}

	// important to use stat.NodeAddress() because effective address may not
	// equal the `nodeAddress` argument value (if there was another record on disk)
	transp = transport.New(string(stat.NodeAddress()))

	timer = time.NewTimer(time.Millisecond)
	if !timer.Stop() {
		<-timer.C
	}

	executor = &Executor{
		storage:                store,
		status:                 stat,
		log:                    log,
		transport:              transp,
		handler:                handlerImplementation,
		stateMachine:           stateMachineImplementation,
		minElectionTimeout:     minElectionTimeout,
		maxElectionTimeout:     maxElectionTimeout,
		timer:                  timer,
		appendEntriesReplyChan: make(chan iface.MsgAppendEntriesReply, 1000),
		requestVoteReplyChan:   make(chan iface.MsgRequestVoteReply, 1000),
		stop:                   nil,
		stopped:                make(chan bool),
	}

	// initialize random seed so that different raft nodes
	// will use different random sequences
	rand.Seed(time.Now().UnixNano())

	return executor, nil
}

// TearDown closes what must be closed for the executor to safely stop
// executing
func (executor *Executor) TearDown() {
	executor.storage.Close()
}

// Run executes the raft protocol loop. This is a blocking function
// (it does NOT create another goroutine for the loop).
func (executor *Executor) Run() {

	if executor.stop != nil {
		panic("executor: called Run() but was already running")
	}

	executor.stop = make(chan bool)

	var (
		actions []interface{}
	)

	executor.transport.Listen()

	// trigger first event change (actually, not a change
	// since there was no prior state in the first place )
	executor.status.SetState(iface.StateFollower)
	actions = executor.forwardStateChanged()
	executor.implementActions(actions, nil, iface.MsgStateChanged{})

	// loop listening for (and reacting to) events
	for {

		select {
		case msg := <-executor.transport.ReceiverChan():
			actions = executor.forwardIncoming(msg)
			executor.implementActions(actions, msg.ReplyChan, msg)

		case <-executor.timer.C:
			actions = executor.forwardTick()
			executor.implementActions(actions, nil, nil)

		case msg := <-executor.appendEntriesReplyChan:
			actions = executor.forwardReply(msg)
			executor.implementActions(actions, nil, msg)

		case msg := <-executor.requestVoteReplyChan:
			actions = executor.forwardReply(msg)
			executor.implementActions(actions, nil, msg)

		case <-executor.stop:
			executor.transport.Close()
			executor.stop = nil
			executor.stopped <- true
			return

		}
	}

}

// Stop stops the execution of Run(). It is idempotent
// and safe to call even if Run() actually is not
// being executed
func (executor *Executor) Stop() {
	if executor.stop != nil {
		executor.stop <- true
		<-executor.stopped
	}
}

func (executor *Executor) randomElectionTimeout() time.Duration {
	return executor.minElectionTimeout +
		time.Duration(rand.Float64()*float64(executor.maxElectionTimeout-executor.minElectionTimeout))
}

// forwardIncoming forwards `msg` to handler and picks up its response.
//
// Returns the list of actions (as received from handler) or nil in case of error
func (executor *Executor) forwardIncoming(msg transport.IncomingMessage) []interface{} {

	var (
		appendEntries       *iface.MsgAppendEntries
		requestVote         *iface.MsgRequestVote
		addServer           *iface.MsgAddServer
		removeServer        *iface.MsgRemoveServer
		stateMachineCommand *iface.MsgStateMachineCommand
		stateMachineProbe   *iface.MsgStateMachineProbe
		actions             []interface{}
		err                 error
	)

	fmt.Printf("forwarding (%T)%+v to %s handler \n\n", msg, struct {
		Endpoint string
		Data     string
	}{
		Endpoint: msg.Endpoint,
		Data:     string(msg.Data),
	}, executor.status.State())

	switch msg.Endpoint {
	case "/appendEntries":
		if err = json.Unmarshal(msg.Data, &appendEntries); err != nil {
			msg.ReplyChan <- []byte("Bad Request")
			return nil
		}
		switch executor.status.State() {
		case iface.StateFollower:
			actions = executor.handler.FollowerOnAppendEntries(
				*appendEntries,
				executor.log,
				executor.status)

		case iface.StateCandidate:
			actions = executor.handler.CandidateOnAppendEntries(
				*appendEntries,
				executor.log,
				executor.status)

		case iface.StateLeader:
			actions = executor.handler.LeaderOnAppendEntries(
				*appendEntries,
				executor.log,
				executor.status)

		}

	case "/requestVote":
		if err = json.Unmarshal(msg.Data, &requestVote); err != nil {
			msg.ReplyChan <- []byte("Bad Request")
			return nil
		}
		switch executor.status.State() {
		case iface.StateFollower:
			actions = executor.handler.FollowerOnRequestVote(
				*requestVote,
				executor.log,
				executor.status)

		case iface.StateCandidate:
			actions = executor.handler.CandidateOnRequestVote(
				*requestVote,
				executor.log,
				executor.status)

		case iface.StateLeader:
			actions = executor.handler.LeaderOnRequestVote(
				*requestVote,
				executor.log,
				executor.status)

		}

	case "/addServer":
		if err = json.Unmarshal(msg.Data, &addServer); err != nil {
			msg.ReplyChan <- []byte("Bad Request")
			return nil
		}
		switch executor.status.State() {
		case iface.StateFollower:
			actions = executor.handler.FollowerOnAddServer(
				*addServer,
				executor.log,
				executor.status)

		case iface.StateCandidate:
			actions = executor.handler.CandidateOnAddServer(
				*addServer,
				executor.log,
				executor.status)

		case iface.StateLeader:
			actions = executor.handler.LeaderOnAddServer(
				*addServer,
				executor.log,
				executor.status)

		}

	case "/removeServer":
		if err = json.Unmarshal(msg.Data, &removeServer); err != nil {
			msg.ReplyChan <- []byte("Bad Request")
			return nil
		}
		switch executor.status.State() {
		case iface.StateFollower:
			actions = executor.handler.FollowerOnRemoveServer(
				*removeServer,
				executor.log,
				executor.status)

		case iface.StateCandidate:
			actions = executor.handler.CandidateOnRemoveServer(
				*removeServer,
				executor.log,
				executor.status)

		case iface.StateLeader:
			actions = executor.handler.LeaderOnRemoveServer(
				*removeServer,
				executor.log,
				executor.status)

		}

	case "/stateMachineCommand":
		if err = json.Unmarshal(msg.Data, &stateMachineCommand); err != nil {
			msg.ReplyChan <- []byte("Bad Request")
			return nil
		}
		switch executor.status.State() {
		case iface.StateFollower:
			actions = executor.handler.FollowerOnStateMachineCommand(
				*stateMachineCommand,
				executor.log,
				executor.status)

		case iface.StateCandidate:
			actions = executor.handler.CandidateOnStateMachineCommand(
				*stateMachineCommand,
				executor.log,
				executor.status)

		case iface.StateLeader:
			actions = executor.handler.LeaderOnStateMachineCommand(
				*stateMachineCommand,
				executor.log,
				executor.status)

		}

	case "/stateMachineProbe":
		if err = json.Unmarshal(msg.Data, &stateMachineProbe); err != nil {
			msg.ReplyChan <- []byte("Bad Request")
			return nil
		}
		switch executor.status.State() {
		case iface.StateFollower:
			actions = executor.handler.FollowerOnStateMachineProbe(
				*stateMachineProbe,
				executor.log,
				executor.status)

		case iface.StateCandidate:
			actions = executor.handler.CandidateOnStateMachineProbe(
				*stateMachineProbe,
				executor.log,
				executor.status)

		case iface.StateLeader:
			actions = executor.handler.LeaderOnStateMachineProbe(
				*stateMachineProbe,
				executor.log,
				executor.status)

		}

	default:
		panic("unknown endpoint: " + msg.Endpoint)
	}

	return actions
}

// forwardReply forwards either a MsgAppendEntriesReply or a
// MsgRequestVoteReply (received from a peer) to the rule handler.
//
// Returns the list of actions (as received from handler)
func (executor *Executor) forwardReply(reply interface{}) []interface{} {

	var (
		actions []interface{}
	)

	fmt.Printf("forwarding (%T)%+v to %s handler \n\n", reply, reply, executor.status.State())

	switch res := reply.(type) {
	case iface.MsgAppendEntriesReply:
		switch executor.status.State() {
		case iface.StateFollower:
			actions = executor.handler.FollowerOnAppendEntriesReply(
				res,
				executor.log,
				executor.status)

		case iface.StateCandidate:
			actions = executor.handler.CandidateOnAppendEntriesReply(
				res,
				executor.log,
				executor.status)

		case iface.StateLeader:
			actions = executor.handler.LeaderOnAppendEntriesReply(
				res,
				executor.log,
				executor.status)

		}

	case iface.MsgRequestVoteReply:
		switch executor.status.State() {
		case iface.StateFollower:
			actions = executor.handler.FollowerOnRequestVoteReply(
				res,
				executor.log,
				executor.status)

		case iface.StateCandidate:
			actions = executor.handler.CandidateOnRequestVoteReply(
				res,
				executor.log,
				executor.status)

		case iface.StateLeader:
			actions = executor.handler.LeaderOnRequestVoteReply(
				res,
				executor.log,
				executor.status)

		}

	default:
		panic("unknown reply type")
	}

	return actions
}

// forwardTick forwards to handler the fact that the timer expired.
//
// Returns the list of actions (as received from handler)
func (executor *Executor) forwardTick() []interface{} {

	var (
		actions []interface{}
	)

	fmt.Printf("forwarding timeout to %s handler \n\n", executor.status.State())

	switch executor.status.State() {
	case iface.StateFollower:
		actions = executor.handler.FollowerOnTimeout(
			iface.MsgTimeout{},
			executor.log,
			executor.status)

	case iface.StateCandidate:
		actions = executor.handler.CandidateOnTimeout(
			iface.MsgTimeout{},
			executor.log,
			executor.status)

	case iface.StateLeader:
		actions = executor.handler.LeaderOnTimeout(
			iface.MsgTimeout{},
			executor.log,
			executor.status)

	}

	return actions
}

// forwardStateChanged forwards a notice that state has changed
// to the rule handler
//
// Returns the list of actions (as received from handler)
func (executor *Executor) forwardStateChanged() []interface{} {

	var (
		actions []interface{}
	)

	fmt.Printf("forwarding state change (to %s) to handler \n\n", executor.status.State())

	msg := iface.MsgStateChanged{}
	switch executor.status.State() {
	case iface.StateFollower:
		actions = executor.handler.FollowerOnStateChanged(
			msg,
			executor.log,
			executor.status)

	case iface.StateCandidate:
		actions = executor.handler.CandidateOnStateChanged(
			msg,
			executor.log,
			executor.status)

	case iface.StateLeader:
		actions = executor.handler.LeaderOnStateChanged(
			msg,
			executor.log,
			executor.status)
	}

	return actions
}

// implementActions executes the actions specified by rule handler
//
// `replyChan` may be nil
//
// `originatingMsg` is the Msg* struct that logically
// demanded the call to implementActions()
func (executor *Executor) implementActions(
	actions []interface{},
	replyChan chan []byte,
	originatingMsg interface{}) {

	var (
		marshal    []byte
		subactions []interface{}
		err        error
	)

	fmt.Printf("implementing actions:\n")
	for _, untypedAction := range actions {
		fmt.Printf("\t(%T)%+v\n", untypedAction, untypedAction)
	}
	fmt.Println("")

	for _, untypedAction := range actions {
		switch action := untypedAction.(type) {

		case iface.ReplyNotLeader:
			if replyChan == nil {
				panic("handler tried to issue a Reply* to a local event")
			}
			replyChan <- []byte("not leader")

		case iface.ReplyCheckLater:
			if replyChan == nil {
				panic("handler tried to issue a Reply* to a local event")
			}
			replyChan <- []byte("check later. index: " +
				strconv.Itoa(int(action.Index)) + ", term: " +
				strconv.Itoa(int(action.Term)))

		case iface.ReplyFailed:
			if replyChan == nil {
				panic("handler tried to issue a Reply* to a local event")
			}
			replyChan <- []byte("failed")

		case iface.ReplyCompleted:
			if replyChan == nil {
				panic("handler tried to issue a Reply* to a local event")
			}
			replyChan <- action.Result

		case iface.ReplyDecidedVote:
			if replyChan == nil {
				panic("handler tried to issue a Reply* to a local event")
			}
			if marshal, err = json.Marshal(action); err != nil {
				panic("malformed ReplyDecidedVote action")
			}
			replyChan <- marshal

		case iface.ReplyAppendEntries:
			if replyChan == nil {
				panic("handler tried to issue a Reply* to a local event")
			}
			if marshal, err = json.Marshal(action); err != nil {
				panic("malformed ReplyAppendEntries action")
			}
			replyChan <- marshal

		case iface.ActionAppendLog:
			for _, entry := range action.Entries {
				if err = executor.log.Append(entry); err != nil {
					panic(err)
				}
			}

		case iface.ActionDeleteLog:
			for index := int64(0); index < action.Count; index++ {
				if err = executor.log.Remove(); err != nil {
					panic(err)
				}
			}

		case iface.ActionSetState:
			// must execute state-change subactions
			// before returning to the remaining actions at this level
			executor.status.SetState(action.NewState)
			msg := iface.MsgStateChanged{}
			subactions = executor.forwardStateChanged()
			executor.implementActions(subactions, nil, msg)

		case iface.ActionSetCurrentTerm:
			executor.status.SetCurrentTerm(action.NewCurrentTerm)

		case iface.ActionSetVotedFor:
			executor.status.SetVotedFor(action.NewVotedFor)

		case iface.ActionSetVoteCount:
			executor.status.SetVoteCount(action.NewVoteCount)

		case iface.ActionSetCommitIndex:
			executor.status.SetCommitIndex(action.NewCommitIndex)

		case iface.ActionSetLastApplied:
			executor.status.SetLastApplied(action.NewLastApplied)

		case iface.ActionAddServer:
			addresses := executor.status.PeerAddresses()
			addresses = append(addresses, action.NewServerAddress)
			executor.status.SetPeerAddresses(addresses)

		case iface.ActionRemoveServer:
			oldAddresses := executor.status.PeerAddresses()
			addresses := make([]iface.PeerAddress, len(oldAddresses)-1)
			index := 0
			for _, addr := range oldAddresses {
				if addr != action.OldServerAddress {
					addresses[index] = addr
					index++
				}
			}
			executor.status.SetPeerAddresses(addresses)

		case iface.ActionSetNextIndex:
			executor.status.SetNextIndex(action.Peer, action.NewNextIndex)

		case iface.ActionSetMatchIndex:
			executor.status.SetMatchIndex(action.Peer, action.NewMatchIndex)

		case iface.ActionSetClusterChange:
			executor.status.SetClusterChange(
				action.NewClusterChangeIndex,
				action.NewClusterChangeTerm)

		case iface.ActionResetTimer:
			if action.HalfTime {
				executor.timer.Reset(time.Duration(executor.minElectionTimeout/2) * time.Millisecond)
			} else {
				executor.timer.Reset(executor.randomElectionTimeout())
			}

		case iface.ActionStateMachineApply:
			entry, err := executor.log.Get(action.EntryIndex)
			if err != nil {
				panic(err)
			}
			result := executor.stateMachine.Apply(entry.Command)
			entry.Result = result
			if err = executor.log.Update(action.EntryIndex, *entry); err != nil {
				panic(err)
			}

		case iface.ActionAppendEntries:
			if marshal, err = json.Marshal(action); err != nil {
				panic(err)
			}

			// dispatch the AppendEntries call to the internet
			// on response, forward to the appropriate executor channel
			replyChan := executor.transport.Send(
				string(action.Destination)+"/appendEntries",
				marshal)

			go func() {
				reply := <-replyChan
				if reply == nil {
					fmt.Printf("executor: transport: receive reply failed\n")
					return
				}
				unmarshal := iface.MsgAppendEntriesReply{}
				if err := json.Unmarshal(reply, &unmarshal); err != nil {
					return
				}
				executor.appendEntriesReplyChan <- unmarshal
			}()

		case iface.ActionRequestVote:
			if marshal, err = json.Marshal(action); err != nil {
				panic(err)
			}

			// dispatch the RequestVote call to the internet
			// on response, forward to the appropriate executor channel
			replyChan := executor.transport.Send(
				string(action.Destination)+"/requestVote",
				marshal)

			go func() {
				reply := <-replyChan
				if reply == nil {
					fmt.Printf("executor: transport: error: received nil reply\n")
					return
				}
				unmarshal := iface.MsgRequestVoteReply{}
				if err := json.Unmarshal(reply, &unmarshal); err != nil {
					fmt.Printf("executor: transport: error: on unmarshal %+v\n", string(reply))
					return
				}
				executor.requestVoteReplyChan <- unmarshal
			}()

		case iface.ActionReprocess:
			switch msg := originatingMsg.(type) {
			// timer
			case nil:
				subactions = executor.forwardTick()

			case iface.MsgStateChanged:
				subactions = executor.forwardStateChanged()

			case iface.MsgAppendEntriesReply:
				subactions = executor.forwardReply(msg)

			case iface.MsgRequestVoteReply:
				subactions = executor.forwardReply(msg)

			case transport.IncomingMessage:
				subactions = executor.forwardIncoming(msg)

			default:
				panic("unknown action type")

			}

			executor.implementActions(subactions, replyChan, originatingMsg)

		default:
			panic("unknown action type: must be one of Action* or Reply* structs")

		}

	}
}

// Status gives access to the underlying Status instance.
// Altering this instance my leave the node in inconsistent state
func (executor *Executor) Status() *status.Status {
	return executor.status
}

// Log gives access to the underlying RaftLog instance.
// Altering this instance my leave the node in inconsistent state
func (executor *Executor) Log() *raftlog.RaftLog {
	return executor.log
}

// Transport gives access to the underlying Transport instance.
// Altering this instance my leave the node in inconsistent state
func (executor *Executor) Transport() *transport.Transport {
	return executor.transport
}
