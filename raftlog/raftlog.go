package raftlog

import (
	"encoding/json"
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

// LogEntry represents an entry in the log.
// `command` is an interface{} because there is no a-priori structure
// to be imposed (on the contrary: each application on top of raft has
// its intended semantics for the command)
type LogEntry struct {
	Term    int64
	Command interface{}
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

// Set writes a log entry to disk (if one exists at `index`, it will be overwritten)
func (log *RaftLog) Set(index int64, entry LogEntry) (err error) {

	var (
		marshal []byte
	)

	marshal, err = json.Marshal(entry)

	if err != nil {
		return err
	}

	err = log.storage.Set(
		[]byte("/raft/log/index="+toString(index)),
		marshal)

	if err != nil {
		return err
	}

	if index > log.lastIndex {
		log.lastIndex = index
		err = log.storage.Set([]byte("/raft/log/lastIndex"),
			[]byte(toString(log.lastIndex)))
	}

	if err != nil {
		return err
	}

	return nil
}

// Get reads a log entry, returning a pointer to it.
//
// If no entry exists at `index`, returns `logEntry == nil`.
func (log *RaftLog) Get(index int64) (logEntry *LogEntry, err error) {
	var (
		marshal []byte
	)

	logEntry = &LogEntry{}

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
