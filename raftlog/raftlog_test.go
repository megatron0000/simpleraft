package raftlog

import (
	"simpleraft/storage"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLog(t *testing.T) {
	var (
		store *storage.Storage
		log   *RaftLog
		entry *LogEntry
		err   error
	)

	store, err = storage.New("/tmp/raftdb")
	defer store.Clear()

	if err != nil {
		t.Fatal(err)
	}

	log, err = New(store, 0x1)

	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, int64(-1), log.LastIndex())

	err = log.Set(1, LogEntry{Term: 2, Command: nil})

	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, int64(1), log.LastIndex())

	// make sure it was saved
	log, err = New(store, 0x1)

	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, int64(1), log.LastIndex())

	entry, err = log.Get(1)

	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, int64(2), entry.Term)

}
