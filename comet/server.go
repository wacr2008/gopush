package main

import (
	log "github.com/ikenchina/golog"
	"github.com/ikenchina/gopush/libs/proto"
)

type Server struct {
	Buckets		[]*Bucket
	round		*RoundRobin
	Config		ConfigServer	// copy it , @TODO reload config need to update Server.Config

	pushRpc		*PushRpcs
	adminRpc	*AdminRpcs
}

func NewServer() *Server {
	s := new(Server)
	return s
}

// @TODO
func (server *Server) UpdateConfig() {
	server.Config = Conf.Server
	
	server.adminRpc.UpdateConfig()
	server.pushRpc.UpdateConfig()
}

func (server *Server) Init() (err error) {
	hasListen := false

	server.Config = Conf.Server
	server.adminRpc = NewAdminRpcs()
	server.pushRpc = NewPushRpcs()

	server.Buckets = make([]*Bucket, server.Config.Bucket.Size)
	for i := 0; i < len(server.Buckets); i++ {
		server.Buckets[i] = NewBucket(server.Config.Bucket)
	}

	// init tcp protocol
	if len(server.Config.Tcp.Bind) > 0 {
		if err = server.InitTCP(); err != nil {
			return
		}
		hasListen = true
	}

	if len(server.Config.WebSocket.Bind) > 0 {
		// init websocket protocol
		if err = server.InitWebsocket(); err != nil {
			return
		}

		// TLS websocket
		if server.Config.WebSocket.Tls {
			if err = server.InitTlsWebsocket(server.Config.WebSocket.Bind, server.Config.WebSocket.TlsCertFile, server.Config.WebSocket.TlsPrivateFile); err != nil {
				return
			}
		}
		hasListen = true
	}

	if !hasListen {
		err = ErrServerNotListen
	} else {
		server.round = NewRoundRobin(server.Config.RoundRobin)

		// init admin rpc
		err = server.adminRpc.Init()
		if err != nil {
			return ErrAdminRpc
		}
		// init push rpc
		err = server.pushRpc.Init()
		if err != nil {
			return ErrPushRpc
		}
		
		Conf.RegisterUpdateNotify(server.UpdateConfig)
	}

	return
}

func (server *Server) UnInit() {
	server.adminRpc.UnInit()
	server.pushRpc.UnInit()
}

func (server *Server) Subscribe(r *proto.SubREQ) (err error) {
	return server.adminRpc.Subscribe(r)
}

func (server *Server) UnSubscribe(r *proto.UnSubREQ) (err error) {
	return server.adminRpc.UnSubscribe(r)
}

// resubscribe all users to admin process
// senario : admin crashed, resubscribe
// @TODO admin stateless :  admin save all information to storage?
func (server *Server) ReSubAll() (err error) {
	log.Infof("CometServer.ReSubAll start...")
	
	for _, bk := range server.Buckets {
		for uid, ch := range bk.channels {

			subreq := &proto.SubREQ{
				UserId: uid,
				Token:  ch.Token,
				Topics: ch.GetTopics(),
				Server: server.Config.Id,
			}
			// @TODO reset hearbeat
			err = server.Subscribe(subreq)
			if err != nil {
				log.Errorln(err)
				break
			}
		}
	}
	log.Infof("CometServer.ReSubAll finish...")
	return
}

func (server *Server) Bucket(userid int64) *Bucket {
	idx := userid % int64(server.Config.Bucket.Size)
	return server.Buckets[idx]
}
