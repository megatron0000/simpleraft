package status

import (
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

	nodeAddress := PeerAddress("localhost:10000")
	peerAddresses := []PeerAddress{"localhost:10001"}

	status := New(nodeAddress, peerAddresses, storage)

	assert.Equal(t, nodeAddress, status.NodeAddress())
	assert.Equal(t, int64(0), status.CurrentTerm())
	assert.Equal(t, PeerAddress(""), status.VotedFor())
	assert.Equal(t, int64(-1), status.CommitIndex())
	assert.Equal(t, int64(-1), status.LastApplied())

	status.SetCurrentTerm(10)
	status.SetVotedFor(peerAddresses[0])
	status.SetCommitIndex(12)
	status.SetLastApplied(9)

	assert.Equal(t, nodeAddress, status.NodeAddress())
	assert.Equal(t, int64(10), status.CurrentTerm())
	assert.Equal(t, peerAddresses[0], status.VotedFor())
	assert.Equal(t, int64(12), status.CommitIndex())
	assert.Equal(t, int64(9), status.LastApplied())

	status = New(nodeAddress, peerAddresses, storage)

	assert.Equal(t, nodeAddress, status.NodeAddress())
	assert.Equal(t, int64(10), status.CurrentTerm())
	assert.Equal(t, peerAddresses[0], status.VotedFor())
	assert.Equal(t, int64(-1), status.CommitIndex())
	assert.Equal(t, int64(-1), status.LastApplied())

}
