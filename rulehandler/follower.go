package rulehandler

import (
	"encoding/json"
	"math"
	"simpleraft/iface"
	"time"
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
	actions = append(actions, iface.ActionSetLeaderLastHeard{
		Instant: time.Now(),
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

		// take care ! Maybe we are removing an entry
		// containing our current cluster configuration.
		// In this case, revert to previous cluster
		// configuration
		containsClusterChange := false
		stabilized := false
		clusterChangeIndex := status.ClusterChangeIndex()
		clusterChangeTerm := status.ClusterChangeTerm()
		cluster := append(status.PeerAddresses(), status.NodeAddress())
		for !stabilized {
			stabilized = true
			if clusterChangeIndex > msg.PrevLogIndex {
				stabilized = false
				containsClusterChange = true
				entry, _ := log.Get(clusterChangeIndex)
				record := &iface.ClusterChangeCommand{}
				json.Unmarshal(entry.Command, &record)
				clusterChangeIndex = record.OldClusterChangeIndex
				clusterChangeTerm = record.OldClusterChangeTerm
				cluster = record.OldCluster
			}
		}

		// if deletion detected, rewind to previous configuration
		if containsClusterChange {
			actions = append(actions, iface.ActionSetClusterChange{
				NewClusterChangeIndex: clusterChangeIndex,
				NewClusterChangeTerm:  clusterChangeTerm,
			})
			peers := []iface.PeerAddress{}
			for _, addr := range cluster {
				if addr != status.NodeAddress() {
					peers = append(peers, addr)
				}
			}
			actions = append(actions, iface.ActionSetPeers{
				PeerAddresses: peers,
			})
		}

		// append all entries sent by leader
		actions = append(actions, iface.ActionAppendLog{
			Entries: msg.Entries,
		})

		// once again, take care ! Maybe we are adding some entry
		// describing a cluster change. In such a case, we must apply
		// the new cluster configuration to ourselves (specifically,
		// the last cluster configuration among the new entries)
		for index := len(msg.Entries) - 1; index >= 0; index-- {
			if msg.Entries[index].Kind != iface.EntryAddServer &&
				msg.Entries[index].Kind != iface.EntryRemoveServer {
				continue
			}
			record := &iface.ClusterChangeCommand{}
			json.Unmarshal(msg.Entries[index].Command, &record)
			actions = append(actions, iface.ActionSetClusterChange{
				NewClusterChangeIndex: msg.PrevLogIndex + int64(index+1),
				NewClusterChangeTerm:  msg.Entries[index].Term,
			})
			peers := []iface.PeerAddress{}
			for _, addr := range record.NewCluster {
				if addr != status.NodeAddress() {
					peers = append(peers, addr)
				}
			}
			actions = append(actions, iface.ActionSetPeers{
				PeerAddresses: peers,
			})
			break
		}

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

	// reject if we recently heard from leader
	// (to avoid "disruptive servers" during cluster configuration change)
	if time.Now().Sub(status.LeaderLastHeard()) < status.MinElectionTimeout() {
		actions = append(actions, iface.ReplyRequestVote{
			VoteGranted: false,
			Term:        status.CurrentTerm(),
		})
		return actions
	}

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
