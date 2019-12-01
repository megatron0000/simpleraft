package rulehandler

import (
	"simpleraft/iface"
)

// FollowerOnStateChanged implements raft rules
func (handler *RuleHandler) FollowerOnStateChanged(msg iface.MsgStateChanged, log iface.RaftLog, status iface.Status) []interface{} {
	actions := []interface{}{}

	// as a new follower, VotedFor is initially empty
	actions = append(actions, iface.ActionSetVotedFor{
		NewVotedFor: "",
	})

	// as a follower, vote count must be set to 0
	actions = append(actions, iface.ActionSetVoteCount{
		NewVoteCount: 0,
	})

	// timeout should be reset (waiting for election time)
	actions = append(actions, iface.ActionResetTimer{
		HalfTime: false,
	})

	return actions
}

// FollowerOnAppendEntries implements raft rules
func (handler *RuleHandler) FollowerOnAppendEntries(msg iface.MsgAppendEntries, log iface.RaftLog, status iface.Status) []interface{} {
	actions := []interface{}{} // list of actions to be returned

	// maybe we are outdated
	if msg.Term > status.CurrentTerm() {
		actions = append(actions, iface.ActionSetCurrentTerm{
			NewCurrentTerm: msg.Term,
		})
	}

	// leader is outdated ?
	if msg.Term < status.CurrentTerm() {
		actions = append(actions, iface.ReplyAppendEntries{
			Address: status.NodeAddress(),
			Success: false,
			Term:    status.CurrentTerm(),
		})
		return actions
	}

	// since we are hearing from the leader, reset timeout
	actions = append(actions, iface.ActionResetTimer{
		HalfTime: false,
	})

	// all is ok. accept
	actions = append(actions, iface.ReplyAppendEntries{
		Address: status.NodeAddress(),
		Success: true,
		Term:    status.CurrentTerm(),
	})

	return actions
}

// FollowerOnRequestVote implements raft rules
func (handler *RuleHandler) FollowerOnRequestVote(msg iface.MsgRequestVote, log iface.RaftLog, status iface.Status) []interface{} {
	actions := []interface{}{} // list of actions to be returned

	// maybe we are outdated
	if msg.Term > status.CurrentTerm() {
		actions = append(actions, iface.ActionSetCurrentTerm{
			NewCurrentTerm: msg.Term,
		})

		actions = append(actions, iface.ActionSetVotedFor{
			NewVotedFor: "",
		})

		actions = append(actions, iface.ActionReprocess{})

		return actions
	}

	// if candidate is still in a previous term, reject vote
	if msg.Term < status.CurrentTerm() {
		actions = append(actions, iface.ReplyRequestVote{
			VoteGranted: false,
			Term:        status.CurrentTerm(),
		})
		return actions
	}

	// reject vote if we voted on another peer already
	if status.VotedFor() != "" && status.VotedFor() != msg.CandidateAddress {
		actions = append(actions, iface.ReplyRequestVote{
			VoteGranted: false,
			Term:        status.CurrentTerm(),
			Address:     status.NodeAddress(),
		})
		return actions
	}

	actions = append(actions, iface.ReplyRequestVote{
		VoteGranted: true,
		Term:        status.CurrentTerm(),
		Address:     status.NodeAddress(),
	})

	return actions
}

// FollowerOnAddServer implements raft rules
func (handler *RuleHandler) FollowerOnAddServer(msg iface.MsgAddServer, log iface.RaftLog, status iface.Status) []interface{} {
	// leader should be responsible for this
	return []interface{}{iface.ReplyNotLeader{}}
}

// FollowerOnRemoveServer implements raft rules
func (handler *RuleHandler) FollowerOnRemoveServer(msg iface.MsgRemoveServer, log iface.RaftLog, status iface.Status) []interface{} {
	// leader should be responsible for this
	return []interface{}{iface.ReplyNotLeader{}}
}

// FollowerOnTimeout implements raft rules
func (handler *RuleHandler) FollowerOnTimeout(msg iface.MsgTimeout, log iface.RaftLog, status iface.Status) []interface{} {
	// timed out without hearing from leader... election tiiiime !
	return []interface{}{iface.ActionSetState{
		NewState: iface.StateCandidate,
	}}
}

// FollowerOnStateMachineCommand implements raft rules
func (handler *RuleHandler) FollowerOnStateMachineCommand(msg iface.MsgStateMachineCommand, log iface.RaftLog, status iface.Status) []interface{} {
	// leader should be responsible for this
	return []interface{}{iface.ReplyNotLeader{}}
}

// FollowerOnStateMachineProbe implements raft rules
func (handler *RuleHandler) FollowerOnStateMachineProbe(msg iface.MsgStateMachineProbe, log iface.RaftLog, status iface.Status) []interface{} {
	// leader should be responsible for this
	return []interface{}{iface.ReplyNotLeader{}}
}

// FollowerOnAppendEntriesReply implements raft rules
func (handler *RuleHandler) FollowerOnAppendEntriesReply(msg iface.MsgAppendEntriesReply, log iface.RaftLog, status iface.Status) []interface{} {
	// delayed append entries reply. ignore it
	return []interface{}{}
}

// FollowerOnRequestVoteReply implements raft rules
func (handler *RuleHandler) FollowerOnRequestVoteReply(msg iface.MsgRequestVoteReply, log iface.RaftLog, status iface.Status) []interface{} {
	// delayed request vote reply. ignore it
	return []interface{}{}
}
