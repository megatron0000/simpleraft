import(
	"math"
)

func FollowerOnStateChanged(msg MsgStateChanged, log RaftLog, status Status) []interface{	
	actions := make([]interface{}, 0) // list of actions created
	
	actions = append(actions, ActionSetVotedFor{NewVotedFor: ""}) // as a new follower, VotedFor is initially empty
	actions = append(actions, ActionSetVoteCount{NewVoteCount: 0}) // as a follower, vote count must be set to 0
	actions = append(actions, ActionResetTimer{HalfTime: false}) // timeout should be reseted
	
	return actions
}

func FollowerOnAppendEntries(msg MsgAppendEntries, log RaftLog, status Status) []interface{
	actions := make([]interface{}, 0) // list of actions created
	
	// If 'append entry' is from a leader with smaller term OR log matching property not achieved yet
	if (MsgAppendEntries.Term < status.CurrentTerm()) || (MsgAppendEntries.Term == status.CurrentTerm() && log.get(MsgAppendEntries.PrevLogIndex).Term != MsgAppendEntries.PrevLogTerm){
		actions = append(actions, ReplyAppendEntries{Success: false, Term: status.CurrentTerm()}) // not successfull append entry
		return actions
	}
	
	// as program jumped the if statement above, we have a successfull append entry
	actions = append(actions, ReplyAppendEntries{Success: true, Term: status.CurrentTerm()}) 
	actions = append(actions, ActionDeleteLog{Count: (log.LastIndex()+1 - (MsgAppendEntries.PrevLogIndex+i))} // delete all entries in log after PrevLogIndex
	actions	= append(actions, ActionAppendLog{Entries: MsgAppendEntries.Entries} // append all entries sent by leader
	
	if MsgAppendEntries.LeaderCommitIndex > status.CommitIndex(){
		actions = append(actions, ActionSetCommitIndex{NewCommitIndex: Min(MsgAppendEntries.LeaderCommitIndex, MsgAppendEntries.PrevLogIndex + len(MsgAppendEntries.Entries))})
	}
	
	return actions
}

func FollowerOnRequestVote(msg MsgRequestVote, log RaftLog, status Status) []interface{
	actions := make([]interface{}, 0) // list of actions created
	
	// if candidate is in a smaller term, vote is not granted
	if MsgRequestVote.Term < status.CurrentTerm(){
		actions = append(actions, ReplyDecidedVote{VoteGranted: false, Term: status.CurrentTerm()}) // not successfull vote
		return actions
	} 
	
	// if candidate is at least as updated as follower, vote is granted
	if (status.VotedFor() == "" || status.VotedFor() == MsgRequestVote.CandidateAddress) 
		&& ((status.CurrentTerm() < MsgRequestVote.LastLogTerm) || ((status.CurrentTerm() == MsgRequestVote.LastLogTerm) && (status.CommitIndex() <= MsgRequestVote.LastLogIndex))) {
		actions = append(actions, ReplyDecidedVote{VoteGranted: true, Term: status.CurrentTerm()})
		actions = append(actions, ActionSetVotedFor{NewVotedFor: MsgRequestVote.CandidateAddress}) // vote is granted
	}
	else{
		actions = append(actions, ReplyDecidedVote{VoteGranted: false, Term: status.CurrentTerm()}) // not successfull vote
	}
	
	return actions
}

func FollowerOnAddServer(msg MsgAddServer, log RaftLog, status Status) []interface{
	actions := make([]interface{}, 0) // list of actions created
	actions = append(actions, ReplyNotLeader{}) // leader should be responsable for this activity
	return actions
}

func FollowerOnRemoveServer(msg MsgRemoveServer, log RaftLog, status Status) []interface{
	actions := make([]interface{}, 0) // list of actions created
	actions = append(actions, ReplyNotLeader{}) // leader should be responsable for this activity
	return actions
}

func FollowerOnTimeout(msg MsgTimeout, log RaftLog, status Status) []interface{
	actions := make([]interface{}, 0) // list of actions created	
	actions = append(actions, ActionSetState{NewState: StateCandidate}) // a timeout for a follower means it should change its state to candidate	
	return actions
}

func FollowerOnStateMachineCommand(msg MsgStateMachineCommand, log RaftLog, status Status) []interface{
	actions := make([]interface{}, 0) // list of actions created
	actions = append(actions, ReplyNotLeader{}) // leader should be responsable for this activity
	return actions	
}
