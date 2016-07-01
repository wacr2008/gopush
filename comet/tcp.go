package main

import (
	
	"io"
	"net"
	"time"
	"github.com/pquerna/ffjson/ffjson"
	log "github.com/ikenchina/golog"
	
	"gopush/libs/bufio"
	"gopush/libs/bytes"
	"gopush/libs/define"
	"gopush/libs/proto"
	itime "gopush/libs/time"
)


// config
/*
"tcp" : {
	"bind" : ["0.0.0.0:8080"],
	"sndbuf" : 2048,
	"rcvbuf" : 256,
	"keepalive" : true
},
*/
		
// InitTCP listen all tcp.bind and start accept connections.
func (server *Server)InitTCP() (err error) {
	var (
		bind		string
		listener	*net.TCPListener
		addr		*net.TCPAddr
	)
	for _, bind = range server.Config.Tcp.Bind {
		if addr, err = net.ResolveTCPAddr("tcp4", bind); err != nil {
			log.Errorf("net.ResolveTCPAddr(tcp4, %s) error(%v)", bind, err)
			return
		}
		if listener, err = net.ListenTCP("tcp4", addr); err != nil {
			log.Errorf("net.ListenTCP(tcp4, %s) error(%v)", bind, err)
			return
		}
		log.Infof("comet start listen tcp : %s", bind)
		
		for i := 0; i < Conf.Common.MaxProc; i++ {
			go server.acceptTCP(listener)
		}
	}
	return
}


func (server *Server) acceptTCP(lis *net.TCPListener) {
	var (
		conn *net.TCPConn
		err  error
		r    int
	)
	for {
		if conn, err = lis.AcceptTCP(); err != nil {
			// if listener close then return
			log.Errorf("listener.Accept address(%s) error : %v", lis.Addr().String(), err)
			return
		}
		if err = conn.SetKeepAlive(server.Config.Tcp.KeepAlive); err != nil {
			log.Errorf("conn.SetKeepAlive() error : %v", err)
			return
		}
		if err = conn.SetReadBuffer(server.Config.Tcp.RcvBuf); err != nil {
			log.Errorf("conn.SetReadBuffer() error : %v", err)
			return
		}
		if err = conn.SetWriteBuffer(server.Config.Tcp.SndBuf); err != nil {
			log.Errorf("conn.SetWriteBuffer() error : %v", err)
			return
		}
		go server.serveTCP(conn, r)
		if r++; r < 0 {
			r = 0
		}
	}
}


func (server *Server) serveTCP(conn *net.TCPConn, r int) {

	var (
		tr = server.round.Timer(r)
		rp = server.round.Reader(r)
		wp = server.round.Writer(r)
		err error
		userid int64
		p   *proto.Proto
		b   *Bucket
		trd *itime.TimerData
		rb  = rp.Get()
		wb  = wp.Get()
		ch  = NewChannel(server.Config.Bucket.Channel.RingBufferSize, server.Config.Bucket.Channel.PushBufferSize)
		rr  = &ch.Reader
		wr  = &ch.Writer
		topics []int32
		token string
	)
	rr.ResetBuffer(conn, rb.Bytes())
	wr.ResetBuffer(conn, wb.Bytes())

	// subscribe to admin
	if p, err = ch.ProtoRing.Set(); err == nil {
		if userid, topics, token, err = server.tcpSubscribe(rr, wr, p); err == nil {
			b = server.Bucket(userid)
			// if already register, close connection.
			if tch := b.Channel(userid); tch != nil {
				conn.Close()
				return
			}
			err = b.Put(userid, ch)
		}
	}
	if err != nil {
		conn.Close()
		rp.Put(rb)
		wp.Put(wb)
		log.Errorf("userid: %d handshake failed error(%v)", userid, err)
		return
	}
	
	trd = tr.Add(time.Duration(server.Config.HeartBeatTimeout), func() {
		conn.Close()
	})

	ch.SetTopics(topics)
	ch.SetToken(token)
	trd.Key = userid
	tr.Set(trd, time.Duration(server.Config.HeartBeatTimeout))
	
	go server.processWriteTcp(userid, conn, wr, wp, wb, ch)
	for {
		if p, err = ch.ProtoRing.Set(); err != nil {
			break
		}
		if err = p.ReadTCP(rr); err != nil {
			break
		}
		if p.OpCode == define.OP_HEARTBEAT_REQ {
			tr.Set(trd, time.Duration(server.Config.HeartBeatTimeout))
			p.Body = nil
			p.OpCode = define.OP_HEARTBEAT_RES
			if isDebug {
				log.Debugf("userid: %v receive heartbeat", userid)
			}
		}
		ch.ProtoRing.SetAdv()
		ch.Ready()
	}
	if err != nil && err != io.EOF {
		log.Errorf("userid: %d server tcp failed : %v", userid, err)
	}
	b.DelSafe(userid, ch)
	tr.Del(trd)
	rp.Put(rb)
	conn.Close()
	ch.Close()
	disArg := &proto.UnSubREQ{userid,  server.Config.Id}
	if err = server.UnSubscribe(disArg); err != nil {
		log.Errorf("unsub user(%d) failed : %v", userid, err)
	}
	if isDebug {
		log.Debugf("userid: %d server tcp goroutine exit", userid)
	}
	return
}


func (server *Server) processWriteTcp(userid int64, conn *net.TCPConn, wr *bufio.Writer, wp *bytes.Pool, wb *bytes.Buffer, ch *Channel) {
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
						err = nil	// must be empty error
						break
					}
					if err = p.WriteTCP(wr); err != nil {
						goto failed
					}
					p.Body = nil // avoid memory leak
					ch.ProtoRing.GetAdv()
				}
			default:
				if err = p.WriteTCP(wr); err != nil {
					goto failed
				}
		}
		// only hungry flush response
		if err = wr.Flush(); err != nil {
			break
		}
	}
failed:
	if err != nil {
		log.Errorf("userid: %v dispatch tcp error(%v)", userid, err)
	}
	conn.Close()
	wp.Put(wb)
	ch.DiscardAllProto()
	
	if isDebug {
		log.Debugf("userid: %v dispatch goroutine exit", userid)
	}
	return
}

// auth
func (server *Server) tcpSubscribe(rr *bufio.Reader, wr *bufio.Writer, p *proto.Proto) (userid int64, topics []int32, token string, err error) {
	if err = p.ReadTCP(rr); err != nil {
		return
	}
	if p.OpCode != define.OP_SUB_REQ {
		log.Warnf("register user operation not valid: %d", p.OpCode)
		err = ErrOperation
		return
	}
	
	arg := new(proto.SubREQ)
	err = ffjson.Unmarshal(p.Body, arg)
	if err != nil {
		return
	}
	
	if err = server.Subscribe(arg); err != nil {
		return
	}
	// outer must be copy it 
	userid = arg.UserId
	topics = arg.Topics
	token = arg.Token
	p.Body = nil
	p.OpCode = define.OP_SUB_RES
	if err = p.WriteTCP(wr); err != nil {
		return
	}
	err = wr.Flush()
	return
}
