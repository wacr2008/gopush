package main

import (
	log "github.com/ikenchina/golog"
	"github.com/pquerna/ffjson/ffjson"
	"math/rand"
	"net/rpc"
	"time"

	"gopush/libs/define"
	"gopush/libs/proto"
)

const (
	CometService               = "RPC"
	CometServicePing           = "RPC.Ping"
	CometServiceSPushMsg       = "RPC.PushMsg"
	CometServiceMPushMsg       = "RPC.MPushMsg"
	CometServiceBroadcast      = "RPC.Broadcast"
	CometServiceBroadcastTopic = "RPC.BroadcastTopic"
)

//type pushMsg struct {
//	ServerId	int32
//	UserId	[]int64
//	Msg		[]byte
//	Topic	int32
//	Ensure	bool
//}

type CometRpcClient struct {
	Client  *rpc.Client
	Quit    chan struct{}
	Network string
	Addr    string
}

func (c *CometRpcClient) Close() {
	c.Quit <- struct{}{}
	c.Client.Close()
}

type CometRpc struct {
	Config  ConfigComets
	Clients map[int32]*CometRpcClient

	PushChannels []chan *proto.MQMsg
}

func NewCometRpc() *CometRpc {
	rpc := new(CometRpc)
	return rpc
}

func (c *CometRpc) Init() (err error) {
	conf := Conf.Comets
	c.Clients = make(map[int32]*CometRpcClient)
	for _, comet := range conf.Comet {
		log.Infof("dial comet rpc (%s@%s)......", comet.Network, comet.Addr)
		client, err := rpc.Dial(comet.Network, comet.Addr)
		if err != nil {
			// dial later
			log.Infof("comet rpc first dial...")
		}
		quit := make(chan struct{}, 1)
		c.Clients[comet.Id] = &CometRpcClient{client, quit, comet.Network, comet.Addr}
		go c.Keepalive(comet.Id, quit, comet.Network, comet.Addr)
		log.Infof("connect to comet [%s@%s] successful.", comet.Network, comet.Addr)
	}

	c.PushChannels = make([]chan *proto.MQMsg, conf.Push.ChannelSize)
	for i := 0; i < len(c.PushChannels); i++ {
		c.PushChannels[i] = make(chan *proto.MQMsg, conf.Push.ChannelBufferSize)
		go c.processPushMsg(c.PushChannels[i])
	}
	c.Config = conf
	return nil
}

func (c *CometRpc) UnInit() {
	for i, cli := range c.Clients {
		cli.Close()
		delete(c.Clients, i)
	}
}

// update configuration.
// @TODO failback, if init failed, keep original cometRpc available
func (c *CometRpc) UpdateConfig() {
	c.UnInit()
	err := c.Init()
	if err != nil {
		log.Fatalf("CometRpc.UpdateConfig failed : %v", err)

	}
}

// keep alive with comet servers
func (c *CometRpc) Keepalive(cometId int32, quit chan struct{}, network, address string) {
	var (
		client *CometRpcClient
		ok     bool
		err    error
		call   *rpc.Call
		done   = make(chan *rpc.Call, 1)
		args   = proto.NoREQ{}
		reply  = proto.NoRES{}
	)
	if client, ok = c.Clients[cometId]; !ok {
		log.Errorf("keepalive : rpc connect to comet(id : %d) failed", cometId)
		panic(ErrConnectComet)
	}

	for {
		select {
		case <-quit:
			return
		default:
			if client.Client != nil {
				call = <-client.Client.Go(CometServicePing, &args, &reply, done).Done
				if call.Error != nil {
					log.Errorf("rpc ping %s error(%v)", address, call.Error)
				}
			}

			if client.Client == nil || (call != nil && call.Error == rpc.ErrShutdown) {
				if client.Client, err = rpc.Dial(network, address); err == nil {
					// @TODO Dial other adminRpc server
				}
			}
		}
		time.Sleep(5 * time.Second)
	}
}

// get comet server client by server id
func (c *CometRpc) getCometByServerId(serverId int32) (*CometRpcClient, error) {
	if client, ok := c.Clients[serverId]; !ok || client == nil {
		return nil, ErrComet
	} else {
		return client, nil
	}
}

//  push
// does invalidate msg, if it is invalid, ignore it
func (c *CometRpc) PushMQ(msg []byte) (err error) {
	m := &proto.MQMsg{}
	if err = ffjson.Unmarshal(msg, m); err != nil {
		log.Errorf("json.Unmarshal(%s) error(%s)", msg, err)
		return
	}
	c.push(m)
	return
}

