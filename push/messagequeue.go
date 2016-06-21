package main

import (
	"time"
	"errors"
	"github.com/nsqio/go-nsq"
	"github.com/Shopify/sarama"
	log "github.com/ikenchina/golog"
	"github.com/wvanbergen/kafka/consumergroup"
)
var (
	ErrUnknowMessageQueue = errors.New("unknow message queue!")
)
const (
	MQPushTopic = "PushTopic"
	NSQChannel = "PushTopicChannel1"
)

type MessageQueue interface {
	Init(Addrs []string) error
	Start() error
	UnInit()
}


func InitMQ() (MessageQueue,error) {
	conf := Conf.MQ
	var mq MessageQueue
	if conf.Type == "kafka" {
		mq = new(MessageQueueKafka)
	} else if conf.Type == "nsq" {
		mq = new(MessageQueueNSQ)
	} else {
		return nil,ErrUnknowMessageQueue
	}
	err := mq.Init(conf.Addrs)
	return mq, err
}


type MessageQueueNSQ struct {
	cs *nsq.Consumer
	addrs []string
}


func (mq *MessageQueueNSQ)Init(Addrs []string) (err error) {
	config := nsq.NewConfig()
	mq.cs, err = nsq.NewConsumer(MQPushTopic, NSQChannel, config)
	mq.addrs = Addrs
	if err == nil {
		log.Infof("init message queue successful(%v)\n", Addrs)
	} else {
		log.Errorf("init message queue failed(%v)\n", Addrs)
		return
	}
	err = mq.Start()
	if err == nil {
		log.Infoln("start message queue successful\n")
	} else {
		log.Errorln("start message queue failed\n")
	}
	return
}

// exit need to chck mq.cs is not nil
func (mq *MessageQueueNSQ)UnInit() {
	if mq.cs != nil {
		mq.cs.Stop()
	}
}
func (mq *MessageQueueNSQ) Start() error {
	log.Infoln("MessageQueueNSQ start")
	mq.cs.AddHandler(nsq.HandlerFunc(func(m *nsq.Message) error {
		if cometRPC == nil {
			return ErrComet
		}
		return cometRPC.PushMQ(m.Body)
		
	}))
	// connect and start
	err := mq.cs.ConnectToNSQLookupds(mq.addrs)
	return err
}



type MessageQueueKafka struct {
	cg  *consumergroup.ConsumerGroup
}
const (
	KAFKA_GROUP_NAME                   = "kafka_topic_push_group"
	OFFSETS_PROCESSING_TIMEOUT_SECONDS = 10 * time.Second
	OFFSETS_COMMIT_INTERVAL            = 10 * time.Second
)

func (mq *MessageQueueKafka) Init(Addrs []string) (err error) {
	
	config := consumergroup.NewConfig()
	config.Offsets.Initial = sarama.OffsetNewest
	config.Offsets.ProcessingTimeout = OFFSETS_PROCESSING_TIMEOUT_SECONDS
	config.Offsets.CommitInterval = OFFSETS_COMMIT_INTERVAL
	config.Zookeeper.Chroot = "/gopush"
	kafkaTopics := []string{MQPushTopic}
	mq.cg, err = consumergroup.JoinConsumerGroup(KAFKA_GROUP_NAME, kafkaTopics, Addrs, config)
	return
}

// exit need to chck mq.cg is not nil
func (mq *MessageQueueKafka) UnInit() {
	if mq.cg != nil {
		mq.cg.Close()
	}
}

func (mq *MessageQueueKafka) Start() error {
	go func() {
		for err := range mq.cg.Errors() {
			log.Errorf("consumer error(%v)", err)
		}
	}()
	go func() {
		for msg := range mq.cg.Messages() {
			log.Infof("deal with topic:%s, partitionId:%d, Offset:%d, Key:%s msg:%s", msg.Topic, msg.Partition, msg.Offset, msg.Key, msg.Value)
			cometRPC.PushMQ(msg.Value)
			mq.cg.CommitUpto(msg)
		}
	}()
	return nil
}

