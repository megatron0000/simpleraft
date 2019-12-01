package rulehandler

import (
	"encoding/json"
	"simpleraft/iface"
)

// LeaderOnStateChanged implements raft rules
func (rulehandler *RuleHandler) LeaderOnStateChanged(msg iface.MsgStateChanged, log iface.RaftLog, status iface.Status) []interface{} {
	actions := []interface{}{}

	//////////////////////////////////////////////////////////
	/////////////// MODIFY HERE (SOMENTE TAREFA 3) ///////////

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
			PrevLogTerm:       -1,
			LeaderAddress:     status.NodeAddress(),
			LeaderCommitIndex: status.CommitIndex(),
			Term:              status.CurrentTerm(),
		})

		/////////////// END MOFIFY ///////////////////////////////
		//////////////////////////////////////////////////////////

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
			Kind:    iface.EntryAddServer,
			Term:    status.CurrentTerm(),
			Command: lastClusterRecord,
			Result:  []byte{},
		}},
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

	//////////////////////////////////////////////////////////
	/////////////// MODIFY HERE (SOMENTE TAREFA 3) ///////////

	// Reset timer for next timeout
	actions = append(actions, iface.ActionResetTimer{
		HalfTime: true,
	})

	// Send append entries to everybody
	for _, address := range status.PeerAddresses() {
		entries := []iface.LogEntry{}

		// Heartbeat
		actions = append(actions, iface.ActionAppendEntries{
			Destination:       address,
			Entries:           entries,
			PrevLogIndex:      log.LastIndex(),
			PrevLogTerm:       status.CurrentTerm(),
			LeaderAddress:     status.NodeAddress(),
			LeaderCommitIndex: status.CommitIndex(),
			Term:              status.CurrentTerm(),
		})

	}

	return actions

	/////////////// END OF MODIFY ////////////////////////////
	//////////////////////////////////////////////////////////
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

	return actions
}

// LeaderOnRequestVoteReply implements raft rules
func (rulehandler *RuleHandler) LeaderOnRequestVoteReply(msg iface.MsgRequestVoteReply, log iface.RaftLog, status iface.Status) []interface{} {
	// delayed request vote reply. ignore it
	return []interface{}{}
}