// @TODO : optimization, operate channel directly, timeout write to messagequeue
func (c *CometRpc) push(msg *proto.MQMsg) {
	idx := rand.Int() % c.Config.Push.ChannelSize
	c.PushChannels[idx] <- msg
}

func (c *CometRpc) processPushMsg(ch chan *proto.MQMsg) {

	for {
		msg := <-ch
		switch msg.OP {
		case define.MQ_MESSAGE:
			if len(msg.ServerId) == 1 {
				pmsg := &proto.PushSMsgREQ{msg.ServerId[0], msg.Msg}
				c.SPush(pmsg)
			}

		case define.MQ_MESSAGE_MULTI:
			if len(msg.ServerId) == 1 {
				pmsg := &proto.PushMMsgREQ{msg.ServerId[0], msg.UserId, msg.Msg}
				c.MPush(pmsg)
			}

		case define.MQ_MESSAGE_BROADCAST:
			pmsg := &proto.PushBroadcastREQ{msg.Msg}
			c.broadcast(pmsg)

		case define.MQ_MESSAGE_BROADCAST_TOPIC:
			pmsg := &proto.PushBroadcastTopicREQ{Topic : msg.Topic, ServerId : msg.ServerId, Msg : msg.Msg}
			c.broadcastTopic(pmsg)
		default:
			log.Errorf("unknown operation:%d", msg.OP)
		}
	}
}

func (c *CometRpc) SPush(msg *proto.PushSMsgREQ) {

	var (
		reply = &proto.NoRES{}
		cc    *CometRpcClient
		err   error
	)
	cc, err = c.getCometByServerId(msg.ServerId)

	if err != nil || cc == nil || cc.Client == nil {
		log.Errorf("getCometByServerId(%d) failed : %v", msg.ServerId, err)
		return
	}
	if err = cc.Client.Call(CometServiceSPushMsg, msg, reply); err != nil {
		log.Errorf("c.Call(%s, %v, reply) failed : %v", CometServiceSPushMsg, msg, err)
	}
}

func (c *CometRpc) MPush(msg *proto.PushMMsgREQ) {
	var (
		reply = &proto.NoRES{}
		cc    *CometRpcClient
		err   error
	)
	cc, err = c.getCometByServerId(msg.ServerId)
	if err != nil || cc == nil || cc.Client == nil {
		log.Errorf("getCometByServerId(%d) failed : %v", msg.ServerId, err)
		return
	}
	if err = cc.Client.Call(CometServiceMPushMsg, msg, reply); err != nil {
		log.Errorf("c.Call(%s, %v, reply) failed : %v", CometServiceMPushMsg, msg, err)
	}
}

// ### broadcast : not guarantee message have been send successful

// broadcast broadcast a message to all
func (c *CometRpc) broadcast(msg *proto.PushBroadcastREQ) {

	for serverId, cc := range c.Clients {
		if cc != nil {
			go c.broadcastComet(cc, msg)
		} else {
			log.Errorf("broadcast to server(%d) failed : rpc.client is nil", serverId)
		}
	}
}

// broadcastComet a message to specified comet
func (c *CometRpc) broadcastComet(cc *CometRpcClient, msg *proto.PushBroadcastREQ) (err error) {
	var reply = proto.NoRES{}
	if cc == nil || cc.Client == nil {
		log.Errorln("rpc client is nil")
		return
	}
	if err = cc.Client.Call(CometServiceBroadcast, msg, &reply); err != nil {
		log.Errorf("c.Call(%s, %v, reply) failed : %v", CometServiceBroadcast, msg, err)
	}
	return
}

// broadcast topic
func (c *CometRpc) broadcastTopic(msg *proto.PushBroadcastTopicREQ) {
	var (
		reply = &proto.NoRES{}
		cc    *CometRpcClient
		err   error
	)

	for _, server := range msg.ServerId {
		go func(server int32) {
			cc, err = c.getCometByServerId(server)
			if err != nil || cc == nil || cc.Client == nil {
				log.Errorf("getCometByServerId(%d) failed : %v", msg.ServerId, err)
				return
			}
			if err = cc.Client.Call(CometServiceBroadcastTopic, msg, &reply); err != nil {
				log.Errorf("c.Call(%s, %v, reply) failed : %v", CometServiceBroadcastTopic, msg, err)
			}
		}(server)
	}
}
