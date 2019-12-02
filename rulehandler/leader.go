package rulehandler

import (
	"encoding/json"
	"math"
	"simpleraft/iface"
)

// LeaderOnStateChanged implements raft rules
func (rulehandler *RuleHandler) LeaderOnStateChanged(msg iface.MsgStateChanged, log iface.RaftLog, status iface.Status) []interface{} {
	actions := []interface{}{}

	// on becoming leader, insert a bogus log entry just to commit something
	// belonging to our own term
	firstEntry := iface.LogEntry{
		Term:    status.CurrentTerm(),
		Kind:    iface.EntryNoOp,
		Command: []byte{},
		Result:  []byte{},
	}

	actions = append(actions, iface.ActionAppendLog{
		Entries: []iface.LogEntry{firstEntry},
	})

	lastEntry, _ := log.Get(log.LastIndex())
	lastTerm := int64(-1)
	if lastEntry != nil {
		lastTerm = lastEntry.Term
	}

	for _, address := range status.PeerAddresses() {
		// for each server, index of the next log entry
		// to send to that server (initialized to leader
		// last log index + 1)
		actions = append(actions, iface.ActionSetNextIndex{
			Peer:         address,
			NewNextIndex: log.LastIndex() + 1,
		})

		// for each server, index of highest log entry
		// known to be replicated on server
		// (initialized to 0, increases monotonically)
		actions = append(actions, iface.ActionSetMatchIndex{
			Peer:          address,
			NewMatchIndex: 0,
		})

		// Upon election: send initial empty AppendEntries RPCs
		// (heartbeat) to each server;
		actions = append(actions, iface.ActionAppendEntries{
			Destination:       address,
			Entries:           []iface.LogEntry{},
			PrevLogIndex:      log.LastIndex(),
			PrevLogTerm:       lastTerm,
			LeaderAddress:     status.NodeAddress(),
			LeaderCommitIndex: status.CommitIndex(),
			Term:              status.CurrentTerm(),
		})

	}

	// Reset timer for next timeout (next timeout == moment to send heartbeats)
	actions = append(actions, iface.ActionResetTimer{
		HalfTime: true,
	})

	return actions
}

// LeaderOnAppendEntries implements raft rules
func (rulehandler *RuleHandler) LeaderOnAppendEntries(msg iface.MsgAppendEntries, log iface.RaftLog, status iface.Status) []interface{} {
	actions := []interface{}{}

	// If RPC request or response contains term T > currentTerm:
	// set currentTerm = T, convert to follower (§5.1)
	if msg.Term > status.CurrentTerm() {
		actions = append(actions, iface.ActionSetState{
			NewState: iface.StateFollower,
		})
		actions = append(actions, iface.ActionReprocess{})
	}

	return actions
}

// LeaderOnRequestVote implements raft rules
func (rulehandler *RuleHandler) LeaderOnRequestVote(msg iface.MsgRequestVote, log iface.RaftLog, status iface.Status) []interface{} {
	actions := []interface{}{}

	// If RPC request or response contains term T > currentTerm:
	// set currentTerm = T, convert to follower (§5.1)
	if msg.Term > status.CurrentTerm() {
		actions = append(actions, iface.ActionSetCurrentTerm{
			NewCurrentTerm: msg.Term,
		})
		actions = append(actions, iface.ActionSetState{
			NewState: iface.StateFollower,
		})
		actions = append(actions, iface.ActionReprocess{})

		return actions
	}

	actions = append(actions, iface.ReplyRequestVote{
		Address:     status.NodeAddress(),
		VoteGranted: false,
		Term:        status.CurrentTerm(),
	})

	return actions
}

