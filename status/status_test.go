package status

import (
	"simpleraft/iface"
	"simpleraft/storage"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRecoverFromDisk(t *testing.T) {
	storage, err := storage.New("/tmp/raftdb")
	defer storage.Clear() // close and delete information

	if err != nil {
		t.Fatal(err)
	}

	nodeAddress := iface.PeerAddress("localhost:10000")
	peerAddresses := []iface.PeerAddress{"localhost:10001"}
	newPeerAddresses := []iface.PeerAddress{"localhost:10001", "localhost:10002"}

	status := New(nodeAddress, peerAddresses, storage)

	assert.Equal(t, iface.StateFollower, status.State())
	assert.Equal(t, nodeAddress, status.NodeAddress())
	assert.Equal(t, int64(0), status.CurrentTerm())
	assert.Equal(t, iface.PeerAddress(""), status.VotedFor())
	assert.Equal(t, int64(0), status.VoteCount())
	assert.Equal(t, peerAddresses, status.PeerAddresses())
	assert.Equal(t, int64(-1), status.CommitIndex())
	assert.Equal(t, int64(-1), status.LastApplied())
	assert.Equal(t, int64(-1), status.ClusterChangeIndex())
	assert.Equal(t, int64(-1), status.ClusterChangeTerm())
	assert.Equal(t, int64(0), status.NextIndex(peerAddresses[0]))
	assert.Equal(t, int64(-1), status.MatchIndex(peerAddresses[0]))

	status.SetState(iface.StateLeader)
	status.SetNodeAddress(iface.PeerAddress("localhost:10"))
	status.SetCurrentTerm(10)
	status.SetVotedFor(peerAddresses[0])
	status.SetVoteCount(4)
	status.SetCommitIndex(12)
	status.SetLastApplied(9)
	status.SetClusterChange(13, 17)
	status.SetNextIndex(peerAddresses[0], 10)
	status.SetMatchIndex(peerAddresses[0], 9)
	status.SetPeerAddresses(newPeerAddresses)

	assert.Equal(t, iface.StateLeader, status.State())
	assert.Equal(t, iface.PeerAddress("localhost:10"), status.NodeAddress())
	assert.Equal(t, int64(10), status.CurrentTerm())
	assert.Equal(t, peerAddresses[0], status.VotedFor())
	assert.Equal(t, int64(4), status.VoteCount())
	assert.Equal(t, int64(12), status.CommitIndex())
	assert.Equal(t, int64(9), status.LastApplied())
	assert.Equal(t, int64(13), status.ClusterChangeIndex())
	assert.Equal(t, int64(17), status.ClusterChangeTerm())
	assert.Equal(t, int64(10), status.NextIndex(newPeerAddresses[0]))
	assert.Equal(t, int64(9), status.MatchIndex(newPeerAddresses[0]))
	assert.Equal(t, int64(0), status.NextIndex(newPeerAddresses[1]))
	assert.Equal(t, int64(-1), status.MatchIndex(newPeerAddresses[1]))

	status = New(nodeAddress, peerAddresses, storage)

	assert.Equal(t, iface.StateFollower, status.State())
	assert.Equal(t, iface.PeerAddress("localhost:10"), status.NodeAddress())
	assert.Equal(t, int64(10), status.CurrentTerm())
	assert.Equal(t, peerAddresses[0], status.VotedFor())
	assert.Equal(t, int64(0), status.VoteCount())
	assert.Equal(t, int64(-1), status.CommitIndex())
	assert.Equal(t, int64(-1), status.LastApplied())
	assert.Equal(t, int64(13), status.ClusterChangeIndex())
	assert.Equal(t, int64(17), status.ClusterChangeTerm())
	assert.Equal(t, int64(0), status.NextIndex(newPeerAddresses[0]))
	assert.Equal(t, int64(-1), status.MatchIndex(newPeerAddresses[0]))
	assert.Equal(t, int64(0), status.NextIndex(newPeerAddresses[1]))
	assert.Equal(t, int64(-1), status.MatchIndex(newPeerAddresses[1]))

}
