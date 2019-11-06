package transport

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTransport(t *testing.T) {
	a1 := "localhost:10123"
	a2 := "localhost:10124"
	t1 := New(a1)
	t2 := New(a2)

	// actions in t1
	go func() {
		replyChanFromT2 := t1.Send("http://"+a2+"/endpoint", []byte("Some message string here"))
		replyFromT2 := <-replyChanFromT2
		assert.Equal(t, "Some response", string(replyFromT2))
	}()

	// actions in t2
	go func() {
		msgInT2 := <-t2.ReceiverChan()
		assert.Equal(t, "Some message string here", string(msgInT2.Data))
		assert.Equal(t, "/endpoint", msgInT2.Endpoint)
		msgInT2.ReplyChan <- []byte("Some response")
	}()

}