// LeaderOnAddServer implements raft rules
func (rulehandler *RuleHandler) LeaderOnAddServer(msg iface.MsgAddServer, log iface.RaftLog, status iface.Status) []interface{} {
	actions := []interface{}{}

	if status.CommitIndex() < status.ClusterChangeIndex() {
		actions = append(actions, iface.ReplyFailed{
			Reason: "previous change in progress",
		})
		return actions
	}

	entry, _ := log.Get(status.CommitIndex())

	if entry == nil || entry.Term != status.CurrentTerm() {
		actions = append(actions, iface.ReplyFailed{
			Reason: "did not commit anything on current term yet",
		})
		return actions
	}

	// put new server in our peer list
	actions = append(actions, iface.ActionAddServer{
		NewServerAddress: msg.NewServerAddress,
	})

	// memorize last time the cluster changed
	// and last configuration
	lastClusterRecord, _ := json.Marshal(iface.ClusterChangeCommand{
		OldClusterChangeIndex: status.ClusterChangeIndex(),
		OldClusterChangeTerm:  status.ClusterChangeTerm(),
		OldCluster:            append(status.PeerAddresses(), status.NodeAddress()),
		NewCluster:            append(status.PeerAddresses(), status.NodeAddress(), msg.NewServerAddress),
	})

	// append an entry
	actions = append(actions, iface.ActionAppendLog{
		Entries: []iface.LogEntry{iface.LogEntry{
			Kind:    iface.EntryAddServer,
			Term:    status.CurrentTerm(),
			Command: lastClusterRecord,
			Result:  []byte{},
		}},
	})

	// record new cluster change index/term
	actions = append(actions, iface.ActionSetClusterChange{
		NewClusterChangeIndex: log.LastIndex() + 1,
		NewClusterChangeTerm:  status.CurrentTerm(),
	})

	// inform client
	actions = append(actions, iface.ReplyCheckLater{
		Index: log.LastIndex() + 1,
		Term:  status.CurrentTerm(),
	})

	return actions
}

// LeaderOnRemoveServer implements raft rules
func (rulehandler *RuleHandler) LeaderOnRemoveServer(msg iface.MsgRemoveServer, log iface.RaftLog, status iface.Status) []interface{} {
	actions := []interface{}{}

	if status.CommitIndex() < status.ClusterChangeIndex() {
		actions = append(actions, iface.ReplyFailed{
			Reason: "previous change in progress",
		})
		return actions
	}

	entry, _ := log.Get(status.CommitIndex())

	if entry == nil || entry.Term != status.CurrentTerm() {
		actions = append(actions, iface.ReplyFailed{
			Reason: "did not commit anything on current term yet",
		})
		return actions
	}

	// remove server from our peer list (if it us ourselves,
	// it won't be in the peer list, but this operation is safe
	// nonetheless - it will be a no-op in this case)
	actions = append(actions, iface.ActionRemoveServer{
		OldServerAddress: msg.OldServerAddress,
	})

	oldCluster := append(status.PeerAddresses(), status.NodeAddress())
	newCluster := []iface.PeerAddress{}
	for _, addr := range oldCluster {
		if addr != msg.OldServerAddress {
			newCluster = append(newCluster, addr)
		}
	}

	// memorize last time the cluster changed
	// and last configuration
	lastClusterRecord, _ := json.Marshal(iface.ClusterChangeCommand{
		OldClusterChangeIndex: status.ClusterChangeIndex(),
		OldClusterChangeTerm:  status.ClusterChangeTerm(),
		OldCluster:            oldCluster,
		NewCluster:            newCluster,
	})

	// append an entry
	actions = append(actions, iface.ActionAppendLog{
		Entries: []iface.LogEntry{iface.LogEntry{
			Kind:    iface.EntryRemoveServer,
			Term:    status.CurrentTerm(),
			Command: lastClusterRecord,
			Result:  []byte{},
		}},
	})

	// record new cluster change index/term
	actions = append(actions, iface.ActionSetClusterChange{
		NewClusterChangeIndex: log.LastIndex() + 1,
		NewClusterChangeTerm:  status.CurrentTerm(),
	})

	// send a last heartbeat to to-be-removed server
	// to minimize the chance it will be disruptive

	lastTerm := int64(-1)
	lastEntry, _ := log.Get(log.LastIndex())
	if lastEntry != nil {
		lastTerm = lastEntry.Term
	}

	// Heartbeat
	actions = append(actions, iface.ActionAppendEntries{
		Destination:       msg.OldServerAddress,
		Entries:           []iface.LogEntry{},
		PrevLogIndex:      log.LastIndex(),
		PrevLogTerm:       lastTerm,
		LeaderAddress:     status.NodeAddress(),
		LeaderCommitIndex: status.CommitIndex(),
		Term:              status.CurrentTerm(),
	})

	// inform client
	actions = append(actions, iface.ReplyCheckLater{
		Index: log.LastIndex() + 1,
		Term:  status.CurrentTerm(),
	})

	return actions
}

