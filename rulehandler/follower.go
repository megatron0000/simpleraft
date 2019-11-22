package rulehandler

import (
	"math"
	"simpleraft/iface"
)

// FollowerOnStateChanged implements raft rules
func (handler *RuleHandler) FollowerOnStateChanged(msg iface.MsgStateChanged, log iface.RaftLog, status iface.Status) []interface{} {
	actions := make([]interface{}, 0) // list of actions created

	actions = append(actions, iface.ActionSetVotedFor{NewVotedFor: ""})  // as a new follower, VotedFor is initially empty
	actions = append(actions, iface.ActionSetVoteCount{NewVoteCount: 0}) // as a follower, vote count must be set to 0
	actions = append(actions, iface.ActionResetTimer{HalfTime: false})   // timeout should be reseted

	return actions
}

// FollowerOnAppendEntries implements raft rules
func (handler *RuleHandler) FollowerOnAppendEntries(msg iface.MsgAppendEntries, log iface.RaftLog, status iface.Status) []interface{} {
	actions := make([]interface{}, 0) // list of actions created

	// If 'append entry' is from a leader with smaller term OR log matching property not achieved yet
	entry, err := log.Get(msg.PrevLogIndex)
	if err != nil {
		panic(err)
	}
	if (msg.Term < status.CurrentTerm()) || (msg.Term == status.CurrentTerm() && entry.Term != msg.PrevLogTerm) {
		actions = append(actions, iface.ReplyAppendEntries{Success: false, Term: status.CurrentTerm()}) // not successfull append entry
		return actions
	}

	// as program jumped the if statement above, we have a successfull append entry
	actions = append(actions, iface.ReplyAppendEntries{Success: true, Term: status.CurrentTerm()})
	actions = append(actions, iface.ActionDeleteLog{Count: (log.LastIndex() + 1 - msg.PrevLogIndex)}) // delete all entries in log after PrevLogIndex
	actions = append(actions, iface.ActionAppendLog{Entries: msg.Entries})                            // append all entries sent by leader

	if msg.LeaderCommitIndex > status.CommitIndex() {
		actions = append(actions, iface.ActionSetCommitIndex{
			NewCommitIndex: int64(math.Min(
				float64(msg.LeaderCommitIndex),
				float64(msg.PrevLogIndex+int64(len(msg.Entries))),
			)),
		})
	}

	return actions
}

// FollowerOnRequestVote implements raft rules
func (handler *RuleHandler) FollowerOnRequestVote(msg iface.MsgRequestVote, log iface.RaftLog, status iface.Status) []interface{} {
	actions := make([]interface{}, 0) // list of actions created

	// if candidate is in a smaller term, vote is not granted
	if msg.Term < status.CurrentTerm() {
		actions = append(actions, iface.ReplyDecidedVote{VoteGranted: false, Term: status.CurrentTerm()}) // not successfull vote
		return actions
	}

	// if candidate is at least as updated as follower, vote is granted
	if (status.VotedFor() == "" || status.VotedFor() == msg.CandidateAddress) &&
		((status.CurrentTerm() < msg.LastLogTerm) ||
			((status.CurrentTerm() == msg.LastLogTerm) &&
				(status.CommitIndex() <= msg.LastLogIndex))) {
		actions = append(actions, iface.ReplyDecidedVote{VoteGranted: true, Term: status.CurrentTerm()})
		actions = append(actions, iface.ActionSetVotedFor{NewVotedFor: msg.CandidateAddress}) // vote is granted
	} else {
		actions = append(actions, iface.ReplyDecidedVote{VoteGranted: false, Term: status.CurrentTerm()}) // not successfull vote
	}

	return actions
}

// FollowerOnAddServer implements raft rules
func (handler *RuleHandler) FollowerOnAddServer(msg iface.MsgAddServer, log iface.RaftLog, status iface.Status) []interface{} {
	actions := make([]interface{}, 0)                 // list of actions created
	actions = append(actions, iface.ReplyNotLeader{}) // leader should be responsable for this activity
	return actions
}

// FollowerOnRemoveServer implements raft rules
func (handler *RuleHandler) FollowerOnRemoveServer(msg iface.MsgRemoveServer, log iface.RaftLog, status iface.Status) []interface{} {
	actions := make([]interface{}, 0)                 // list of actions created
	actions = append(actions, iface.ReplyNotLeader{}) // leader should be responsable for this activity
	return actions
}

// FollowerOnTimeout implements raft rules
func (handler *RuleHandler) FollowerOnTimeout(msg iface.MsgTimeout, log iface.RaftLog, status iface.Status) []interface{} {
	actions := make([]interface{}, 0)                                               // list of actions created
	actions = append(actions, iface.ActionSetState{NewState: iface.StateCandidate}) // a timeout for a follower means it should change its state to candidate
	return actions
}

// FollowerOnStateMachineCommand implements raft rules
func (handler *RuleHandler) FollowerOnStateMachineCommand(msg iface.MsgStateMachineCommand, log iface.RaftLog, status iface.Status) []interface{} {
	actions := make([]interface{}, 0)                 // list of actions created
	actions = append(actions, iface.ReplyNotLeader{}) // leader should be responsable for this activity
	return actions
}
