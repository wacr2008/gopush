package main

import (
	"time"
	. "github.com/ikenchina/gopush/libs/define"
	log "github.com/ikenchina/golog"
	"golang.org/x/net/websocket"
	"fmt"
	"sync/atomic"
)

var (
	msgCount int64
	errCount int64
)


func initWebsocket(userid int64, topics []int32) {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("%v\n", err)
		}	
	}()
	var (
		err error
	)
	origin := "http://" + Conf.Protocol.Addr + "/subscribe"
	url := "ws://" + Conf.Protocol.Addr + "/subscribe"
	conn, err := websocket.Dial(url, "", origin)
	//log.Debugf("dial : %v", url)

	if err != nil {
		log.Errorf("websocket.Dial(\"%s\") error(%v)", Conf.Protocol.Addr, err)
		return
	}

	proto := new(Proto)
	proto.Ver = 1
	
	proto.Operation = OP_SUB_REQ
	msgId := int64(0)
	proto.MsgId = msgId

	topicstring := ""
	for i, t := range topics {
		if i != 0 {
			topicstring = fmt.Sprintf("%s,%d", topicstring, t)
		} else {
			topicstring = fmt.Sprintf("%d",  t)
		}
		
	}
	bodystr := fmt.Sprintf(`{"userid":%v, "token":"%v", "topics":[%v]}`, userid, userid, topicstring)
	proto.Body = []byte(bodystr)
	
	//log.Debugf("body : %s\n", string(proto.Body))
	
	
	if err = websocketWriteProto(conn, proto); err != nil {
		log.Errorf("websocketWriteProto() error(%v)", err)
		return
	}
	if err = websocketReadProto(conn, proto); err != nil {
		log.Errorf("websocketReadProto() error(%v)", err)
		return
	}
	//log.Debugf("auth ok, proto: %v", proto)
	msgId++
	// writer
	go func() {
		proto1 := new(Proto)
		for {
			// heartbeat
			proto1.Operation = OP_HEARTBEAT_REQ
			proto1.MsgId = msgId
			proto1.Body = nil
			if err = websocketWriteProto(conn, proto1); err != nil {
				log.Errorf("tcpWriteProto() error(%v)", err)
				return
			}
			msgId++
			time.Sleep(100 * time.Second)
		}
	}()
	// reader
	st := time.Now()
	et := time.Now()
	for {
		if err = websocketReadProto(conn, proto); err != nil {
			//if errCount % 100 == 0 {
				log.Errorf("wsReadProto(%d) error(%v)", errCount, err)
			//}
			atomic.AddInt64(&errCount, 1)
			return
		}
		
		if proto.Operation == OP_HEARTBEAT_RES {
			if err = conn.SetReadDeadline(time.Now().Add(10 * time.Hour)); err != nil {
				log.Errorf("conn.SetReadDeadline() error(%v)", err)
				return
			}
		} else {
			if msgCount %10000 == 0 {
				et = time.Now()
				log.Debugf("-->%d----->%v", msgCount, et.Sub(st))
				st = time.Now()
				
			}
atomic.AddInt64(&msgCount, 1)
		}
	}
}

func websocketReadProto(conn *websocket.Conn, p *Proto) error {
	return websocket.JSON.Receive(conn, p)
}

func websocketWriteProto(conn *websocket.Conn, p *Proto) error {
	if p.Body == nil {
		p.Body = []byte("{}")
	}
	return websocket.JSON.Send(conn, p)
}