// LeaderOnTimeout implements raft rules
func (rulehandler *RuleHandler) LeaderOnTimeout(msg iface.MsgTimeout, log iface.RaftLog, status iface.Status) []interface{} {
	actions := []interface{}{}

	// Reset timer for next timeout
	actions = append(actions, iface.ActionResetTimer{
		HalfTime: true,
	})

	// Send append entries to everybody
	for _, address := range status.PeerAddresses() {
		entries := []iface.LogEntry{}

		// if we have something to send, then send it !
		if log.LastIndex() >= status.NextIndex(address) {
			// If last log index ≥ nextIndex for a follower: send
			// AppendEntries RPC with log entries starting at nextIndex
			lastLog, _ := log.Get(status.NextIndex(address) - 1)
			lastTerm := int64(-1)
			if lastLog != nil {
				lastTerm = lastLog.Term
			}
			for i := status.NextIndex(address); i <= log.LastIndex(); i++ {
				entry, _ := log.Get(i)
				entries = append(entries, *entry)
			}
			actions = append(actions, iface.ActionAppendEntries{
				Destination:       address,
				Entries:           entries,
				PrevLogIndex:      status.NextIndex(address) - 1,
				PrevLogTerm:       lastTerm,
				LeaderAddress:     status.NodeAddress(),
				LeaderCommitIndex: status.CommitIndex(),
				Term:              status.CurrentTerm(),
			})

			// if not, then just send an empty heartbeat
		} else {

			lastTerm := int64(-1)
			entry, _ := log.Get(log.LastIndex())
			if entry != nil {
				lastTerm = entry.Term
			}

			// Heartbeat
			actions = append(actions, iface.ActionAppendEntries{
				Destination:       address,
				Entries:           entries,
				PrevLogIndex:      log.LastIndex(),
				PrevLogTerm:       lastTerm,
				LeaderAddress:     status.NodeAddress(),
				LeaderCommitIndex: status.CommitIndex(),
				Term:              status.CurrentTerm(),
			})
		}

	}

	return actions
}

// LeaderOnStateMachineCommand implements raft rules
func (rulehandler *RuleHandler) LeaderOnStateMachineCommand(msg iface.MsgStateMachineCommand, log iface.RaftLog, status iface.Status) []interface{} {
	actions := []interface{}{}

	// we should put this command in the log to be committed
	// (we will apply it later when it gets committed)
	actions = append(actions, iface.ActionAppendLog{
		Entries: []iface.LogEntry{iface.LogEntry{
			Term:    status.CurrentTerm(),
			Kind:    iface.EntryStateMachineCommand,
			Command: msg.Command,
			Result:  []byte{},
		}},
	})

	// also respond to the client so he/she knows to check
	// later for command completion
	actions = append(actions, iface.ReplyCheckLater{
		Index: log.LastIndex() + 1,
		Term:  status.CurrentTerm(),
	})

	return actions
}

