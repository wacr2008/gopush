package main

import (
	"net/rpc"
	"time"
	"strconv"
	log "github.com/ikenchina/golog"
	"github.com/stathat/consistent"
	"github.com/ikenchina/gopush/libs/proto"
)

var (
	adminRpcPing		=	"RPC.Ping"
	adminRpcSub		=	"RPC.Sub"
	adminRpcUnSub	=	"RPC.UnSub"
)

func NewAdminRpcs() *AdminRpcs {
	return new(AdminRpcs)
}

type AdminRpcs struct {
	adminRpc	map[string]*AdminRpc
	CHash		*consistent.Consistent
	Config		[]ConfigRpcAdmin
}

func (r *AdminRpcs) Init() (err error) {
	r.Config = Conf.Server.RpcAdmin
	r.CHash = consistent.New()
	r.adminRpc = make(map[string]*AdminRpc)
	for _, node := range r.Config {
		
		adminRpc := NewAdminRpc()
		if err = adminRpc.Init(node); err != nil {
			return 
		} else {
			r.adminRpc[node.Name] = adminRpc
			r.CHash.Add(node.Name)
		}
	}
	if len(r.adminRpc) == 0 {
		return ErrAdminRpc
	}
	return nil
}

func (r *AdminRpcs) UnInit() {
	for _, rpc := range r.adminRpc {
		rpc.Close()
	}
	for _,m := range r.CHash.Members() {
		r.CHash.Remove(m)
	}
	r.adminRpc = nil
}


// @TODO only update AdminRpc which need to update
func (r *AdminRpcs) UpdateConfig() {
	var err error
	config := Conf.Server.RpcAdmin
	rpcs := make(map[string]*AdminRpc, 0)

	// init admin rpc
	for _, node := range config {
		adminRpc := NewAdminRpc()
		if err = adminRpc.Init(node); err != nil {
			break
		} else {
			rpcs[node.Name] = adminRpc
		}
	}
	if err == nil && len(rpcs) != 0 {
		r.UnInit()

		r.Config = config
		r.adminRpc = rpcs
		for k, _ := range r.adminRpc {
			r.CHash.Add(k)
		}
	} else {
		log.Fatalf("admin rpc update config failed")
	}
}

func (r *AdminRpcs) Subscribe(a *proto.SubREQ) (err error) {
	var (
		uid string
		n string
	)
	uid = strconv.FormatInt(a.UserId, 10)
	n, err = r.CHash.Get(uid)
	if err != nil {
		return
	} 
	c, ok := r.adminRpc[n]
	if !ok {
		err = ErrInternal
		return
	}
	err = c.Subscribe(a)
	return
}

func (r *AdminRpcs) UnSubscribe(a *proto.UnSubREQ) (err error) {
	var (
		uid string
		n string
	)
	uid = strconv.FormatInt(a.UserId, 10)
	n, err = r.CHash.Get(uid)
	if err != nil {
		return
	} 
	c, ok := r.adminRpc[n]
	if !ok {
		err = ErrInternal
		return
	}
	
	var has bool
	if has, err = c.UnSubscribe(a); err != nil {
		log.Errorf("AdminRpc.UnSubscribe user(%d) to server(%d) error : %v", a.UserId, a.Server, err)
		return
	}
	if !has {
		log.Warnf("AdminRpc.UnSubscribe user: %v not exists", a.UserId)
	}
	return
}




// rpc client
func NewAdminRpc() *AdminRpc {
	return new(AdminRpc)
}

type AdminRpc struct {
	Client		*rpc.Client
	adminRpcQuit	chan struct{}
	Config		ConfigRpcAdmin
}

// init Rpc client 
func (r *AdminRpc) Init(conf ConfigRpcAdmin) (err error) {
	
	network := conf.Network
	addr := conf.Addr

	r.Client, err = rpc.Dial(network, addr)
	if err != nil || r.Client == nil {
		log.Fatalf("adminRpc.Dial(%s@%s) failed : %s", network, addr, err)
		panic(err)
	}
	r.adminRpcQuit  = make(chan struct{}, 1)
	go r.Keepalive(r.adminRpcQuit, network, addr)
	log.Infof("Init admin Rpc (%s@%s) sub successfuls", network, addr)
	return
}


func (r *AdminRpc) Close() {
	r.adminRpcQuit <- struct{}{}
}

// if comet crash : call http://admin/server?id=xx as DELETE method, and restart comet
// keep rpc available
func (r *AdminRpc) Keepalive(quit chan struct{}, network, address string) {
	if r.Client == nil {
		log.Fatalf("keepalive rpc with admin : rpc.Client is nil")
		panic(ErrAdminRpc)
	}
	var (
		err    error
		call   *rpc.Call
		done     = make(chan *rpc.Call, 1)
		args   = proto.NoREQ{}
		reply  = proto.PingRES{}
		firstPingTime int64
	)
	for {
		select {
			case <-quit:
				return
			default:
				if r.Client != nil {
					call = <-r.Client.Go(adminRpcPing, &args, &reply, done).Done
					if call.Error != nil {
						log.Errorf("adminRpc ping %s failed : %v", address, call.Error)
					} else {
						if firstPingTime == 0 {
							firstPingTime = reply.TimeUnix
						}
						// if firstPingTime difference : admin is crash. lost all user infomation
						if firstPingTime != reply.TimeUnix {
							// @TODO  client maybe disconnect while doing ReSubAll,
							// so, comet need to send all online users to admin later?
							err := CometServer.ReSubAll()
							if err == nil {
								firstPingTime = reply.TimeUnix
							}
						}
					}
				}
				 
				if r.Client == nil  || (call != nil && call.Error == rpc.ErrShutdown) {
					//if isDebug {
					//	log.Debugf("rpc.Dial (%s@%s) failed : %v", network, address, err)
					//}
					if r.Client, err = rpc.Dial(network, address); err == nil {
					} else {
						log.Errorf("adminRpc dial error : %v", err)
					}
				}
		}
		// @TODO : configuration heartbeat between comet and admin
		time.Sleep(5 * time.Second)
	}
}

// subscribe to admin
func (r *AdminRpc)Subscribe(arg *proto.SubREQ) (err error) {
	if r.Client == nil {
		err = ErrAdminRpc
		return
	}

	// @TODO memory pool
	reply := &proto.SubRES{}
	if err = r.Client.Call(adminRpcSub, arg, reply); err != nil {
		if isDebug {
			log.Errorf("adminRpc.Call(%s, %v) failed : %v", adminRpcSub, arg, err)
		}
		return
	}
	if !reply.Pass {
		err = ErrAuthFailed
		return
	}
	
	return
}

// unsubscribe to admin
func (r *AdminRpc)UnSubscribe(arg *proto.UnSubREQ) (has bool, err error) {
	if r.Client == nil {
		err = ErrAdminRpc
		return
	}
	reply := &proto.UnSubRES{}
	if err = r.Client.Call(adminRpcUnSub, arg, reply); err != nil {
		log.Errorf("adminRpc.Call(%s, %v) failed : %v", adminRpcUnSub, arg, err)
		return
	}
	has = reply.Has
	return
}
