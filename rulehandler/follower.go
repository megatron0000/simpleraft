package rulehandler

import (
	"math"
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
			Address:      status.NodeAddress(),
			Success:      false,
			Term:         status.CurrentTerm(),
			Length:       int64(len(msg.Entries)),
			PrevLogIndex: msg.PrevLogIndex,
			PrevLogTerm:  msg.PrevLogTerm,
		})
		return actions
	}

	// since we are hearing from the leader, reset timeout
	actions = append(actions, iface.ActionResetTimer{
		HalfTime: false,
	})

	prevEntry, _ := log.Get(msg.PrevLogIndex)

	// I dont have previous log entry (but should)
	if prevEntry == nil && msg.PrevLogIndex != -1 {
		actions = append(actions, iface.ReplyAppendEntries{
			Address:      status.NodeAddress(),
			Success:      false,
			Term:         status.CurrentTerm(),
			Length:       int64(len(msg.Entries)),
			PrevLogIndex: msg.PrevLogIndex,
			PrevLogTerm:  msg.PrevLogTerm,
		})
		return actions
	}

	// I have previous log entry, but it does not match
	if prevEntry != nil && prevEntry.Term != msg.PrevLogTerm {
		actions = append(actions, iface.ReplyAppendEntries{
			Address:      status.NodeAddress(),
			Success:      false,
			Term:         status.CurrentTerm(),
			Length:       int64(len(msg.Entries)),
			PrevLogIndex: msg.PrevLogIndex,
			PrevLogTerm:  msg.PrevLogTerm,
		})
		return actions
	}

	// all is ok. accept
	actions = append(actions, iface.ReplyAppendEntries{
		Address:      status.NodeAddress(),
		Success:      true,
		Term:         status.CurrentTerm(),
		Length:       int64(len(msg.Entries)),
		PrevLogIndex: msg.PrevLogIndex,
		PrevLogTerm:  msg.PrevLogTerm,
	})

	// if there is anything to append, do it
	if len(msg.Entries) > 0 {
		// delete all entries in log after PrevLogIndex
		actions = append(actions, iface.ActionDeleteLog{
			Count: log.LastIndex() - msg.PrevLogIndex,
		})

		// append all entries sent by leader
		actions = append(actions, iface.ActionAppendLog{
			Entries: msg.Entries,
		})

	}

	// if leader has committed more than we know, update our index
	// and demand state-machine application
	if msg.LeaderCommitIndex > status.CommitIndex() {
		actions = append(actions, iface.ActionSetCommitIndex{
			NewCommitIndex: int64(math.Min(
				float64(msg.LeaderCommitIndex),
				float64(msg.PrevLogIndex+int64(len(msg.Entries))),
			)),
		})
		// order the state machine to apply the new committed entries
		// (only if they are state machine commands)
		for index := status.CommitIndex() + 1; index < msg.LeaderCommitIndex; index++ {
			var entry *iface.LogEntry

			// get from my log
			if index <= msg.PrevLogIndex {
				entry, _ = log.Get(index)

				// get from leader
			} else {
				entry = &msg.Entries[index-msg.PrevLogIndex-1]
			}

			switch entry.Kind {
			case iface.EntryStateMachineCommand:
				actions = append(actions, iface.ActionStateMachineApply{
					EntryIndex: index,
				})
			}
		}
	}

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

	lastEntry, _ := log.Get(log.LastIndex())

	// if we have log but peer's is not as updated as ours, reject
	if lastEntry != nil && (msg.LastLogTerm < lastEntry.Term || (msg.LastLogTerm == lastEntry.Term && msg.LastLogIndex < log.LastIndex())) {
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

	actions = append(actions, iface.ActionResetTimer{
		HalfTime: false,
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
