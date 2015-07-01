package common

import (
	"time"
)

type Message struct {
	ReplyChannel chan interface{}
	Request		 interface{}
}
func (msg *Message) Reply(response interface{}) {
	msg.ReplyChannel<-response
}
func (msg *Message) GetWithTimeout(timeout time.Duration) (reply interface{}, err error)  {
	select {
	case reply := <- msg.ReplyChannel:
		{
			return reply, nil
		}
	case <-time.After(timeout):
		{
			return nil, &TimeOutError{}
		}
	}
}

func (msg *Message) Get() (reply interface{}, err error) {
	reply = <- msg.ReplyChannel
	return reply, nil

}
