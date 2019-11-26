package statemachine

import (
	"encoding/json"
)

// Command specifies the format of commands
// understood by StateMachine
type Command struct {
	// Operation must be "read" or "write"
	Operation string
	// Key is the key whose value will be either read or written to
	Key string
	// Value is the new value to be associated to `Key`. Only has meaning
	// when `Operation` == "write"
	Value string
}

// StateMachine implements a simple in-memory key-value store
// (both keys and values are strings)
type StateMachine struct {
	database map[string]string
}

// New constructs a StateMachine instance
func New() *StateMachine {
	return &StateMachine{database: make(map[string]string)}
}

// Apply is iface.StateMachine's only method. By implementing it,
// the StateMachine struct can be used as a state machine for raft
func (machine *StateMachine) Apply(command []byte) []byte {

	com := Command{}
	err := json.Unmarshal(command, &com)
	if err != nil {
		panic(err)
	}

	if com.Operation == "write" {
		machine.database[com.Key] = com.Value
		return []byte(com.Value)
	}

	val, ok := machine.database[com.Key]
	if !ok {
		return []byte{}
	}
	return []byte(val)
}
