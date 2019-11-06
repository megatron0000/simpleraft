package transport

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"strconv"
)

// Transport represents an active transport service listening on a port
type Transport struct {
	receiver chan IncomingMessage
	server   *http.Server
}

type handler struct {
	transport *Transport
}

// IncomingMessage represents a message received via a Transport service,
// coupled with a `ReplyChan` through which data can be sent back to the
// communicating peer
type IncomingMessage struct {
	Endpoint  string
	Data      []byte
	ReplyChan chan []byte
}

func (handler handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(200)
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return
	}
	replyChan := make(chan []byte)
	handler.transport.receiver <- IncomingMessage{
		Endpoint:  req.URL.EscapedPath(),
		Data:      body,
		ReplyChan: replyChan,
	}

	response := <-replyChan
	w.Header().Set("Content-Length", strconv.Itoa(len(response)))
	w.Header().Set("Connection", "close")
	w.Write(response)
}

// New contructs a Transport instance and initiates it
func New(address string) *Transport {
	handler := handler{}
	server := &http.Server{
		Addr:    address,
		Handler: &handler,
	}
	t := &Transport{receiver: make(chan IncomingMessage), server: server}
	handler.transport = t

	go func() { server.ListenAndServe() }()

	return t
}

// Send sends a message to `address` and returns a channel from
// which the response can be read (once)
func (transport *Transport) Send(address string, data []byte) (replyChan chan []byte) {
	client := &http.Client{}
	replyChan = make(chan []byte)

	// wait for response
	go func() {
		res, err := client.Post(address, "application/json", bytes.NewReader(data))
		if err != nil {
			close(replyChan)
			return
		}
		defer res.Body.Close()
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			close(replyChan)
			return
		}
		replyChan <- body
	}()

	return replyChan
}

// ReceiverChan returns the channel on which `transport` receives requests
// Reading from it is the way to be notified of incoming requests.
//
// Once a request is received, it MUST be answered (you must reply to it)
func (transport *Transport) ReceiverChan() chan IncomingMessage {
	return transport.receiver
}
