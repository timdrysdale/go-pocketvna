package stream

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"../pocket"
	"../reconws"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

var upgrader = websocket.Upgrader{}

func TestRun(t *testing.T) {

	// Create test server with the echo handler.
	s := httptest.NewServer(http.HandlerFunc(reasonableRange))
	defer s.Close()

	// Convert http://127.0.0.1 to ws://127.0.0.
	u := "ws" + strings.TrimPrefix(s.URL, "http")

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	go Run(u, ctx)

	time.Sleep(2 * time.Second)

}

// TODO test the pipe functions

func TestPipeInterfaceToWs(t *testing.T) {
	timeout := 100 * time.Millisecond

	chanWs := make(chan reconws.WsMessage)
	chanInterface := make(chan interface{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go PipeInterfaceToWs(chanInterface, chanWs, ctx)

	/* Test ReasonableFrequencyRange */

	chanInterface <- pocket.ReasonableFrequencyRange{
		Command: pocket.Command{Command: "rr"}, Result: pocket.Range{Start: 100000, End: 4000000}}

	select {

	case <-time.After(timeout):
		t.Error("timeout awaiting response")
	case reply := <-chanWs:

		expected := "{\"id\":\"\",\"t\":0,\"cmd\":\"rr\",\"range\":{\"Start\":100000,\"End\":4000000}}"

		assert.Equal(t, expected, string(reply.Data))
	}

	/* Test SingleQuery */
	chanInterface <- pocket.SingleQuery{
		Command: pocket.Command{Command: "sq"},
		Freq:    100000,
		Avg:     1,
		Select:  pocket.SParamSelect{S11: true, S12: false, S21: true, S22: false},
		Result: pocket.SParam{
			S11: pocket.Complex{Real: -1, Imag: 2},
			S21: pocket.Complex{Real: 0.34, Imag: 0.12},
		},
	}

	select {

	case <-time.After(timeout):
		t.Error("timeout awaiting response")
	case reply := <-chanWs:

		expected := "{\"id\":\"\",\"t\":0,\"cmd\":\"sq\",\"freq\":100000,\"avg\":1,\"sparam\":{\"S11\":true,\"S12\":false,\"S21\":true,\"S22\":false},\"result\":{\"S11\":{\"Real\":-1,\"Imag\":2},\"S12\":{\"Real\":0,\"Imag\":0},\"S21\":{\"Real\":0.34,\"Imag\":0.12},\"S22\":{\"Real\":0,\"Imag\":0}}}"
		assert.Equal(t, expected, string(reply.Data))
	}

}

func TestPipeWsToInterface(t *testing.T) {
	timeout := 100 * time.Millisecond

	chanWs := make(chan reconws.WsMessage)
	chanInterface := make(chan interface{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go PipeWsToInterface(chanWs, chanInterface, ctx)

	mt := int(websocket.TextMessage)

	/* Test ReasonableFrequencyRange */
	message := []byte("{\"cmd\":\"rr\"}")

	ws := reconws.WsMessage{
		Data: message,
		Type: mt,
	}

	chanWs <- ws

	select {

	case <-time.After(timeout):
		t.Error("timeout awaiting response")
	case reply := <-chanInterface:
		assert.Equal(t, reflect.TypeOf(reply), reflect.TypeOf(pocket.ReasonableFrequencyRange{}))
		fmt.Println(reply)
	}

	/* Test SingleQuery */
	message = []byte("{\"cmd\":\"sq\",\"freq\":100000,\"avg\":1,\"sparam\":{\"S11\":true,\"S21\":true}}")

	ws = reconws.WsMessage{
		Data: message,
		Type: mt,
	}

	chanWs <- ws

	select {

	case <-time.After(timeout):
		t.Error("timeout awaiting response")
	case reply := <-chanInterface:
		assert.Equal(t, reflect.TypeOf(reply), reflect.TypeOf(pocket.SingleQuery{}))
		fmt.Println(reply)
	}

}

func reasonableRange(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer c.Close()

	mt := int(websocket.TextMessage)

	message := []byte("{\"cmd\":\"rr\"}")

	for {

		err = c.WriteMessage(mt, message)
		if err != nil {
			break
		}
		_, message, err := c.ReadMessage()
		if err != nil {
			break
		}

		fmt.Println("Hello!")
		fmt.Println(message)

	}
}