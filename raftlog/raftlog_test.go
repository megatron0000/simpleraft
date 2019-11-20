package raftlog

import (
	"simpleraft/iface"
	"simpleraft/storage"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLog(t *testing.T) {
	var (
		store *storage.Storage
		log   *RaftLog
		entry *iface.LogEntry
		err   error
	)

	store, err = storage.New("/tmp/raftdb")
	defer store.Clear()

	if err != nil {
		t.Fatal(err)
	}

	log, err = New(store)

	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, int64(-1), log.LastIndex())

	err = log.Append(iface.LogEntry{
		Term:    2,
		Command: []byte("command"),
		Kind:    iface.EntryStateMachineCommand})

	if err != nil {
		t.Fatal(err)
	}

	err = log.Append(iface.LogEntry{
		Term:    3,
		Command: []byte("other command"),
		Kind:    iface.EntryStateMachineCommand})

	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, int64(1), log.LastIndex())

	// make sure it was saved
	log, err = New(store)

	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, int64(1), log.LastIndex())

	entry, err = log.Get(1)

	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, int64(3), entry.Term)
	assert.Equal(t, []byte("other command"), entry.Command)
	assert.Equal(t, iface.EntryStateMachineCommand, entry.Kind)

	err = log.Remove()

	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, int64(0), log.LastIndex())

	entry, err = log.Get(0)

	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, int64(2), entry.Term)
	assert.Equal(t, []byte("command"), entry.Command)
	assert.Equal(t, iface.EntryStateMachineCommand, entry.Kind)

}
