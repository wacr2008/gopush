package main

import (
	"net"
	"net/rpc"
	"time"
	log "github.com/ikenchina/golog"
	"github.com/ikenchina/gopush/libs/proto"
)


type RPC struct {
	Config	ConfigRPC
	TimeUnix	int64
	notifyOnline	*OnlineNotify
}

func InitRpc() (err error) {
	r := &RPC{Config : Conf.RPC, TimeUnix : time.Now().UnixNano(),}
	
	rpc.Register(r)
	for _, b := range r.Config.Bind {
		go r.rpcListen(b.Network, b.Addr)
	}
	
	r.notifyOnline = NewOnlineNotify()
	
	return
}

func UnInitRpc() {
	
}

func (r *RPC)rpcListen(network, addr string) {
	l, err := net.Listen(network, addr)
	if err != nil {
		log.Fatalf("adminRpc listen %s@%s failed : %v", network, addr, err)
		panic(err)
	}
	// if process exit, then close the rpc bind
	defer func() {
		log.Infof("adminRpc addr: %s close", addr)
		if err := l.Close(); err != nil {
			log.Errorf("adminRpc close failed : %v", err)
		}
	}()
	rpc.Accept(l)
}


func (r *RPC) bucket(userId int64) *Bucket {
	return DefaultServer.GetBucket(userId)
}

func (r *RPC) Ping(arg *proto.NoREQ, reply *proto.PingRES) error {
	reply.TimeUnix = r.TimeUnix
	return nil
}


// authorize and subscribe
func (r *RPC) Sub(arg *proto.SubREQ, reply *proto.SubRES) (err error) {
	if arg == nil {
		err = ErrSubREQ
		log.Errorf("sub request failed : %v", err)
		return
	}
	reply.Pass = DefaultServer.Auther.Auth(arg.UserId, arg.Token)
	
	if reply.Pass {
		err = DefaultServer.Sub(arg.UserId, arg.Server, arg.Topics)
		if err == nil {
			// notify user on line
			r.notifyOnline.Notify(arg.UserId)
		}
	}
	return
}

// unsubscribe
func (r *RPC) UnSub(arg *proto.UnSubREQ, res *proto.UnSubRES) (err error) {
	if arg == nil {
		err = ErrUnSubREQ
		log.Errorf("unsub failed : %v", err)
		return
	}
	err = DefaultServer.UnSub(arg.UserId, arg.Server)
	res.Has = true
	if err == ErrUserNotExist {
		res.Has = false
	}  else if err != nil {
		log.Debugf("UnSub failed : %v", err)
	}
	
	return
}
