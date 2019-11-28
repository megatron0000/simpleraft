package webapp

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"simpleraft/executor"
	"simpleraft/iface"
	"simpleraft/transport"

	socketio "github.com/googollee/go-socket.io"
)

// RaftWebApp is the backend for the monitor web app
type RaftWebApp struct {
	// an executor instance is monitored by the web app
	executor *executor.Executor
	// port where the webapp will be served
	port string
	// directory where web app files (html, css, js) are located
	wwwDir string
}

// New instantiates a web app instance, but DOES NOT initiate it
func New(exec *executor.Executor, port string, wwwDir string) *RaftWebApp {
	return &RaftWebApp{executor: exec, port: port, wwwDir: wwwDir}
}

// Start activates the backend server. This is a blocking function
func (app *RaftWebApp) Start() {
	fmt.Printf("webapp: starting at port %s\n", app.port)
	app.executor.Stop()
	executorIsRunning := false

	server, err := socketio.NewServer(nil)

	if err != nil {
		log.Fatal(err)
	}

	server.OnConnect("/", func(s socketio.Conn) error {
		s.SetContext("")
		fmt.Println("webapp: connected: ", s.ID())
		return nil
	})

	server.OnEvent("/", "get-executor-is-running", func(s socketio.Conn) {
		s.SetContext("")
		s.Emit("executor-is-running-info", executorIsRunning)
	})

	server.OnEvent("/", "set-executor-is-running", func(s socketio.Conn, msg bool) {
		s.SetContext("")
		if executorIsRunning == true && msg == false {
			app.executor.Stop()
		}
		if executorIsRunning == false && msg == true {
			go app.executor.Run()
		}
		executorIsRunning = msg

		s.Emit("executor-is-running-info", executorIsRunning)
		s.Emit("ack")
	})

	server.OnEvent("/", "get-node-state", func(s socketio.Conn) {
		s.SetContext("")
		s.Emit("node-state-info", app.executor.Status().State())
	})

	server.OnEvent("/", "set-node-state", func(s socketio.Conn, msg string) {
		s.SetContext("")
		app.executor.Status().SetState(msg)
		s.Emit("node-state-info", app.executor.Status().State())
		s.Emit("ack")
	})

	server.OnEvent("/", "get-node-address", func(s socketio.Conn) {
		s.SetContext("")
		s.Emit("node-address-info", app.executor.Status().NodeAddress())
	})

	server.OnEvent("/", "set-node-address", func(s socketio.Conn, msg string) {
		s.SetContext("")
		app.executor.Status().SetNodeAddress(iface.PeerAddress(msg))
		app.executor.Transport().ChangeAddress(msg)
		s.Emit("node-address-info", app.executor.Status().NodeAddress())
		s.Emit("ack")
	})

	server.OnEvent("/", "get-current-term", func(s socketio.Conn) {
		s.SetContext("")
		s.Emit("current-term-info", app.executor.Status().CurrentTerm())
	})

	server.OnEvent("/", "set-current-term", func(s socketio.Conn, msg int64) {
		s.SetContext("")
		app.executor.Status().SetCurrentTerm(msg)
		s.Emit("current-term-info", app.executor.Status().CurrentTerm())
		s.Emit("ack")
	})

	server.OnEvent("/", "get-voted-for", func(s socketio.Conn) {
		s.SetContext("")
		s.Emit("voted-for-info", app.executor.Status().VotedFor())
	})

	server.OnEvent("/", "set-voted-for", func(s socketio.Conn, msg string) {
		s.SetContext("")
		app.executor.Status().SetVotedFor(iface.PeerAddress(msg))
		s.Emit("voted-for-info", app.executor.Status().VotedFor())
		s.Emit("ack")
	})

	server.OnEvent("/", "get-vote-count", func(s socketio.Conn) {
		s.SetContext("")
		s.Emit("vote-count-info", app.executor.Status().VoteCount())
	})

	server.OnEvent("/", "set-vote-count", func(s socketio.Conn, msg int64) {
		s.SetContext("")
		app.executor.Status().SetVoteCount(msg)
		s.Emit("vote-count-info", app.executor.Status().VoteCount())
		s.Emit("ack")
	})

	server.OnEvent("/", "get-commit-index", func(s socketio.Conn) {
		s.SetContext("")
		s.Emit("commit-index-info", app.executor.Status().CommitIndex())
	})

	server.OnEvent("/", "set-commit-index", func(s socketio.Conn, msg int64) {
		s.SetContext("")
		app.executor.Status().SetCommitIndex(msg)
		s.Emit("commit-index-info", app.executor.Status().CommitIndex())
		s.Emit("ack")
	})

	server.OnEvent("/", "get-last-applied", func(s socketio.Conn) {
		s.SetContext("")
		s.Emit("last-applied-info", app.executor.Status().LastApplied())
	})

	server.OnEvent("/", "set-last-applied", func(s socketio.Conn, msg int64) {
		s.SetContext("")
		app.executor.Status().SetLastApplied(msg)
		s.Emit("last-applied-info", app.executor.Status().LastApplied())
		s.Emit("ack")
	})

	server.OnEvent("/", "get-peer-addresses", func(s socketio.Conn) {
		s.SetContext("")
		s.Emit("peer-addresses-info", app.executor.Status().PeerAddresses())
	})

	server.OnEvent("/", "set-peer-addresses", func(s socketio.Conn, msg string) {
		s.SetContext("")
		addresses := []iface.PeerAddress{}
		err := json.Unmarshal([]byte(msg), &addresses)
		if err != nil {
			return
		}
		app.executor.Status().SetPeerAddresses(addresses)
		s.Emit("peer-addresses-info", app.executor.Status().PeerAddresses())

		// same code as 'get-next-indexes'
		s.SetContext("")
		type Res struct {
			PeerAddress iface.PeerAddress
			NextIndex   int64
		}
		list := []Res{}
		for _, addr := range app.executor.Status().PeerAddresses() {
			list = append(list, Res{
				PeerAddress: addr,
				NextIndex:   app.executor.Status().NextIndex(addr),
			})
		}
		s.Emit("next-indexes-info", list)

		// same code as 'get-match-indexes'
		s.SetContext("")
		type Res2 struct {
			PeerAddress iface.PeerAddress
			MatchIndex  int64
		}
		list2 := []Res2{}
		for _, addr := range app.executor.Status().PeerAddresses() {
			list2 = append(list2, Res2{
				PeerAddress: addr,
				MatchIndex:  app.executor.Status().MatchIndex(addr),
			})
		}
		s.Emit("match-indexes-info", list2)

		s.Emit("ack")
	})

	server.OnEvent("/", "get-next-indexes", func(s socketio.Conn) {
		s.SetContext("")
		type Res struct {
			PeerAddress iface.PeerAddress
			NextIndex   int64
		}
		list := []Res{}
		for _, addr := range app.executor.Status().PeerAddresses() {
			list = append(list, Res{
				PeerAddress: addr,
				NextIndex:   app.executor.Status().NextIndex(addr),
			})
		}
		s.Emit("next-indexes-info", list)
	})

	server.OnEvent("/", "set-next-index", func(s socketio.Conn, addr string, nextIndex int64) {
		s.SetContext("")
		app.executor.Status().SetNextIndex(iface.PeerAddress(addr), nextIndex)
		type Res struct {
			PeerAddress iface.PeerAddress
			NextIndex   int64
		}
		list := []Res{}
		for _, addr := range app.executor.Status().PeerAddresses() {
			list = append(list, Res{
				PeerAddress: addr,
				NextIndex:   app.executor.Status().NextIndex(addr),
			})
		}
		s.Emit("next-indexes-info", list)
		s.Emit("ack")
	})

	server.OnEvent("/", "get-match-indexes", func(s socketio.Conn) {
		s.SetContext("")
		type Res struct {
			PeerAddress iface.PeerAddress
			MatchIndex  int64
		}
		list := []Res{}
		for _, addr := range app.executor.Status().PeerAddresses() {
			list = append(list, Res{
				PeerAddress: addr,
				MatchIndex:  app.executor.Status().MatchIndex(addr),
			})
		}
		s.Emit("match-indexes-info", list)
	})

	server.OnEvent("/", "set-match-index", func(s socketio.Conn, addr string, matchIndex int64) {
		s.SetContext("")
		app.executor.Status().SetMatchIndex(iface.PeerAddress(addr), matchIndex)
		type Res struct {
			PeerAddress iface.PeerAddress
			MatchIndex  int64
		}
		list := []Res{}
		for _, addr := range app.executor.Status().PeerAddresses() {
			list = append(list, Res{
				PeerAddress: addr,
				MatchIndex:  app.executor.Status().MatchIndex(addr),
			})
		}
		s.Emit("match-indexes-info", list)
		s.Emit("ack")
	})

	server.OnEvent("/", "get-cluster-change-index", func(s socketio.Conn) {
		s.SetContext("")
		s.Emit("cluster-change-index-info", app.executor.Status().ClusterChangeIndex())
	})

	server.OnEvent("/", "set-cluster-change-index", func(s socketio.Conn, msg int64) {
		s.SetContext("")
		app.executor.Status().SetClusterChange(msg, app.executor.Status().ClusterChangeTerm())
		s.Emit("cluster-change-index-info", app.executor.Status().ClusterChangeIndex())
		s.Emit("ack")
	})

	server.OnEvent("/", "get-cluster-change-term", func(s socketio.Conn) {
		s.SetContext("")
		s.Emit("cluster-change-term-info", app.executor.Status().ClusterChangeTerm())
	})

	server.OnEvent("/", "set-cluster-change-term", func(s socketio.Conn, msg int64) {
		s.SetContext("")
		app.executor.Status().SetClusterChange(app.executor.Status().ClusterChangeIndex(), msg)
		s.Emit("cluster-change-term-info", app.executor.Status().ClusterChangeTerm())
		s.Emit("ack")
	})

	server.OnEvent("/", "get-logs", func(s socketio.Conn, msg string) {
		s.SetContext("")
		req := struct {
			StartIndex int
			EndIndex   int
		}{}

		type Log struct {
			Term    int64
			Command string
			Kind    string
			Result  string
		}

		err := json.Unmarshal([]byte(msg), &req)
		if err != nil {
			return
		}

		// grab logs
		maxIndex := app.executor.Log().LastIndex()

		// endIndex -1 means "until the last one"
		if req.EndIndex == -1 {
			req.EndIndex = int(maxIndex)
		}

		logs := []Log{}

		for index := req.StartIndex; index <= req.EndIndex && int64(index) <= maxIndex; index++ {
			log, _ := app.executor.Log().Get(int64(index))
			logs = append(logs, Log{
				Term:    log.Term,
				Command: string(log.Command),
				Kind:    log.Kind,
				Result:  string(log.Result),
			})
		}

		s.Emit("logs-info", logs)

	})

	server.OnEvent("/", "remove-logs", func(s socketio.Conn, count int) {
		s.SetContext("")
		length := app.executor.Log().LastIndex() + 1
		for index := 0; index < count && index < int(length); index++ {
			app.executor.Log().Remove()
		}

		type Log struct {
			Term    int64
			Command string
			Kind    string
			Result  string
		}

		logs := []Log{}

		for index := int64(0); index <= app.executor.Log().LastIndex(); index++ {
			log, _ := app.executor.Log().Get(index)
			logs = append(logs, Log{
				Term:    log.Term,
				Command: string(log.Command),
				Kind:    log.Kind,
				Result:  string(log.Result),
			})
		}

		s.Emit("logs-info", logs)
		s.Emit("ack")
	})

	server.OnEvent("/", "new-client-command", func(s socketio.Conn, operation string,
		key string, value string) {
		s.SetContext("")
		trans := transport.New("")
		type Req struct {
			Operation string
			Key       string
			Value     string
		}
		type Encapsulation struct {
			Command []byte
		}
		req := Req{Operation: operation, Key: key, Value: value}
		marshal, err := json.Marshal(req)
		if err != nil {
			s.Emit("client-info", err)
			s.Emit("ack")
			return
		}
		encaps := Encapsulation{Command: marshal}
		marshal2, err := json.Marshal(encaps)
		if err != nil {
			s.Emit("client-info", err)
			s.Emit("ack")
			return
		}
		replyChan := trans.Send(string(app.executor.Status().NodeAddress())+"/stateMachineCommand",
			marshal2)
		res, ok := <-replyChan
		if !ok {
			s.Emit("client-info", "internal error")
			s.Emit("ack")
			return
		}
		s.Emit("client-info", res)
		s.Emit("ack")
	})

	server.OnEvent("/", "new-client-probe", func(s socketio.Conn, index int64, term int64) {
		s.SetContext("")
		trans := transport.New("")
		req := iface.MsgStateMachineProbe{Index: index, Term: term}
		marshal, err := json.Marshal(req)
		if err != nil {
			s.Emit("client-info", err)
			s.Emit("ack")
			return
		}
		replyChan := trans.Send(string(app.executor.Status().NodeAddress())+"/stateMachineProbe",
			marshal)
		res, ok := <-replyChan
		if !ok {
			s.Emit("client-info", "internal error")
			s.Emit("ack")
			return
		}
		s.Emit("client-info", res)
		s.Emit("ack")
	})

	server.OnError("/", func(s socketio.Conn, e error) {
		// fmt.Println("meet error:", e)
	})
	server.OnDisconnect("/", func(s socketio.Conn, reason string) {
		// fmt.Println("closed", reason)
	})

	go server.Serve()
	defer server.Close()

	http.Handle("/socket.io/", server)
	http.Handle("/", http.FileServer(http.Dir(app.wwwDir)))

	http.ListenAndServe(":"+app.port, nil)
}
