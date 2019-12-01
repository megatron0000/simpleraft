package rulehandler

import (
	"simpleraft/iface"
)

// CandidateOnStateChanged implements raft rules
func (handler *RuleHandler) CandidateOnStateChanged(msg iface.MsgStateChanged, log iface.RaftLog, status iface.Status) []interface{} {
	actions := []interface{}{}

	// this is a new term
	actions = append(actions, iface.ActionSetCurrentTerm{
		NewCurrentTerm: status.CurrentTerm() + 1,
	})
	// vote for myself
	actions = append(actions, iface.ActionSetVotedFor{
		NewVotedFor: status.NodeAddress(),
	})
	actions = append(actions, iface.ActionSetVoteCount{
		NewVoteCount: 1,
	})
	// count down. if election is not over by then, we will try another election
	actions = append(actions, iface.ActionResetTimer{
		HalfTime: false,
	})

	lastIndex := int64(-1)
	lastTerm := int64(-1)
	entry, _ := log.Get(log.LastIndex())
	if entry != nil {
		lastIndex = log.LastIndex()
		lastTerm = entry.Term
	}

	// request everyone's vote
	for _, addr := range status.PeerAddresses() {
		actions = append(actions, iface.ActionRequestVote{
			Term:             status.CurrentTerm() + 1,
			CandidateAddress: status.NodeAddress(),
			LastLogIndex:     lastIndex,
			LastLogTerm:      lastTerm,
			Destination:      addr,
		})
	}

	return actions
}

// CandidateOnAppendEntries implements raft rules
func (handler *RuleHandler) CandidateOnAppendEntries(msg iface.MsgAppendEntries, log iface.RaftLog, status iface.Status) []interface{} {
	actions := []interface{}{} // list of actions to be returned

	//////////////////////////////////////////////////////////
	/////////////// MODIFY HERE //////////////////////////////

	actions = append(actions, iface.ReplyAppendEntries{
		Address: status.NodeAddress(),
		Success: true,
		Term:    status.CurrentTerm(),
	})

	return actions

	/////////////// END OF MODIFY ////////////////////////////
	//////////////////////////////////////////////////////////
}

// CandidateOnRequestVote implements raft rules
func (handler *RuleHandler) CandidateOnRequestVote(msg iface.MsgRequestVote, log iface.RaftLog, status iface.Status) []interface{} {
	actions := []interface{}{} // list of actions to be returned

	//////////////////////////////////////////////////////////
	/////////////// MODIFY HERE //////////////////////////////

	actions = append(actions, iface.ReplyRequestVote{
		VoteGranted: false,
		Term:        status.CurrentTerm(),
		Address:     status.NodeAddress(),
	})

	return actions

	/////////////// END OF MODIFY ////////////////////////////
	//////////////////////////////////////////////////////////

}

// CandidateOnAddServer implements raft rules
func (handler *RuleHandler) CandidateOnAddServer(msg iface.MsgAddServer, log iface.RaftLog, status iface.Status) []interface{} {
	// leader should be responsible for this
	return []interface{}{iface.ReplyNotLeader{}}
}

// CandidateOnRemoveServer implements raft rules
func (handler *RuleHandler) CandidateOnRemoveServer(msg iface.MsgRemoveServer, log iface.RaftLog, status iface.Status) []interface{} {
	// leader should be responsible for this
	return []interface{}{iface.ReplyNotLeader{}}
}

// CandidateOnTimeout implements raft rules
func (handler *RuleHandler) CandidateOnTimeout(msg iface.MsgTimeout, log iface.RaftLog, status iface.Status) []interface{} {
	// timed out =( Try another election !
	return []interface{}{iface.ActionSetState{
		NewState: iface.StateCandidate,
	}}
}

// CandidateOnStateMachineCommand implements raft rules
func (handler *RuleHandler) CandidateOnStateMachineCommand(msg iface.MsgStateMachineCommand, log iface.RaftLog, status iface.Status) []interface{} {
	// leader should be responsible for this
	return []interface{}{iface.ReplyNotLeader{}}
}

// CandidateOnStateMachineProbe implements raft rules
func (handler *RuleHandler) CandidateOnStateMachineProbe(msg iface.MsgStateMachineProbe, log iface.RaftLog, status iface.Status) []interface{} {
	// leader should be responsible for this
	return []interface{}{iface.ReplyNotLeader{}}
}

// CandidateOnAppendEntriesReply implements raft rules
func (handler *RuleHandler) CandidateOnAppendEntriesReply(msg iface.MsgAppendEntriesReply, log iface.RaftLog, status iface.Status) []interface{} {
	// delayed append entries reply. ignore it
	return []interface{}{}
}

// CandidateOnRequestVoteReply implements raft rules
func (handler *RuleHandler) CandidateOnRequestVoteReply(msg iface.MsgRequestVoteReply, log iface.RaftLog, status iface.Status) []interface{} {
	actions := []interface{}{} // list of actions to be returned

	//////////////////////////////////////////////////////////
	/////////////// MODIFY HERE //////////////////////////////

	// maybe we are outdated. If so, then too bad for us: step down
	if msg.Term > status.CurrentTerm() {
		actions = append(actions, iface.ActionSetCurrentTerm{
			NewCurrentTerm: msg.Term,
		})
		actions = append(actions, iface.ActionSetState{
			NewState: iface.StateFollower,
		})
		return actions
	}

	newVoteCount := status.VoteCount() + 1

	actions = append(actions, iface.ActionSetVoteCount{
		NewVoteCount: newVoteCount,
	})

	return actions

	/////////////// END OF MODIFY ////////////////////////////
	//////////////////////////////////////////////////////////
}
