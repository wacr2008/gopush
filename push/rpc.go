package main

import (
	"net"
	"net/rpc"
	log "github.com/ikenchina/golog"
	"github.com/ikenchina/gopush/libs/proto"
)

func InitRPC() (err error) {
	conf := Conf.RPC
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
// @TODO : don't update address of rpc.Listen now.
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

// @TODO : if comet has not call this api, the message will be send to user next time, (so it is duplicate message)
func (this *RPC) PushEnsure(arg *proto.PushMsgEnsureRES, reply *proto.NoRES) (err error) {
	//log.Debugf("push ensure received")
	//log.Debugf("pushensure : %d", arg.Id)
	if arg == nil {
		err = ErrInvalidMsg
		return
	}
	
	err = MsgStorage.DelMsg(arg.UserId, arg.Id)
	
	return
}
