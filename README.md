# Features

Simpleraft implements:
- Leader election
- Log replication
- Cluster membership changes

# Setup

`go get modernc.org/kv` to install the key-value store we use to persist elements to disk

`go get github.com/stretchr/testify/assert` to install the library used for tests

`go get github.com/googollee/go-socket.io` to install the library used for the webapp

# Contributing

We recommend Visual Studio Code editor with Go extension by Microsoft (extension is installed through the editor itself).

# Architecture

See `achitecture.txt`. Diagram made with asciiflow.

# Testing

Run `bash testall.sh`. Maybe only works on Linux, since we make use of "/tmp/..."-style paths