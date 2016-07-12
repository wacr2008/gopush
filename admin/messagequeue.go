package main

import (
	"errors"
	"github.com/nsqio/go-nsq"
	"github.com/pquerna/ffjson/ffjson"
	"github.com/Shopify/sarama"
	log "github.com/ikenchina/golog"
	"github.com/ikenchina/gopush/libs/define"
	"github.com/ikenchina/gopush/libs/proto"
)

var (
	ErrUnknowMessageQueue = errors.New("unknow message queue!")
)
const (
	MQPushTopic = "PushTopic"
)
type MessageQueue interface {
	Init(Addrs []string) error
	UnInit()
	Push(serverId int32, msg *proto.PushMsg) error
	
	// push multi msg to one user
	// @todo PushMMsg(serverid int32, userid int64, msgs []*proto.PushMsg) error
	
	MPush(serverId int32, userIds []int64, msg *proto.PushMsg) error
	Broadcast(msg *proto.PushMsg) error
	BroadcastTopic(serverId []int32, topic int32, msg *proto.PushMsg) error
}


func InitMQ() (MessageQueue,error) {
	conf := Conf.MQ
	//log.Infoln(conf)
	var mq MessageQueue
	if conf.Type == "kafka" {
		mq = new(MessageQueueKafka)
	} else if conf.Type == "nsq" {
		mq = new(MessageQueueNSQ)
	} else {
		return nil, ErrUnknowMessageQueue
	}
	err := mq.Init(conf.Addrs)
	return mq, err
}



type MessageQueueNSQ struct {
	producers []*nsq.Producer
}


func (mq *MessageQueueNSQ)Init(Addrs []string) error {
	var reterr error
	for _, addr := range Addrs {
		config := nsq.NewConfig()
		producer, err := nsq.NewProducer(addr, config)
		if err == nil {
			mq.producers = append(mq.producers, producer)
			log.Infof("init MessageQueueNSQ successful(%s)\n", addr)
		} else {
			reterr = err
			log.Errorf("init MessageQueueNSQ failed(%s) : %v\n", addr, err)
		}
	}
	
	leastOneOk := false
	for _, p := range mq.producers {
		if err := p.Ping(); err != nil {
			log.Errorf("ping message queue(%s) failed : %v", p.String(), err)
			reterr = err
		} else {
			leastOneOk = true
		}
	}
	if leastOneOk {
		reterr = nil
	}
	
	return reterr
}

func (mq *MessageQueueNSQ)UnInit() {
	for _,pr := range mq.producers {
		pr.Stop()
	}
	mq.producers = nil
}

func (mq *MessageQueueNSQ) push(msg *proto.MQMsg) (err error) {
	var data []byte
	if data, err = ffjson.Marshal(msg); err != nil {
		return
	}

	for _, pr := range mq.producers {
		err = pr.Publish(MQPushTopic, data)
		if err == nil {
			return
		}	
	}
	return
}


func (mq *MessageQueueNSQ) Push(sid int32, msg *proto.PushMsg) (error) {
	v := &proto.MQMsg{OP: define.MQ_MESSAGE, ServerId: []int32{sid}, Msg : msg}
	return mq.push(v)
}
func (mq *MessageQueueNSQ) MPush (serverId int32, userId []int64, msg *proto.PushMsg) (error) {
	v  := &proto.MQMsg{OP: define.MQ_MESSAGE_MULTI, ServerId: []int32{serverId}, UserId: userId, Msg: msg}
	return mq.push(v)
}
func (mq *MessageQueueNSQ) Broadcast (msg *proto.PushMsg) (error) {
	v  := &proto.MQMsg{OP: define.MQ_MESSAGE_BROADCAST, Msg: msg}
	return mq.push(v)
}
func (mq *MessageQueueNSQ) BroadcastTopic (serverId []int32, topic int32, msg *proto.PushMsg) (error) {
	v  := &proto.MQMsg{OP: define.MQ_MESSAGE_BROADCAST_TOPIC, ServerId: serverId, Topic : topic, Msg: msg}
	return mq.push(v)
}



type MessageQueueKafka struct {
	producer sarama.AsyncProducer
}

func (mq *MessageQueueKafka)Init(kafkaAddrs []string) (err error) {
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.NoResponse
	config.Producer.Partitioner = sarama.NewHashPartitioner
	config.Producer.Return.Successes = true
	config.Producer.Return.Errors = true
	mq.producer, err = sarama.NewAsyncProducer(kafkaAddrs, config)
	go mq.handleSuccess()
	go mq.handleError()
	return
}

func (mq *MessageQueueKafka)UnInit() {
	err := mq.producer.Close()
	if err != nil {
		log.Errorln(err)
	}
}

func (mq *MessageQueueKafka)handleSuccess() {
	var pm *sarama.ProducerMessage
	
	for {
		pm = <-mq.producer.Successes()
		if pm != nil {
			log.Infof("producer message success, partition:%d offset:%d key:%v valus:%s", pm.Partition, pm.Offset, pm.Key, pm.Value)
		}
	}
}

func (mq *MessageQueueKafka)handleError() {
	var err *sarama.ProducerError
	
	for {
		err = <-mq.producer.Errors()
		if err != nil {
			log.Errorf("producer message error, partition:%d offset:%d key:%v valus:%s failed(%v)", err.Msg.Partition, err.Msg.Offset, err.Msg.Key, err.Msg.Value, err.Err)
		}
	}
}

func (mq *MessageQueueKafka) push(msg *proto.MQMsg) (err error) {
	var data []byte
	if data, err = ffjson.Marshal(msg); err != nil {
		return
	}
	mq.producer.Input() <- &sarama.ProducerMessage{Topic: MQPushTopic, Value: sarama.ByteEncoder(data)}
	return err
}

func (mq *MessageQueueKafka) Push(sid int32, msg *proto.PushMsg) (error) {
	v := &proto.MQMsg{OP: define.MQ_MESSAGE, ServerId: []int32{sid}, Msg : msg}
	return mq.push(v)
}

func (mq *MessageQueueKafka) MPush (serverId int32, userId []int64, msg *proto.PushMsg) (error) {
	v  := &proto.MQMsg{OP: define.MQ_MESSAGE_MULTI, ServerId: []int32{serverId}, UserId: userId, Msg: msg}
	return mq.push(v)
}

func (mq *MessageQueueKafka) Broadcast(msg *proto.PushMsg) (error) {
	v  := &proto.MQMsg{OP: define.MQ_MESSAGE_BROADCAST, Msg: msg}
	return mq.push(v)
}

func (mq *MessageQueueKafka) BroadcastTopic(serverId []int32, topic int32, msg *proto.PushMsg) (error) {
	v  := &proto.MQMsg{OP: define.MQ_MESSAGE_BROADCAST_TOPIC, ServerId: serverId, Topic : topic, Msg: msg}
	return mq.push(v)
}




