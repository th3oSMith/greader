package pure_test

import (
	"fmt"
	"github.com/th3osmith/greader/pure"
	"testing"
)

type MyHandler struct {
	data  map[string]interface{}
	conns []pure.PureConnection
	owner map[string]pure.PureConnection
}

func NewPureHandler() MyHandler {
	return MyHandler{data: make(map[string]interface{}), owner: make(map[string]pure.PureConnection)}

}

func (h MyHandler) Create(m pure.PureReq, rw pure.ResponseWriter) {

	fmt.Println("Create")
	rww := rw.(*pure.PureResponseWriter)
	msg := m.Msg
	id := msg.RequestMap["id"].(string)

	if _, ok := h.data[id]; ok {
		rww.Fail()
		return
	}

	h.data[id] = msg.RequestMap["value"]
	h.owner[id] = m.Conn

	return

}
func (h MyHandler) Update(m pure.PureReq) {
	return

}
func (h MyHandler) Delete(m pure.PureReq) {
	return

}
func (h MyHandler) Retrieve(m pure.PureReq, rw pure.ResponseWriter) {

	fmt.Println("Retrieve")
	rww := rw.(*pure.PureResponseWriter)

	msg := m.Msg
	id := msg.RequestMap["id"].(string)

	vData, ok := h.data[id]

	if !ok {
		rww.Fail()
		return
	}

	rww.AddValue("data", vData)

	// Send the response to the Owner too
	owner, ok := h.owner[id]

	if ok {
		rw.AddConn(owner)
	}

	return

}
func (h MyHandler) Flush(m pure.PureReq) {
	return

}

func TestPureMux(t *testing.T) {

	mux := pure.NewPureMux()
	h := MyHandler{data: make(map[string]interface{}), owner: make(map[string]pure.PureConnection)}
	mux.RegisterHandler("data", h)

	c1 := pure.GoConn{Response: make(chan pure.PureMsg, 1), Muxer: mux}
	c2 := pure.GoConn{Response: make(chan pure.PureMsg, 1), Muxer: mux}

	mm := make(map[string]interface{})
	mm["id"] = "tata"
	mm["value"] = "yoyo"

	c1.SendReq(pure.PureMsg{DataType: "data", Action: "create", RequestMap: mm})
	resp := c1.ReadResp()

	if resp.Action != "CREATED" {
		t.Error("Create failed", resp)
	}

	c1.SendReq(pure.PureMsg{DataType: "data", Action: "create", RequestMap: mm})
	resp = c1.ReadResp()

	if resp.Action != "CREATE_FAIL" {
		t.Error("Create succeeded", resp)
	}

	c2.SendReq(pure.PureMsg{DataType: "data", Action: "retrieve", RequestMap: mm})
	resp = c2.ReadResp()

	if resp.ResponseMap["data"] != "yoyo" {
		t.Error("Retrieve Fail", resp)
	}

	resp = c1.ReadResp()
	if resp.ResponseMap["data"] != "yoyo" {
		t.Error("Broadcast Fail", resp)
	}

}
