package main_test

import (
	"bytes"
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/th3osmith/greader"
	"github.com/th3osmith/greader/pure"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
)

var buf bytes.Buffer

func TestListening(t *testing.T) {

	resp, err := http.Get("http://localhost:3000/test")
	if err != nil {
		t.Error(err)
	}

	if resp.StatusCode != http.StatusTeapot {
		t.Error("Error Setting/Getting the Status Code")
	}

	if output := buf.String(); !strings.Contains(output, "[GET] 418 \"/test\"") {
		t.Error("Invalid Log output", output)
	}
	buf.Reset()

	t.Logf(buf.String())

}

func TestPanicRecover(t *testing.T) {

	_, err := http.Get("http://localhost:3000/panic")
	if err != nil {
		t.Error(err)
	}
}

func TestAuth(t *testing.T) {

	resp, err := http.Get("http://localhost:3000/protected")
	if err != nil {
		t.Error(err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Error("Bad Status", resp.StatusCode)
	}

	client := &http.Client{}
	req, err := http.NewRequest("GET", "http://localhost:3000/protected", nil)
	req.SetBasicAuth("tata", "yoyo")

	reqA, err := http.NewRequest("GET", "http://localhost:3000/protected", nil)
	reqA.SetBasicAuth("toto", "yoyo")

	resp, err = client.Do(req)
	if err != nil {
		t.Error(err)
	}

	if resp.StatusCode != http.StatusTeapot {
		t.Error("Bad Status", resp.StatusCode)
	}

	resp, err = client.Do(reqA)
	if err != nil {
		t.Error(err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Error("Bad Status", resp.StatusCode)
	}

}

func TestWebSocket(t *testing.T) {

	u := url.URL{Scheme: "ws", Host: "localhost:3000", Path: "/websocket"}

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Error("dial:", err)
		return
	}
	defer c.Close()

	confirmation := make(chan bool)

	go func() {
		defer c.Close()

		for {
			_, _, err := c.ReadMessage()
			if err != nil {
				t.Error("Error Reading websocket:", err)
				confirmation <- false
				return
			}

			confirmation <- true
			return
		}

	}()

	err = c.WriteMessage(websocket.TextMessage, []byte("Coucou"))

	<-confirmation

}

func TestPureWebSocket(t *testing.T) {

	u := url.URL{Scheme: "ws", Host: "localhost:3000", Path: "/pure"}

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Error("dial:", err)
		return
	}
	defer c.Close()

	confirmation := make(chan bool)

	go func() {
		defer c.Close()

		for {
			_, p, err := c.ReadMessage()
			if err != nil {
				t.Error("Error Reading websocket:", err)
				confirmation <- false
				return
			}

			if string(p) != "{\"Action\":\"CREATED\",\"DataType\":\"data\",\"LogList\":null,\"RequestMap\":null,\"ResponseMap\":{},\"TransactionMap\":null}" {
				t.Error("Wrong response received: ", string(p))
			}

			confirmation <- true
			return
		}

	}()

	mm := make(map[string]interface{})
	mm["id"] = "tata"
	mm["value"] = "yoyo"

	// Creating Request
	req := pure.PureMsg{DataType: "data", Action: "create", RequestMap: mm}
	jsonReq, err := json.Marshal(req)

	if err != nil {
		t.Error(err)
	}

	err = c.WriteMessage(websocket.TextMessage, jsonReq)

	<-confirmation

}

func TestMain(m *testing.M) {

	// Test Setup
	go main.Serve()
	log.SetOutput(&buf)

	os.Exit(m.Run())

}
