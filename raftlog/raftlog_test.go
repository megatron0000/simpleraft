package raftlog

import (
	"simpleraft/iface"
	"simpleraft/storage"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLog(t *testing.T) {
	var (
		store          *storage.Storage
		log            *RaftLog
		entry          *iface.LogEntry
		command        map[string]float64
		commandContent interface{}
		err            error
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

	command = make(map[string]float64)
	command["a"] = 1
	command["b"] = 2

	err = log.Set(1, iface.LogEntry{Term: 2, Command: command})

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

	assert.Equal(t, int64(2), entry.Term)

	commandContent = entry.Command.(map[string]interface{})["a"]
	assert.Equal(t, float64(1), commandContent.(float64))

	commandContent = entry.Command.(map[string]interface{})["b"]
	assert.Equal(t, float64(2), commandContent.(float64))

}
