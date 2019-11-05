package status

import (
	"simpleraft/storage"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRecoverFromDisk(t *testing.T) {
	db, err := storage.GetStore("/tmp/raftdb")
	defer storage.ClearStore("/tmp/raftdb")

	if err != nil {
		t.Error(err)
	}

	status := New(1, db)

	assert.Equal(t, int64(1), status.NodeID())
	assert.Equal(t, int64(0), status.CurrentTerm())
	assert.Equal(t, int64(-1), status.VotedFor())
	assert.Equal(t, int64(-1), status.CommitIndex())
	assert.Equal(t, int64(-1), status.LastApplied())

	status.SetCurrentTerm(10)
	status.SetVotedFor(101)
	status.SetCommitIndex(12)
	status.SetLastApplied(9)

	assert.Equal(t, int64(1), status.NodeID())
	assert.Equal(t, int64(10), status.CurrentTerm())
	assert.Equal(t, int64(101), status.VotedFor())
	assert.Equal(t, int64(12), status.CommitIndex())
	assert.Equal(t, int64(9), status.LastApplied())

	status = New(1, db)

	assert.Equal(t, int64(1), status.NodeID())
	assert.Equal(t, int64(10), status.CurrentTerm())
	assert.Equal(t, int64(101), status.VotedFor())
	assert.Equal(t, int64(-1), status.CommitIndex())
	assert.Equal(t, int64(-1), status.LastApplied())

}
