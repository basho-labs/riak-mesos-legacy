package common

/* Some OTP goodness. Right now, this is just a "framework" but we can make it more self-contained a la
   OTP's gen_server.
*/

type TimeOutError struct {
}

func (e *TimeOutError) Error() string {
	return "timeout"
}
type Cast struct {
	Request interface{}
}
type GenServer struct {
	Messages chan Message
	Casts    chan Cast
}

func NewGenServerWithQlen(qlen int) GenServer {
	genserver := GenServer{
		Messages: make(chan Message, qlen),
		Casts:    make(chan Cast, qlen),
	}
	return genserver
}
func NewGenServer() GenServer {
	return NewGenServerWithQlen(1)
}

func (genServer *GenServer) Cast(request interface{}) {
	msg := Cast{request}
	genServer.Casts <- msg
}

func (genServer *GenServer) Call(request interface{}) *Message {
	msg := Message{ReplyChannel: make(chan interface{}, 1), Request: request}
	genServer.Messages <- msg
	return &msg
}
