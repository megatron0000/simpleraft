package main

import (
	"flag"
	"simpleraft/executor"
	"simpleraft/iface"
	"simpleraft/rulehandler"
	"simpleraft/statemachine"
	"simpleraft/webapp"
	"strings"
)

var (
	wwwdir             = flag.String("wwwdir", "./webapp", "define the directory where webapp files (js, css, html) are located")
	webappPort         = flag.String("webapp-port", "8081", "define the port where raft web app (for monitoring) will be launched.")
	raftdb             = flag.String("raftdb", "./raftdb", "file path to use as persistent storage for raft node. Each node must have an exclusive database file. The file need not exist, it will be created by raft")
	defaultAddress     = flag.String("default-address", "localhost:10000", "default address where the raft node will listen. Only used if raftdb has no prior address already recorded")
	defaultPeers       = flag.String("default-peers", "localhost:10001", "default network addresses of raft peers (not including current node), separated by comma. Example: localhost:10001,localhost:10002. Only used if raftdb has no prior record of peers")
	minElectionTimeout = flag.Int("min-election-timeout", 4000, "When a raft node chooses a random timeout, this is the minimum value this timeout may be set to")
	maxElectionTimeout = flag.Int("max-election-timeout", 8000, "When a raft node chooses a random timeout, this is the maximum value this timeout may be set to")
)

func main() {
	var (
		exec      *executor.Executor
		app       *webapp.RaftWebApp
		addr      []string
		addresses []iface.PeerAddress
		err       error
	)

	flag.Parse()

	addr = strings.Split(*defaultPeers, ",")
	addresses = make([]iface.PeerAddress, len(addr))
	for index, add := range addr {
		addresses[index] = iface.PeerAddress(add)
	}

	exec, err = executor.New(
		iface.PeerAddress(*defaultAddress),
		addresses,
		*raftdb,
		*minElectionTimeout,
		*maxElectionTimeout,
		rulehandler.New(),
		statemachine.New(),
	)
	defer func() {
		if exec != nil {
			exec.TearDown()
		}
	}()
	if err != nil {
		panic(err)
	}

	app = webapp.New(exec, *webappPort, *wwwdir)

	app.Start()

}
