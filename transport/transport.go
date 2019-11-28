package transport

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

// Transport represents an active transport service listening on a port
type Transport struct {
	receiver chan IncomingMessage
	server   *http.Server
	address  string
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

// New contructs a Transport instance but DOES NOT initiate
func New(address string) *Transport {
	handler := handler{}
	server := &http.Server{
		Addr:    address,
		Handler: &handler,
	}
	t := &Transport{receiver: make(chan IncomingMessage), server: server, address: address}
	handler.transport = t

	return t
}

// Listen activates the transport service
func (transport *Transport) Listen() {
	go transport.server.ListenAndServe()
}

// Close immediately stops Listen()
// TODO: test this function
func (transport *Transport) Close() {
	transport.server.Close()
	handler := &handler{}
	transport.server = &http.Server{
		Addr:    transport.address,
		Handler: handler,
	}
	handler.transport = transport
}

// ChangeAddress changes the internet address where the
// transport service will listen. This change will take
// effect only when the transport service is restarted
func (transport *Transport) ChangeAddress(newAddress string) {
	transport.address = newAddress
	handler := handler{}
	server := &http.Server{
		Addr:    newAddress,
		Handler: &handler,
	}
	handler.transport = transport
	transport.server = server
}

// Send sends a message to `address` and returns a channel from
// which the response can be read (once)
func (transport *Transport) Send(address string, data []byte) (replyChan chan []byte) {
	client := &http.Client{}
	replyChan = make(chan []byte)

	if !strings.HasPrefix(address, "http://") {
		address = "http://" + address
	}

	// wait for response
	go func() {
		res, err := client.Post(address, "application/json", bytes.NewReader(data))
		if err != nil {
			fmt.Printf("transport: error: %+v\n", err)
			close(replyChan)
			return
		}
		defer res.Body.Close()
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			fmt.Printf("transport: error: %+v\n", err)
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
