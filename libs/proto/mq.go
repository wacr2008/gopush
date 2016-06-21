package proto

import (
	"encoding/json"
)
 
// if you modify it . please run ffjson mq.go to update

// message queue 
type MQMsg struct {
	OP		int		`json:"op"`
	ServerId	[]int32	`json:"servers,omitempty"`
	Topic	int32		`json:"topic,omitempty"`
	UserId	[]int64	`json:"users,omitempty"`
	
	Msg		*PushMsg
}

// push message
type PushMsg struct {
	Id			int64		`json:",omitempty"`		// message id
	UserId		int64		`json:",omitempty"`		// user id
	Ensure		bool		`json:",omitempty"`		// if it is true, ensure message send to user
	Time			int64		`json:",omitempty"`		// 
	Expire		int64		`json:",omitempty"`		// expire time of message 
	RedoCount	int32		`json:",omitempty"`		// times of retry to  send
	RedoMax		int32		`json:",omitempty"`		// maximum times of retry to send
	Body			json.RawMessage				// message body
}

// response of ensure message
type PushMsgEnsureRES struct {
	Id		int64
	UserId	int64
}




