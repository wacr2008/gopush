package main

import (
	"net/rpc"
	"time"
	log "github.com/ikenchina/golog"
	"gopush/libs/proto"
)

var (
	pushRpcPing		=	"RPC.Ping"
	pushRpcPushEnsure	=	"RPC.PushEnsure"
)

func NewPushRpcs() *PushRpcs {
	return new(PushRpcs)
}

type PushRpcs struct {
	pushRpc	[]*PushRpc
	Config		[]ConfigRpcPush
}


func (r *PushRpcs) Init() (err error) {
	r.Config = Conf.Server.RpcPush
	r.pushRpc = make([]*PushRpc, 0)
	for _, node := range r.Config {
		rpc := NewPushRpc()
		if err = rpc.Init(node); err != nil {
			log.Errorf("init push rpc failed : %v", err)
			panic(err)
		} else {
			r.pushRpc = append(r.pushRpc, rpc)
		}
	}
	// if len(r.pushRpc) == 0 {
	// 	log.Errorln(r.Config)
	// 	return ErrPushRpc
	// }
	return nil
}

func (r *PushRpcs) UnInit() {
	for _, rr := range r.pushRpc {
		rr.Close()
	}
	r.pushRpc = nil
}

func (r *PushRpcs) UpdateConfig() {
	var err error
	config := Conf.Server.RpcPush
	pushrpc := make([]*PushRpc, 0)
	for _, node := range config {
		rpc := NewPushRpc()
		if err = rpc.Init(node); err != nil {
			continue
		} else {
			pushrpc = append(pushrpc, rpc)
		}
	}
	if err == nil && len(pushrpc) != 0 {
		for _, rpc := range r.pushRpc {
			rpc.Close()
		}
		r.Config = config
		r.pushRpc = pushrpc
	} else {
		log.Fatalf("push rpc update config failed")
	}
}

func (r *PushRpcs) PushEnsure(p *proto.Proto) (err error) {
	length := len(r.pushRpc)
	if length == 0 {
		return ErrPushRpc
	}
	idx := p.UserId % int64(length)

	pp := &proto.PushMsgEnsureRES{p.MsgId, p.UserId}
	return r.pushRpc[idx].PushEnsure(pp)
}



func NewPushRpc() *PushRpc {
	return new(PushRpc)
}

type PushRpc struct {
	Client	*rpc.Client
	rpcQuit	chan struct{}
	Config	ConfigRpcPush
}

func (r *PushRpc) Init(conf ConfigRpcPush) (err error) {
	
	network := conf.Network
	addr := conf.Addr

	r.Client, err = rpc.Dial(network, addr)
	if err != nil || r.Client == nil {
		log.Errorf("pushRpc.Dial(%s@%s) failed : %s", network, addr, err)
		//panic(err)
	}
	r.rpcQuit  = make(chan struct{}, 1)
	go r.Keepalive(r.rpcQuit, network, addr)
	log.Infof("Init push Rpc (%s@%s) sub successfuls", network, addr)
	return
}


func (r *PushRpc) Close() {
	r.rpcQuit <- struct{}{}
}

func (r *PushRpc) Keepalive(quit chan struct{}, network, address string) {
	//if r.Client == nil {
	//	log.Errorf("keepalive rpc with push : rpc.Client is nil")
	//	//panic(ErrPushRpc)
	//}
	var (
		err    error
		call   *rpc.Call
		done     = make(chan *rpc.Call, 1)
		args   = proto.NoREQ{}
		reply  = proto.PingRES{}
		//firstPingTime int64
	)
	for {
		select {
			case <-quit:
				return
			default:
				if r.Client != nil {
					call = <-r.Client.Go(pushRpcPing, &args, &reply, done).Done
					if call.Error != nil {
						log.Errorf("pushRpc ping %s failed : %v", address, call.Error)
					} 
				}
				 
				if (call != nil && call.Error == rpc.ErrShutdown) || r.Client == nil {
					//if isDebug {
					//	log.Debugf("rpc.Dial (%s@%s) failed : %v", network, address, err)
					//}
					if r.Client, err = rpc.Dial(network, address); err == nil {
						// @TODO Dial other pushRpc server : but must to avoid avalanche effect
					} else {
						log.Errorf("pushRpc dial error : %v", err)
					}
				}
		}
		// @TODO : configuration heartbeat between comet and push
		time.Sleep(5 * time.Second)
	}
}


func (r *PushRpc)PushEnsure(arg *proto.PushMsgEnsureRES) (err error) {
	if r.Client == nil {
		err = ErrAdminRpc
		return
	}

	// @TODO message queue
	// @TODO memory pool
	reply := &proto.NoRES{}
	if err = r.Client.Call(pushRpcPushEnsure, arg, reply); err != nil {
		//if isDebug {
		//	log.Errorf("pushRpc.Call(%s, %v) failed : %v", pushRpcPushEnsure, arg, err)
		//}
		return
	}
	
	return
}