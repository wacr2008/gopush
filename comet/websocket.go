package main

import (
	"crypto/tls"
	"math/rand"
	"net"
	"net/http"
	"time"
	"github.com/pquerna/ffjson/ffjson"
	log "github.com/ikenchina/golog"
	"golang.org/x/net/websocket"
	
	"gopush/libs/define"
	"gopush/libs/proto"
	itime "gopush/libs/time"
	
)

func (server *Server) InitWebsocket() (err error) {
	var (
		listener     *net.TCPListener
		addr         *net.TCPAddr
		httpServeMux = http.NewServeMux()
	)
	httpServeMux.Handle("/subscribe", websocket.Handler(server.subscribeWS))
	for _, bind := range server.Config.WebSocket.Bind {
		if addr, err = net.ResolveTCPAddr("tcp4", bind); err != nil {
			log.Errorf("net.ResolveTCPAddr(tcp4, %s) failed : %v", bind, err)
			return
		}
		if listener, err = net.ListenTCP("tcp4", addr); err != nil {
			log.Errorf("net.ListenTCP(tcp4, %s) failed : %v", bind, err)
			return
		}
		httpserver := &http.Server{Handler: httpServeMux}
		log.Infof("comet server start websocket listen: %s", bind)

		go func(bind string) {
			if err = httpserver.Serve(listener); err != nil {
				log.Errorf("httpserver.Serve(%s) failed : %v", bind, err)
				panic(err)
			}
		}(bind)
	}
	return
}

func (server *Server) InitTlsWebsocket(addrs []string, cert, priv string) (err error) {
	var (
		httpServeMux = http.NewServeMux()
	)
	httpServeMux.Handle("/subscribe", websocket.Handler(server.subscribeWS))
	config := &tls.Config{}
	config.Certificates = make([]tls.Certificate, 1)
	if config.Certificates[0], err = tls.LoadX509KeyPair(cert, priv); err != nil {
		return
	}
	for _, bind := range addrs {
		server := &http.Server{Addr: bind, Handler: httpServeMux}
		server.SetKeepAlivesEnabled(true)
		log.Infof("comet server start websocket TLS listen: %s", bind)

		go func(bind string) {
			ln, err := net.Listen("tcp", bind)
			if err != nil {
				return
			}

			tlsListener := tls.NewListener(ln, config)
			if err = server.Serve(tlsListener); err != nil {
				log.Errorf("server.Serve(%s) failed : %v", bind, err)
				return
			}
		}(bind)
	}
	return
}

func (server *Server) subscribeWS(conn *websocket.Conn) {
	var (
		err    error
		userid int64
		p      *proto.Proto
		b      *Bucket
		trd    *itime.TimerData
		ch     = NewChannel(server.Config.Bucket.Channel.RingBufferSize, server.Config.Bucket.Channel.PushBufferSize)
		topics []int32
		tr     = server.round.Timer(rand.Int())
		token string
	)

	//
	if p, err = ch.ProtoRing.Set(); err == nil {
		if userid, topics, token,  err = server.wsSubscribe(conn, p); err == nil {
			b = server.Bucket(userid)
			if tch := b.Channel(userid); tch != nil {
				conn.Close()
				return
			}
			err = b.Put(userid, ch)
		} else {
			log.Errorf("wsSubscribe failed user(%d) failed : %v", userid, err)
		}
	}
	if err != nil {
		conn.Close()
		log.Errorf("handshake failed : %v", err)
		return
	}

	// 
	trd = tr.Add(time.Duration(server.Config.HeartBeatTimeout), func() {
		conn.Close()
	})

	ch.SetTopics(topics)
	ch.SetToken(token)
	trd.Key = userid
	tr.Set(trd, time.Duration(server.Config.HeartBeatTimeout))

	go server.processWriteWebsocket(userid, conn, ch)
	for {
		if p, err = ch.ProtoRing.Set(); err != nil {
			break
		}
		if err = p.ReadWebsocket(conn); err != nil {
			break
		}

		if p.OpCode == define.OP_HEARTBEAT_REQ { 
			// @TODO heartbeat timeout need to close connection
			tr.Set(trd, time.Duration(server.Config.HeartBeatTimeout))
			p.Body = nil
			p.OpCode = define.OP_HEARTBEAT_RES
		}
		ch.ProtoRing.SetAdv()
		ch.Ready()
	}
	conn.Close()
	ch.Close()
	b.DelSafe(userid, ch)

	disArg := &proto.UnSubREQ{userid, server.Config.Id}
	if err = server.UnSubscribe(disArg); err != nil {
		log.Errorf("UnSubscribe user(%d)  failed : %v", userid, err)
	}
	if isDebug {
		//log.Debugf("userid: %d server websocket goroutine exit", userid)
	}
	return
}

func (server *Server) processWriteWebsocket(userid int64, conn *websocket.Conn, ch *Channel) {
	var (
		p   *proto.Proto
		err error
	)
	for {
		p = ch.GetProto()

		switch p {
			case proto.ProtoFinish:
				goto failed
			case proto.ProtoReady:
				for {
					if p, err = ch.ProtoRing.Get(); err != nil {
						break
					}
					
					if err = server.pushWebSocket(p, conn); err != nil {
						log.Debugf("server.pushWebSocket  failed : %v", err)
						goto failed
					}
					
					p.Body = nil
					ch.ProtoRing.GetAdv()
				}
			default:
				if err = server.pushWebSocket(p, conn); err != nil {
					log.Debugf("server.pushWebSocket  failed : %v", err)
					goto failed
				}
		}
	}
failed:
	conn.Close()
	ch.DiscardAllProto()
	if isDebug {
		//log.Debugf("userid: %v processWriteWebsocket goroutine exit", userid)
	}
	return
}


func (server *Server) pushWebSocket(p *proto.Proto, conn *websocket.Conn) (err error) {

	err = p.WriteWebsocket(conn); 
	//log.Debugln(p)
	if err == nil {
		if p.Ensure {
			// if failed : message will be send next time
			server.pushRpc.PushEnsure(p)
		}
	}
	return err	
}

func (server *Server) wsSubscribe(conn *websocket.Conn, p *proto.Proto) (userid int64, topics []int32, token string, err error) {
	if err = p.ReadWebsocket(conn); err != nil {
		return
	}
	if p.OpCode != define.OP_SUB_REQ {
		err = ErrOperation
		return
	}

	sub := new(proto.SubREQ)
	err = ffjson.Unmarshal(p.Body, sub)
	if err != nil {
		return
	}
	sub.Server = server.Config.Id
	if err = server.Subscribe(sub); err != nil {
		return
	}
	userid = sub.UserId
	topics = sub.Topics
	token = sub.Token
	p.Body = nil
	p.OpCode = define.OP_SUB_RES
	err = p.WriteWebsocket(conn)
	return
}
