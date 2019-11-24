package rulehandler

import (
	"simpleraft/iface"
)

func (handler *RuleHandler) CandidateOnStateChanged(msg iface.MsgStateChanged, log iface.RaftLog, status iface.Status) []interface{} {
	actions := make([]interface{}, 0) // list of actions created

	actions = append(actions, iface.ActionSetCurrentTerm{NewCurrentTerm: status.CurrentTerm() + 1}) // updating term
	actions = append(actions, iface.ActionSetVotedFor{NewVotedFor: status.NodeAddress()})           // candidate should vote for itself
	actions = append(actions, iface.ActionSetVoteCount{NewVoteCount: 1})
	actions = append(actions, iface.ActionResetTimer{HalfTime: false}) // timeout should be reseted

	// requesting vote for each of its peers
	entry, err := log.Get(log.LastIndex())
	if err != nil {
		panic(err)
	}
	lastLogTerm := int64(-1) // meaning "does not have"
	if entry != nil {
		lastLogTerm = entry.Term
	}
	for _, addr := range status.PeerAddresses() {
		actions = append(actions, iface.ActionRequestVote{Term: status.CurrentTerm(), CandidateAddress: status.NodeAddress(), LastLogIndex: log.LastIndex(), LastLogTerm: lastLogTerm, Destination: addr})
	}

	return actions
}

func (handler *RuleHandler) CandidateOnAppendEntries(msg iface.MsgAppendEntries, log iface.RaftLog, status iface.Status) []interface{} {
	actions := make([]interface{}, 0) // list of actions created

	if msg.Term < status.CurrentTerm() {
		actions = append(actions, iface.ActionSetCurrentTerm{NewCurrentTerm: msg.Term})
		actions = append(actions, iface.ActionSetState{NewState: iface.StateFollower}) // step down to follower
		actions = append(actions, iface.ActionReprocess{})                             // append entry received should be reprocessed with server as follower
	}

	return actions
}

func (handler *RuleHandler) CandidateOnRequestVote(msg iface.MsgRequestVote, log iface.RaftLog, status iface.Status) []interface{} {
	actions := make([]interface{}, 0) // list of actions created

	actions = append(actions, iface.ActionResetTimer{HalfTime: false}) // timeout should be reseted
	// if candidate is in a smaller term, vote is not granted
	if msg.Term < status.CurrentTerm() {
		actions = append(actions, iface.ReplyDecidedVote{VoteGranted: false, Term: status.CurrentTerm()}) // not successfull vote
		return actions
	}

	// if candidate is at least as updated as other candidated, vote is granted
	if (status.VotedFor() == "" || status.VotedFor() == msg.CandidateAddress) &&
		((status.CurrentTerm() < msg.LastLogTerm) ||
			((status.CurrentTerm() == msg.LastLogTerm) &&
				(status.CommitIndex() <= msg.LastLogIndex))) {
		actions = append(actions, iface.ReplyDecidedVote{VoteGranted: true, Term: status.CurrentTerm()})
		actions = append(actions, iface.ActionSetState{NewState: iface.StateFollower})        // step down to follower
		actions = append(actions, iface.ActionSetVotedFor{NewVotedFor: msg.CandidateAddress}) // vote is granted
	} else {
		actions = append(actions, iface.ReplyDecidedVote{VoteGranted: false, Term: status.CurrentTerm()}) // not successfull vote
	}

	return actions
}

func (handler *RuleHandler) CandidateOnAddServer(msg iface.MsgAddServer, log iface.RaftLog, status iface.Status) []interface{} {
	actions := make([]interface{}, 0)                 // list of actions created
	actions = append(actions, iface.ReplyNotLeader{}) // leader should be responsable for this activity
	return actions
}

func (handler *RuleHandler) CandidateOnRemoveServer(msg iface.MsgRemoveServer, log iface.RaftLog, status iface.Status) []interface{} {
	actions := make([]interface{}, 0)                 // list of actions created
	actions = append(actions, iface.ReplyNotLeader{}) // leader should be responsable for this activity
	return actions
}

func (handler *RuleHandler) CandidateOnTimeout(msg iface.MsgTimeout, log iface.RaftLog, status iface.Status) []interface{} {
	actions := make([]interface{}, 0)                                              // list of actions created
	actions = append(actions, iface.ActionSetState{NewState: iface.StateFollower}) // step down to follower
	return actions
}

func (handler *RuleHandler) CandidateOnStateMachineCommand(msg iface.MsgStateMachineCommand, log iface.RaftLog, status iface.Status) []interface{} {
	actions := make([]interface{}, 0)                 // list of actions created
	actions = append(actions, iface.ReplyNotLeader{}) // leader should be responsable for this activity
	return actions
}

func (handler *RuleHandler) CandidateOnStateMachineProbe(msg iface.MsgStateMachineProbe, log iface.RaftLog, status iface.Status) []interface{} {
	actions := make([]interface{}, 0)                 // list of actions created
	actions = append(actions, iface.ReplyNotLeader{}) // leader should be responsable for this activity
	return actions
}

func (handler *RuleHandler) CandidateOnAppendEntriesReply(msg iface.MsgAppendEntriesReply, log iface.RaftLog, status iface.Status) []interface{} {
	actions := make([]interface{}, 0)                 // list of actions created
	actions = append(actions, iface.ReplyNotLeader{}) // leader should be responsable for this activity
	return actions
}

func (handler *RuleHandler) CandidateOnRequestVoteReply(msg iface.MsgRequestVoteReply, log iface.RaftLog, status iface.Status) []interface{} {
	actions := make([]interface{}, 0) // list of actions created

	if msg.VoteGranted {
		actions = append(actions, iface.ActionSetVoteCount{NewVoteCount: status.VoteCount() + 1})
	}

	if 2*(status.VoteCount()+1) > int64(len(status.PeerAddresses())) {
		actions = append(actions, iface.ActionSetState{NewState: iface.StateLeader})
	}

	return actions
}
