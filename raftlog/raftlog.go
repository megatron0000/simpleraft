package raftlog

import (
	"encoding/json"
	"simpleraft/iface"
	"simpleraft/storage"
	"strconv"
)

// helper
func toString(x int64) string {
	return strconv.FormatInt(x, 10)
}

// RaftLog provides log read/delete/append functionalities.
//
// A particular instance of the struct represents the log of 1
// raft node
type RaftLog struct {
	lastIndex int64
	storage   *storage.Storage
}

// New creates a RaftLog instance
func New(storage *storage.Storage) (log *RaftLog, err error) {
	var (
		lastIndexMarshal []byte
		lastIndex        int64
	)

	lastIndexMarshal, err = storage.Get(
		nil,
		[]byte("/raft/log/lastIndex"))

	if err != nil {
		return nil, err
	}

	if lastIndexMarshal == nil {
		lastIndex = -1
	} else {
		lastIndex, err = strconv.ParseInt(string(lastIndexMarshal), 10, 64)
	}

	if err != nil {
		return nil, err
	}

	return &RaftLog{storage: storage, lastIndex: lastIndex}, nil
}

// Append writes a log entry to the end of the log
func (log *RaftLog) Append(entry iface.LogEntry) (err error) {
	switch entry.Kind {
	case iface.EntryStateMachineCommand:
	case iface.EntryAddServer:
	case iface.EntryRemoveServer:
	case iface.EntryNoOp:
	default:
		panic("unknown log entry kind: " + entry.Kind)
	}

	var (
		marshal []byte
	)

	marshal, err = json.Marshal(entry)

	if err != nil {
		return err
	}

	if err = log.storage.BeginTransaction(); err != nil {
		return err
	}

	defer func() {
		switch err {
		case nil:
			log.storage.Commit()
		default:
			log.storage.Rollback()
		}
	}()

	err = log.storage.Set(
		[]byte("/raft/log/index="+toString(log.lastIndex+1)),
		marshal)

	if err != nil {
		return err
	}

	log.lastIndex++

	err = log.storage.Set([]byte("/raft/log/lastIndex"),
		[]byte(toString(log.lastIndex)))

	if err != nil {
		return err
	}

	return nil
}

// Remove deletes the log entry with greatest index (i.e. removes from the end)
func (log *RaftLog) Remove() (err error) {

	if log.lastIndex == -1 {
		panic("raft log: called Remove() on an empty log")
	}

	if err = log.storage.BeginTransaction(); err != nil {
		return err
	}
	defer func() {
		switch err {
		case nil:
			log.storage.Commit()
		default:
			log.storage.Rollback()
		}
	}()

	if err = log.storage.Delete(
		[]byte("/raft/log/index=" + toString(log.lastIndex))); err != nil {
		return err
	}

	if err = log.storage.Set([]byte("/raft/log/lastIndex"),
		[]byte(toString(log.lastIndex-1))); err != nil {
		return err
	}

	log.lastIndex--

	return nil
}

// Get reads a log entry, returning a pointer to it.
//
// If no entry exists at `index`, returns `logEntry == nil`.
func (log *RaftLog) Get(index int64) (logEntry *iface.LogEntry, err error) {
	var (
		marshal []byte
	)

	logEntry = &iface.LogEntry{}

	marshal, err = log.storage.Get(
		nil,
		[]byte("/raft/log/index="+toString(index)))

	if err != nil {
		return nil, err
	}

	if marshal == nil {
		return nil, nil
	}

	err = json.Unmarshal(marshal, logEntry)

	if err != nil {
		return nil, err
	}

	return logEntry, nil
}

// LastIndex returns either -1 or the highest index of any log entry present in disk
func (log *RaftLog) LastIndex() int64 {
	return log.lastIndex
}
