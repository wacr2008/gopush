package main

import (
	"encoding/json"
	"time"
	//log "github.com/ikenchina/golog"
)


const (
	rawHeaderLen = int16(16)
)

const (
	ProtoTCP          = 0
	ProtoWebsocket    = 1
	ProtoWebsocketTLS = 2
)

type Proto struct {
	HeaderLen int16           `json:"-"`  
	Ver       int16           `json:"ver"` 
	Operation int32           `json:"op"`   
	MsgId     int64           `json:"msgid"`  
	Body      json.RawMessage `json:"body"` 
	Time      time.Time       `json:"-"`  
}

