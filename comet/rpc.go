package main

import (
	"net"
	"net/rpc"
	"time"
	log "github.com/ikenchina/golog"
	"github.com/ikenchina/gopush/libs/proto"
	"github.com/ikenchina/gopush/libs/define"
)

func InitRPC() (err error) {
	conf := Conf.Rpc
	c  := &RPC{}
	rpc.Register(c)
	for _, cc := range conf {
		go rpcListen(cc.Network, cc.Addr)
	}
	return
}

//@TODO
func UnInitRPC() {
	
}


// update configuration
// @TODO : 
func UpdateRpcConfig() {
//	c  := &PushRPC{}
//	rpc.Register(c)
}

func rpcListen(network, addr string) {
	l, err := net.Listen(network, addr)
	if err != nil {
		log.Fatalf("Rpc listen(%s, %s) failed(%v)", network, addr, err)
		panic(err)
	}
	log.Infof("Rpc listen(%s@%s) successful", network, addr)
	defer func() {
		log.Infof("net.Listen(%s, %s)  closed", network, addr)
		if err := l.Close(); err != nil {
			log.Errorf("Rpc Close() failed(%v)", err)
		}
	}()
	
	rpc.Accept(l)
}


//   RPC server
type RPC struct {
}

func (this *RPC) Ping(arg *proto.NoREQ, reply *proto.NoRES) error {
	return nil
}


func (this *RPC) PushMsg(arg *proto.PushSMsgREQ, reply *proto.NoRES) (err error) {
	var bucket  *Bucket
	if arg == nil {
		err = ErrCometRpcMsgREQ
		return
	}
	
	//log.Debugln(arg.Msg)
	bucket = CometServer.Bucket(arg.Msg.UserId)
	p := &proto.Proto{
		Ver : proto.ProtocolVersion,
		OpCode : define.OP_RAW_MSG,
		MsgId : arg.Msg.Id,
		UserId : arg.Msg.UserId,
		Ensure : arg.Msg.Ensure,
		Body : arg.Msg.Body,
		Time : time.Now(),
	}
	err = bucket.Push(arg.Msg.UserId, p)
	return
}


func (this *RPC) MPushMsg(arg *proto.PushMMsgREQ, reply *proto.NoRES) (err error) {
	var (
		bucket  *Bucket
		user     int64
	)

	if arg == nil {
		err = ErrMCometRpcMsgREQ
		return
	}
	go func() {
		for _, user = range arg.UserId {
			bucket = CometServer.Bucket(user)
			p := &proto.Proto{
				Ver : proto.ProtocolVersion,
				OpCode : define.OP_RAW_MSG,
				MsgId : arg.Msg.Id,
				UserId : user,
				Ensure : arg.Msg.Ensure,
				Body : arg.Msg.Body,
				Time : time.Now(),
			}
			bucket.Push(user, p)
		}
	}()

	return
}


func (this *RPC) Broadcast(arg *proto.PushBroadcastREQ, reply *proto.NoRES) (err error) {
	var bucket *Bucket
	p := &proto.Proto{
		Ver : proto.ProtocolVersion,
		OpCode : define.OP_RAW_MSG,
		MsgId : arg.Msg.Id,
		Ensure : arg.Msg.Ensure,
		Body : arg.Msg.Body,
		Time : time.Now(),
	}
	for _, bucket = range CometServer.Buckets {
		go bucket.Broadcast(p)
	}
	return
}

func (this *RPC) BroadcastTopic(arg *proto.PushBroadcastTopicREQ, reply *proto.NoRES) (err error) {
	
	var bucket *Bucket
	p := &proto.Proto{
		Ver : proto.ProtocolVersion,
		OpCode : define.OP_RAW_MSG,
		MsgId : arg.Msg.Id,
		Ensure : arg.Msg.Ensure,
		Body : arg.Msg.Body,
		Time : time.Now(),
	}
	for _, bucket = range CometServer.Buckets {
		go bucket.BroadcastTopic(arg.Topic, p)
	}
	return
	
}



