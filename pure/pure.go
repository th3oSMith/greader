package pure

import (
	"fmt"
)

const (
	Error = iota
	Debug = iota
)

type LogMessage struct {
	Level   int
	Id      int
	Message string
}

type PureMsg struct {
	Action         string
	DataType       string
	LogList        []LogMessage
	RequestMap     map[string]interface{}
	ResponseMap    map[string]interface{}
	TransactionMap map[string]string
}

// Can implement user check
type PureConnection interface {
	Send(msg PureMsg)
	Handle(msg PureMsg)
}

// msg can be nil if resp to no request
type PureReq struct {
	Msg  PureMsg
	Conn PureConnection
}

// The handler can implement PUSH by keeping track of the owner of data
type PureHandler interface {
	Create(PureReq, ResponseWriter)
	Retrieve(PureReq, ResponseWriter)
	Update(PureReq)
	Delete(PureReq)
	Flush(PureReq)
}

type PureMux struct {
	handlers map[string]PureHandler
}

func (p *PureMux) Handle(m PureReq) {

	handler, ok := p.handlers[m.Msg.DataType]

	if !ok {
		fmt.Println("Error Handler not Found")
		return
	}

	rw := &PureResponseWriter{msg: &PureMsg{ResponseMap: make(map[string]interface{})}}
	rw.connections = append(rw.connections, m.Conn)
	rw.msg.TransactionMap = m.Msg.TransactionMap
	rw.msg.DataType = m.Msg.DataType
	rw.msg.Action = m.Msg.Action
	rw.success = true

	if m.Msg.Action == "create" {
		handler.Create(m, rw)

	} else if m.Msg.Action == "retrieve" {
		handler.Retrieve(m, rw)
	}

	for _, conn := range rw.Conns() {
		conn.Send(rw.GetMsg())
	}

}

func NewPureMux() *PureMux {
	return &PureMux{handlers: make(map[string]PureHandler)}
}

func (p *PureMux) RegisterHandler(dataType string, handler PureHandler) {
	p.handlers[dataType] = handler
}

type GoConn struct {
	Response chan PureMsg
	Muxer    *PureMux
}

func (c *GoConn) Send(msg PureMsg) {
	fmt.Println("Send")
	c.Response <- msg
}

func (c *GoConn) Handle(msg PureMsg) {
	fmt.Println("Handle")
	req := PureReq{Msg: msg, Conn: c}
	c.Muxer.Handle(req)
}

func (c *GoConn) SendReq(msg PureMsg) {
	c.Handle(msg)
}

func (c *GoConn) ReadResp() PureMsg {
	fmt.Println("Read")
	return <-c.Response
}

type ResponseWriter interface {
	AddConn(PureConnection) // Add a destination for the response
	GetMsg() PureMsg
	Conns() []PureConnection
}

type PureResponseWriter struct {
	msg         *PureMsg
	connections []PureConnection
	success     bool
}

func (rw *PureResponseWriter) AddConn(conn PureConnection) {
	rw.connections = append(rw.connections, conn)
}

func (rw *PureResponseWriter) Conns() []PureConnection {
	return rw.connections
}

func (rw *PureResponseWriter) GetMsg() PureMsg {
	rw.msg.Action = GetResponseAction(rw.msg.Action, rw.success)
	return *rw.msg
}

func (rw *PureResponseWriter) AddValue(key string, value interface{}) {
	rw.msg.ResponseMap[key] = value
}

func (rw *PureResponseWriter) Fail() {
	rw.success = false
}

var ResponseAction = map[bool]map[string]string{
	true: map[string]string{
		"create":  "CREATED",
		"delete":  "DELETED",
		"update":  "UPDATED",
		"retrive": "RETRIEVED",
		"flush":   "FLUSHED",
	},
	false: map[string]string{
		"create":  "CREATE_FAIL",
		"delete":  "DELETE_FAIL",
		"update":  "UPDATE_FAIL",
		"retrive": "RETRIEVE_FAIL",
		"flush":   "FLUSHED_FAIL",
	},
}

func GetResponseAction(requestAction string, success bool) string {
	return ResponseAction[success][requestAction]
}
