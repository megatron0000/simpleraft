package rulehandler

import (
	"simpleraft/iface"
)

func (rulehandler *RuleHandler) LeaderOnStateChanged(msg iface.MsgStateChanged, log iface.RaftLog, status iface.Status) []interface{} {
	//Slice to be returned
	actions := make([]interface{}, 0)
	firstEntry := iface.LogEntry{
		Term:    status.CurrentTerm(),
		Kind:    iface.EntryNoOp,
		Command: make([]byte, 0),
		Result:  make([]byte, 0)}
	actions = append(actions, firstEntry)
	for _, address := range status.PeerAddresses() {
		//for each server, index of the next log entry
		//to send to that server (initialized to leader
		//last log index + 1)
		actions = append(actions, iface.ActionSetNextIndex{
			Peer:         address,
			NewNextIndex: log.LastIndex() + 1})
		//for each server, index of highest log entry
		//known to be replicated on server
		//(initialized to 0, increases monotonically)
		actions = append(actions, iface.ActionSetMatchIndex{
			Peer:          address,
			NewMatchIndex: 0})
		//Upon election: send initial empty AppendEntries RPCs
		//(heartbeat) to each server;
		actions = append(actions, iface.ActionAppendEntries{
			Destination:  address,
			Entries:      make([]iface.LogEntry, 0),
			PrevLogIndex: log.LastIndex(),
			PrevLogTerm:  firstEntry.Term})
	}
	return actions
}
func (rulehandler *RuleHandler) LeaderOnAppendEntries(msg iface.MsgAppendEntries, log iface.RaftLog, status iface.Status) []interface{} {
	actions := make([]interface{}, 0)
	//If RPC request or response contains term T > currentTerm:
	//set currentTerm = T, convert to follower (§5.1)
	if msg.Term > status.CurrentTerm() {
		actions = append(actions, iface.ActionSetState{
			NewState: iface.StateFollower})
		actions = append(actions, iface.ActionReprocess{})
	}
	/*
		//Reply false if term < currentTerm (§5.1)
		if msg.Term < status.CurrentTerm() {
			actions = append(actions, iface.ReplyAppendEntries{
				Term:    status.CurrentTerm(),
				Success: false})
			return actions
		}
		//Reply false if log doesn’t contain an entry at prevLogIndex
		//whose term matches prevLogTerm (§5.3)
		prevLog, _ := log.Get(msg.PrevLogIndex)
		if prevLog == nil || msg.PrevLogTerm != prevLog.Term {
			actions = append(actions, iface.ReplyAppendEntries{
				Term:    status.CurrentTerm(),
				Success: false})
			return actions
		}
		//If an existing entry conflicts with a new one (same index
		//but different terms), delete the existing entry and all that
		//follow it (§5.3)
		nextLogIndex := msg.PrevLogIndex
		var nextLog *iface.LogEntry
		for _, entry := range msg.Entries {
			nextLogIndex++
			nextLog, _ = log.Get(nextLogIndex)
			if nextLog == nil || entry.Term != nextLog.Term {
				actions = append(actions, iface.ActionDeleteLog{
					Count: log.LastIndex() - nextLogIndex + 1})
				break
			}
		}
		//Append any new entries not already in the log
		nextLogIndex := msg.PrevLogIndex
		for indx, entry := range msg.Entries {
			nextLogIndex++
			if log.LastIndex() < nextLogIndex {
				actions = append(actions, iface.ActionAppendLog{
					Entries: msg.Entries[:]})
				break
			}
			nextLogIndex++
			nextLog, _ = log.Get(nextLogIndex)
		}*/

	return actions
}
func (rulehandler *RuleHandler) LeaderOnRequestVote(msg iface.MsgRequestVote, log iface.RaftLog, status iface.Status) []interface{} {
	actions := make([]interface{}, 0)
	//If RPC request or response contains term T > currentTerm:
	//set currentTerm = T, convert to follower (§5.1)
	if msg.Term > status.CurrentTerm() {
		actions = append(actions, iface.ActionSetState{
			NewState: iface.StateFollower})
		actions = append(actions, iface.ActionReprocess{})
	} else {
		actions = append(actions, iface.ReplyDecidedVote{
			Address:     msg.CandidateAddress,
			VoteGranted: false,
			Term:        status.CurrentTerm()})
	}
	return actions
}
func (rulehandler *RuleHandler) LeaderOnAddServer(msg iface.MsgAddServer, log iface.RaftLog, status iface.Status) []interface{} {
	actions := make([]interface{}, 0)
	return actions
}
func (rulehandler *RuleHandler) LeaderOnRemoveServer(msg iface.MsgRemoveServer, log iface.RaftLog, status iface.Status) []interface{} {
	actions := make([]interface{}, 0)
	return actions
}
func (rulehandler *RuleHandler) LeaderOnTimeout(msg iface.MsgTimeout, log iface.RaftLog, status iface.Status) []interface{} {
	actions := make([]interface{}, 0)
	for _, address := range status.PeerAddresses() {
		entries := make([]iface.LogEntry, 0)
		if log.LastIndex() >= status.NextIndex(address) {
			//If last log index ≥ nextIndex for a follower: send
			//AppendEntries RPC with log entries starting at nextIndex
			lastLog, _ := log.Get(status.NextIndex(address) - 1)
			for i := status.NextIndex(address); i <= log.LastIndex(); i++ {
				entry, _ := log.Get(i)
				entries = append(entries, *entry)
			}
			actions = append(actions, iface.ActionAppendEntries{
				Destination:  address,
				Entries:      entries,
				PrevLogIndex: status.NextIndex(address) - 1,
				PrevLogTerm:  lastLog.Term})
		} else {
			//Heartbeat
			actions = append(actions, iface.ActionAppendEntries{
				Destination:  address,
				Entries:      entries,
				PrevLogIndex: log.LastIndex(),
				PrevLogTerm:  status.CurrentTerm()})
		}
	}
	return actions
}
func (rulehandler *RuleHandler) LeaderOnStateMachineCommand(msg iface.MsgStateMachineCommand, log iface.RaftLog, status iface.Status) []interface{} {
	actions := make([]interface{}, 0)
	entries := make([]iface.LogEntry, 0)
	entries = append(entries, iface.LogEntry{
		Term:    status.CurrentTerm(),
		Kind:    iface.EntryStateMachineCommand,
		Command: msg.Command,
		Result:  make([]byte, 0)})
	actions = append(actions, iface.ActionAppendLog{
		Entries: entries})
	return actions
}
func (rulehandler *RuleHandler) LeaderOnStateMachineProbe(msg iface.MsgStateMachineProbe, log iface.RaftLog, status iface.Status) []interface{} {
	actions := make([]interface{}, 0)
	logClientCommand, _ := log.Get(msg.Index)
	if msg.Term != logClientCommand.Term {
		//Client command overwritten
		actions = append(actions, iface.ReplyFailed{})
	} else if len(logClientCommand.Result) == 0 {
		//No results yet
		actions = append(actions, iface.ReplyCheckLater{})
	} else {
		actions = append(actions, iface.ReplyCompleted{
			Result: logClientCommand.Result})
	}
	return actions
}
func (rulehandler *RuleHandler) LeaderOnAppendEntriesReply(msg iface.MsgAppendEntriesReply, log iface.RaftLog, status iface.Status) []interface{} {
	actions := make([]interface{}, 0)
	if msg.Success && msg.Term == status.CurrentTerm() {
		//If successful: update nextIndex and matchIndex for
		//follower (§5.3)
		actions = append(actions, iface.ActionSetNextIndex{
			Peer:         msg.Address,
			NewNextIndex: log.LastIndex() + 1})
		actions = append(actions, iface.ActionSetMatchIndex{
			Peer:          msg.Address,
			NewMatchIndex: log.LastIndex()})
	} else {
		//If AppendEntries fails because of log inconsistency:
		//decrement nextIndex and retry (§5.3)
		actions = append(actions, iface.ActionSetNextIndex{
			Peer:         msg.Address,
			NewNextIndex: status.NextIndex(msg.Address) - 1})
	}
	return actions
}
func (rulehandler *RuleHandler) LeaderOnRequestVoteReply(msg iface.MsgRequestVoteReply, log iface.RaftLog, status iface.Status) []interface{} {
	actions := make([]interface{}, 0)
	return actions
}