// LeaderOnStateMachineProbe implements raft rules
func (rulehandler *RuleHandler) LeaderOnStateMachineProbe(msg iface.MsgStateMachineProbe, log iface.RaftLog, status iface.Status) []interface{} {
	actions := []interface{}{}

	logClientCommand, _ := log.Get(msg.Index)

	// Client command not in log
	if logClientCommand == nil {
		actions = append(actions, iface.ReplyFailed{
			Reason: "not in log",
		})
		return actions
	}

	// No results yet
	if status.LastApplied() < msg.Index {
		actions = append(actions, iface.ReplyCheckLater{
			Index: msg.Index,
			Term:  msg.Term,
		})
		return actions
	}

	// Client command overwritten
	if msg.Term != logClientCommand.Term {
		actions = append(actions, iface.ReplyFailed{
			Reason: "overwritten",
		})
		return actions
	}

	// Result available
	actions = append(actions, iface.ReplyCompleted{
		Result: logClientCommand.Result,
	})

	return actions
}

// LeaderOnAppendEntriesReply implements raft rules
func (rulehandler *RuleHandler) LeaderOnAppendEntriesReply(msg iface.MsgAppendEntriesReply, log iface.RaftLog, status iface.Status) []interface{} {
	actions := []interface{}{}

	// maybe we are outdated... step down !
	if msg.Term > status.CurrentTerm() {
		actions = append(actions, iface.ActionSetCurrentTerm{
			NewCurrentTerm: msg.Term,
		})
		actions = append(actions, iface.ActionSetState{
			NewState: iface.StateFollower,
		})

		return actions
	}

	// detected an out-of-order response
	if status.NextIndex(msg.Address)-1 != msg.PrevLogIndex {
		return actions
	}

	// If AppendEntries fails because of log inconsistency:
	// decrement nextIndex and retry (§5.3)
	if !msg.Success {
		actions = append(actions, iface.ActionSetNextIndex{
			Peer:         msg.Address,
			NewNextIndex: status.NextIndex(msg.Address) - 1,
		})
		return actions
	}

	// If successful: update nextIndex and matchIndex for
	// follower (§5.3)
	newNextIndex := msg.PrevLogIndex + msg.Length + 1
	actions = append(actions, iface.ActionSetNextIndex{
		Peer:         msg.Address,
		NewNextIndex: newNextIndex,
	})
	newMatchIndex := newNextIndex - 1
	actions = append(actions, iface.ActionSetMatchIndex{
		Peer:          msg.Address,
		NewMatchIndex: newMatchIndex,
	})

	// If there exists an N such that N > commitIndex, a majority
	// of matchIndex[i] ≥ N, and log[N].term == currentTerm:
	// set commitIndex = N (§5.3, §5.4).
	majority := math.Ceil(float64(len(status.PeerAddresses())+1) / 2)
	newCommitIndex := status.CommitIndex()
	for N := newCommitIndex + 1; N <= log.LastIndex(); N++ {
		count := 1 // because there is me myself !
		for _, address := range status.PeerAddresses() {
			if address != msg.Address && status.MatchIndex(address) >= N {
				count++
			}
			if address == msg.Address && newMatchIndex >= N {
				count++
			}
		}
		logEntry, _ := log.Get(N)
		if logEntry != nil {
			if float64(count) >= majority && logEntry.Term == status.CurrentTerm() {
				newCommitIndex = N
			}
		}
	}
	if newCommitIndex > status.CommitIndex() {
		actions = append(actions, iface.ActionSetCommitIndex{
			NewCommitIndex: newCommitIndex,
		})
	}

	// If commitIndex > lastApplied: increment lastApplied, apply
	// log[lastApplied] to state machine (§5.3)
	// (only if it is a state machine command)
	for i := status.LastApplied() + 1; i <= newCommitIndex; i++ {
		actions = append(actions, iface.ActionSetLastApplied{
			NewLastApplied: i,
		})
		entry, _ := log.Get(i)
		if entry == nil {
			break
		}
		switch entry.Kind {
		case iface.EntryStateMachineCommand:
			actions = append(actions, iface.ActionStateMachineApply{
				EntryIndex: i,
			})
		}
	}

	return actions
}

// LeaderOnRequestVoteReply implements raft rules
func (rulehandler *RuleHandler) LeaderOnRequestVoteReply(msg iface.MsgRequestVoteReply, log iface.RaftLog, status iface.Status) []interface{} {
	// delayed request vote reply. ignore it
	return []interface{}{}
}
